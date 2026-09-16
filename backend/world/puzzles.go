package world

import (
	"fmt"

	"backend/lexicon"
)

// The four built-in puzzles. They are small on purpose: the player is always a
// few steps from the goal, the obvious answer always works in one object, and
// nothing here is lethal, timed or fiddly. Every authored answer in this file is
// replayed by the test suite against the real rule engine.
//
// Terrain maps are drawn with symbols so they can be read at a glance:
//
//	. air    # ground   ~ water   T tree trunk   W wall   a ash

// layoutEntity is one object a puzzle starts with. role marks the entities the
// goal referee watches: the star, the candles, the chest.
type layoutEntity struct {
	phrase string
	x      int
	y      int
	role   string
	unlit  bool // the puzzle's candles are the candle before the match
	locked bool // a container that starts closed
}

// layout is the static shape of a puzzle: terrain, starting objects, player.
type layout struct {
	width  int
	height int
	rows   []string
	items  []layoutEntity
	player point
}

// template is a built-in puzzle plus everything the engine needs to referee it.
type template struct {
	puzzle    Puzzle
	plan      goalPlan
	hint      hintPlan
	chars     layout
	solutions []Solution
}

// builtinPuzzles returns the authored puzzles. Each call rebuilds them, so a
// caller cannot mutate a puzzle some other caller is playing.
func builtinPuzzles() []template {
	return []template{
		starInTree(),
		crossTheRiver(),
		lightTheCandles(),
		openTheChest(),
	}
}

// puzzleOneLayout is the tree puzzle's world.
func puzzleOneLayout() layout {
	return layout{
		width:  20,
		height: 10,
		rows: []string{
			"....................",
			"....................",
			"....................",
			"....................",
			"....................",
			"....................",
			".......T............",
			".......T............",
			".......T............",
			"####################",
		},
		items: []layoutEntity{
			{phrase: "star", x: 6, y: 6, role: roleStar},
		},
		player: point{3, 8},
	}
}

const (
	roleStar   = "star"
	roleCandle = "candle"
	roleChest  = "chest"
)

func starInTree() template {
	return template{
		puzzle: Puzzle{
			ID:       "star-in-tree",
			Title:    "Get the star out of the tree",
			Brief:    "The star is stuck in the tree just above your reach. Draw something to reach it.",
			Hint:     "Anything you can stand on will do: a ladder, a box, a chair, a rope. A ladder standing on the grass lifts you right to it.",
			GoalKind: GoalStar,
			Width:    20,
			Height:   10,
		},
		plan:  goalPlan{kind: GoalStar},
		hint:  hintPlan{placement: HintPlacement{X: 5, Y: 8}, decoys: []string{"cake", "pillow", "fish", "umbrella"}, rationale: "One of these gets you up to the star. Which one would you stand on?"},
		chars: puzzleOneLayout(),
		solutions: []Solution{
			{
				Description: "Stand a ladder on the grass and climb it: one object, a few ticks.",
				Phrases:     []Placement{{Phrase: "ladder", X: 5, Y: 8}},
			},
			{
				Description: "Stand a chair under the branch and climb on.",
				Phrases:     []Placement{{Phrase: "chair", X: 5, Y: 8}},
			},
			{
				Description: "Stack a box and stand on it.",
				Phrases:     []Placement{{Phrase: "box", X: 5, Y: 8}},
			},
			{
				Description: "Throw a rope over the branch and climb.",
				Phrases:     []Placement{{Phrase: "rope", X: 5, Y: 8}},
			},
			{
				Description: "Hook the branch with a grappling hook.",
				Phrases:     []Placement{{Phrase: "grappling hook", X: 5, Y: 8}},
			},
			{
				Description: "Put up scaffolding, the tall way.",
				Phrases:     []Placement{{Phrase: "scaffolding", X: 5, Y: 8}},
			},
			{
				Description: "Strap on a jetpack and let it lift you.",
				Phrases:     []Placement{{Phrase: "jetpack", X: 5, Y: 8}},
			},
			{
				Description: "Bounce up on a trampoline, which is the silly way.",
				Phrases:     []Placement{{Phrase: "trampoline", X: 4, Y: 8}},
			},
		},
	}
}

