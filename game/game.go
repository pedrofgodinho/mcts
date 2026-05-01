package game

type Player int8

const (
	Player1 Player = 1
	Player2 Player = -1
	None    Player = 0
)

// GameState represents the state of a game and defines the necessary methods for a game to be used with MCTS.
type GameState[S GameState[S, M], M any] interface {
	// LegalMoves appends the legal moves at this state to buf and returns the result.
	// buf is treated as a buffer; existing contents are discarded (callers should
	// pass nil or a slice from a previous call, optionally re-sliced to [:0]).
	// Returns an empty slice (not nil) iff IsTerminal() is true.
	LegalMoves(buf []M) []M
	// Apply takes a move and returns the new game state resulting from applying that move.
	Apply(M) S
	// IsTerminal returns true if the game has reached a terminal state (i.e., a win, loss, or draw), and false otherwise.
	IsTerminal() bool
	// CurrentPlayer returns the player whose turn it is.
	CurrentPlayer() Player
	// Result returns +1.0 if Player1 wins, -1.0 if Player2 wins, and 0.0 for a draw or non-terminal state.
	Result() float64
}
