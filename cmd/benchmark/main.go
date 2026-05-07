package main

import (
	"flag"
	"fmt"
	"time"

	"github.com/pedrofgodinho/mcts/games/connect4"
	"github.com/pedrofgodinho/mcts/games/tictactoe"
	"github.com/pedrofgodinho/mcts/mcts"
)

func main() {
	game := flag.String("game", "connect4", "game to run: ttt or connect4")
	iters := flag.Int("iterations", 50000, "MCTS iterations (0 = use budget)")
	budget := flag.Duration("budget", 0, "MCTS time budget (0 = use iterations)")
	virtualLoss := flag.Int("virtual-loss", 1, "virtual loss per selection (0 = disabled)")
	workers := flag.Int("workers", 1, "parallel MCTS workers (1 = sequential)")
	flag.Parse()

	opts := mcts.SearchOptions{
		Iterations:  *iters,
		Budget:      *budget,
		VirtualLoss: *virtualLoss,
		Workers:     *workers,
	}
	if *budget > 0 {
		opts.Iterations = 0
	}

	switch *game {
	case "ttt":
		runTTT(opts)
	case "connect4":
		runC4(opts)
	default:
		fmt.Println("unknown game:", *game)
	}
}

func runTTT(opts mcts.SearchOptions) {
	base := mcts.RandomRolloutEvaluator[tictactoe.Game, tictactoe.Move]{}
	eval := mcts.NewInstrumentedEvaluator[tictactoe.Game, tictactoe.Move](base)
	agent := mcts.NewAgent[tictactoe.Game, tictactoe.Move](tictactoe.New(), eval)
	_, stats := agent.Search(opts)
	report("tic-tac-toe", stats, eval.Metrics())
}

func runC4(opts mcts.SearchOptions) {
	base := mcts.RandomRolloutEvaluator[connect4.Game, connect4.Move]{}
	eval := mcts.NewInstrumentedEvaluator[connect4.Game, connect4.Move](base)
	agent := mcts.NewAgent[connect4.Game, connect4.Move](connect4.New(), eval)
	_, stats := agent.Search(opts)
	report("connect 4", stats, eval.Metrics())
}

func report[M comparable](name string, s mcts.SearchStats[M], m mcts.EvaluatorMetrics) {
	fmt.Printf("=== %s ===\n", name)
	fmt.Printf("  search:    %d iterations in %v (%.0f iters/sec)\n",
		s.Iterations, s.Duration.Round(time.Microsecond),
		float64(s.Iterations)/s.Duration.Seconds())
	fmt.Printf("  evaluator: %d calls, %v total (%.1f%% of search time)\n",
		m.Calls, m.TotalDuration.Round(time.Microsecond),
		100*float64(m.TotalDuration)/float64(s.Duration))
	if m.Calls > 0 {
		fmt.Printf("  avg/call:  %v\n", m.TotalDuration/time.Duration(m.Calls))
	}
}
