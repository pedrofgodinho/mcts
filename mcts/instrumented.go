package mcts

import (
	"math/rand/v2"
	"time"

	"github.com/pedrofgodinho/mcts/game"
)

// EvaluatorMetrics summarises the work done by an [InstrumentedEvaluator].
//
// Rates and averages can be derived from the raw fields:
//
//	avg per call = TotalDuration / Calls
//	calls/sec    = Calls / TotalDuration
type EvaluatorMetrics struct {
	// Calls is the number of times Evaluate was invoked.
	Calls int
	// TotalDuration is the cumulative wall time spent inside the inner
	// evaluator's Evaluate method.
	TotalDuration time.Duration
}

// InstrumentedEvaluator wraps another Evaluator and records how many times
// it was called and how much wall time was spent inside it. It is intended
// for benchmarking and profiling: pair it with a Search call to measure
// what fraction of search time is spent in the evaluator versus everywhere
// else in MCTS (selection, expansion, backpropagation, tree bookkeeping).
//
// InstrumentedEvaluator is NOT safe for concurrent use. If MCTS is run in
// parallel, the underlying counter writes will race; callers should use a
// per-worker instrumented evaluator and aggregate the results, or swap the
// counter fields for atomics.
type InstrumentedEvaluator[S game.GameState[S, M], M comparable] struct {
	Inner   Evaluator[S, M]
	metrics EvaluatorMetrics
}

// NewInstrumentedEvaluator wraps inner with metric collection.
func NewInstrumentedEvaluator[S game.GameState[S, M], M comparable](inner Evaluator[S, M]) *InstrumentedEvaluator[S, M] {
	return &InstrumentedEvaluator[S, M]{Inner: inner}
}

// Evaluate delegates to the wrapped evaluator and records timing.
func (e *InstrumentedEvaluator[S, M]) Evaluate(state S, rng *rand.Rand, buf []M) (Evaluation[M], []M) {
	start := time.Now()
	eval, buf := e.Inner.Evaluate(state, rng, buf)
	e.metrics.Calls++
	e.metrics.TotalDuration += time.Since(start)
	return eval, buf
}

// Metrics returns a snapshot of the counters collected so far.
func (e *InstrumentedEvaluator[S, M]) Metrics() EvaluatorMetrics {
	return e.metrics
}

// Reset zeroes the counters. Useful between Search calls to measure each
// search in isolation.
func (e *InstrumentedEvaluator[S, M]) Reset() {
	e.metrics = EvaluatorMetrics{}
}
