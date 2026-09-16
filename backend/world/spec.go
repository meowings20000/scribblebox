package world

import (
	"fmt"
	"strings"

	"backend/lexicon"
)

// Externally authored puzzles. An AI writes a PuzzleSpec; this file decides
// whether the engine can honestly referee it, and refuses anything it cannot.
//
// Two families of rule live here. The structural ones keep a spec playable: a
// grid, terrain codes that exist, phrases the notebook can read, and entities the
// goal kind actually needs. The easiness ones keep a spec kind: this game is for
// primary-school players, so a puzzle whose star is twenty cells away is rejected
// with an explanation rather than handed to a child.

// Bounds. Entity count and map size also bound the work a hostile spec can make
// the tick loop do.
const (
	minSpecWidth  = 12
	maxSpecWidth  = 40
	minSpecHeight = 8
	maxSpecHeight = 20
	maxSpecItems  = 25
	maxGoalRange  = 12 // Chebyshev cells from the player to the goal
	maxWaterWidth = 4  // the stream must be jumpable-by-bridge, not a lake
)

// NewFromSpec builds a world from an externally authored puzzle. It returns a
// world that behaves exactly like a built-in one: the same Spawn, Step, Act,
// Reset and Snapshot, and the same referee.
//
// The puzzle carries no answer key: Solutions stays empty, because a generated
// puzzle has no authored answers and the hint refuses to guess.
func NewFromSpec(spec PuzzleSpec) (*World, error) {
	chars, kind, err := specLayout(spec)
	if err != nil {
		return nil, err
	}
	title := strings.TrimSpace(spec.Title)
	if title == "" {
		title = "A puzzle from the notebook"
	}
	brief := strings.TrimSpace(spec.Brief)
	if brief == "" {
		brief = "Something here needs doing. Draw what you need."
	}
	hint := strings.TrimSpace(spec.Hint)
	if hint == "" {
		hint = "Look at what the puzzle is asking for and draw the obvious thing."
	}
	puzzle := Puzzle{
		ID:       specID(spec),
		Title:    title,
		Brief:    brief,
		Hint:     hint,
		GoalKind: kind,
		Width:    spec.Width,
		Height:   spec.Height,
	}
	return newWorld(puzzle, goalPlan{kind: kind}, hintPlan{}, chars), nil
}

// specID makes a stable id for a spec that did not bring one.
func specID(spec PuzzleSpec) string {
	if trimmed := strings.TrimSpace(spec.ID); trimmed != "" {
		return trimmed
	}
	return "generated-" + strings.ToLower(spec.GoalKind)
}

// specLayout validates a spec and turns it into the same layout the built-in
// puzzles use.
func specLayout(spec PuzzleSpec) (layout, string, error) {
	kind := strings.TrimSpace(spec.GoalKind)
	if err := validateSpec(spec, kind); err != nil {
		return layout{}, "", err
	}

	items := make([]layoutEntity, 0, len(spec.Entities))
	seen := map[string]bool{}
	for _, raw := range spec.Entities {
		obj, ok := lexicon.Parse(raw.Phrase)
		if !ok {
			continue // validation already refused unreadable phrases
		}
		item := layoutEntity{phrase: raw.Phrase, x: raw.X, y: raw.Y}
		switch {
		case hasObjectTag(obj, lexicon.TagCollectible):
			if !seen[roleStar] && kind == GoalStar {
				item.role = roleStar
				seen[roleStar] = true
			}
		case obj.Shape == lexicon.ShapeCandle:
			item.role = roleCandle
			item.unlit = true
		case hasObjectTag(obj, lexicon.TagContainer) && (obj.Shape == lexicon.ShapeChest || !seen[roleChest]):
			if !seen[roleChest] && kind == GoalChest {
				item.role = roleChest
				item.locked = true
				seen[roleChest] = true
			}
		}
		items = append(items, item)
	}

	rows, err := terrainRows(spec.Terrain, spec.Width, spec.Height)
	if err != nil {
		return layout{}, "", err
	}
	return layout{
		width:  spec.Width,
		height: spec.Height,
		rows:   rows,
		items:  items,
		player: point{spec.PlayerX, spec.PlayerY},
	}, kind, nil
}

