package main

import (
	"bufio"
	"container/heap"
	"fmt"
	"net/http"
	"os"
	"sort"
	"strings"
)
import _ "net/http/pprof"

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

/**
Priority queue, highest score first
*/
type CandidateHeap []*Candidate

func (h CandidateHeap) Len() int           { return len(h) }
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
			sl := newBoxes[:]

			// necessary for state hashing, boxes order is NOT important, so we can sort them
			sortCoords(sl[:boxCount])

			return newBoxes
		}
	}
	return newBoxes
}

func goTo(direction Direction, grid Grid, state State) State {
	destination := getNeighbor(direction, state.pusher)

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

func sortCoords(coords []Coord) {
	sort.SliceStable(coords, func(i, j int) bool {
		y1 := coords[i].y
		y2 := coords[j].y
		x1 := coords[i].x
		x2 := coords[j].x
		if y1 != y2 {
			return y1 < y2
		} else {
			return x1 < x2
		}
	})
}

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
	if getGrid(grid, box) == "*" {
		return false
	}

	if !isWall(grid, getNeighbor(Up, box)) && !isWall(grid, getNeighbor(Down, box)) {
		return false
	}

	if !isWall(grid, getNeighbor(Left, box)) && !isWall(grid, getNeighbor(Right, box)) {
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

func findBestAction(grid Grid, state State) Candidate {
	for maxDepth := 10; maxDepth <= 400; maxDepth += 10 {
		seenStates := make(map[State]struct{}, 300000)
		internalHeap := make(CandidateHeap, 0, 10000)
		candidates := &internalHeap
		heap.Init(candidates)

		initState := Candidate{
			actions: []Direction{},
			score:   scoreState(grid, state),
			state:   state,
		}
		heap.Push(candidates, &initState)

		s := initState.state.boxes[:initState.state.boxCount]
		sortCoords(s)

		seenStates[initState.state] = struct{}{}

		for len(*candidates) > 0 {
			c := heap.Pop(candidates).(*Candidate)

			// win!
			if c.score == c.state.boxCount {
				log("won", c)
				solution = c.actions[1:]
				log("seenStates len", len(seenStates))
				log("heap cap", cap(internalHeap))
				return *c
			}

			if len(c.actions) < maxDepth {

				for _, d := range directions {
					newState := goTo(d, grid, c.state)

					if newState.pusher != c.state.pusher {
						_, childSeen := seenStates[newState]

						if !childSeen {
							seenStates[newState] = struct{}{}
							if !stateIsLost(grid, newState) {
								score := scoreState(grid, newState)
								newCandidate := &Candidate{
									actions: make([]Direction, len(c.actions), maxDepth),
									score:   score,
									state:   newState,
								}

								copy(newCandidate.actions, c.actions)
								newCandidate.actions = append(newCandidate.actions, d)

								heap.Push(candidates, newCandidate)

							}
						}
					}
				}

				const MAX_BUFFER = 100000
				if len(*candidates) > MAX_BUFFER {
					for len(*candidates) > (MAX_BUFFER / 2) {
						heap.Remove(candidates, len(*candidates)-1)
					}
				}
			}
		}
	}

	panic("no solution found")
}

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
	go func() {
		println(http.ListenAndServe("localhost:6060", nil))
	}()

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
