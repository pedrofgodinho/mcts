package connect4

import "github.com/pedrofgodinho/mcts/game"

const (
	Cols = 7
	Rows = 6
)

// Game represents the state of a Connect 4 game. It implements the GameState interface defined in the game package.
type Game struct {
	board   [Cols][Rows]game.Player
	heights [Cols]uint8
	turn    game.Player
}

// Ensure that Game implements the GameState interface for use with MCTS.
var _ game.GameState[Game, Move] = Game{}

// Move represents a move in Connect 4, which is simply the index of the square to place a mark (0-8).
type Move int8

// New initializes and returns a new Connect 4 game with an empty board and Player1 starting.
func New() Game {
	return Game{
		turn: game.Player1,
	}
}

// Height returns the current height of the specified column
func (g Game) Height(col int) uint8 {
	return g.heights[col]
}

// At returns the player at the given column and row, where row 0 is the
// bottom of the board.
func (g Game) At(col, row int) game.Player {
	return g.board[col][row]
}

// LegalMoves returns a slice of legal moves (columns that are not full) that can be made from the current game state.
func (g Game) LegalMoves(buf []Move) []Move {
	buf = buf[:0]
	if g.IsTerminal() {
		return buf
	}
	for c := range Cols {
		if g.heights[c] < Rows {
			buf = append(buf, Move(c))
		}
	}
	return buf
}

// Apply takes a move and returns the new game state resulting from applying that move. It updates the board and switches the turn to the other player.
// It assumes the move is legal (the column is not full) and does not perform any bounds checking.
func (g Game) Apply(m Move) Game {
	c := int(m)
	g.board[c][g.heights[c]] = g.turn
	g.heights[c]++
	g.turn = -g.turn
	return g
}

// CurrentPlayer returns the player whose turn it is to move in the current game state.
func (g Game) CurrentPlayer() game.Player {
	return g.turn
}

// IsTerminal checks if the game has reached a terminal state, which occurs when either player has won by
// connecting four pieces in a row, column, or diagonal, or when the board is full resulting in a draw.
// It returns true if the game is over and false otherwise.
func (g Game) IsTerminal() bool {
	if g.Winner() != game.None {
		return true
	}
	for c := range Cols {
		if g.heights[c] < Rows {
			return false
		}
	}
	return true
}

func (g Game) Result() float64 {
	return float64(g.Winner())
}

// Winner scans the board for a four-in-a-row in any direction and returns
// the winning player, or game.None if there is no winner.
func (g Game) Winner() game.Player {
	// For each occupied square, check the four directions whose starting
	// square would be this one: east, north, north-east, north-west.
	// This visits every possible four-in-a-row exactly once.
	for c := range Cols {
		h := int(g.heights[c])
		for r := 0; r < h; r++ {
			p := g.board[c][r]
			// East: (c, r), (c+1, r), (c+2, r), (c+3, r)
			if c+3 < Cols &&
				g.board[c+1][r] == p &&
				g.board[c+2][r] == p &&
				g.board[c+3][r] == p {
				return p
			}
			// North: (c, r), (c, r+1), (c, r+2), (c, r+3)
			if r+3 < Rows &&
				g.board[c][r+1] == p &&
				g.board[c][r+2] == p &&
				g.board[c][r+3] == p {
				return p
			}
			// North-East: (c, r), (c+1, r+1), (c+2, r+2), (c+3, r+3)
			if c+3 < Cols && r+3 < Rows &&
				g.board[c+1][r+1] == p &&
				g.board[c+2][r+2] == p &&
				g.board[c+3][r+3] == p {
				return p
			}
			// North-West: (c, r), (c-1, r+1), (c-2, r+2), (c-3, r+3)
			if c-3 >= 0 && r+3 < Rows &&
				g.board[c-1][r+1] == p &&
				g.board[c-2][r+2] == p &&
				g.board[c-3][r+3] == p {
				return p
			}
		}
	}
	return game.None
}