func hasObjectTag(obj lexicon.Object, tag string) bool {
	for _, t := range obj.Tags {
		if t == tag {
			return true
		}
	}
	return false
}

// terrainRows groups a flat terrain slice into the drawn rows the layout uses.
func terrainRows(terrain []int, width, height int) ([]string, error) {
	symbols := map[int]rune{
		TerrainAir:    '.',
		TerrainGround: '#',
		TerrainWater:  '~',
		TerrainTrunk:  'T',
		TerrainWall:   'W',
		TerrainAsh:    'a',
	}
	rows := make([]string, 0, height)
	for y := 0; y < height; y++ {
		row := make([]rune, 0, width)
		for x := 0; x < width; x++ {
			symbol, ok := symbols[terrain[y*width+x]]
			if !ok {
				return nil, fmt.Errorf("terrain code %d is not one the world knows", terrain[y*width+x])
			}
			row = append(row, symbol)
		}
		rows = append(rows, string(row))
	}
	return rows, nil
}

// validateSpec is the whole gate. Every rejection says what is wrong and, where
// it can, how to fix it.
func validateSpec(spec PuzzleSpec, kind string) error {
	if !knownGoalKind(kind) {
		return fmt.Errorf("this puzzle asks for a %q goal, which the engine cannot referee: use one of %s", kind, strings.Join(GoalKinds, ", "))
	}
	if spec.Width < minSpecWidth || spec.Width > maxSpecWidth {
		return tooHard("make the world between %d and %d cells wide (it is %d)", minSpecWidth, maxSpecWidth, spec.Width)
	}
	if spec.Height < minSpecHeight || spec.Height > maxSpecHeight {
		return tooHard("make the world between %d and %d cells tall (it is %d)", minSpecHeight, maxSpecHeight, spec.Height)
	}
	if len(spec.Terrain) != spec.Width*spec.Height {
		return fmt.Errorf("the terrain has %d cells but a %dx%d world needs %d", len(spec.Terrain), spec.Width, spec.Height, spec.Width*spec.Height)
	}
	if len(spec.Entities) > maxSpecItems {
		return tooHard("use at most %d objects (this puzzle asks for %d)", maxSpecItems, len(spec.Entities))
	}
	for i, code := range spec.Terrain {
		if code < TerrainAir || code > TerrainAsh {
			return fmt.Errorf("terrain code %d at cell %d is not one of 0..5", code, i)
		}
	}

	grid := newGrid(spec.Terrain, spec.Width, spec.Height)
	start := point{spec.PlayerX, spec.PlayerY}
	if !grid.inside(start) {
		return fmt.Errorf("the player starts at (%d, %d), which is outside the %dx%d world", spec.PlayerX, spec.PlayerY, spec.Width, spec.Height)
	}
	if isSolid(grid.at(start)) {
		return fmt.Errorf("the player starts inside solid %s at (%d, %d): start them in the air", terrainName(grid.at(start)), spec.PlayerX, spec.PlayerY)
	}
	if !grid.hasGround() {
		return fmt.Errorf("this puzzle has no ground anywhere: a player needs something to stand on")
	}

	for i, raw := range spec.Entities {
		phrase := strings.TrimSpace(raw.Phrase)
		if phrase == "" {
			return fmt.Errorf("object %d has no name", i+1)
		}
		obj, ok := lexicon.Parse(phrase)
		if !ok {
			return fmt.Errorf("the notebook cannot read the object name %q", phrase)
		}
		width, height := footprint(obj)
		if raw.X < 0 || raw.Y < 0 || raw.X+width-1 >= spec.Width || raw.Y-height+1 < 0 {
			return fmt.Errorf("object %d (%s) sits at (%d, %d), which does not fit inside the %dx%d world", i+1, phrase, raw.X, raw.Y, spec.Width, spec.Height)
		}
		if solidFootprint(grid, obj, raw.X, raw.Y) {
			return fmt.Errorf("object %d (%s) is buried in solid ground at (%d, %d)", i+1, phrase, raw.X, raw.Y)
		}
	}

	switch kind {
	case GoalStar:
		return validateStarSpec(grid, spec, start)
	case GoalCandles:
		return validateCandlesSpec(grid, spec, start)
	case GoalChest:
		return validateChestSpec(grid, spec, start)
	case GoalCross:
		return validateCrossSpec(grid, spec, start)
	}
	return nil
}

