package mcts

import (
	"math"
	"math/rand"

	"github.com/pedrofgodinho/mcts/game"
)

var explorationConstant = math.Sqrt(2)

type node[S game.GameState[S, M], M any] struct {
	state        S
	parent       *node[S, M]
	move         M
	children     []*node[S, M]
	untriedMoves []M
	visits       int
	valueSum     float64
}

func newNode[S game.GameState[S, M], M any](state S, parent *node[S, M], move M) *node[S, M] {
	return &node[S, M]{
		state:        state,
		parent:       parent,
		move:         move,
		untriedMoves: state.LegalMoves(),
	}
}

// Search performs the Monte Carlo Tree Search algorithm starting from the given root game state and running for the specified number of iterations.
// It returns the move that was most visited during the search, which is considered the best move to play from the root state.
func Search[S game.GameState[S, M], M any](root S, iterations int) M {
	rootNode := newNode[S, M](root, nil, *new(M))

	for range iterations {
		leaf := selectLeaf(rootNode)
		expanded := expand(leaf)
		result := simulate(expanded.state)
		backpropagate(expanded, result)
	}

	return mostVisitedChild(rootNode).move
}

func selectLeaf[S game.GameState[S, M], M any](n *node[S, M]) *node[S, M] {
	for len(n.untriedMoves) == 0 && len(n.children) > 0 {
		n = bestUCBChild(n)
	}
	return n
}

func expand[S game.GameState[S, M], M any](n *node[S, M]) *node[S, M] {
	if len(n.untriedMoves) == 0 {
		return n
	}

	// Selecting random move rather than first to avoid move gen order bias
	idx := rand.Intn(len(n.untriedMoves))
	move := n.untriedMoves[idx]
	n.untriedMoves[idx] = n.untriedMoves[len(n.untriedMoves)-1]
	n.untriedMoves = n.untriedMoves[:len(n.untriedMoves)-1]

	childState := n.state.Apply(move)
	child := newNode(childState, n, move)
	n.children = append(n.children, child)
	return child
}

func simulate[S game.GameState[S, M], M any](state S) float64 {
	for !state.IsTerminal() {
		moves := state.LegalMoves()
		state = state.Apply(moves[rand.Intn(len(moves))])
	}
	return state.Result()
}

func backpropagate[S game.GameState[S, M], M any](n *node[S, M], result float64) {
	for n != nil {
		n.visits++
		n.valueSum += result
		n = n.parent
	}
}

func bestUCBChild[S game.GameState[S, M], M any](n *node[S, M]) *node[S, M] {
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

func mostVisitedChild[S game.GameState[S, M], M any](n *node[S, M]) *node[S, M] {
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
