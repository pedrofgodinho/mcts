package mcts

import (
	"math"
	"math/rand/v2"
	"sort"
	"sync"
	"sync/atomic"
	"time"

	"github.com/pedrofgodinho/mcts/game"
)

var explorationConstant = math.Sqrt(2)

type SearchOptions struct {
	Iterations int           // 0 means run until time budget is exhausted
	Budget     time.Duration // 0 means run until iteration budget is exhausted
	Rand       *rand.Rand    // nil means a fresh PCG seeded from time
}

type node[S game.GameState[S, M], M comparable] struct {
	// Immutable after newNode.
	state  S
	parent *node[S, M]
	move   M

	// Statistics, atomic. Read during selection, written during backpropagation.
	visits atomic.Int64
	// valueSum is float64 but stored as uint64 using math.Float64bits to allow atomic updates.
	valueSum atomic.Uint64

	// Tree structure, protected by mu. children and untriedMoves are locked
	// together since they are modified together during expansion.
	mu           sync.RWMutex
	children     []*node[S, M]
	untriedMoves []M
}

func newNode[S game.GameState[S, M], M comparable](state S, parent *node[S, M], move M) *node[S, M] {
	// No locking needed since the new node is not yet reachable by other goroutines.
	return &node[S, M]{
		state:        state,
		parent:       parent,
		move:         move,
		untriedMoves: state.LegalMoves(nil),
	}
}

// addValue atomically adds v to valueSum using a CAS loop.
// Retries if another goroutine updates valueSum between Load and CompareAndSwap.
func (n *node[S, M]) addValue(v float64) {
	for {
		old := n.valueSum.Load()
		newBits := math.Float64bits(math.Float64frombits(old) + v)
		if n.valueSum.CompareAndSwap(old, newBits) {
			break
		}
	}
}

// loadValue atomically loads valueSum and converts it to float64.
func (n *node[S, M]) loadValue() float64 {
	return math.Float64frombits(n.valueSum.Load())
}

type Agent[S game.GameState[S, M], M comparable] struct {
	root      *node[S, M]
	evaluator Evaluator[S, M]
}

func NewAgent[S game.GameState[S, M], M comparable](initialState S, evaluator Evaluator[S, M]) *Agent[S, M] {
	return &Agent[S, M]{
		root:      newNode(initialState, nil, *new(M)),
		evaluator: evaluator,
	}
}

func (a *Agent[S, M]) State() S {
	return a.root.state
}

// Search runs MCTS under the given options and returns the most-visited move
// along with statistics about the search.
// It panics if the agent's current state is terminal.
func (a *Agent[S, M]) Search(opts SearchOptions) (M, SearchStats[M]) {
	if a.root.state.IsTerminal() {
		panic("mcts: Search called on terminal state")
	}
	if opts.Iterations <= 0 && opts.Budget <= 0 {
		panic("mcts: SearchOptions must set Iterations or Budget")
	}
	rng := opts.Rand
	if rng == nil {
		rng = newDefaultRand()
	}

	deadline := time.Time{}
	if opts.Budget > 0 {
		deadline = time.Now().Add(opts.Budget)
	}

	var rolloutBuf []M

	start := time.Now()
	iters := 0
	for opts.Iterations <= 0 || iters < opts.Iterations {
		if !deadline.IsZero() && time.Now().After(deadline) {
			break
		}
		leaf := selectLeaf(a.root)
		expanded := expand(leaf, rng)
		var result float64
		if expanded.state.IsTerminal() {
			result = expanded.state.Result()
		} else {
			var eval Evaluation[M]
			eval, rolloutBuf = a.evaluator.Evaluate(expanded.state, rng, rolloutBuf)
			result = eval.Value
		}
		backpropagate(expanded, result)
		iters++
	}

	a.root.mu.RLock()
	best := mostVisitedChildLocked(a.root)
	bestMove := best.move
	a.root.mu.RUnlock()
	return bestMove, a.collectStats(iters, time.Since(start))
}

// collectStats builds a SearchStats snapshot from the current root.
func (a *Agent[S, M]) collectStats(iterations int, duration time.Duration) SearchStats[M] {
	perspective := float64(a.root.state.CurrentPlayer())

	a.root.mu.RLock()
	children := make([]ChildStats[M], len(a.root.children))
	for i, c := range a.root.children {
		visits := c.visits.Load()
		v := (c.loadValue() * perspective) / float64(visits)
		children[i] = ChildStats[M]{
			Move:    c.move,
			Visits:  visits,
			WinRate: (v + 1) / 2,
		}
	}
	rootVisits := a.root.visits.Load()
	a.root.mu.RUnlock()

	sort.Slice(children, func(i, j int) bool {
		return children[i].Visits > children[j].Visits
	})
	return SearchStats[M]{
		Iterations:         iterations,
		Duration:           duration,
		RootVisits:         rootVisits,
		Children:           children,
		PrincipalVariation: collectPV(a.root),
	}
}

