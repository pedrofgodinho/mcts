package mcts

import (
	"math"
	"math/rand/v2"
	"testing"

	"github.com/pedrofgodinho/mcts/games/tictactoe"
)

func TestMCTSvsMCTSAlwaysDraws(t *testing.T) {
	for trial := range 5 {
		evaluator := RandomRolloutEvaluator[tictactoe.Game, tictactoe.Move]{}
		agent := NewAgent[tictactoe.Game, tictactoe.Move](tictactoe.New(), evaluator)
		for !agent.State().IsTerminal() {
			seed := uint64(trial)*1000 + uint64(agent.State().LegalMoves(nil)[0])
			move, _ := agent.Search(SearchOptions{
				Iterations: 5000,
				Rand:       rand.New(rand.NewPCG(seed, seed^0x9E3779B97F4A7C15)),
			})
			agent.Advance(move)
		}
		if agent.State().Result() != 0 {
			t.Errorf("trial %d: expected draw, got result %v", trial, agent.State().Result())
		}
	}
}

func TestMCTSvsMCTSAlwaysDrawsWithVirtualLoss(t *testing.T) {
	for trial := range 5 {
		evaluator := RandomRolloutEvaluator[tictactoe.Game, tictactoe.Move]{}
		agent := NewAgent[tictactoe.Game, tictactoe.Move](tictactoe.New(), evaluator)
		for !agent.State().IsTerminal() {
			seed := uint64(trial)*1000 + uint64(agent.State().LegalMoves(nil)[0])
			move, _ := agent.Search(SearchOptions{
				Iterations:  5000,
				Rand:        rand.New(rand.NewPCG(seed, seed^0x9E3779B97F4A7C15)),
				VirtualLoss: 2,
			})
			agent.Advance(move)
		}
		if agent.State().Result() != 0 {
			t.Errorf("trial %d: expected draw, got result %v", trial, agent.State().Result())
		}
	}
}

func TestVirtualLossRoundTrip(t *testing.T) {
	var n node[tictactoe.Game, tictactoe.Move]
	n.addValue(0.3)
	n.visits.Add(5)

	n.applyVirtualLoss(3, +1) // Player1 perspective
	if got := n.visits.Load(); got != 8 {
		t.Errorf("after apply: visits = %d, want 8", got)
	}
	if got := n.loadValue(); math.Abs(got-(-2.7)) > 1e-9 {
		t.Errorf("after apply: valueSum = %v, want -2.7", got)
	}

	n.revertVirtualLoss(3, +1)
	if got := n.visits.Load(); got != 5 {
		t.Errorf("after revert: visits = %d, want 5", got)
	}
	if got := n.loadValue(); math.Abs(got-0.3) > 1e-9 {
		t.Errorf("after revert: valueSum = %v, want 0.3", got)
	}
}
