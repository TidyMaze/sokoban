package main

import (
	"bufio"
	"fmt"
	"os"
	"runtime"
	"runtime/pprof"
	"strings"
)

type Cell = string
type Grid = [][]Cell

type Coord struct {
	x, y int
}

var boxCount = 0

type State struct {
	pusher Coord
	boxes  [5]Coord
}

type Candidate struct {
	actions []Direction
	score   int
	state   State
}

func log(msg string, v interface{}) {
	fmt.Fprintf(os.Stderr, "%s: %+v\n", msg, v)
}

func showGrid(grid [][]Cell) string {
	res := ""
	for i := 0; i < len(grid); i++ {
		for j := 0; j < len(grid[i]); j++ {
			res += grid[i][j]
		}
		res += "\n"
	}
	return res
}

type Direction int8

const (
	Up Direction = iota
	Down
	Left
	Right
)

var directions = [4]Direction{Up, Down, Left, Right}

func showDir(direction Direction) string {
	return [4]string{"U", "D", "L", "R"}[direction]
}

func getNeighbor(direction Direction, coord Coord) Coord {
	if direction == Up {
		return Coord{coord.x, coord.y - 1}
	} else if direction == Down {
		return Coord{coord.x, coord.y + 1}
	} else if direction == Left {
		return Coord{coord.x - 1, coord.y}
	} else if direction == Right {
		return Coord{coord.x + 1, coord.y}
	}
	panic("No case for dir " + fmt.Sprint(direction))
}

func contains(s [5]Coord, e Coord) bool {
	for i := 0; i < boxCount; i++ {
		if s[i] == e {
			return true
		}
	}
	return false
}

func isWall(grid Grid, coord Coord) bool {
	// if coord.y < 0 || coord.x < 0 {
	// 	panic(fmt.Sprintf("negative coord %v", coord))
	// }
	return grid[coord.y][coord.x] == "#"
}

func isBox(boxes [5]Coord, coord Coord) bool {
	return contains(boxes, coord)
}

func moveBox(boxes [5]Coord, from Coord, to Coord) [5]Coord {
	newBoxes := boxes
	for i := 0; i < boxCount; i++ {
		if newBoxes[i] == from {
			newBoxes[i] = to
			return newBoxes
		}
	}
	return newBoxes
}

func goTo(direction Direction, grid Grid, state State) State {
	// if state.pusher.x == 0 && state.pusher.y == 0 {
	// 	panic(fmt.Sprintf("null pusher coord %v %v", direction, state.pusher))
	// }
	destination := getNeighbor(direction, state.pusher)

	// if destination.y < 0 || destination.x < 0 {
	// 	panic(fmt.Sprintf("negative coord %v %v %v", destination, direction, state.pusher))
	// }

	if isWall(grid, destination) {
		// cannot move if wall
		return state
	} else if isBox(state.boxes, destination) {
		// check if next is available if cell is box
		afterBoxCoord := getNeighbor(direction, destination)
		if isWall(grid, afterBoxCoord) || isBox(state.boxes, afterBoxCoord) {
			// cannot push box if next is wall or another box
			return state
		} else {
			// can push box, copy and replace boxes list
			return State{
				pusher: destination,
				boxes:  moveBox(state.boxes, destination, afterBoxCoord),
			}
		}
	} else {
		// can move if empty cell
		return State{
			pusher: destination,
			boxes:  state.boxes,
		}
	}
}

func hashCoord(coord Coord) int {
	hash := 11
	hash = 43*hash + coord.x
	hash = 43*hash + coord.y
	hash = 43*hash + coord.x
	hash = 43*hash + coord.y
	return hash
}

func hashCoords(coords [5]Coord) int {
	hash := 47
	for i := 0; i < boxCount; i++ {
		hash = 97*hash + hashCoord(coords[i])
		hash = 97*hash + hashCoord(coords[i])
	}
	return hash
}

