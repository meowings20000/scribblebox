// Deterministic grid world internals: the mutable state, spawning (with the
// stacking rule that makes a tower), snapshots, resets, the sandbox used by
// hints, and the outside judge.
//
// Every exported method locks the world's own mutex, so one world can be stepped
// and snapshotted from different goroutines safely (the HTTP layer does exactly
// that).
package world

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"

	"backend/lexicon"
)

// maxEvents is how many happenings a world remembers. The frontend shows the
// tail of this list, so it is a display budget, not a log.
const maxEvents = 24

// goalPlan is the internal, unexported knowledge the engine needs in order to
// referee a puzzle: which entities are the goal, and which way the player should
// be walking. It is never serialised.
type goalPlan struct {
	kind      string
	starID    int
	candleIDs []int
	chestID   int
	// cross puzzles
	farSide   int // +1 when the far bank is to the right, -1 when to the left
	waterMinX int
	waterMaxX int
	bankY     int
}

// hintPlan is the authored four-choice hint: where an object is meant to be
// spawned, the pool of phrases that provably do NOT solve the puzzle from there,
// and the friendly line that goes with the offer.
type hintPlan struct {
	placement HintPlacement
	decoys    []string
	rationale string
}

// worldState is everything that changes while a puzzle is played. Keeping it in
// one struct is what makes Reset exact: the initial state is a value copy, so
// restoring it cannot drift out of step with the fields.
//
// burn, fell, wade and facing are bookkeeping the wire form does not need: how
// long each thing has been on fire, what moved down this tick (so magical things
// can float), the half-speed water clock, and which way the player last walked.
type worldState struct {
	terrain  []int
	entities []Entity
	player   Player
	goalMet  bool
	solved   bool
	judged   bool
	judge    string
	ticks    int
	events   []string
	notebook []NotebookEntry
	nextID   int
	burn     map[int]int
	fell     map[int]bool
	wade     int
	facing   int
}

// World is one playable puzzle with its own state. Methods are safe for
// concurrent use.
type World struct {
	mu     sync.Mutex
	id     string
	puzzle Puzzle
	plan   goalPlan
	hint   hintPlan
	layout layout
	state  worldState
	start  worldState
}

// worldCounter hands out unique world ids. Two worlds of the same puzzle must
// never share an id: the HTTP store keys sessions by it.
var worldCounter atomic.Int64

// New builds a fresh world for a built-in puzzle.
func New(puzzleID string) (*World, error) {
	trimmed := strings.TrimSpace(puzzleID)
	for _, template := range builtinPuzzles() {
		if template.puzzle.ID != trimmed {
			continue
		}
		puzzle := template.puzzle
		puzzle.Solutions = template.solutions
		return newWorld(puzzle, template.plan, template.hint, template.chars), nil
	}
	return nil, fmt.Errorf("no puzzle called %q", trimmed)
}

// newWorld assembles a world from an already validated puzzle, goal plan, hint
// plan and layout. Both New and NewFromSpec end up here, which is what makes a
// spec world behave exactly like a built-in one.
func newWorld(puzzle Puzzle, plan goalPlan, hint hintPlan, chars layout) *World {
	w := &World{
		id:     fmt.Sprintf("%s-%d", puzzle.ID, worldCounter.Add(1)),
		puzzle: puzzle,
		plan:   plan,
		hint:   hint,
		layout: chars,
	}
	w.state = w.initialState()
	w.start = w.state.clone()
	return w
}

// ID is the unique world id the HTTP layer stores and looks up.
func (w *World) ID() string {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.id
}

// Puzzles returns the built-in puzzles, without their answer keys. Solutions is
// tagged json:"-" as well, but building the copy here means a caller cannot
// reach the answers even by reflecting over the slice.
func Puzzles() []Puzzle {
	templates := builtinPuzzles()
	out := make([]Puzzle, 0, len(templates))
	for _, template := range templates {
		out = append(out, template.puzzle.public())
	}
	return out
}

// public returns the puzzle with its answer key removed.
func (p Puzzle) public() Puzzle {
	return Puzzle{
		ID:       p.ID,
		Title:    p.Title,
		Brief:    p.Brief,
		Hint:     p.Hint,
		GoalKind: p.GoalKind,
		Width:    p.Width,
		Height:   p.Height,
	}
}

