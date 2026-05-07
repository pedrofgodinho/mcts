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
	Iterations  int           // 0 means run until time budget is exhausted
	Budget      time.Duration // 0 means run until iteration budget is exhausted
	Rand        *rand.Rand    // nil means a fresh PCG seeded from time
	VirtualLoss int           // Number of virtual loss visits to apply during selection. 0 means no virtual loss. 1-3 is a common range to try
	Workers     int           // Number of concurrent workers to use. 0 or 1 means sequential execution. >1 means parallel MCTS with virtual loss.
}

type node[S game.GameState[S, M], M comparable] struct {
	state  S
	parent *node[S, M]
	move   M

	visits   atomic.Int64
	valueSum atomic.Uint64

	mu           sync.RWMutex
	children     []*node[S, M]
	untriedMoves []M
}

func newNode[S game.GameState[S, M], M comparable](state S, parent *node[S, M], move M) *node[S, M] {
	return &node[S, M]{state: state, parent: parent, move: move, untriedMoves: state.LegalMoves(nil)}
}

func (n *node[S, M]) addValue(v float64) {
	for {
		old := n.valueSum.Load()
		newBits := math.Float64bits(math.Float64frombits(old) + v)
		if n.valueSum.CompareAndSwap(old, newBits) {
			break
		}
	}
}

func (n *node[S, M]) loadValue() float64 { return math.Float64frombits(n.valueSum.Load()) }

func (n *node[S, M]) applyVirtualLoss(vl int, parentPerspective float64) {
	n.visits.Add(int64(vl))
	n.addValue(-float64(vl) * parentPerspective)
}

func (n *node[S, M]) revertVirtualLoss(vl int, parentPerspective float64) {
	n.visits.Add(int64(-vl))
	n.addValue(float64(vl) * parentPerspective)
}

type Agent[S game.GameState[S, M], M comparable] struct {
	root      *node[S, M]
	evaluator Evaluator[S, M]
}

func NewAgent[S game.GameState[S, M], M comparable](initialState S, evaluator Evaluator[S, M]) *Agent[S, M] {
	return &Agent[S, M]{root: newNode(initialState, nil, *new(M)), evaluator: evaluator}
}

func (a *Agent[S, M]) State() S { return a.root.state }

func (a *Agent[S, M]) iterate(rng *rand.Rand, rolloutBuf []M, vl int) []M {
	path := selectAndExpand(a.root, rng, vl)
	leaf := path[len(path)-1]
	var result float64
	if leaf.state.IsTerminal() {
		result = leaf.state.Result()
	} else {
		var eval Evaluation[M]
		eval, rolloutBuf = a.evaluator.Evaluate(leaf.state, rng, rolloutBuf)
		result = eval.Value
	}
	backpropagatePath(path, result, vl)
	return rolloutBuf
}

func selectAndExpand[S game.GameState[S, M], M comparable](root *node[S, M], rng *rand.Rand, vl int) []*node[S, M] {
	path := []*node[S, M]{root}
	n := root
	for {
		n.mu.RLock()
		canDescend := len(n.untriedMoves) == 0 && len(n.children) > 0
		var next *node[S, M]
		if canDescend {
			next = bestUCBChildLocked(n)
		}
		parentPerspective := float64(n.state.CurrentPlayer())
		n.mu.RUnlock()
		if !canDescend {
			child := expand(n, rng, vl, parentPerspective)
			if child != n {
				path = append(path, child)
			}
			return path
		}
		next.applyVirtualLoss(vl, parentPerspective)
		path = append(path, next)
		n = next
	}
}