// validateStarSpec: there is a prize, it is close enough, and it is not sealed
// away where nothing can reach it.
func validateStarSpec(grid grid, spec PuzzleSpec, start point) error {
	index, ok := findSpecEntity(spec, func(obj lexicon.Object) bool {
		return hasObjectTag(obj, lexicon.TagCollectible)
	})
	if !ok {
		return fmt.Errorf("a star puzzle needs a collectible to fetch, and this one has none")
	}
	prize := spec.Entities[index]
	goal := point{prize.X, prize.Y}
	if !grid.inside(goal) {
		return fmt.Errorf("the prize sits outside the world at (%d, %d)", prize.X, prize.Y)
	}
	if sealedIn(grid, spec, goal) {
		return tooHard("the prize is walled in at (%d, %d) with no way to touch it: leave at least one empty cell beside it", goal.X, goal.Y)
	}
	return checkGoalRange(start, goal, "the prize")
}

// validateCandlesSpec: there are candles, the first one is close enough, and it
// can be reached.
func validateCandlesSpec(grid grid, spec PuzzleSpec, start point) error {
	index, ok := findSpecEntity(spec, func(obj lexicon.Object) bool { return obj.Shape == lexicon.ShapeCandle })
	if !ok {
		return fmt.Errorf("a candles puzzle needs at least one candle to light, and this one has none")
	}
	candle := spec.Entities[index]
	goal := point{candle.X, candle.Y}
	if sealedIn(grid, spec, goal) {
		return tooHard("the candle at (%d, %d) is walled in: leave at least one empty cell beside it", goal.X, goal.Y)
	}
	return checkGoalRange(start, goal, "the first candle")
}

// validateChestSpec: there is a container to open, it is close enough, and it is
// not walled in.
func validateChestSpec(grid grid, spec PuzzleSpec, start point) error {
	index, ok := findSpecEntity(spec, func(obj lexicon.Object) bool {
		return hasObjectTag(obj, lexicon.TagContainer) || obj.Shape == lexicon.ShapeChest
	})
	if !ok {
		return fmt.Errorf("a chest puzzle needs a container to open, and this one has no chest or box")
	}
	chest := spec.Entities[index]
	goal := point{chest.X, chest.Y}
	if sealedIn(grid, spec, goal) {
		return tooHard("the chest at (%d, %d) is walled in: leave at least one empty cell beside it", goal.X, goal.Y)
	}
	return checkGoalRange(start, goal, "the chest")
}

// validateCrossSpec: there is a stream, it is narrow enough to bridge, the two
// banks really are banks, and the far side is close enough to walk to.
func validateCrossSpec(grid grid, spec PuzzleSpec, start point) error {
	band, ok := grid.waterBand()
	if !ok {
		return fmt.Errorf("a crossing puzzle needs water to cross, and this one has none")
	}
	if band.width > maxWaterWidth {
		return tooHard("the stream is %d cells wide: keep it to %d cells so a plank can reach", band.width, maxWaterWidth)
	}
	if !grid.bridgeWouldConnect(band) {
		return tooHard("the water at column %d is not a stream between two banks: put ground on both sides so a bridge has something to rest on", band.minX)
	}
	farX := band.maxX + 1
	if start.X > band.maxX {
		farX = band.minX - 1
	}
	far := point{farX, start.Y}
	return checkGoalRange(start, far, "the far bank")
}

// checkGoalRange enforces "the goal is a few steps away, not a walk".
func checkGoalRange(start, goal point, what string) error {
	distance := chebyshev(start, goal)
	if distance > maxGoalRange {
		return tooHard("move %s within %d cells of the player (it is %d cells away)", what, maxGoalRange, distance)
	}
	return nil
}

