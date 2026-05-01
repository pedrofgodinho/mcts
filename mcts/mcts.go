package mcts

import (
	"math"
	"math/rand/v2"
	"sort"
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
	state        S
	parent       *node[S, M]
	move         M
	children     []*node[S, M]
	untriedMoves []M
	visits       int
	valueSum     float64
}

func newNode[S game.GameState[S, M], M comparable](state S, parent *node[S, M], move M) *node[S, M] {
	return &node[S, M]{
		state:        state,
		parent:       parent,
		move:         move,
		untriedMoves: state.LegalMoves(nil),
	}
}

type Agent[S game.GameState[S, M], M comparable] struct {
	root *node[S, M]
}

func NewAgent[S game.GameState[S, M], M comparable](initialState S) *Agent[S, M] {
	return &Agent[S, M]{
		root: newNode(initialState, nil, *new(M)),
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
		result, rolloutBuf = simulate(expanded.state, rng, rolloutBuf)
		backpropagate(expanded, result)
		iters++
	}

	best := mostVisitedChild(a.root)
	return best.move, a.collectStats(iters, time.Since(start))
}

// collectStats builds a SearchStats snapshot from the current root.
func (a *Agent[S, M]) collectStats(iterations int, duration time.Duration) SearchStats[M] {
	perspective := float64(a.root.state.CurrentPlayer())
	children := make([]ChildStats[M], len(a.root.children))
	for i, c := range a.root.children {
		v := (c.valueSum * perspective) / float64(c.visits)
		children[i] = ChildStats[M]{
			Move:    c.move,
			Visits:  c.visits,
			WinRate: (v + 1) / 2,
		}
	}
	sort.Slice(children, func(i, j int) bool {
		return children[i].Visits > children[j].Visits
	})
	return SearchStats[M]{
		Iterations:         iterations,
		Duration:           duration,
		RootVisits:         a.root.visits,
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
	for len(n.children) > 0 {
		best := mostVisitedChild(n)
		// WinRate from the perspective of the player to move at n
		// (i.e., the player choosing this move).
		perspective := float64(n.state.CurrentPlayer())
		v := (best.valueSum * perspective) / float64(best.visits)
		pv = append(pv, PVStep[M]{
			Move:    best.move,
			Visits:  best.visits,
			WinRate: (v + 1) / 2,
		})
		n = best
	}
	return pv
}

// Advance applies the given move to the current state and promotes the
// corresponding child (if it exists in the tree) to be the new root.
// If no child m exists, a fresh root is built from Apply(m).
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
	for len(n.untriedMoves) == 0 && len(n.children) > 0 {
		n = bestUCBChild(n)
	}
	return n
}

func expand[S game.GameState[S, M], M comparable](n *node[S, M], rng *rand.Rand) *node[S, M] {
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

func simulate[S game.GameState[S, M], M comparable](state S, rng *rand.Rand, buf []M) (float64, []M) {
	for !state.IsTerminal() {
		buf = state.LegalMoves(buf)
		state = state.Apply(buf[rng.IntN(len(buf))])
	}
	return state.Result(), buf
}

func backpropagate[S game.GameState[S, M], M comparable](n *node[S, M], result float64) {
	for n != nil {
		n.visits++
		n.valueSum += result
		n = n.parent
	}
}

func bestUCBChild[S game.GameState[S, M], M comparable](n *node[S, M]) *node[S, M] {
	var best *node[S, M]
	bestScore := math.Inf(-1)
	parentVisitsLog := math.Log(float64(n.visits))
	perspective := float64(n.state.CurrentPlayer())

	for _, child := range n.children {
		exploitation := (child.valueSum * perspective) / float64(child.visits)
		exploration := explorationConstant * math.Sqrt(parentVisitsLog/float64(child.visits))
		score := exploitation + exploration
		if score > bestScore {
			bestScore = score
			best = child
		}
	}
	return best
}

func mostVisitedChild[S game.GameState[S, M], M comparable](n *node[S, M]) *node[S, M] {
	var best *node[S, M]
	bestVisits := -1
	for _, c := range n.children {
		if c.visits > bestVisits {
			bestVisits = c.visits
			best = c
		}
	}
	return best
}
