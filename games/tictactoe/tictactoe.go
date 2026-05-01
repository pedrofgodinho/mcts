package tictactoe

import "github.com/pedrofgodinho/mcts/game"

// Game represents the state of a tic-tac-toe game. It implements the GameState interface defined in the game package.
type Game struct {
	board [9]game.Player
	turn  game.Player
}

// Ensure that Game implements the GameState interface for use with MCTS.
var _ game.GameState[Game, Move] = Game{}

// Move represents a move in tic-tac-toe, which is simply the index of the square to place a mark (0-8).
type Move int8

// New initializes and returns a new tic-tac-toe game with an empty board and Player1 starting.
func New() Game {
	return Game{
		board: [9]game.Player{},
		turn:  game.Player1,
	}
}

func (g Game) At(i int) game.Player {
	return g.board[i]
}

var winLines = [8][3]int{
	{0, 1, 2}, // rows
	{3, 4, 5},
	{6, 7, 8},
	{0, 3, 6}, // columns
	{1, 4, 7},
	{2, 5, 8},
	{0, 4, 8}, // diagonals
	{2, 4, 6},
}

// LegalMoves returns a slice of legal moves (empty squares) that can be made from the current game state.
func (g Game) LegalMoves() []Move {
	if g.IsTerminal() {
		return nil
	}

	moves := make([]Move, 0, 9)
	for i, sq := range g.board {
		if sq == game.None {
			moves = append(moves, Move(i))
		}
	}
	return moves
}

// Apply takes a move and returns the new game state resulting from applying that move. It updates the board and switches the turn to the other player.
func (g Game) Apply(m Move) Game {
	g.board[m] = g.turn
	g.turn = -g.turn
	return g
}

// CurrentPlayer returns the player whose turn it is.
func (g Game) CurrentPlayer() game.Player {
	return g.turn
}

// IsTerminal returns true if the game has reached a terminal state (i.e., a win, loss, or draw), and false otherwise.
func (g Game) IsTerminal() bool {
	if g.Winner() != game.None {
		return true
	}

	for _, sq := range g.board {
		if sq == game.None {
			return false
		}
	}
	return true
}

// Result returns +1.0 if Player1 wins, -1.0 if Player2 wins, and 0.0 for a draw or non-terminal state.
func (g Game) Result() float64 {
	return float64(g.Winner())
}

// Winner checks all possible winning lines (rows, columns, diagonals) to determine if there is a winner. It returns the player who has won or game.None if there is no winner.
func (g Game) Winner() game.Player {
	for _, line := range winLines {
		a, b, c := g.board[line[0]], g.board[line[1]], g.board[line[2]]
		if a != game.None && a == b && b == c {
			return a
		}
	}
	return game.None
}
