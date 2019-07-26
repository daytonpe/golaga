package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"os"
	"os/exec"
	"time"
)

// Player is the player character
// Can only move left and right
type Player struct {
	col int
}

var player Player

// Alien is the enemy that is trying to invade earth
type Alien struct {
	row int
	col int
}

// Laser is the bolt of energy that kills Alien ships
type Laser struct {
	row int
	col int
}

var aliens []*Alien
var lasers []*Laser
var playerRow = 30 // denotes the row in which the player slides
var level []string
var score int
var numDots int
var lives = 1

// Config holds the emoji configuration
type Config struct {
	Player   string `json:"player"`
	Alien    string `json:"alien"`
	Wall     string `json:"wall"`
	Dot      string `json:"dot"`
	Death    string `json:"death"`
	Space    string `json:"space"`
	UseEmoji bool   `json:"use_emoji"`
}

var cfg Config

func loadConfig() error {
	f, err := os.Open("config_noemoji.json")
	if err != nil {
		return err
	}
	defer f.Close()

	decoder := json.NewDecoder(f)
	err = decoder.Decode(&cfg)
	if err != nil {
		return err
	}

	return nil
}

func loadLevel() error {
	f, err := os.Open("level01.txt")
	if err != nil {
		return err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		level = append(level, line)
	}

	for row, line := range level {
		for col, char := range line {
			switch char {
			case 'U':
				player = Player{col}
			case 'G':
				aliens = append(aliens, &Alien{row, col})
			case '.': // laser shots
				numDots++
			}
		}
	}

	return nil
}

func clearScreen() {
	fmt.Printf("\x1b[2J")
	moveCursor(0, 0)
}

func moveCursor(row, col int) {
	if cfg.UseEmoji {
		fmt.Printf("\x1b[%d;%df", row+1, col*2+1)
	} else {
		fmt.Printf("\x1b[%d;%df", row+1, col+1)
	}
}

func printScreen() {
	clearScreen()
	for _, line := range level {
		for _, chr := range line {
			switch chr {
			case '#':
				fmt.Printf("\x1b[44m" + cfg.Wall + "\x1b[0m") // seems this adds color to the wall
			case 'Y':
				fmt.Printf(cfg.Alien)
			case '.':
				fmt.Printf(cfg.Dot)
			default:
				fmt.Printf(cfg.Space)
			}
		}
		fmt.Printf("\n")
	}

	moveCursor(playerRow, player.col)
	fmt.Printf(cfg.Player)

	for _, g := range aliens {
		moveCursor(g.row, g.col)
		fmt.Printf(cfg.Alien)
	}

	moveCursor(len(level)+1, 0)
	fmt.Printf("Score: %v\tLives: %v\n", score, lives)
}

func readInput() (string, error) {
	buffer := make([]byte, 100)

	cnt, err := os.Stdin.Read(buffer)
	if err != nil {
		return "", err
	}

	if cnt == 1 && buffer[0] == 0x1b {
		return "ESC", nil
	} else if cnt >= 3 {
		if buffer[0] == 0x1b && buffer[1] == '[' {
			switch buffer[2] {
			case 'A': // up
				fallthrough
			case 'B': // down
				return "FIRE", nil
			case 'C':
				return "RIGHT", nil
			case 'D':
				return "LEFT", nil
			}
		}
	}

	return "", nil
}

func fireLaser() {
	// pass in the column that we want to spawn the lasor
	// maze[player.row] = maze[player.row][0:player.col] + " " + maze[player.row][player.col+1:]
	level[playerRow-1] = level[playerRow-1][0:player.col] + "." + level[playerRow-1][player.col+1:]
	return
}

func makeMove(oldRow, oldCol int, dir string) (newRow, newCol int) {
	newRow, newCol = oldRow, oldCol

	switch dir {
	case "FIRE":
		fireLaser()

	case "RIGHT":
		newCol = newCol + 1
		if newCol == len(level[0]) { // hit right edge
			newCol = 0
		}
	case "LEFT":
		newCol = newCol - 1
		if newCol < 0 { // hit left edge
			newCol = len(level[0]) - 1
		}
	}

	if level[newRow][newCol] == '#' {
		newRow = oldRow
		newCol = oldCol
	}

	return
}

func movePlayer(dir string) {
	playerRow, player.col = makeMove(playerRow, player.col, dir)
	switch level[playerRow][player.col] {
	case 'Y': //Run into alien
		lives = 0
	}
}

func drawDirection() string {
	dir := rand.Intn(4)
	move := map[int]string{
		0: "UP",
		1: "DOWN",
		2: "RIGHT",
		3: "LEFT",
	}
	return move[dir]
}

func moveAliens() {
	for _, g := range aliens {
		dir := drawDirection()
		g.row, g.col = makeMove(g.row, g.col, dir)
	}
}

func init() {
	cbTerm := exec.Command("/bin/stty", "cbreak", "-echo")
	cbTerm.Stdin = os.Stdin

	err := cbTerm.Run()
	if err != nil {
		log.Fatalf("Unable to activate cbreak mode terminal: %v\n", err)
	}
}

func cleanup() {
	cookedTerm := exec.Command("/bin/stty", "-cbreak", "echo")
	cookedTerm.Stdin = os.Stdin

	err := cookedTerm.Run()
	if err != nil {
		log.Fatalf("Unable to activate cooked mode terminal: %v\n", err)
	}
}

func main() {

	// initialize game
	defer cleanup()

	// load resources
	err := loadLevel()
	if err != nil {
		log.Printf("Error loading level: %v\n", err)
		return
	}

	err = loadConfig()
	if err != nil {
		log.Printf("Error loading configuration: %v\n", err)
		return
	}

	// process input (async)
	input := make(chan string)
	go func(ch chan<- string) {
		for {
			input, err := readInput()
			if err != nil {
				log.Printf("Error reading input: %v", err)
				ch <- "ESC"
			}
			ch <- input
		}
	}(input)

	// game loop
	for {
		// process movement
		select {
		case inp := <-input:
			if inp == "ESC" {
				lives = 0
			}
			movePlayer(inp)
		default:
		}

		moveAliens()

		// process collisions
		// TODO set this to if alien makes contact, die
		for _, g := range aliens {
			if playerRow == g.row && player.col == g.col {
				lives = 0
			}
		}

		// update screen
		printScreen()

		// check game over
		if numDots == 0 || lives == 0 {
			if lives == 0 {
				moveCursor(playerRow, player.col)
				fmt.Printf(cfg.Death)
				moveCursor(len(level)+2, 0)
			}
			break
		}

		// repeat
		time.Sleep(200 * time.Millisecond)
	}
}
