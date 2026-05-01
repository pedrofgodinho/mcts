package mcts

import (
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
