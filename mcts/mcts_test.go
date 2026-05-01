package mcts

import (
	"testing"

	"github.com/pedrofgodinho/mcts/games/tictactoe"
)

func TestMCTSvsMCTSAlwaysDraws(t *testing.T) {
	for trial := range 5 {
		g := tictactoe.New()
		for !g.IsTerminal() {
			move := Search(g, 5000)
			g = g.Apply(move)
		}
		if g.Result() != 0 {
			t.Errorf("trial %d: expected draw, got result %v", trial, g.Result())
		}
	}
}
