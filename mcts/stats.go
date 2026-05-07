package mcts

import "time"

// SearchStats summarizes a single Search call.
type SearchStats[M comparable] struct {
	Iterations         int
	Duration           time.Duration
	RootVisits         int64
	Children           []ChildStats[M]
	PrincipalVariation []PVStep[M]
}

// ChildStats summarizes the stats for a single child of the root node after a Search.
// WinRate is from the perspective of the player to move at the root
type ChildStats[M comparable] struct {
	Move    M
	Visits  int64
	WinRate float64
}

// PVStep is one move along the principal variation, with stats from the
// perspective of the player to move at that step.
type PVStep[M comparable] struct {
	Move    M
	Visits  int64
	WinRate float64
}
