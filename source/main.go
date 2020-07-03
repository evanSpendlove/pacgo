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

/*
	Extra emoji: 🌵, 🏔, 🛸, 🚀, 🛰, ☄, 🌑
*/

/*
	Regular theme
	{
  "player": "😃",
  "ghost": "👻",
  "ghost_blue": "🤖",
  "wall": "🌵",
  "dot": "🍕",
  "pill": "💊",
  "death": "💀",
  "space": "  ",
  "use_emoji": true,
  "pill_duration_secs": 10
}

	Japanese theme (⛩, 🏮, 🏯, 🍡)
	{
		"player": "👾",
		"ghost": "🦖",
		"ghost_blue": "🦕",
		"wall": "⛩⛩",
		"dot": "🍙",
		"pill": "🍡",
		"death": "☠",
		"space": "  ",
		"use_emoji": true,
		"pill_duration_secs": 10
	}

	Mummy theme (🗿, ⚱, 🏺)

	Space Theme (🌕, 🌌)
	{
    "player": "🚀",
    "ghost": "🛰",
    "wall" : "🌌",
    "dot": "🌕",
    "pill": "☄☄",
    "death": "☠",
    "space": "  ",
    "use_emoji": true
	}
*/

/*
	Todo:
	[ ] Centre the screen
	[ ] Add themes option - just change the files that are loaded
	[ ] Add pathfinding to AI
	[ ] Add new maps - could have a level select
*/

// Imports
import (
	"bufio"
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"os"
	"os/exec"
	"strconv"
	"sync"
	"time"

	"github.com/danicat/simpleansi"
)

//import "simpleansi" // Library used for clearing the terminal screen so we can redraw the game each frame

// Structs

type sprite struct {
	row      int
	col      int
	startRow int
	startCol int
}

// Config struct for holding Json data
// Note that public members were used here - required for json decoder to work!
type Config struct {
	Player           string        `json:"player"`
	Ghost            string        `json:"ghost"`
	GhostBlue        string        `json:"ghost_blue"`
	Wall             string        `json:"wall"`
	Dot              string        `json:"dot"`
	Pill             string        `json:"pill"`
	Death            string        `json:"death"`
	Space            string        `json:"space"`
	UseEmoji         bool          `json:"use_emoji"`
	PillDurationSecs time.Duration `json:"pill_duration_secs"`
}

type GhostStatus string // Define the Ghost status as a string

// Constant definition of these two variables
const (
	GhostStatusNormal GhostStatus = "Normal"
	GhostStatusBlue   GhostStatus = "Blue"
)

type ghost struct {
	position sprite
	status   GhostStatus
}

// Global variables
var maze []string // The maze is stored as an array of strings read in from a file
var player sprite
var ghosts []*ghost // Slice of pointers to ghost objects
var cfg Config

var score int
var numDots int
var lives = 3

var pillTimer *time.Timer
var pillMx sync.Mutex          // Mutex lock
var ghostStatusMx sync.RWMutex // Read/Write Mutex

/* Command line flags defined here
 	 Paramaters of .String() : flag name, default value, description (exhibited when --help is used)
 			Returns a pointer to a string holding the value of the flag
			Value only filled after calling flag.Parse()
*/
var (
	configFile = flag.String("config-file", "source/config.json", "path to custom configuration file")
	mazeFile   = flag.String("maze-file", "source/maze01.txt", "path to a custom maze file")
)

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
				player = sprite{row, col, row, col}
			case 'G':
				ghosts = append(ghosts, &ghost{sprite{row, col, row, col}, GhostStatusNormal}) // & here means we are adding a pointer to an object
			case '.':
				numDots++
			}
		}
	}

	return nil
}