// clone deep-copies a state so a sandbox or a Reset cannot alias the live world.
func (s worldState) clone() worldState {
	out := s
	out.terrain = append([]int{}, s.terrain...)
	out.entities = make([]Entity, len(s.entities))
	for i, e := range s.entities {
		copied := e
		copied.Object.Tags = append([]string{}, e.Object.Tags...)
		copied.Object.Modifiers = append([]string{}, e.Object.Modifiers...)
		out.entities[i] = copied
	}
	out.events = append([]string{}, s.events...)
	out.notebook = append([]NotebookEntry{}, s.notebook...)
	out.burn = make(map[int]int, len(s.burn))
	for id, ticks := range s.burn {
		out.burn[id] = ticks
	}
	out.fell = make(map[int]bool, len(s.fell))
	for id, moved := range s.fell {
		out.fell[id] = moved
	}
	return out
}

// Spawn puts an object in the world at (x, y), reading the phrase the way the
// notebook reads it. It returns the object that was created, a line of flavour
// for the notebook, whether the noun was improvised, and an error.
//
// Placement rules, in order:
//  1. a phrase the book cannot read at all (empty, punctuation only) is an error;
//  2. coordinates outside the grid, or a footprint that would not fit, are an
//     error naming the problem;
//  3. solid terrain (ground, trunk, wall) is an error: you cannot draw inside
//     the ground;
//  4. anything already there is stood on: the object is pushed up until it no
//     longer overlaps, which is what makes "spawn three boxes in one column"
//     build a tower rather than a pile of one.
func (w *World) Spawn(phrase string, x, y int) (lexicon.Object, string, bool, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.spawnLocked(phrase, x, y, true)
}

// spawnLocked is the whole of Spawn; record says whether the notebook and the
// events should remember it (the sandbox does not).
func (w *World) spawnLocked(phrase string, x, y int, record bool) (lexicon.Object, string, bool, error) {
	obj, ok := lexicon.Parse(phrase)
	if !ok {
		return lexicon.Object{}, "", false, fmt.Errorf("the notebook cannot read %q", strings.TrimSpace(phrase))
	}
	obj = applyAffordances(obj)
	width, height := footprint(obj)

	if x < 0 || y < 0 || x >= w.puzzle.Width || y >= w.puzzle.Height {
		return lexicon.Object{}, "", false, fmt.Errorf("there is nothing at (%d, %d): the world is %d wide and %d tall", x, y, w.puzzle.Width, w.puzzle.Height)
	}
	if x+width-1 >= w.puzzle.Width || y-height+1 < 0 {
		return lexicon.Object{}, "", false, fmt.Errorf("a %s needs more room than (%d, %d) has", obj.Name, x, y)
	}

	placed := point{x, y}
	candidate := Entity{Object: obj, X: x, Y: y, W: width, H: height}
	if w.footprintSolid(candidate) {
		return lexicon.Object{}, "", false, fmt.Errorf("that spot is solid: try an empty cell")
	}
	// Stand on whatever is already there.
	for w.footprintTaken(candidate) {
		placed.Y--
		candidate.Y = placed.Y
		if placed.Y-height+1 < 0 {
			return lexicon.Object{}, "", false, fmt.Errorf("there is no room above (%d, %d): the stack has reached the sky", x, y)
		}
		if w.footprintSolid(candidate) {
			return lexicon.Object{}, "", false, fmt.Errorf("there is no room above (%d, %d): something solid is in the way", x, y)
		}
	}

	candidate.ID = w.state.nextID
	w.state.nextID++
	w.state.entities = append(w.state.entities, candidate)

	if record {
		w.state.notebook = append(w.state.notebook, NotebookEntry{
			Phrase:      strings.TrimSpace(phrase),
			Key:         obj.Key,
			Approximate: obj.Approximate,
			X:           x,
			Y:           y,
		})
		note := obj.Note
		if note == "" {
			note = "a " + obj.Name + " appears"
		}
		w.sayLocked(fmt.Sprintf("you draw a %s (%d, %d) - %s", obj.Name, candidate.X, candidate.Y, note))
		w.checkGoalLocked()
	}
	return obj, obj.Note, obj.Approximate, nil
}

// footprintSolid reports whether any cell of a candidate entity sits in solid
// terrain.
func (w *World) footprintSolid(candidate Entity) bool {
	for _, c := range candidate.cells() {
		if !w.inside(c) {
			return true
		}
		if isSolid(w.state.terrain[indexOf(c, w.puzzle.Width)]) {
			return true
		}
	}
	return false
}