// collectPV walks from the given node, taking the most-visited child each step,
// and returns the sequence as PVSteps. The starting node itself is not included
// (the PV is the moves *from* the root, not the root state).
func collectPV[S game.GameState[S, M], M comparable](root *node[S, M]) []PVStep[M] {
	var pv []PVStep[M]
	n := root
	for {
		n.mu.RLock()
		var best *node[S, M]
		if len(n.children) > 0 {
			best = mostVisitedChildLocked(n)
		}
		if best == nil {
			n.mu.RUnlock()
			return pv
		}
		// WinRate from the perspective of the player to move at n
		// (i.e., the player choosing this move).
		perspective := float64(n.state.CurrentPlayer())
		visits := best.visits.Load()
		v := (best.loadValue() * perspective) / float64(visits)
		n.mu.RUnlock()

		pv = append(pv, PVStep[M]{
			Move:    best.move,
			Visits:  visits,
			WinRate: (v + 1) / 2,
		})
		n = best
	}
}

// Advance applies the given move to the current state and promotes the
// corresponding child (if it exists in the tree) to be the new root.
// If no child m exists, a fresh root is built from Apply(m).
//
// Advance must not be called concurrently with Search; callers must wait
// for any in-flight Search to return before calling Advance.
func (a *Agent[S, M]) Advance(m M) {
	for _, c := range a.root.children {
		if c.move == m {
			c.parent = nil
			a.root = c
			return
		}
	}
	newState := a.root.state.Apply(m)
	a.root = newNode(newState, nil, *new(M))
}

func newDefaultRand() *rand.Rand {
	now := uint64(time.Now().UnixNano())
	return rand.New(rand.NewPCG(now, now^0x9E3779B97F4A7C15))
}

func selectLeaf[S game.GameState[S, M], M comparable](n *node[S, M]) *node[S, M] {
	for {
		n.mu.RLock()
		canDescend := len(n.untriedMoves) == 0 && len(n.children) > 0
		var next *node[S, M]
		if canDescend {
			next = bestUCBChildLocked(n)
		}
		n.mu.RUnlock()
		if !canDescend {
			return n
		}
		n = next
	}
}

func expand[S game.GameState[S, M], M comparable](n *node[S, M], rng *rand.Rand) *node[S, M] {
	n.mu.Lock()
	defer n.mu.Unlock()
	if len(n.untriedMoves) == 0 {
		return n
	}
	idx := rng.IntN(len(n.untriedMoves))
	move := n.untriedMoves[idx]
	n.untriedMoves[idx] = n.untriedMoves[len(n.untriedMoves)-1]
	n.untriedMoves = n.untriedMoves[:len(n.untriedMoves)-1]

	childState := n.state.Apply(move)
	child := newNode(childState, n, move)
	n.children = append(n.children, child)
	return child
}

func backpropagate[S game.GameState[S, M], M comparable](n *node[S, M], result float64) {
	for n != nil {
		n.visits.Add(1)
		// visits and valueSum are not updated atomically together; a reader
		// can briefly see one updated and not the other. MCTS tolerates this
		// noise.
		n.addValue(result)
		n = n.parent
	}
}

// bestUCBChildLocked returns the child with the highest UCB score.
// The caller must hold n.mu.
func bestUCBChildLocked[S game.GameState[S, M], M comparable](n *node[S, M]) *node[S, M] {
	var best *node[S, M]
	bestScore := math.Inf(-1)
	parentVisitsLog := math.Log(float64(n.visits.Load()))
	perspective := float64(n.state.CurrentPlayer())

	for _, child := range n.children {
		v := child.visits.Load()
		exploitation := (child.loadValue() * perspective) / float64(v)
		exploration := explorationConstant * math.Sqrt(parentVisitsLog/float64(v))
		score := exploitation + exploration
		if score > bestScore {
			bestScore = score
			best = child
		}
	}
	return best
}

// mostVisitedChildLocked returns the child with the highest visit count.
// The caller must hold n.mu.
func mostVisitedChildLocked[S game.GameState[S, M], M comparable](n *node[S, M]) *node[S, M] {
	var best *node[S, M]
	var bestVisits int64 = -1
	for _, c := range n.children {
		v := c.visits.Load()
		if v > bestVisits {
			bestVisits = v
			best = c
		}
	}
	return best
}
