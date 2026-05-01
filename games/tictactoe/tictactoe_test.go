package tictactoe

import (
	"testing"

	"github.com/pedrofgodinho/mcts/game"
)

func TestNewGame(t *testing.T) {
	g := New()
	if g.CurrentPlayer() != game.Player1 {
		t.Errorf("expected Player1 to go first, got %v", g.CurrentPlayer())
	}
	if g.IsTerminal() {
		t.Error("new game should not be terminal")
	}
	if g.Winner() != game.None {
		t.Errorf("new game should have no winner, got %v", g.Winner())
	}
	moves := g.LegalMoves(nil)
	if len(moves) != 9 {
		t.Errorf("expected 9 legal moves on new board, got %d", len(moves))
	}
}

func TestTurnAlternates(t *testing.T) {
	g := New()
	if g.CurrentPlayer() != game.Player1 {
		t.Errorf("expected Player1's turn, got %v", g.CurrentPlayer())
	}
	g = g.Apply(0)
	if g.CurrentPlayer() != game.Player2 {
		t.Errorf("expected Player2's turn after move, got %v", g.CurrentPlayer())
	}
	g = g.Apply(1)
	if g.CurrentPlayer() != game.Player1 {
		t.Errorf("expected Player1's turn after two moves, got %v", g.CurrentPlayer())
	}
}

func TestApplyReducesLegalMoves(t *testing.T) {
	g := New()
	g = g.Apply(4)
	moves := g.LegalMoves(nil)
	if len(moves) != 8 {
		t.Errorf("expected 8 legal moves after one move, got %d", len(moves))
	}
	for _, m := range moves {
		if m == 4 {
			t.Error("square 4 should not be a legal move after being played")
		}
	}
}

func TestApplyIsNonMutating(t *testing.T) {
	g := New()
	g.Apply(0)
	if g.CurrentPlayer() != game.Player1 {
		t.Error("Apply should not mutate the receiver")
	}
	moves := g.LegalMoves(nil)
	if len(moves) != 9 {
		t.Error("Apply should not mutate the receiver's board")
	}
}

func TestLegalMovesEmptyOnTerminal(t *testing.T) {
	g := New()
	for _, m := range []Move{0, 3, 1, 4, 2} {
		g = g.Apply(m)
	}
	if g.LegalMoves(nil) != nil {
		t.Error("terminal game should return nil legal moves")
	}
}

var winTests = []struct {
	name   string
	moves  []Move
	winner game.Player
}{
	// All rows
	{"player1 wins row 0", []Move{0, 3, 1, 4, 2}, game.Player1},
	{"player1 wins row 1", []Move{3, 0, 4, 1, 5}, game.Player1},
	{"player1 wins row 2", []Move{6, 0, 7, 1, 8}, game.Player1},
	// All columns
	{"player1 wins col 0", []Move{0, 1, 3, 2, 6}, game.Player1},
	{"player1 wins col 1", []Move{1, 0, 4, 2, 7}, game.Player1},
	{"player1 wins col 2", []Move{2, 0, 5, 1, 8}, game.Player1},
	// All diagonals
	{"player1 wins diag main", []Move{0, 1, 4, 2, 8}, game.Player1},
	{"player1 wins diag anti", []Move{2, 0, 4, 1, 6}, game.Player1},
	// Player2 wins
	{"player2 wins row 0", []Move{3, 0, 4, 1, 8, 2}, game.Player2},
	{"player2 wins col 0", []Move{1, 0, 2, 3, 8, 6}, game.Player2},
	{"player2 wins diag main", []Move{1, 0, 2, 4, 3, 8}, game.Player2},
}

func TestWinLines(t *testing.T) {
	for _, tt := range winTests {
		t.Run(tt.name, func(t *testing.T) {
			g := New()
			for _, m := range tt.moves {
				g = g.Apply(m)
			}
			if !g.IsTerminal() {
				t.Fatal("game should be terminal")
			}
			if g.Winner() != tt.winner {
				t.Errorf("expected winner %v, got %v", tt.winner, g.Winner())
			}
			if g.Result() != float64(tt.winner) {
				t.Errorf("expected result %v, got %v", float64(tt.winner), g.Result())
			}
		})
	}
}

func TestDraw(t *testing.T) {
	g := New()
	// X O X / X O O / O X X
	for _, m := range []Move{4, 0, 8, 2, 1, 7, 3, 5, 6} {
		g = g.Apply(m)
	}
	if !g.IsTerminal() {
		t.Fatal("game should be terminal")
	}
	if g.Winner() != game.None {
		t.Errorf("expected no winner, got %v", g.Winner())
	}
	if g.Result() != 0 {
		t.Errorf("expected result 0 for draw, got %v", g.Result())
	}
}

func TestResultNonTerminal(t *testing.T) {
	g := New()
	if g.Result() != 0 {
		t.Errorf("expected result 0 for non-terminal game, got %v", g.Result())
	}
}