// footprintTaken reports whether a candidate entity overlaps another entity or
// the player.
func (w *World) footprintTaken(candidate Entity) bool {
	for _, other := range w.state.entities {
		if candidate.ID != 0 && other.ID == candidate.ID {
			continue
		}
		if entitiesOverlap(candidate, other) {
			return true
		}
	}
	for _, c := range candidate.cells() {
		if c.X == w.state.player.X && c.Y == w.state.player.Y {
			return true
		}
	}
	return false
}

func entitiesOverlap(a, b Entity) bool {
	for _, ca := range a.cells() {
		for _, cb := range b.cells() {
			if ca == cb {
				return true
			}
		}
	}
	return false
}

func (w *World) inside(p point) bool {
	return p.X >= 0 && p.Y >= 0 && p.X < w.puzzle.Width && p.Y < w.puzzle.Height
}

func indexOf(p point, width int) int { return p.Y*width + p.X }

// terrainAt reads a cell, treating anything outside the grid as unbreakable
// wall, which keeps every rule from needing its own bounds check.
func (w *World) terrainAt(p point) int {
	if !w.inside(p) {
		return TerrainWall
	}
	return w.state.terrain[indexOf(p, w.puzzle.Width)]
}

func (w *World) setTerrain(p point, code int) {
	if w.inside(p) {
		w.state.terrain[indexOf(p, w.puzzle.Width)] = code
	}
}

// entityAt finds the first entity occupying a cell.
func (w *World) entityAt(p point) (int, bool) {
	for i, e := range w.state.entities {
		if e.occupies(p) {
			return i, true
		}
	}
	return -1, false
}

// entityByID finds an entity by its id.
func (w *World) entityByID(id int) (int, bool) {
	for i, e := range w.state.entities {
		if e.ID == id {
			return i, true
		}
	}
	return -1, false
}

// removeEntity drops an entity and forgets whatever the player was holding if it
// was that entity.
func (w *World) removeEntity(id int) {
	kept := w.state.entities[:0]
	for _, e := range w.state.entities {
		if e.ID == id {
			continue
		}
		kept = append(kept, e)
	}
	w.state.entities = kept
	if w.state.player.Holding == id {
		w.state.player.Holding = 0
	}
}

// sayLocked records a happening, keeping only the most recent few.
func (w *World) sayLocked(message string) {
	w.state.events = append(w.state.events, message)
	if len(w.state.events) > maxEvents {
		w.state.events = w.state.events[len(w.state.events)-maxEvents:]
	}
}

// Events returns the last few happenings, newest last.
func (w *World) Events() []string {
	w.mu.Lock()
	defer w.mu.Unlock()
	return append([]string{}, w.state.events...)
}

// Step advances the world by ticks ticks. Each tick runs the documented rule
// order exactly once; see rules.go for that order and the reasoning behind it.
func (w *World) Step(ticks int) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if ticks < 0 {
		ticks = 0
	}
	for i := 0; i < ticks; i++ {
		w.tickLocked()
	}
}

// Reset restores the puzzle to the moment it was created: same terrain, same
// entities, same player position, empty notebook, unsolved.
func (w *World) Reset() {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.state = w.start.clone()
	w.sayLocked("the notebook is wiped clean and the puzzle starts again")
}

// Snapshot is the complete wire form of the world. The answer key is absent by
// construction.
func (w *World) Snapshot() State {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.snapshotLocked()
}

func (w *World) snapshotLocked() State {
	terrain := append([]int{}, w.state.terrain...)
	entities := make([]Entity, len(w.state.entities))
	for i, e := range w.state.entities {
		copied := e
		copied.Object.Tags = append([]string{}, e.Object.Tags...)
		copied.Object.Modifiers = append([]string{}, e.Object.Modifiers...)
		entities[i] = copied
	}
	return State{
		ID:        w.id,
		Puzzle:    w.puzzle.public(),
		Width:     w.puzzle.Width,
		Height:    w.puzzle.Height,
		Terrain:   terrain,
		Entities:  entities,
		Player:    w.state.player,
		Solved:    w.state.solved,
		Judged:    w.state.judged,
		JudgeNote: w.state.judge,
		Ticks:     w.state.ticks,
		Events:    append([]string{}, w.state.events...),
		Notebook:  append([]NotebookEntry{}, w.state.notebook...),
		Goal:      w.goalLocked(),
	}
}

