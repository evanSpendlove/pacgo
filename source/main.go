package main

/*
	Notes:
		-> Go is a garbage collected language.
		-> Pointers work more like references. Automatically deleted (safer) and cannot do math on pointers.
		-> Goroutines are similar to thread (use keyword go)
		-> Have no control over order of execution of async goroutines

		Channels:
			-> Allow communication (read/write) between goroutines
			-> Each channel has a type and buffer size
			-> Can be a blocking operation if the buffer is full (blocked until read from another routine)
			-> Write is ch <- something || Read is something <- ch
*/

// Imports
import ("bufio"
				"fmt"
				"log"
				"math/rand"
				"os"
				"os/exec"
				"time"
)

import "github.com/danicat/simpleansi"
//import "simpleansi" // Library used for clearing the terminal screen so we can redraw the game each frame

// Structs

type sprite struct {
	row int
	col int
}

// Global variables
var maze []string // The maze is stored as an array of strings read in from a file
var player sprite
var ghosts []*sprite // Slice of pointers to sprite objects

var score int
var numDots int
var lives = 1

// ---------------- Initialisation functions ----------------

/*
	Initialises the terminal to Cbreak mode rather than Cooked mode.
	Turns off echo for key presses, so we can accept input without it being displayed in the terminal.
	Error logged if unable to activate Cbreak.
*/
func initialise() {
	cbTerm := exec.Command("stty", "cbreak", "-echo")
  cbTerm.Stdin = os.Stdin

	err := cbTerm.Run()
	if err != nil {
		log.Fatalln("unable to activate Cbreak mode : ", err)
	}
}

/*
	Loads the global variable, maze, with strings read in from a file

	Returns an error if the file cannot be opened.
*/
func loadMaze(file string) error {
	f, err := os.Open(file)
	if err != nil { // If error ocurred, then return the error.
		return err
	}

	defer f.Close() // Defer closing until the remaining code is run.

	scanner := bufio.NewScanner(f)

	for scanner.Scan() {
		line := scanner.Text()
		maze = append(maze, line)
	}

	for row, line := range maze {
		for col, char := range line {
			switch char {
			case 'P':
				player = sprite{row, col}
			case 'G':
				ghosts = append(ghosts, &sprite{row, col}) // & here means we are adding a pointer to an object
			case '.':
					numDots++
			}
		}
	}

	return nil
}

// ---------------- Game Logic functions ----------------

/*
	Returns the updated row and col of a player according to the maze and the direction of movement passed.
*/
func makeMove(oldRow, oldCol int, dir string) (newRow, newCol int) {
	newRow, newCol = oldRow, oldCol

	// Switch based on direction, with circular movement if out of bounds
	switch dir {
	case "UP":
		newRow = newRow - 1
		if newRow < 0 {
			newRow = len(maze) - 1
		}
	case "DOWN":
		newRow = newRow + 1
		if newRow == len(maze){
			newRow = 0
		}
	case "RIGHT":
		newCol = newCol + 1
		if newCol == len(maze[0]) {
			newCol = 0
		}
	case "LEFT":
		newCol = newCol - 1
		if newCol < 0 {
			newCol = len(maze[0]) - 1
		}
	}

	// If wall, then ignore movement
	if maze[newRow][newCol] == '#' {
		newRow = oldRow
		newCol = oldCol
	}

	// Can use fallthrough keyword to skip the implicit break in the switch statement

	return
}

/*
	Moves the player according to the direction passed.
	Updates the player's row and col
*/
func movePlayer(dir string) {
	player.row, player.col = makeMove(player.row, player.col, dir)

	// Check for dots
	switch maze[player.row][player.col] {
	case '.':
		numDots--
		score++
		maze[player.row] = maze[player.row][0:player.col] + " " + maze[player.row][player.col + 1:] // Remove dot from maze
	}
}

/*
	Returns a string with a randomly chosen direction
*/
func drawDirection() string {
	dir := rand.Intn(4) // Generate random number in range [0, 4), i.e. {0, 1, 2, 3}

	move := map[int]string { // Map from int to String
		0: "UP",
		1: "DOWN",
		2: "RIGHT",
		3: "LEFT",
	}

	return move[dir]
}

/*
	Moves the ghosts by drawing a random direction and moving them in that direction
*/
func moveGhosts() {
	for _, g := range ghosts {
		dir := drawDirection()
		g.row, g.col = makeMove(g.row, g.col, dir)
	}
}

// ---------------- IO functions ----------------

/*
	Prints the maze to the screen.
*/
func printScreen(){
	simpleansi.ClearScreen() // Clear the screen before we print

	// Print the maze
	for _, line := range maze {
		for _, chr := range line {
			switch chr {
			case '#':
				fallthrough
			case '.':
				fmt.Printf("%c", chr)
			default:
				fmt.Print(" ")
			}
		}
		fmt.Println()
	}

	// Draw player
	simpleansi.MoveCursor(player.row, player.col)
	fmt.Print("P")

	// Draw ghosts
	for _, g := range ghosts {
		simpleansi.MoveCursor(g.row, g.col)
		fmt.Print("G")
	}

	// Move cursor outside of maze drawing area
	simpleansi.MoveCursor(len(maze)+1, 0)
	fmt.Println("Score: ", score, "\tLives: ", lives) // Print score and lives
}

/*
	Reads input from Stdin (100 byte buffer)
	Returns the command read in (ESC, Up, down, etc.) and an error code
*/
func readInput() (string, error) {
	buffer := make([] byte, 100)
	cnt, err := os.Stdin.Read(buffer)

	// If error, return error and empty string
	if err != nil {
		return "", err
	}

	// If the key press is esc
	if cnt == 1 && buffer[0] == 0x1b {
		return "ESC", nil
	} else if cnt >= 3 {
			if buffer[0] == 0x1b && buffer[1] == '[' {
				switch buffer[2] {
				case 'A' :
					return "UP", nil
				case 'B' :
					return "DOWN", nil
				case 'C' :
					return "RIGHT", nil
				case 'D' :
					return "LEFT", nil
				}
			}
	}

	return "", nil // Nothing read in
}

// ---------------- Close-down functions ----------------

func cleanup() {
	cookedTerm := exec.Command("stty", "-cbreak", "echo")
  cookedTerm.Stdin = os.Stdin

	err := cookedTerm.Run()
	if err != nil {
		log.Fatalln("unable to restored cooked mode: ", err)
	}
}

// Main function

func main() {
	// initialise game

	initialise()
	defer cleanup()

	// Load maze with error checking
	err := loadMaze("source/maze01.txt")
	if(err != nil){
		log.Println("failed to load maze:", err)
		return
	}

	// load resources

	// process input (async)
	input := make(chan string)
	go func(ch chan <- string) {
		for {
			input, err := readInput()
			if err != nil {
				log.Println("error reading input:", err)
				ch <- "ESC"
			}
			ch <- input
		}
	}(input)

	// game loop
	for{
		// process movement
		select { // Select is a switch statement for channels
		case inp := <-input:
			if inp == "ESC" {
				lives = 0
			}
			movePlayer(inp)
		default:
		}

		moveGhosts()

		// process collisions
		for _, g := range ghosts {
			if player == *g { // Interesting that you can compare objects like this (not reference based comparison like Java)
				lives--; // Decrement lives
			}
		}

		// update screen
		printScreen()

		// check game over
		if numDots == 0 || lives <= 0 {
			break;
		}

		// repeat
		time.Sleep(150 * time.Millisecond)
	}
}