func hashState(state State) int {
	//log("generating hash for", state)
	hash := 17
	hash = 41*hash + hashCoord(state.pusher)
	//log("- hash with the pusher", hash)
	hash = 89*hash + hashCoords(state.boxes)
	//log("- hash with the box", hash)

	hash = 41*hash + hashCoord(state.pusher)
	hash = 89*hash + hashCoords(state.boxes)
	return hash
}

func Equal(a, b [5]Coord) bool {
	if len(a) != len(b) {
		return false
	}
	for i := 0; i < boxCount; i++ {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func sameState(state1 State, state2 State) bool {
	return state1.pusher == state2.pusher && Equal(state1.boxes, state2.boxes)
}

// func copyState(state State) State {
// 	copiedBoxes := make([]Coord, len(state.boxes))
// 	copy(copiedBoxes, state.boxes)
// 	return State{
// 		pusher: state.pusher,
// 		boxes:  copiedBoxes,
// 	}
// }

func scoreState(grid Grid, state State) int {
	count := 0
	for i := 0; i < boxCount; i++ {
		if grid[state.boxes[i].y][state.boxes[i].x] == "*" {
			count += 1
		}
	}
	return count
}

func getGrid(grid Grid, coord Coord) Cell {
	return grid[coord.y][coord.x]
}

func boxStuck(grid Grid, box Coord) bool {
	isTarget := getGrid(grid, box) == "*"

	if isTarget {
		return false
	}

	touchVertical := isWall(grid, getNeighbor(Up, box)) || isWall(grid, getNeighbor(Down, box))

	if !touchVertical {
		return false
	}

	touchHorizontal := isWall(grid, getNeighbor(Left, box)) || isWall(grid, getNeighbor(Right, box))

	if !touchHorizontal {
		return false
	}

	return true
}

func stateIsLost(grid Grid, state State) bool {
	for i := 0; i < boxCount; i++ {
		if boxStuck(grid, state.boxes[i]) {
			return true
		}
	}
	return false
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func findBestAction(grid Grid, state State) Candidate {
	seenStates := make(map[State]bool, 5000000)

	const MAX_NEW_CANDIDATES = 10
	const MAX_DEPTH = 400

	candidates := make([]Candidate, 0, 1000)
	initState := Candidate{
		actions: []Direction{},
		score:   scoreState(grid, state),
		state:   state,
	}
	candidates = append(candidates, initState)
	seenStates[initState.state] = true

	for len(candidates) > 0 {
		// log("candidates", candidates)
		// c := candidates[len(candidates)-1]
		c := candidates[len(candidates)-1]

		if (len(seenStates) % 100000) == 0 {
			log("len candidates", fmt.Sprintf("%d candidates: %v seen %d", len(candidates), c, len(seenStates)))
		}
		// log("len candidates", fmt.Sprintf("%d seen %v", len(candidates), len(seenStates)))
		// candidates[0] = Candidate{}
		candidates = candidates[:len(candidates)-1]
		// candidates.Remove(ci)

		// win!
		if c.score == boxCount {
			log("won", c)
			solution = c.actions[1:]
			log("seenStates length", len(seenStates))
			return c
		}

		if len(c.actions) < MAX_DEPTH {

			for _, d := range directions {
				// if c.state.pusher.x == 0 && c.state.pusher.y == 0 {
				// 	panic(fmt.Sprintf("null oldstate coord %v %v %v", d, c.state.pusher, c))
				// }

				newState := goTo(d, grid, c.state)

				// if newState.pusher.x == 0 && newState.pusher.y == 0 {
				// 	panic(fmt.Sprintf("null newState coord %v %v", d, newState.pusher))
				// }

				if newState.pusher != c.state.pusher {
					// hChild := hashState(newState)
					// log("hChild", hChild)

					_, childSeen := seenStates[newState]

					// hStored := hashState(storedState)

					// if childSeen && !sameState(storedState, newState) {
					// 	log("hash previous", hStored)
					// 	log("state previous", storedState)
					// 	log("hash new", hChild)
					// 	log("state new", newState)
					// 	log("hash1", hashState(storedState))
					// 	log("hash2", hashState(newState))
					// 	panic(fmt.Sprintf("%v and %v had the same hash %d", storedState, newState, hChild))
					// }

					if !childSeen {
						if !stateIsLost(grid, newState) {
							seenStates[newState] = true
							newHistory := make([]Direction, len(c.actions), MAX_DEPTH)
							copy(newHistory, c.actions)
							newHistory = append(newHistory, d)
							score := scoreState(grid, newState)
							newCandidate := Candidate{
								actions: newHistory,
								score:   score,
								state:   newState,
							}

							// if len(candidates) == cap(candidates) {
							// 	log("too small", MAX_NEW_CANDIDATES)
							// }

							candidates = append(candidates, newCandidate)
						}

					}
				}
			}

			// sort.Slice(candidates, func(i, j int) bool {
			// 	return candidates[i].score < candidates[j].score
			// })

			const MAX_BUFFER = 400
			if len(candidates) > MAX_BUFFER {
				candidates = candidates[len(candidates)-MAX_BUFFER:]
			}
		}
	}

	panic("no solution found")
}

// func randInt(min int, max int) int {
// 	return rand.Intn(max-min) + min
// }

var solution = []Direction{}

func main() {
	//mainCg()

	mainProfile()
}

func mainCg() {
	scanner := bufio.NewScanner(os.Stdin)
	scanner.Buffer(make([]byte, 1000000), 1000000)

	var width, height int
	scanner.Scan()
	fmt.Sscan(scanner.Text(), &width, &height, &boxCount)

	grid := make(Grid, 0, height)
	for i := 0; i < height; i++ {
		scanner.Scan()
		line := scanner.Text()
		grid = append(grid, strings.Split(line, ""))
	}

	log("grid", showGrid(grid))

	for {
		var pusherX, pusherY int
		scanner.Scan()
		fmt.Sscan(scanner.Text(), &pusherX, &pusherY)
		p := Coord{pusherX, pusherY}

		boxes := [5]Coord{}
		for i := 0; i < boxCount; i++ {
			var boxX, boxY int
			scanner.Scan()
			fmt.Sscan(scanner.Text(), &boxX, &boxY)
			boxes[i] = Coord{boxX, boxY}
		}

		log("boxes", boxes)

		state := State{
			pusher: p,
			boxes:  boxes,
		}

		if len(solution) > 0 {
			log("solution", solution)
			fmt.Println(showDir(solution[0]))
			solution = solution[1:]
		} else {
			bestCandidate := findBestAction(grid, state)
			log("best", bestCandidate)
			// fmt.Fprintln(os.Stderr, "Debug messages...")
			fmt.Println(showDir(bestCandidate.actions[0]))
		}

	}
}

func parseGrid(raw string) Grid {
	lines := strings.Split(raw, "\n")
	res := [][]string{}
	for _, l := range lines {
		res = append(res, strings.Split(l, ""))
	}
	return res
}

func mainProfile() {
	log("starting CPU profile", true)

	f, err := os.Create("out.prof")
	if err != nil {
		log("could not create CPU profile: ", err)
	}
	defer f.Close()

	runtime.SetCPUProfileRate(500)

	if err := pprof.StartCPUProfile(f); err != nil {
		log("could not start CPU profile: ", err)
	}

	defer pprof.StopCPUProfile()

	gridRaw := `..#######
..#.....#
.##.###.#
##....#.#
#..*****#
#.....#.#
#..##...#
#..######
####.....`

	state := State{
		pusher: Coord{7, 4},
		boxes:  [5]Coord{{x: 2, y: 4}, {x: 3, y: 4}, {x: 4, y: 4}, {x: 5, y: 4}, {x: 6, y: 4}},
	}

	boxCount = 5

	grid := parseGrid(gridRaw)

	for i := 0; i < 3; i++ {
		best := findBestAction(grid, state)
		log("best profile", best)
		log("actions len", len(best.actions))
	}

}
