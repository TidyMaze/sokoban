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

const TARGET_CELL = "*"

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
			sortBoxesForStableHash(newBoxes[:boxCount])
			return newBoxes
		}
	}
	return newBoxes
}

func goTo(direction Direction, grid Grid, state State) State {
	destination := getNeighbor(direction, state.pusher)

	if isWall(grid, destination) {
		return state
	} else if isBox(state.boxes, state.boxCount, destination) {
		afterBoxCoord := getNeighbor(direction, destination)
		if afterBoxIsOccupied(grid, afterBoxCoord, state) {
			return state
		} else {
			return moveByPushingBox(destination, state, afterBoxCoord)
		}
	} else {
		return moveToEmptyCell(destination, state)
	}
}

func moveToEmptyCell(destination Coord, state State) State {
	return State{
		pusher:   destination,
		boxes:    state.boxes,
		boxCount: state.boxCount,
	}
}

func moveByPushingBox(destination Coord, state State, afterBoxCoord Coord) State {
	return State{
		pusher:   destination,
		boxes:    moveBox(state.boxes, state.boxCount, destination, afterBoxCoord),
		boxCount: state.boxCount,
	}
}

func afterBoxIsOccupied(grid Grid, afterBoxCoord Coord, state State) bool {
	return isWall(grid, afterBoxCoord) || isBox(state.boxes, state.boxCount, afterBoxCoord)
}

func sortBoxesForStableHash(coords []Coord) {
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
	return getGrid(grid, box) != TARGET_CELL &&
		touchingWallVertical(grid, box) &&
		touchingWallHorizontal(grid, box)
}

func touchingWallVertical(grid Grid, box Coord) bool {
	return isWall(grid, getNeighbor(Up, box)) || isWall(grid, getNeighbor(Down, box))
}

func touchingWallHorizontal(grid Grid, box Coord) bool {
	return isWall(grid, getNeighbor(Left, box)) || isWall(grid, getNeighbor(Right, box))
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
	candidate, done := findBestActionIterativeDeepening(grid, state)
	if done {
		return candidate
	}

	panic("no solution found")
}

func findBestActionIterativeDeepening(grid Grid, state State) (Candidate, bool) {
	for maxDepth := 100; maxDepth <= 400; maxDepth += 100 {
		candidate, done := findBestActionWithMaxDepth(grid, state, maxDepth)
		if done {
			return candidate, true
		}
	}
	return Candidate{}, false
}

func findBestActionWithMaxDepth(grid Grid, state State, maxDepth int) (Candidate, bool) {
	seenStates := buildSeenMap()
	candidateHeap := buildCandidatesQueue()

	initState := buildInitialCandidate(grid, state, &candidateHeap)

	markStateAsSeen(seenStates, initState.state)

	for thereAreStillCandidates(&candidateHeap) {
		candidate := getBestCandidate(&candidateHeap)
		if won(candidate, candidate.state) {
			storeWin(candidate, seenStates, candidateHeap)
			return *candidate, true
		}

		if !reachedMaxDepth(candidate, maxDepth) {
			for _, d := range directions {
				exploreInDirection(grid, d, candidate.state, seenStates, candidate, maxDepth, &candidateHeap)
			}
		}
	}
	return Candidate{}, false
}

func buildCandidatesQueue() CandidateHeap {
	candidateHeap := make(CandidateHeap, 0, 10000)
	heap.Init(&candidateHeap)
	return candidateHeap
}

func buildSeenMap() map[State]struct{} {
	return make(map[State]struct{}, 300000)
}

func buildInitialCandidate(grid Grid, state State, candidates *CandidateHeap) Candidate {
	candidate := Candidate{
		actions: []Direction{},
		score:   scoreState(grid, state),
		state:   state,
	}

	sortBoxesForStableHash(candidate.state.boxes[:candidate.state.boxCount])

	heap.Push(candidates, &candidate)
	return candidate
}

func thereAreStillCandidates(candidates *CandidateHeap) bool {
	return len(*candidates) > 0
}

func getBestCandidate(candidates *CandidateHeap) *Candidate {
	return heap.Pop(candidates).(*Candidate)
}

func storeWin(c *Candidate, seenStates map[State]struct{}, candidates CandidateHeap) {
	log("won", c)
	solution = c.actions[1:]
	log("seenStates len", len(seenStates))
	log("heap cap", cap(candidates))
}

func won(c *Candidate, candidateState State) bool {
	return c.score == candidateState.boxCount
}

func reachedMaxDepth(c *Candidate, maxDepth int) bool {
	return len(c.actions) >= maxDepth
}

func exploreInDirection(grid Grid, d Direction, candidateState State, seenStates map[State]struct{}, c *Candidate, maxDepth int, candidates *CandidateHeap) {
	newState := goTo(d, grid, candidateState)
	if discoveredNewValidState(newState, candidateState, seenStates) {
		markStateAsSeen(seenStates, newState)
		if !stateIsLost(grid, newState) {
			newCandidate := buildNewCandidate(grid, newState, c, maxDepth, d)
			heap.Push(candidates, &newCandidate)
		}
	}
}

func discoveredNewValidState(newState State, candidateState State, seenStates map[State]struct{}) bool {
	return pusherMoved(newState, candidateState) && !isStateSeen(seenStates, newState)
}

func pusherMoved(newState State, candidateState State) bool {
	return newState.pusher != candidateState.pusher
}

func isStateSeen(seenStates map[State]struct{}, newState State) bool {
	_, childSeen := seenStates[newState]
	return childSeen
}

func markStateAsSeen(seenStates map[State]struct{}, newState State) {
	seenStates[newState] = struct{}{}
}

func buildNewCandidate(grid Grid, newState State, candidate *Candidate, maxDepth int, direction Direction) Candidate {
	return Candidate{
		actions: appendAction(&candidate.actions, maxDepth, direction),
		score:   scoreState(grid, newState),
		state:   newState,
	}
}

func appendAction(actions *[]Direction, newCap int, element Direction) []Direction {
	newActions := make([]Direction, len(*actions), newCap)
	copy(newActions, *actions)
	newActions = append(newActions, element)
	return newActions
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