// puzzleTwoLayout is the stream puzzle's world: two banks, a three-deep stream.
func puzzleTwoLayout() layout {
	return layout{
		width:  20,
		height: 10,
		rows: []string{
			"....................",
			"....................",
			"....................",
			"....................",
			"....................",
			"....................",
			"....................",
			"....................",
			"########~~##########",
			"########~~##########",
		},
		player: point{4, 7},
	}
}

func crossTheRiver() template {
	return template{
		puzzle: Puzzle{
			ID:       "cross-the-river",
			Title:    "Cross the river",
			Brief:    "A little stream is in the way. Reach the grass on the other side.",
			Hint:     "A bridge laid over the water is the tidiest way. A boat or a plank floats just as well, and something cold freezes the stream into a path.",
			GoalKind: GoalCross,
			Width:    20,
			Height:   10,
		},
		plan:  goalPlan{kind: GoalCross, farSide: 1},
		hint:  hintPlan{placement: HintPlacement{X: 7, Y: 7}, decoys: []string{"pillow", "sandwich", "cat"}, rationale: "The stream is right there. One of these gets you across it. Which one?"},
		chars: puzzleTwoLayout(),
		solutions: []Solution{
			{
				Description: "Lay a bridge over the water.",
				Phrases:     []Placement{{Phrase: "bridge", X: 7, Y: 7}},
			},
			{
				Description: "Float across on a boat.",
				Phrases:     []Placement{{Phrase: "boat", X: 7, Y: 7}},
			},
			{
				Description: "Paddle a canoe: two cells of floating boat is exactly enough.",
				Phrases:     []Placement{{Phrase: "canoe", X: 7, Y: 7}},
			},
			{
				Description: "Roll out a raft and walk on it.",
				Phrases:     []Placement{{Phrase: "raft", X: 7, Y: 7}},
			},
			{
				Description: "Put a plank across the stream.",
				Phrases:     []Placement{{Phrase: "plank", X: 7, Y: 7}},
			},
			{
				Description: "Freeze the stream with ice and walk over.",
				Phrases:     []Placement{{Phrase: "ice", X: 8, Y: 7}},
			},
			{
				Description: "Freeze it with a snowball instead.",
				Phrases:     []Placement{{Phrase: "snowball", X: 8, Y: 7}},
			},
			{
				Description: "Hover across with a jetpack, which is the impossible-sounding one.",
				Phrases:     []Placement{{Phrase: "jetpack", X: 7, Y: 7}},
			},
		},
	}
}

// puzzleThreeLayout is the candle puzzle's world: three candles in a row.
func puzzleThreeLayout() layout {
	return layout{
		width:  20,
		height: 10,
		rows: []string{
			"....................",
			"....................",
			"....................",
			"....................",
			"....................",
			"....................",
			"....................",
			"....................",
			"....................",
			"####################",
		},
		items: []layoutEntity{
			{phrase: "candle", x: 9, y: 8, role: roleCandle, unlit: true},
			{phrase: "candle", x: 10, y: 8, role: roleCandle, unlit: true},
			{phrase: "candle", x: 11, y: 8, role: roleCandle, unlit: true},
		},
		player: point{4, 8},
	}
}

func lightTheCandles() template {
	return template{
		puzzle: Puzzle{
			ID:       "light-the-candles",
			Title:    "Light the three candles",
			Brief:    "Three candles are waiting. Draw something that makes fire, or brings light.",
			Hint:     "A match beside the first candle will do it: fire hops along the row on its own. A lamp works too, if you light all three at once.",
			GoalKind: GoalCandles,
			Width:    20,
			Height:   10,
		},
		plan:  goalPlan{kind: GoalCandles},
		hint:  hintPlan{placement: HintPlacement{X: 9, Y: 7}, decoys: []string{"ice", "water", "blanket"}, rationale: "The candles need something that lights them. Which of these would?"},
		chars: puzzleThreeLayout(),
		solutions: []Solution{
			{
				Description: "Strike a match beside the first candle.",
				Phrases:     []Placement{{Phrase: "match", X: 9, Y: 7}},
			},
			{
				Description: "Use a lighter.",
				Phrases:     []Placement{{Phrase: "lighter", X: 9, Y: 7}},
			},
			{
				Description: "Bring a torch.",
				Phrases:     []Placement{{Phrase: "torch", X: 9, Y: 7}},
			},
			{
				Description: "Draw a candle that is already alight and let it share.",
				Phrases:     []Placement{{Phrase: "flaming candle", X: 9, Y: 7}},
			},
			{
				Description: "Hold up a lamp between all three, so light reaches every wick.",
				Phrases:     []Placement{{Phrase: "lamp", X: 10, Y: 7}},
			},
			{
				Description: "Borrow the sun.",
				Phrases:     []Placement{{Phrase: "sun", X: 10, Y: 7}},
			},
			{
				Description: "Focus sunlight through a magnifying glass.",
				Phrases:     []Placement{{Phrase: "magnifying glass", X: 9, Y: 7}},
			},
		},
	}
}

