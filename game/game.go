package game

type GameState[M any] interface {
	LeaglMoves() []M
	Apply(M) GameState[M]
	IsTerminal() bool
	CurrentPlayer() int
}
