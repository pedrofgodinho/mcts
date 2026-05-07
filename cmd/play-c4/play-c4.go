// Command play-c4 runs an interactive Connect 4 game between a human and the MCTS agent.
package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/pedrofgodinho/mcts/game"
	"github.com/pedrofgodinho/mcts/games/connect4"
	"github.com/pedrofgodinho/mcts/mcts"
)

func main() {
	iterations := flag.Int("iterations", 50000, "MCTS iterations per move")
	budget := flag.Duration("budget", 0, "MCTS time budget per move (0 = use iterations)")
	humanFirst := flag.Bool("first", true, "human plays first (X)")
	verbose := flag.Bool("verbose", false, "print MCTS statistics after each move")
	flag.Parse()

	humanPlayer := game.Player1
	if !*humanFirst {
		humanPlayer = game.Player2
	}

	opts := mcts.SearchOptions{Iterations: *iterations}
	if *budget > 0 {
		opts.Budget = *budget
		opts.Iterations = 0
	}

	reader := bufio.NewReader(os.Stdin)
	evaluator := mcts.RandomRolloutEvaluator[connect4.Game, connect4.Move]{}
	agent := mcts.NewAgent[connect4.Game, connect4.Move](connect4.New(), evaluator)

	fmt.Println("Connect 4 vs. MCTS. Enter a column number (0-6) to drop a piece.")
	printBoard(agent.State())

	for !agent.State().IsTerminal() {
		var move connect4.Move
		if agent.State().CurrentPlayer() == humanPlayer {
			move = readHumanMove(reader, agent.State())
		} else {
			var stats mcts.SearchStats[connect4.Move]
			move, stats = agent.Search(opts)
			fmt.Printf("MCTS plays column %d.\n", move)
			if *verbose {
				printStats(stats)
			}
		}
		agent.Advance(move)
		printBoard(agent.State())
	}

	fmt.Println(resultMessage(agent.State(), humanPlayer))
}

func printStats(s mcts.SearchStats[connect4.Move]) {
	fmt.Printf("  %d iterations in %v (root visits: %d)\n",
		s.Iterations, s.Duration.Round(time.Millisecond), s.RootVisits)
	for _, c := range s.Children {
		marker := " "
		if c.Visits == s.Children[0].Visits {
			marker = "►"
		}
		fmt.Printf("  %s col %d: %6d visits, win rate %.1f%%\n",
			marker, c.Move, c.Visits, c.WinRate*100)
	}
	if len(s.PrincipalVariation) > 0 {
		fmt.Print("  PV:")
		for _, step := range s.PrincipalVariation {
			fmt.Printf(" %d(%d, %.0f%%)", step.Move, step.Visits, step.WinRate*100)
		}
		fmt.Println()
	}
}

// readHumanMove prompts until the human enters a legal move, then returns it.
func readHumanMove(reader *bufio.Reader, g connect4.Game) connect4.Move {
	legal := make(map[connect4.Move]bool)
	for _, m := range g.LegalMoves(nil) {
		legal[m] = true
	}
	for {
		fmt.Print("Your move (column 0-6): ")
		line, err := reader.ReadString('\n')
		if err != nil {
			fmt.Println("input error, exiting")
			os.Exit(1)
		}
		n, err := strconv.Atoi(strings.TrimSpace(line))
		if err != nil || n < 0 || n >= connect4.Cols {
			fmt.Printf("enter a number 0-%d\n", connect4.Cols-1)
			continue
		}
		m := connect4.Move(n)
		if !legal[m] {
			fmt.Println("that column is full")
			continue
		}
		return m
	}
}

// printBoard renders the board top-down with X/O for played squares and `.`
// for empty ones. A column-index footer lets the player see which number to
// type without counting.
func printBoard(g connect4.Game) {
	fmt.Println()
	for r := connect4.Rows - 1; r >= 0; r-- {
		fmt.Print(" ")
		for c := 0; c < connect4.Cols; c++ {
			switch g.At(c, r) {
			case game.Player1:
				fmt.Print("X ")
			case game.Player2:
				fmt.Print("O ")
			default:
				fmt.Print(". ")
			}
		}
		fmt.Println()
	}
	fmt.Print(" ")
	for c := 0; c < connect4.Cols; c++ {
		fmt.Printf("%d ", c)
	}
	fmt.Println()
	fmt.Println()
}

func resultMessage(g connect4.Game, human game.Player) string {
	r := g.Result()
	if r == 0 {
		return "Draw."
	}
	if game.Player(r) == human {
		return "You win."
	}
	return "MCTS wins."
}
