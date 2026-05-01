// Command play-ttt runs an interactive tic-tac-toe game between a human and the MCTS agent.
package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/pedrofgodinho/mcts/game"
	"github.com/pedrofgodinho/mcts/games/tictactoe"
	"github.com/pedrofgodinho/mcts/mcts"
)

func main() {
	iterations := flag.Int("iterations", 10000, "MCTS iterations per move")
	humanFirst := flag.Bool("first", true, "human plays first (X)")
	flag.Parse()

	humanPlayer := game.Player1
	if !*humanFirst {
		humanPlayer = game.Player2
	}

	reader := bufio.NewReader(os.Stdin)
	g := tictactoe.New()

	fmt.Println("Tic-tac-toe vs. MCTS. Enter a square number (0-8) to play.")
	printBoard(g)

	for !g.IsTerminal() {
		var move tictactoe.Move
		if g.CurrentPlayer() == humanPlayer {
			move = readHumanMove(reader, g)
		} else {
			fmt.Println("MCTS is thinking...")
			move = mcts.Search(g, *iterations)
			fmt.Printf("MCTS plays %d.\n", move)
		}
		g = g.Apply(move)
		printBoard(g)
	}

	fmt.Println(resultMessage(g, humanPlayer))
}

// readHumanMove prompts until the human enters a legal move, then returns it.
func readHumanMove(reader *bufio.Reader, g tictactoe.Game) tictactoe.Move {
	legal := make(map[tictactoe.Move]bool)
	for _, m := range g.LegalMoves() {
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
