// Command play-ttt runs an interactive tic-tac-toe game between a human and the MCTS agent.
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
	"github.com/pedrofgodinho/mcts/games/tictactoe"
	"github.com/pedrofgodinho/mcts/mcts"
)

func main() {
	iterations := flag.Int("iterations", 10000, "MCTS iterations per move (0 = use budget)")
	budget := flag.Duration("budget", 0, "MCTS time budget per move (0 = use iterations)")
	virtualLoss := flag.Int("virtual-loss", 1, "virtual loss per selection (0 = disabled)")
	workers := flag.Int("workers", 1, "parallel MCTS workers (1 = sequential)")
	humanFirst := flag.Bool("first", true, "human plays first (X)")
	verbose := flag.Bool("verbose", false, "print MCTS statistics after each move")
	flag.Parse()

	humanPlayer := game.Player1
	if !*humanFirst {
		humanPlayer = game.Player2
	}

	opts := mcts.SearchOptions{
		Iterations:  *iterations,
		VirtualLoss: *virtualLoss,
		Workers:     *workers,
	}
	if *budget > 0 {
		opts.Budget = *budget
		opts.Iterations = 0
	}

	reader := bufio.NewReader(os.Stdin)
	evaluator := mcts.RandomRolloutEvaluator[tictactoe.Game, tictactoe.Move]{}
	agent := mcts.NewAgent[tictactoe.Game, tictactoe.Move](tictactoe.New(), evaluator)

	fmt.Println("Tic-tac-toe vs. MCTS. Enter a square number (0-8) to play.")
	printBoard(agent.State())

	for !agent.State().IsTerminal() {
		var move tictactoe.Move
		if agent.State().CurrentPlayer() == humanPlayer {
			move = readHumanMove(reader, agent.State())
		} else {
			var stats mcts.SearchStats[tictactoe.Move]
			move, stats = agent.Search(opts)
			fmt.Printf("MCTS plays %d.\n", move)
			if *verbose {
				printStats(stats)
			}
		}
		agent.Advance(move)
		printBoard(agent.State())
	}

	fmt.Println(resultMessage(agent.State(), humanPlayer))
}

func printStats(s mcts.SearchStats[tictactoe.Move]) {
	fmt.Printf("  %d iterations in %v (root visits: %d)\n",
		s.Iterations, s.Duration.Round(time.Millisecond), s.RootVisits)
	for _, c := range s.Children {
		marker := " "
		if c.Visits == s.Children[0].Visits {
			marker = "►"
		}
		fmt.Printf("  %s move %d: %5d visits, win rate %.1f%%\n",
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
func readHumanMove(reader *bufio.Reader, g tictactoe.Game) tictactoe.Move {
	legal := make(map[tictactoe.Move]bool)
	for _, m := range g.LegalMoves(nil) {
		legal[m] = true
	}
	for {
		fmt.Print("Your move: ")
		line, err := reader.ReadString('\n')
		if err != nil {
			fmt.Println("input error, exiting")
			os.Exit(1)
		}
		n, err := strconv.Atoi(strings.TrimSpace(line))
		if err != nil || n < 0 || n > 8 {
			fmt.Println("enter a number 0-8")
			continue
		}
		m := tictactoe.Move(n)
		if !legal[m] {
			fmt.Println("that square is already taken")
			continue
		}
		return m
	}
}

// printBoard renders the board with X/O for played squares and the index for empty ones.
func printBoard(g tictactoe.Game) {
	cells := make([]string, 9)
	for i := range 9 {
		switch g.At(i) {
		case game.Player1:
			cells[i] = "X"
		case game.Player2:
			cells[i] = "O"
		default:
			cells[i] = strconv.Itoa(i)
		}
	}
	fmt.Println()
	fmt.Printf(" %s | %s | %s\n", cells[0], cells[1], cells[2])
	fmt.Println("-----------")
	fmt.Printf(" %s | %s | %s\n", cells[3], cells[4], cells[5])
	fmt.Println("-----------")
	fmt.Printf(" %s | %s | %s\n", cells[6], cells[7], cells[8])
	fmt.Println()
}

func resultMessage(g tictactoe.Game, human game.Player) string {
	r := g.Result()
	if r == 0 {
		return "Draw."
	}
	if game.Player(r) == human {
		return "You win."
	}
	return "MCTS wins."
}
