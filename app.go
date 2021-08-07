package main

import (
	"bufio"
	"container/heap"
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

type State struct {
	pusher   Coord
	boxes    [5]Coord
	boxCount int
}

type Candidate struct {
	actions []Direction
	score   int
	state   State
}

type Puzzle struct {
	rawGrid    string
	boxes      [5]Coord
	boxCount   int
	startCoord Coord
}

type CandidateHeap []*Candidate

func (h CandidateHeap) Len() int { return len(h) }

/**
WTF we pop the last element so the last should be the highest, not the lowest!
Yet this impl works a lot better
*/
func (h CandidateHeap) Less(i, j int) bool { return h[i].score > h[j].score }
func (h CandidateHeap) Swap(i, j int)      { h[i], h[j] = h[j], h[i] }

func (h *CandidateHeap) Push(x interface{}) {
	*h = append(*h, x.(*Candidate))
}

func (h *CandidateHeap) Pop() interface{} {
	old := *h
	n := len(old)
	x := old[n-1]
	old[n-1] = nil
	*h = old[0 : n-1]
	return x
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

func contains(s [5]Coord, boxCount int, e Coord) bool {
	for i := 0; i < boxCount; i++ {
		if s[i] == e {
			return true
		}
	}
	return false
}

func isWall(grid Grid, coord Coord) bool {
	if coord.y < 0 || coord.x < 0 {
		panic(fmt.Sprintf("negative coord %v", coord))
	}
	return grid[coord.y][coord.x] == "#"
}

func isBox(boxes [5]Coord, boxCount int, coord Coord) bool {
	return contains(boxes, boxCount, coord)
}

func moveBox(boxes [5]Coord, boxCount int, from Coord, to Coord) [5]Coord {
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
	} else if isBox(state.boxes, state.boxCount, destination) {
		// check if next is available if cell is box
		afterBoxCoord := getNeighbor(direction, destination)
		if isWall(grid, afterBoxCoord) || isBox(state.boxes, state.boxCount, afterBoxCoord) {
			// cannot push box if next is wall or another box
			return state
		} else {
			// can push box, copy and replace boxes list
			return State{
				pusher:   destination,
				boxes:    moveBox(state.boxes, state.boxCount, destination, afterBoxCoord),
				boxCount: state.boxCount,
			}
		}
	} else {
		// can move if empty cell
		return State{
			pusher:   destination,
			boxes:    state.boxes,
			boxCount: state.boxCount,
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

func hashCoords(coords [5]Coord, boxCount int) int {
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
	hash = 89*hash + hashCoords(state.boxes, state.boxCount)
	//log("- hash with the box", hash)

	hash = 41*hash + hashCoord(state.pusher)
	hash = 89*hash + hashCoords(state.boxes, state.boxCount)
	return hash
}

func Equal(a, b [5]Coord, boxCount int) bool {
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
	return state1.pusher == state2.pusher && Equal(state1.boxes, state2.boxes, state1.boxCount)
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
	for i := 0; i < state.boxCount; i++ {
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
	for i := 0; i < state.boxCount; i++ {
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
	seenStates := make(map[int]bool, 20000000)

	const MAX_NEW_CANDIDATES = 10
	const MAX_DEPTH = 400

	internalHeap := make(CandidateHeap, 0, 500000)
	candidates := &internalHeap
	heap.Init(candidates)

	initState := Candidate{
		actions: []Direction{},
		score:   scoreState(grid, state),
		state:   state,
	}
	heap.Push(candidates, &initState)
	seenStates[hashState(initState.state)] = true

	for len(*candidates) > 0 {
		// log("candidates", candidates)
		// c := candidates[len(candidates)-1]
		c := heap.Pop(candidates).(*Candidate)

		if (len(*candidates) % 100000) == 0 {
			log("len candidates", fmt.Sprintf("%d candidates: %v seen %d", len(*candidates), c, len(seenStates)))
		}
		// log("len candidates", fmt.Sprintf("%d seen %v", len(candidates), len(seenStates)))
		// candidates[0] = Candidate{}
		//candidates = candidates[:len(candidates)-1]
		// candidates.Remove(ci)

		// win!
		if c.score == c.state.boxCount {
			log("won", c)
			solution = c.actions[1:]
			log("seenStates length", len(seenStates))
			return *c
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
					hChild := hashState(newState)
					// log("hChild", hChild)

					_, childSeen := seenStates[hChild]

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
						seenStates[hChild] = true
						if !stateIsLost(grid, newState) {
							newHistory := make([]Direction, len(c.actions), MAX_DEPTH)
							copy(newHistory, c.actions)
							newHistory = append(newHistory, d)
							score := scoreState(grid, newState)
							newCandidate := Candidate{
								actions: newHistory,
								score:   score,
								state:   newState,
							}

							heap.Push(candidates, &newCandidate)
						}
					}
				}
			}

			//sort.Slice(candidates, func(i, j int) bool {
			//	return candidates[i].score < candidates[j].score
			//})

			const MAX_BUFFER = 100000
			if len(internalHeap) > MAX_BUFFER {
				internalHeap = internalHeap[:MAX_BUFFER]
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
	if len(os.Args) > 1 && strings.Compare(os.Args[1], "profile") == 0 {
		mainProfile()
	} else {
		mainCg()
	}
}

func mainCg() {
	scanner := bufio.NewScanner(os.Stdin)
	scanner.Buffer(make([]byte, 1000000), 1000000)

	var width, height, boxCount int
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
			pusher:   p,
			boxes:    boxes,
			boxCount: boxCount,
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

	f, err := os.Create("out/out.prof")
	if err != nil {
		log("could not create CPU profile: ", err)
	}
	defer f.Close()

	runtime.SetCPUProfileRate(500)

	if err := pprof.StartCPUProfile(f); err != nil {
		log("could not start CPU profile: ", err)
	}

	defer pprof.StopCPUProfile()

	easyPuzzle := Puzzle{
		`..#######
..#.....#
.##.###.#
##....#.#
#..*****#
#.....#.#
#..##...#
#..######
####.....`,
		[5]Coord{{x: 2, y: 4}, {x: 3, y: 4}, {x: 4, y: 4}, {x: 5, y: 4}, {x: 6, y: 4}},
		5,
		Coord{7, 4},
	}

	mediumPuzzle := Puzzle{
		`####....
#..#....
#..#####
#......#
##.*.*.#
#..*#.##
#*....#.
#######.`,
		[5]Coord{{x: 3, y: 3}, {x: 2, y: 4}, {x: 3, y: 4}, {x: 5, y: 4}},
		4,
		Coord{4, 6},
	}

	mediumPuzzle2 := Puzzle{
		`.####....
.#..#....
.#..#####
##..#...#
#....*..#
#.#.*.###
#..**.#..
####..#..
...####..`,
		[5]Coord{{x: 2, y: 4}, {x: 4, y: 4}, {x: 5, y: 4}, {x: 3, y: 5}},
		4,
		Coord{5, 3},
	}

	hardPuzzle := Puzzle{
		`#########.
#.......#.
#.#####.#.
#.#...#.##
#.#.#.#..#
#.#*.**..#
#.*......#
####.*.###
...#..##..
...####...`,
		[5]Coord{{x: 5, y: 5}, {x: 2, y: 6}, {x: 5, y: 6}, {x: 4, y: 7}, {x: 5, y: 7}},
		5,
		Coord{4, 8},
	}

	puzzles := []Puzzle{hardPuzzle, mediumPuzzle2, mediumPuzzle, easyPuzzle}
	//puzzles := []Puzzle{easyPuzzle, mediumPuzzle, mediumPuzzle2}

	for _, puzzle := range puzzles {
		state := State{
			pusher:   puzzle.startCoord,
			boxes:    puzzle.boxes,
			boxCount: puzzle.boxCount,
		}

		grid := parseGrid(puzzle.rawGrid)

		for i := 0; i < 3; i++ {
			best := findBestAction(grid, state)
			log("best profile", best)
			log("actions len", len(best.actions))
		}
	}

}