func backpropagatePath[S game.GameState[S, M], M comparable](path []*node[S, M], result float64, vl int) {
	for i := 1; i < len(path); i++ {
		parent := path[i-1]
		parentPerspective := float64(parent.state.CurrentPlayer())
		path[i].revertVirtualLoss(vl, parentPerspective)
	}
	for _, n := range path {
		n.visits.Add(1)
		n.addValue(result)
	}
}

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

	workers := max(opts.Workers, 1)

	workerRngs := make([]*rand.Rand, workers)
	for i := range workerRngs {
		workerRngs[i] = rand.New(rand.NewPCG(rng.Uint64(), rng.Uint64()))
	}

	deadline := time.Time{}
	if opts.Budget > 0 {
		deadline = time.Now().Add(opts.Budget)
	}

	var iters atomic.Int64
	var wg sync.WaitGroup
	start := time.Now()

	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func(workerRng *rand.Rand) {
			defer wg.Done()
			a.workerLoop(workerRng, opts, deadline, &iters)
		}(workerRngs[i])
	}
	wg.Wait()

	a.root.mu.RLock()
	best := mostVisitedChildLocked(a.root)
	bestMove := best.move
	a.root.mu.RUnlock()
	return bestMove, a.collectStats(int(iters.Load()), time.Since(start))
}

func (a *Agent[S, M]) workerLoop(rng *rand.Rand, opts SearchOptions, deadline time.Time, iters *atomic.Int64) {
	var rolloutBuf []M
	for {
		if !deadline.IsZero() && time.Now().After(deadline) {
			return
		}
		n := iters.Add(1)
		if opts.Iterations > 0 && n > int64(opts.Iterations) {
			iters.Add(-1)
			return
		}
		rolloutBuf = a.iterate(rng, rolloutBuf, opts.VirtualLoss)
	}
}

func (a *Agent[S, M]) collectStats(iterations int, duration time.Duration) SearchStats[M] {
	perspective := float64(a.root.state.CurrentPlayer())
	a.root.mu.RLock()
	children := make([]ChildStats[M], len(a.root.children))
	for i, c := range a.root.children {
		visits := c.visits.Load()
		v := (c.loadValue() * perspective) / float64(visits)
		children[i] = ChildStats[M]{Move: c.move, Visits: visits, WinRate: (v + 1) / 2}
	}
	rootVisits := a.root.visits.Load()
	a.root.mu.RUnlock()
	sort.Slice(children, func(i, j int) bool { return children[i].Visits > children[j].Visits })
	return SearchStats[M]{
		Iterations: iterations, Duration: duration, RootVisits: rootVisits,
		Children: children, PrincipalVariation: collectPV(a.root),
	}
}

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
		perspective := float64(n.state.CurrentPlayer())
		visits := best.visits.Load()
		v := (best.loadValue() * perspective) / float64(visits)
		n.mu.RUnlock()
		pv = append(pv, PVStep[M]{Move: best.move, Visits: visits, WinRate: (v + 1) / 2})
		n = best
	}
}

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

// expand pops one untried move from n and adds the resulting child to n.children.
// VL is applied to the new child while still holding the write lock, so other
// workers cannot see a child with visits==0. Returns n itself if there are no
// untried moves to expand (e.g. terminal state, or another worker already
// drained the moves under tree parallelism).
func expand[S game.GameState[S, M], M comparable](n *node[S, M], rng *rand.Rand, vl int, parentPerspective float64) *node[S, M] {
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
	child.applyVirtualLoss(vl, parentPerspective)
	n.children = append(n.children, child)
	return child
}

// bestUCBChildLocked returns the child with the highest UCB score.
// Unvisited children (visits==0) are treated as having infinite score, so
// they are always selected before any visited child. This both matches
// classical UCT semantics ("explore everything once first") and defends
// against the brief window during expansion where a child may exist in
// n.children before its first visit count has been written (only possible
// when VirtualLoss is 0, since otherwise the apply happens under the
// expand lock).
//
// The caller must hold n.mu.
func bestUCBChildLocked[S game.GameState[S, M], M comparable](n *node[S, M]) *node[S, M] {
	var best *node[S, M]
	bestScore := math.Inf(-1)
	parentVisitsLog := math.Log(float64(n.visits.Load()))
	perspective := float64(n.state.CurrentPlayer())
	for _, child := range n.children {
		v := child.visits.Load()
		var score float64
		if v == 0 {
			score = math.Inf(+1)
		} else {
			exploitation := (child.loadValue() * perspective) / float64(v)
			exploration := explorationConstant * math.Sqrt(parentVisitsLog/float64(v))
			score = exploitation + exploration
		}
		if score > bestScore {
			bestScore = score
			best = child
		}
	}
	return best
}

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