/*
	Loads the json config, decodes it, and stores it in the cfg variable

	Returns an error if the file cannot be opened or there is an error when decoding the json file.
*/
func loadConfig(file string) error {
	f, err := os.Open(file) // Open file

	// Error checking, returns error
	if err != nil {
		return err
	}
	defer f.Close() // Defer file closing to end of function

	decoder := json.NewDecoder(f) // Create json decoder
	err = decoder.Decode(&cfg)    // Decode the json into the config struct

	// Error checking
	if err != nil {
		return err
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
		if newRow == len(maze) {
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

	// Creating an inline function here!
	removeDot := func(row, col int) {
		maze[row] = maze[row][0:col] + " " + maze[row][col+1:]
	}

	// Check for dots
	switch maze[player.row][player.col] {
	case '.':
		numDots--
		score++
		removeDot(player.row, player.col)
	case 'X':
		score += 10
		removeDot(player.row, player.col)
		go processPill() // Call goroutine
	}
}

/*
	Returns a string with a randomly chosen direction
*/
func drawDirection() string {
	dir := rand.Intn(4) // Generate random number in range [0, 4), i.e. {0, 1, 2, 3}

	move := map[int]string{ // Map from int to String
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
		g.position.row, g.position.col = makeMove(g.position.row, g.position.col, dir)
	}
}

/*
	Update the status of all ghosts with the status supplied
*/
func updateGhosts(ghosts []*ghost, ghostStatus GhostStatus) {
	ghostStatusMx.Lock()         // Lock RW mutex
	defer ghostStatusMx.Unlock() // Defer unlocking until the end of the function
	for _, g := range ghosts {
		g.status = ghostStatus
	}
}

/*
	Asynchronously processes a pill consumed by the pacman using mutex locks and channels.
*/
func processPill() {
	pillMx.Lock()                         // Lock Mutex
	updateGhosts(ghosts, GhostStatusBlue) // Make ghosts blue
	if pillTimer != nil {                 // If a pill is already active
		pillTimer.Stop() // Stop it and reset it
	}
	pillTimer = time.NewTimer(time.Second * cfg.PillDurationSecs) // Creates new timer that will send the current time on its channel after at least the specified duration
	pillMx.Unlock()
	<-pillTimer.C // Request message from channel (blocking)
	pillMx.Lock()
	pillTimer.Stop()                        // Stop the timer once done
	updateGhosts(ghosts, GhostStatusNormal) // Return ghosts to normal
	pillMx.Unlock()
}

// ---------------- IO functions ----------------

/*
	Correct for moving the cursor twice the horizontal displacement if the use_emoji flag is set
*/
func moveCursor(row, col int) {
	if cfg.UseEmoji {
		simpleansi.MoveCursor(row, col*2)
	} else {
		simpleansi.MoveCursor(row, col)
	}
}

/*
	Prints the maze to the screen.
*/
func printScreen() {
	simpleansi.ClearScreen() // Clear the screen before we print

	// Print the maze
	for _, line := range maze {
		for _, chr := range line {
			switch chr {
			case '#':
				//fmt.Print(simpleansi.WithBlueBackground(cfg.Wall)) // Print wall from struct
				fmt.Print(cfg.Wall) // Print wall from struct
			case '.':
				fmt.Print(cfg.Dot)
			case 'X':
				fmt.Print(cfg.Pill)
			default:
				fmt.Print(cfg.Space)
			}
		}
		fmt.Println()
	}

	// Draw player
	moveCursor(player.row, player.col)
	fmt.Print(cfg.Player)

	// Draw ghosts
	ghostStatusMx.RLock() // Lock Read lock
	for _, g := range ghosts {
		moveCursor(g.position.row, g.position.col)
		if g.status == GhostStatusNormal {
			fmt.Print(cfg.Ghost)
		} else if g.status == GhostStatusBlue {
			fmt.Print(cfg.GhostBlue)
		}
	}
	ghostStatusMx.RUnlock() // Unlock Read lock

	// Move cursor outside of maze drawing area
	moveCursor(len(maze)+1, 0)

	livesRemaining := strconv.Itoa(lives) // Converts lives int to a string
	if cfg.UseEmoji {                     // If use_emoji flag set
		livesRemaining = getLivesAsEmoji() // Convert to emoji
	}

	fmt.Println("Score: ", score, "\tLives: ", livesRemaining) // Print score and lives
}

/*
	Converts the number of lives remaining into a string of emoji

	Returns a string containing number of lives remaining as emoji
*/
func getLivesAsEmoji() string {
	buf := bytes.Buffer{}        // Create empty buffer
	for i := lives; i > 0; i-- { // Write the number of lives remaining to the buffer as emoji
		buf.WriteString(cfg.Player)
	}
	return buf.String() // Return the buffer
}

/*
	Reads input from Stdin (100 byte buffer)
	Returns the command read in (ESC, Up, down, etc.) and an error code
*/
func readInput() (string, error) {
	buffer := make([]byte, 100)
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
			case 'A':
				return "UP", nil
			case 'B':
				return "DOWN", nil
			case 'C':
				return "RIGHT", nil
			case 'D':
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
	// Initialise command line flags
	flag.Parse() // Need to call this ** before ** changing the console to cbreak mode as it calls os.Exit() on error

	// initialise game

	initialise()
	defer cleanup()

	// Load maze with error checking
	err := loadMaze(*mazeFile)
	if err != nil {
		log.Println("failed to load maze:", err)
		return
	}

	// load resources
	err = loadConfig(*configFile) // Load json config
	if err != nil {
		log.Println("failed to load configuration:", err)
		return
	}

	// process input (async)
	input := make(chan string)
	go func(ch chan<- string) {
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
	for {
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

		// Interesting that you can compare objects as player == *g (not reference based comparison like Java)

		for _, g := range ghosts {
			if player.row == g.position.row && player.col == g.position.col {
				ghostStatusMx.RLock() // Lock Read mutex
				if g.status == GhostStatusNormal {
					lives--
					if lives != 0 {
						moveCursor(player.row, player.col)
						fmt.Print(cfg.Death)
						moveCursor(len(maze)+2, 0)
						ghostStatusMx.RUnlock()
						time.Sleep(1000 * time.Millisecond) // Long respawn timer
						player.row, player.col = player.startRow, player.startRow
					}
				} else if g.status == GhostStatusBlue {
					ghostStatusMx.RUnlock()
					updateGhosts([]*ghost{g}, GhostStatusNormal)
					g.position.row, g.position.col = g.position.startRow, g.position.startCol
				}
			}
		}

		// update screen
		printScreen()

		// check game over
		if numDots == 0 || lives <= 0 {
			// If dead, print the death emoji
			if lives == 0 {
				moveCursor(player.row, player.col)
				fmt.Print(cfg.Death)
				moveCursor(len(maze)+2, 0)
			}
			break
		}

		// repeat
		time.Sleep(150 * time.Millisecond)
	}
}