// sealedIn reports whether every neighbour of a cell is solid: a goal nothing
// can reach is not a puzzle.
func sealedIn(grid grid, spec PuzzleSpec, goal point) bool {
	if isSolid(grid.at(goal)) && grid.at(goal) != TerrainWater {
		return true
	}
	for dy := -1; dy <= 1; dy++ {
		for dx := -1; dx <= 1; dx++ {
			if dx == 0 && dy == 0 {
				continue
			}
			p := point{goal.X + dx, goal.Y + dy}
			if grid.inside(p) && !isSolid(grid.at(p)) {
				return false
			}
		}
	}
	return true
}

// findSpecEntity finds the first spec entity whose parsed object matches.
func findSpecEntity(spec PuzzleSpec, match func(lexicon.Object) bool) (int, bool) {
	for i, raw := range spec.Entities {
		obj, ok := lexicon.Parse(raw.Phrase)
		if !ok {
			continue
		}
		if match(obj) {
			return i, true
		}
	}
	return -1, false
}

func solidFootprint(grid grid, obj lexicon.Object, x, y int) bool {
	width, height := footprint(obj)
	for dy := 0; dy < height; dy++ {
		for dx := 0; dx < width; dx++ {
			if isSolid(grid.at(point{x + dx, y - dy})) {
				return true
			}
		}
	}
	return false
}

func knownGoalKind(kind string) bool {
	for _, candidate := range GoalKinds {
		if candidate == kind {
			return true
		}
	}
	return false
}

// tooHard marks a rejection that is about difficulty rather than correctness.
func tooHard(format string, args ...any) error {
	return fmt.Errorf("this puzzle would be too hard for the game's difficulty: %s", fmt.Sprintf(format, args...))
}

// grid is a read-only view of spec terrain, used by validation.
type grid struct {
	terrain []int
	width   int
	height  int
}

func newGrid(terrain []int, width, height int) grid {
	return grid{terrain: terrain, width: width, height: height}
}

func (g grid) inside(p point) bool {
	return p.X >= 0 && p.Y >= 0 && p.X < g.width && p.Y < g.height
}

// at reads a cell, treating outside as wall so validation never needs its own
// bounds check.
func (g grid) at(p point) int {
	if !g.inside(p) {
		return TerrainWall
	}
	return g.terrain[p.Y*g.width+p.X]
}

func (g grid) hasGround() bool {
	for _, code := range g.terrain {
		if code == TerrainGround {
			return true
		}
	}
	return false
}

// waterBand describes the stream of a crossing puzzle: the widest run of water
// in any row, and the rows it spans.
type waterBand struct {
	minX  int
	maxX  int
	width int
	topY  int
	botY  int
}

// waterBand measures the water. It reports the widest run, which is what a
// bridge has to span.
func (g grid) waterBand() (waterBand, bool) {
	band := waterBand{minX: -1, topY: -1, botY: -1, width: 0}
	for y := 0; y < g.height; y++ {
		runStart := -1
		for x := 0; x <= g.width; x++ {
			isWater := x < g.width && g.terrain[y*g.width+x] == TerrainWater
			if isWater && runStart == -1 {
				runStart = x
			}
			if isWater {
				continue
			}
			if runStart == -1 {
				continue
			}
			width := x - runStart
			if width > band.width {
				band.width = width
				band.minX = runStart
				band.maxX = x - 1
			}
			if band.topY == -1 || y < band.topY {
				band.topY = y
			}
			if y > band.botY {
				band.botY = y
			}
			runStart = -1
		}
	}
	return band, band.width > 0
}

// bridgeWouldConnect reports whether a bridge laid across the stream at its
// widest row would have land at both ends. This is the honest version of "the
// water must be crossable": if both ends are water or off the map, no bridge can
// help.
func (g grid) bridgeWouldConnect(band waterBand) bool {
	for y := 0; y < g.height; y++ {
		left := point{band.minX - 1, y}
		right := point{band.maxX + 1, y}
		if !g.inside(left) || !g.inside(right) {
			continue
		}
		if g.at(left) == TerrainWater || g.at(right) == TerrainWater {
			continue
		}
		return true
	}
	return false
}