// AcceptJudge records that an external judge accepted the player's solution and
// marks the world solved. The engine's own goal check is untouched: a rule
// engine that cannot simulate an idea should not be able to veto it, and it
// should not be able to confirm it either, so this is the only door in.
func (w *World) AcceptJudge(reason string) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.state.solved {
		return errors.New("this puzzle is already solved")
	}
	trimmed := strings.TrimSpace(reason)
	if trimmed == "" {
		trimmed = "the judge liked it"
	}
	w.state.solved = true
	w.state.goalMet = true
	w.state.judged = true
	w.state.judge = trimmed
	w.sayLocked("the judge accepts your idea: " + trimmed)
	return nil
}

// Trial dry-runs a phrase in a sandbox copy of the world: it spawns the phrase
// at (x, y), steps until the world settles (up to trialTicks), and reports
// whether the goal is met. The real world is never touched, so a hint can be
// verified before it is offered.
func (w *World) Trial(phrase string, x, y int) (bool, error) {
	w.mu.Lock()
	sandbox := w.sandboxLocked()
	w.mu.Unlock()

	if _, _, _, err := sandbox.spawnLocked(phrase, x, y, false); err != nil {
		return false, err
	}
	sandbox.Step(trialTicks)
	return sandbox.Snapshot().Solved, nil
}

// trialTicks is how long a dry run is allowed to take: one spawn, then enough
// ticks for falling, climbing, riding and burning to finish.
const trialTicks = 40

// sandboxLocked copies the world so a dry run cannot disturb the real one.
func (w *World) sandboxLocked() *World {
	clone := &World{
		id:     w.id,
		puzzle: w.puzzle,
		plan:   w.plan,
		hint:   w.hint,
		state:  w.state.clone(),
		start:  w.start.clone(),
	}
	return clone
}

// goalLocked reads the current goal state for the wire form. An accepted judge
// verdict is reflected here: a solution an outside judge accepted is a solved
// puzzle, and the player should be told so in words.
func (w *World) goalLocked() Goal {
	if w.state.judged {
		return Goal{
			Title:    goalTitle(w.plan.kind),
			Met:      true,
			Progress: judgeProgress(w.state.judge),
		}
	}
	met, progress := w.evaluateGoalLocked()
	return Goal{
		Title:    goalTitle(w.plan.kind),
		Met:      met,
		Progress: progress,
	}
}

// checkGoalLocked runs the goal check after a mutation and latches a win. Once
// solved, a world stays solved: a candle that burns down to ash should not take
// the win away from the player who lit it.
func (w *World) checkGoalLocked() {
	met, _ := w.evaluateGoalLocked()
	if met {
		w.state.goalMet = true
		if !w.state.solved {
			w.state.solved = true
			w.sayLocked("that is the puzzle solved: " + goalTitle(w.plan.kind))
		}
		return
	}
	w.state.goalMet = false
}

// Describe renders a debug line for tests and tooling: terrain, entities, player
// and goal in one string. It is deliberately stable so a golden comparison is
// meaningful.
func (w *World) Describe() string {
	w.mu.Lock()
	defer w.mu.Unlock()
	var b strings.Builder
	fmt.Fprintf(&b, "world %s puzzle %s %dx%d ticks=%d solved=%v\n", w.id, w.puzzle.ID, w.puzzle.Width, w.puzzle.Height, w.state.ticks, w.state.solved)
	for y := 0; y < w.puzzle.Height; y++ {
		for x := 0; x < w.puzzle.Width; x++ {
			b.WriteString(strconv.Itoa(w.state.terrain[indexOf(point{x, y}, w.puzzle.Width)]))
		}
		b.WriteString("\n")
	}
	fmt.Fprintf(&b, "player (%d,%d) holding=%d onGround=%v\n", w.state.player.X, w.state.player.Y, w.state.player.Holding, w.state.player.OnGround)
	for _, e := range w.state.entities {
		fmt.Fprintf(&b, "entity %d %s at (%d,%d) %dx%d burning=%v opened=%v cut=%v powered=%v held=%v\n",
			e.ID, e.Object.Key, e.X, e.Y, e.W, e.H, e.Burning, e.Opened, e.Cut, e.Powered, e.Held)
	}
	return b.String()
}
