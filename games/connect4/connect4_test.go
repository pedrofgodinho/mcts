package connect4

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
	if len(moves) != Cols {
		t.Errorf("expected %d legal moves on new board, got %d", Cols, len(moves))
	}
	for c := range Cols {
		if g.Height(c) != 0 {
			t.Errorf("expected column %d to have height 0, got %d", c, g.Height(c))
		}
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

func TestApplyStacksInColumn(t *testing.T) {
	g := New()
	g = g.Apply(3)
	g = g.Apply(3)
	g = g.Apply(3)
	if g.Height(3) != 3 {
		t.Errorf("expected column 3 height 3, got %d", g.Height(3))
	}
	if g.At(3, 0) != game.Player1 {
		t.Errorf("expected Player1 at (3,0), got %v", g.At(3, 0))
	}
	if g.At(3, 1) != game.Player2 {
		t.Errorf("expected Player2 at (3,1), got %v", g.At(3, 1))
	}
	if g.At(3, 2) != game.Player1 {
		t.Errorf("expected Player1 at (3,2), got %v", g.At(3, 2))
	}
}

func TestApplyIsNonMutating(t *testing.T) {
	g := New()
	g.Apply(0)
	if g.CurrentPlayer() != game.Player1 {
		t.Error("Apply should not mutate the receiver")
	}
	if g.Height(0) != 0 {
		t.Error("Apply should not mutate the receiver's heights")
	}
	moves := g.LegalMoves(nil)
	if len(moves) != Cols {
		t.Error("Apply should not mutate the receiver's board")
	}
}

func TestFullColumnNotLegal(t *testing.T) {
	g := New()
	// Fill column 0 by alternating with column 1 to avoid winning.
	for range Rows {
		g = g.Apply(0)
		g = g.Apply(1)
	}
	if g.Height(0) != Rows {
		t.Fatalf("expected column 0 to be full, got height %d", g.Height(0))
	}
	moves := g.LegalMoves(nil)
	for _, m := range moves {
		if m == 0 {
			t.Error("full column 0 should not appear in legal moves")
		}
	}
}

func TestLegalMovesEmptyOnTerminal(t *testing.T) {
	g := New()
	// Player1 plays cols 0-3 on the bottom row (with P2 stacking on col 0).
	for _, m := range []Move{0, 0, 1, 0, 2, 0, 3} {
		g = g.Apply(m)
	}
	if !g.IsTerminal() {
		t.Fatal("game should be terminal")
	}
	if g.LegalMoves(nil) != nil {
		t.Error("terminal game should return nil legal moves")
	}
}

// TestWinLines verifies all four win directions for both players.
// For each direction, we build a sequence that uses some non-line column
// as a "trash" column for filler moves, so the fillers don't accidentally
// form a four-in-a-row themselves.
func TestWinLines(t *testing.T) {
	cases := []struct {
		name   string
		moves  []Move
		winner game.Player
	}{
		// Horizontal: P1 plays (0,0),(1,0),(2,0),(3,0). P2 stacks col 0.
		{"player1 horizontal", []Move{0, 0, 1, 0, 2, 0, 3}, game.Player1},

		// Vertical: P1 stacks col 3 four times. P2 plays col 0 in between.
		{"player1 vertical", []Move{3, 0, 3, 0, 3, 0, 3}, game.Player1},

		// Diagonal NE (/). P1 wins on (0,0),(1,1),(2,2),(3,3).
		// Trace:
		//   1: P1 col0 -> (0,0)=P1 ✓
		//   2: P2 col1 -> (1,0)=P2
		//   3: P1 col1 -> (1,1)=P1 ✓
		//   4: P2 col2 -> (2,0)=P2
		//   5: P1 col4 -> filler
		//   6: P2 col2 -> (2,1)=P2
		//   7: P1 col2 -> (2,2)=P1 ✓
		//   8: P2 col3 -> (3,0)=P2
		//   9: P1 col5 -> filler
		//  10: P2 col3 -> (3,1)=P2
		//  11: P1 col6 -> filler
		//  12: P2 col3 -> (3,2)=P2
		//  13: P1 col3 -> (3,3)=P1 ✓ wins
		{"player1 diagonal NE", []Move{0, 1, 1, 2, 4, 2, 2, 3, 5, 3, 6, 3, 3}, game.Player1},

		// Diagonal NW (\). P1 wins on (3,0),(2,1),(1,2),(0,3).
		// Use col 6 as the trash column; col 6 ends up with 3 stacked P1s
		// (no vertical 4) and contributes nothing to the line.
		{"player1 diagonal NW", []Move{3, 2, 2, 1, 6, 1, 1, 0, 6, 0, 6, 0, 0}, game.Player1},

		// Player2 horizontal: P2 plays cols 0-3 on the bottom row.
		// P1 fillers stack col 6 three times, then col 5, to avoid a
		// vertical four on col 6.
		{"player2 horizontal", []Move{6, 0, 6, 1, 6, 2, 5, 3}, game.Player2},

		// Player2 vertical on column 4. P1 fillers stack col 6 three times,
		// then col 5.
		{"player2 vertical", []Move{6, 4, 6, 4, 6, 4, 5, 4}, game.Player2},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			g := New()
			for i, m := range tt.moves {
				if g.IsTerminal() {
					t.Fatalf("game became terminal early at move %d (winner %v)", i, g.Winner())
				}
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

// TestDraw fills the board with a sequence that produces no four-in-a-row.
// The script was found by depth-first search.
//
// The pattern: fill columns 0,1,2 fully, then a single piece in col 4 (to
// break a would-be diagonal), then col 3 fully, then the rest of col 4,
// then col 5 and col 6. This produces a checker pattern with column 3
// inverted relative to its neighbours:
//
//	O O O X O O O
//	X X X O X X X
//	O O O X O O O
//	X X X O X X X
//	O O O X O O O
//	X X X O X X X
func TestDraw(t *testing.T) {
	moves := []Move{
		0, 0, 0, 0, 0, 0,
		1, 1, 1, 1, 1, 1,
		2, 2, 2, 2, 2, 2,
		4,
		3, 3, 3, 3, 3, 3,
		4, 4, 4, 4, 4,
		5, 5, 5, 5, 5, 5,
		6, 6, 6, 6, 6, 6,
	}
	if len(moves) != Rows*Cols {
		t.Fatalf("draw script has %d moves, want %d", len(moves), Rows*Cols)
	}
	g := New()
	for i, m := range moves {
		if g.IsTerminal() {
			t.Fatalf("game became terminal early at move %d (winner %v)", i, g.Winner())
		}
		g = g.Apply(m)
	}
	if !g.IsTerminal() {
		t.Fatal("game should be terminal after 42 moves")
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