// puzzleFourLayout is the chest puzzle's world.
func puzzleFourLayout() layout {
	return layout{
		width:  20,
		height: 10,
		rows: []string{
			"....................",
			"....................",
			"....................",
			"....................",
			"....................",
			"....................",
			"....................",
			"....................",
			"....................",
			"####################",
		},
		items: []layoutEntity{
			{phrase: "chest", x: 10, y: 8, role: roleChest, locked: true},
		},
		player: point{8, 8},
	}
}

func openTheChest() template {
	return template{
		puzzle: Puzzle{
			ID:       "open-the-chest",
			Title:    "Open the locked chest",
			Brief:    "That chest is locked. Draw something that opens it.",
			Hint:     "A key is the polite answer and a crowbar is the rude one. Blunt tools, a drill, even a wand will do it too.",
			GoalKind: GoalChest,
			Width:    20,
			Height:   10,
		},
		plan:  goalPlan{kind: GoalChest},
		hint:  hintPlan{placement: HintPlacement{X: 9, Y: 8}, decoys: []string{"pillow", "cake", "flower"}, rationale: "The chest is locked. One of these opens it. Which one?"},
		chars: puzzleFourLayout(),
		solutions: []Solution{
			{
				Description: "Unlock it with a key.",
				Phrases:     []Placement{{Phrase: "key", X: 9, Y: 8}},
			},
			{
				Description: "Prize it open with a crowbar.",
				Phrases:     []Placement{{Phrase: "crowbar", X: 9, Y: 8}},
			},
			{
				Description: "Use the spare master key.",
				Phrases:     []Placement{{Phrase: "master key", X: 9, Y: 8}},
			},
			{
				Description: "Pick the lock.",
				Phrases:     []Placement{{Phrase: "lockpick", X: 9, Y: 8}},
			},
			{
				Description: "Batter the hasp with a hammer.",
				Phrases:     []Placement{{Phrase: "hammer", X: 9, Y: 8}},
			},
			{
				Description: "Drill the lock out.",
				Phrases:     []Placement{{Phrase: "drill", X: 9, Y: 8}},
			},
			{
				Description: "Wave a wand at it.",
				Phrases:     []Placement{{Phrase: "wand", X: 9, Y: 8}},
			},
			{
				Description: "Blow it open: dynamite beside it, and a match to light the fuse.",
				Phrases: []Placement{
					{Phrase: "dynamite", X: 9, Y: 8},
					{Phrase: "match", X: 9, Y: 7},
				},
			},
			{
				Description: "Same idea with a bomb and a candle.",
				Phrases: []Placement{
					{Phrase: "bomb", X: 9, Y: 8},
					{Phrase: "candle", X: 9, Y: 7},
				},
			},
		},
	}
}

// parseTerrain turns the drawing into terrain codes.
func parseTerrain(rows []string, width int) ([]int, error) {
	out := make([]int, 0, len(rows)*width)
	for y, row := range rows {
		if len(row) != width {
			return nil, fmt.Errorf("terrain row %d is %d cells wide, want %d", y, len(row), width)
		}
		for _, r := range row {
			switch r {
			case '.':
				out = append(out, TerrainAir)
			case '#':
				out = append(out, TerrainGround)
			case '~':
				out = append(out, TerrainWater)
			case 'T':
				out = append(out, TerrainTrunk)
			case 'W':
				out = append(out, TerrainWall)
			case 'a':
				out = append(out, TerrainAsh)
			default:
				return nil, fmt.Errorf("unknown terrain symbol %q in row %d", string(r), y)
			}
		}
	}
	return out, nil
}

