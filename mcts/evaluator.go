package mcts

import (
	"math/rand/v2"

	"github.com/pedrofgodinho/mcts/game"
)

// Evaluation is the result of evaluating a non-terminal position.
type Evaluation[M comparable] struct {
	// Value estimates the game's outcome from this position, in the same
	// convention as GameState.Result(): +1 for Player1 win, -1 for Player2,
	// 0 for draw. Estimators may return any value in [-1, +1].
	Value float64
	// Policy is an optional prior distribution over the legal moves at
	// this position. If non-nil, MCTS uses it to bias selection.
	// Probabilities should sum to 1 across the legal moves; absent moves
	// are treated as having probability 0.
	// Nil means "uniform prior" — MCTS uses plain UCB1.
	Policy map[M]float64
}

// Evaluator estimates the value of a position (and optionally a policy
// over moves) for use as a leaf evaluation in MCTS.
//
// Evaluators must be safe for concurrent use if MCTS is run in parallel.
// (Single-threaded MCTS makes no concurrency requirement.)
type Evaluator[S game.GameState[S, M], M comparable] interface {
	// Evaluate takes a game state and returns an evaluation of that state.
	// A buffer is provided for the evaluator to use when calling LegalMoves;
	// it may be reused across calls, but the evaluator must not retain references to it after returning.
	// Evaluate is called only on non-terminal states. MCTS handles terminal states directly by using GameState.Result()
	Evaluate(state S, rng *rand.Rand, buf []M) (Evaluation[M], []M)
}

// RandomRolloutEvaluator plays uniformly random moves from the given state
// until termination, then returns the terminal result as the value.
// Policy is always nil (no prior).
//
// This evaluator is purely a function of its input — it holds no state —
// and is safe for concurrent use.
type RandomRolloutEvaluator[S game.GameState[S, M], M comparable] struct{}

func (RandomRolloutEvaluator[S, M]) Evaluate(state S, rng *rand.Rand, buf []M) (Evaluation[M], []M) {
	for !state.IsTerminal() {
		buf = state.LegalMoves(buf)
		state = state.Apply(buf[rng.IntN(len(buf))])
	}
	return Evaluation[M]{Value: state.Result()}, buf
}