// initialState builds the starting state from the puzzle's layout: the very
// state Reset returns to, and the state every authored answer is replayed from.
func (w *World) initialState() worldState {
	state := worldState{
		terrain: mustTerrain(w.layout),
		burn:    map[int]int{},
		fell:    map[int]bool{},
		nextID:  1,
	}
	state.player = Player{X: w.layout.player.X, Y: w.layout.player.Y}
	state.player.OnGround = true

	for _, item := range w.layout.items {
		entity, err := buildEntity(item, state.nextID)
		if err != nil {
			// An authored layout that will not parse is a programming error, and
			// the tests catch it; keep the world usable rather than panicking.
			state.events = append(state.events, "the notebook skipped a "+item.phrase+": "+err.Error())
			continue
		}
		switch item.role {
		case roleStar:
			w.plan.starID = entity.ID
		case roleCandle:
			w.plan.candleIDs = append(w.plan.candleIDs, entity.ID)
		case roleChest:
			w.plan.chestID = entity.ID
		}
		state.nextID++
		state.entities = append(state.entities, entity)
	}

	if w.plan.kind == GoalCross {
		w.analyseCross(state)
	}
	state.events = append(state.events, fmt.Sprintf("puzzle: %s", goalTitle(w.plan.kind)))
	return state
}

// mustTerrain parses the authored terrain, and refuses to hand back a broken
// grid: a puzzle without a floor is not playable.
func mustTerrain(l layout) []int {
	terrain, err := parseTerrain(l.rows, l.width)
	if err != nil {
		// The built-in layouts are compile-time constants covered by tests; a
		// spec is validated long before it gets here.
		return make([]int, l.width*l.height)
	}
	return terrain
}

// buildEntity turns a layout item into a living entity.
func buildEntity(item layoutEntity, id int) (Entity, error) {
	obj, ok := lexicon.Parse(item.phrase)
	if !ok {
		return Entity{}, fmt.Errorf("the book cannot read %q", item.phrase)
	}
	obj = applyAffordances(obj)
	if item.unlit {
		obj = unlitCandle(obj)
	}
	width, height := footprint(obj)
	entity := Entity{
		ID:     id,
		Object: obj,
		X:      item.x,
		Y:      item.y,
		W:      width,
		H:      height,
	}
	if item.locked {
		entity.Opened = false
	}
	return entity, nil
}

// unlitCandle is the candle before the match: the same object the player would
// draw, with the flame taken out of it. The word book's candle is a burning one,
// which is exactly right when a player draws it and exactly wrong for a puzzle
// that asks you to light it.
func unlitCandle(obj lexicon.Object) lexicon.Object {
	tags := make([]string, 0, len(obj.Tags))
	for _, tag := range obj.Tags {
		if tag == lexicon.TagFire || tag == lexicon.TagLightSource {
			continue
		}
		tags = append(tags, tag)
	}
	obj.Tags = tags
	return obj
}

// analyseCross reads a crossing puzzle's water: where it starts and ends, and
// which way the far bank is. Everything the crossing referee needs comes from the
// terrain itself, so an authored spec gets the same treatment as the built-in.
func (w *World) analyseCross(state worldState) {
	minX, maxX := -1, -1
	for y := 0; y < w.puzzle.Height; y++ {
		for x := 0; x < w.puzzle.Width; x++ {
			if state.terrain[indexOf(point{x, y}, w.puzzle.Width)] != TerrainWater {
				continue
			}
			if minX == -1 || x < minX {
				minX = x
			}
			if x > maxX {
				maxX = x
			}
		}
	}
	w.plan.waterMinX = minX
	w.plan.waterMaxX = maxX
	w.plan.bankY = state.player.Y
	if minX != -1 && state.player.X <= minX {
		w.plan.farSide = 1
	} else if maxX != -1 {
		w.plan.farSide = -1
	}
}
