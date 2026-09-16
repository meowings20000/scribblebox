package world

import (
	"encoding/json"
	"strings"
	"sync"
	"testing"
)

// One test per rule family in the physics list. Each one drives the real engine
// through the public API and reads the result out of a snapshot.

// --- Rule 4: fire -----------------------------------------------------------

func TestMatchBurnsARopeToAsh(t *testing.T) {
	w := mustWorld(t, "star-in-tree")
	mustSpawn(t, w, "rope", 12, 8)
	mustSpawn(t, w, "match", 13, 8)

	w.Step(1)
	rope, ok := findByKey(w, "rope")
	if !ok {
		t.Fatalf("the rope vanished on the first tick\n%s", describeState(w.Snapshot()))
	}
	if !rope.Burning {
		t.Fatalf("a match beside a rope should set it alight\n%s", describeState(w.Snapshot()))
	}
	if !anyEventContains(eventsOf(w), "catches fire") {
		t.Fatalf("no event said anything caught fire: %v", eventsOf(w))
	}

	w.Step(2) // three burning ticks in total
	if _, ok := findByKey(w, "rope"); ok {
		t.Fatalf("a burning rope should be ash after %d ticks\n%s", burnTicksToAsh, describeState(w.Snapshot()))
	}
	if code := terrainAtState(w.Snapshot(), 12, 8); code != TerrainAsh {
		t.Fatalf("the rope should leave ash terrain, found code %d", code)
	}
	if !anyEventContains(eventsOf(w), "burns down to ash") {
		t.Fatalf("no event mentioned ash: %v", eventsOf(w))
	}
}

func TestFireHopsAlongARowOfCandles(t *testing.T) {
	w := mustWorld(t, "light-the-candles")
	mustSpawn(t, w, "match", 11, 7) // beside the far candle only

	w.Step(1)
	if state := w.Snapshot(); state.Goal.Met {
		t.Fatalf("one match beside one candle should not light all three yet: %+v", state.Goal)
	}
	w.Step(3)
	state := w.Snapshot()
	if !state.Goal.Met || !state.Solved {
		t.Fatalf("fire should spread along the row of candles\n%s", describeState(state))
	}
}

func TestFireInWaterIsPutOut(t *testing.T) {
	w := mustWorld(t, "cross-the-river")
	// (8, 8) is water in this puzzle: a flame drawn into the stream.
	if _, _, _, err := w.Spawn("match", 8, 8); err != nil {
		t.Fatalf("spawning a match in water should be allowed: %v", err)
	}
	w.Step(1)
	if _, ok := findByKey(w, "match"); ok {
		t.Fatalf("a flame in the water should be extinguished\n%s", describeState(w.Snapshot()))
	}
	if !anyEventContains(eventsOf(w), "put out by the water") {
		t.Fatalf("no event mentioned the water: %v", eventsOf(w))
	}
}

func TestFlammableBesideWaterNeverIgnites(t *testing.T) {
	w := mustWorld(t, "cross-the-river")
	mustSpawn(t, w, "plank", 7, 7) // its far cell hangs over the water
	mustSpawn(t, w, "match", 6, 7)

	w.Step(1)
	plank, ok := findByKey(w, "plank")
	if !ok {
		t.Fatalf("a plank beside water should not burn away\n%s", describeState(w.Snapshot()))
	}
	if plank.Burning {
		t.Fatalf("a flammable thing beside water should be out, not burning")
	}
	w.Step(20)
	if _, ok := findByKey(w, "plank"); !ok {
		t.Fatalf("the wet plank should still be there after twenty ticks\n%s", describeState(w.Snapshot()))
	}
}

// --- Rule 5: cutting -------------------------------------------------------

func TestKnifeCutsARopeAndItStopsHoldingAnythingUp(t *testing.T) {
	w := mustWorld(t, "star-in-tree")
	mustSpawn(t, w, "rope", 12, 8)
	mustSpawn(t, w, "knife", 13, 8)

	w.Step(1)
	rope, ok := findByKey(w, "rope")
	if !ok {
		t.Fatalf("the knife should cut the rope, not delete it\n%s", describeState(w.Snapshot()))
	}
	if !rope.Cut {
		t.Fatalf("a knife beside a rope should cut it\n%s", describeState(w.Snapshot()))
	}
	if standable(rope) || blocking(rope) {
		t.Fatalf("a cut rope stops being climbable, a platform and solid")
	}
	if !anyEventContains(eventsOf(w), "cuts the") {
		t.Fatalf("no event mentioned the cut: %v", eventsOf(w))
	}
	// What a cut rope was holding now falls: the rule is "gravity continues
	// normally", so a box above the cut rope should end up on the ground.
	mustSpawn(t, w, "box", 12, 5)
	before := entitiesByKey(w, "box")[0]
	w.Step(10)
	after := entitiesByKey(w, "box")[0]
	if after.Y <= before.Y {
		t.Fatalf("the box above the cut rope did not fall: %+v then %+v", before, after)
	}
}

// --- Rule 3: cold and ice --------------------------------------------------

func TestColdObjectFreezesRiverWaterIntoSolidGround(t *testing.T) {
	w := mustWorld(t, "cross-the-river")
	mustSpawn(t, w, "ice", 8, 7)
	w.Step(1)
	state := w.Snapshot()
	if terrainAtState(state, 8, 8) != TerrainGround || terrainAtState(state, 9, 8) != TerrainGround {
		t.Fatalf("ice beside the stream should turn the surface into solid ice\n%s", describeState(state))
	}
	if !anyEventContains(eventsOf(w), "freezes the water") {
		t.Fatalf("no event mentioned freezing: %v", eventsOf(w))
	}
	// And the frozen stream is then walkable: the crossing answer uses exactly
	// this, so a second world proves the ice really is a bridge.
	w2 := mustWorld(t, "cross-the-river")
	mustSpawn(t, w2, "ice", 8, 7)
	w2.Step(30)
	if !w2.Snapshot().Solved {
		t.Fatalf("frozen water should be crossable\n%s", describeState(w2.Snapshot()))
	}
}

// --- Rule 7: explosions ----------------------------------------------------

func TestDynamiteBlowsChestOpenAndLeavesWallsStanding(t *testing.T) {
	rows := flatRows(20, 12)
	rows[8] = "..........W........." // a wall inside the blast radius
	w := specWorld(t, GoalChest, rows, []SpecEntity{{Phrase: "chest", X: 9, Y: 10}}, 7, 10)

	mustSpawn(t, w, "dynamite", 8, 10)
	mustSpawn(t, w, "match", 8, 9)
	mustSpawn(t, w, "coin", 10, 9)

	w.Step(2)
	state := w.Snapshot()
	for _, key := range []string{"dynamite", "match", "coin"} {
		if _, ok := findByKey(w, key); ok {
			t.Fatalf("the blast should have removed the %s\n%s", key, describeState(state))
		}
	}
	chest, ok := findByKey(w, "chest")
	if !ok {
		t.Fatalf("a blast blows a chest open rather than destroying it\n%s", describeState(state))
	}
	if !chest.Opened {
		t.Fatalf("a locked chest in the blast radius should end up open\n%s", describeState(state))
	}
	if code := terrainAtState(state, 10, 8); code != TerrainWall {
		t.Fatalf("terrain 4 walls are untouched by explosions, found %d", code)
	}
	if code := terrainAtState(state, 10, 9); code != TerrainAsh {
		t.Fatalf("the blast should leave ash where the coin was, found %d", code)
	}
	if !state.Solved {
		t.Fatalf("blowing the chest open solves the chest puzzle\n%s", describeState(state))
	}
}

func TestExplosiveNeedsAFlame(t *testing.T) {
	w := chestSpec(t, 20, 12, 9, 10, 7, 10)
	mustSpawn(t, w, "dynamite", 8, 10)
	mustSpawn(t, w, "cake", 10, 10)
	w.Step(10)
	if _, ok := findByKey(w, "cake"); !ok {
		t.Fatalf("dynamite should sit there quietly until something lights it")
	}
	if _, ok := findByKey(w, "dynamite"); !ok {
		t.Fatalf("an unlit stick of dynamite should not go off on its own")
	}
}

// --- Rule 8: power ---------------------------------------------------------

func TestBatteryThroughWireRunsALiftAndTakingTheBatteryStopsIt(t *testing.T) {
	// The chest sits directly above the player, so the walker has no sideways
	// reason to move and this test measures power, not pathfinding.
	rows := flatRows(20, 12)
	rows[8] = "..........#........."
	w := specWorld(t, GoalChest, rows, []SpecEntity{{Phrase: "chest", X: 10, Y: 7}}, 10, 10)

	mustSpawn(t, w, "battery", 11, 10)
	mustSpawn(t, w, "wire", 12, 10)
	ground := mustSpawn(t, w, "elevator", 13, 10).Y

	w.Step(1)
	lift, ok := findByKey(w, "elevator")
	if !ok {
		t.Fatal("the elevator should still be there")
	}
	if !lift.Powered {
		t.Fatalf("battery -> wire -> machine should power the machine\n%s", describeState(w.Snapshot()))
	}
	if lift.Y >= ground {
		t.Fatalf("a powered lift should rise off the ground: y went %d -> %d", ground, lift.Y)
	}
	risen := lift.Y

	// It rises as far as its supply reaches and then stops: no climbing out of
	// contact and dropping back forever.
	w.Step(3)
	if settled := entitiesByKey(w, "elevator")[0]; settled.Y != risen {
		t.Fatalf("a lift fed through a wire should stop when the wire runs out: y went %d -> %d", risen, settled.Y)
	}

	// Take the battery away. Power is recomputed from adjacency every tick, so
	// the machine must go quiet and drop.
	if err := w.Act("take"); err != nil {
		t.Fatalf("the battery is size 1 and should be pickable: %v", err)
	}
	held, ok := findByKey(w, "battery")
	if !ok || !held.Held {
		t.Fatalf("the player should be holding the battery: %+v", held)
	}
	w.Step(1)
	if lift = entitiesByKey(w, "elevator")[0]; lift.Powered {
		t.Fatalf("taking the battery away should unpower the machine\n%s", describeState(w.Snapshot()))
	}
	if !anyEventContains(eventsOf(w), "goes quiet") {
		t.Fatalf("no event mentioned the machine going quiet: %v", eventsOf(w))
	}
	w.Step(6)
	lift = entitiesByKey(w, "elevator")[0]
	if lift.Y != ground {
		t.Fatalf("an unpowered lift should drop back to the ground row %d, it is at y=%d", ground, lift.Y)
	}
	w.Step(3)
	if again := entitiesByKey(w, "elevator")[0]; again.Y != ground {
		t.Fatalf("an unpowered lift should sit still: y %d -> %d", lift.Y, again.Y)
	}
}

// --- Rules 1, 10: gravity, stacking, heavy, magical, sticky -----------------

func TestBoxInMidAirFallsAndStops(t *testing.T) {
	w := mustWorld(t, "star-in-tree")
	box := mustSpawn(t, w, "box", 12, 4)
	if box.Y != 4 {
		t.Fatalf("a spawned box should start where it was asked for, got y=%d", box.Y)
	}
	w.Step(20)
	rested := entitiesByKey(w, "box")[0]
	if rested.Y != 8 {
		t.Fatalf("the box should rest on the ground row 8, it is at y=%d\n%s", rested.Y, describeState(w.Snapshot()))
	}
	w.Step(5)
	if again := entitiesByKey(w, "box")[0]; again.Y != rested.Y {
		t.Fatalf("a box on the ground should stay there: %d then %d", rested.Y, again.Y)
	}
}

func TestThreeBoxesInOneColumnStackAtThreeHeights(t *testing.T) {
	w := mustWorld(t, "star-in-tree")
	for i := 0; i < 3; i++ {
		mustSpawn(t, w, "box", 14, 4)
	}
	boxes := entitiesByKey(w, "box")
	if len(boxes) != 3 {
		t.Fatalf("expected three boxes, got %d", len(boxes))
	}
	heights := map[int]bool{}
	for _, box := range boxes {
		heights[box.Y] = true
	}
	if len(heights) != 3 {
		t.Fatalf("three boxes in one column should stack at three heights, got %v", heights)
	}
	w.Step(20)
	settled := entitiesByKey(w, "box")
	heights = map[int]bool{}
	lowest := -1
	for _, box := range settled {
		heights[box.Y] = true
		if box.Y > lowest {
			lowest = box.Y
		}
	}
	if len(heights) != 3 {
		t.Fatalf("the stack should settle as three stacked boxes, got %v", heights)
	}
	if lowest != 8 {
		t.Fatalf("the bottom box should rest on the ground at y=8, got %d", lowest)
	}
}

func TestHeavyObjectCrushesAFragileOneBeneathIt(t *testing.T) {
	w := mustWorld(t, "star-in-tree")
	mustSpawn(t, w, "snowball", 12, 8)
	mustSpawn(t, w, "pillar", 12, 5)
	w.Step(5)
	if _, ok := findByKey(w, "snowball"); ok {
		t.Fatalf("a heavy thing should crush a fragile thing beneath it\n%s", describeState(w.Snapshot()))
	}
	if !anyEventContains(eventsOf(w), "crushes") {
		t.Fatalf("no event mentioned the crush: %v", eventsOf(w))
	}
}

func TestMagicalObjectFloatsAndStickyObjectHoldsOn(t *testing.T) {
	w := mustWorld(t, "star-in-tree")
	moon := mustSpawn(t, w, "moon", 12, 5)
	w.Step(10)
	if floated := entitiesByKey(w, "moon")[0]; floated.Y != moon.Y {
		t.Fatalf("a magical thing should ignore gravity: y went %d -> %d", moon.Y, floated.Y)
	}
	glue := mustSpawn(t, w, "glue", 8, 6) // beside the tree trunk
	w.Step(10)
	if stuck := entitiesByKey(w, "glue")[0]; stuck.Y != glue.Y {
		t.Fatalf("a sticky thing beside something solid should not fall: y went %d -> %d", glue.Y, stuck.Y)
	}
}

func TestBuoyantThingFloatsAndHeavyThingSinks(t *testing.T) {
	rows := flatRows(20, 12)
	rows[9] = "######~~~~~~~~######"
	rows[10] = "######~~~~~~~~######"
	rows[11] = "######~~~~~~~~######"
	// A chest puzzle, so the water can be as wide as the test needs it.
	w := specWorld(t, GoalChest, rows, []SpecEntity{{Phrase: "chest", X: 15, Y: 8}}, 4, 8)

	mustSpawn(t, w, "raft", 7, 5)   // two cells wide, over the west end of the pool
	mustSpawn(t, w, "stone", 11, 5) // three cells wide, over the east end
	w.Step(12)
	raft := entitiesByKey(w, "raft")[0]
	if raft.Y > 8 {
		t.Fatalf("a buoyant thing should come up to the surface, it is at y=%d", raft.Y)
	}
	stone := entitiesByKey(w, "stone")[0]
	if stone.Y < 11 {
		t.Fatalf("a thing that cannot float should sink to the bottom, it is at y=%d", stone.Y)
	}
}

// --- Rule 9: player movement -----------------------------------------------

func TestPlayerCannotWalkIntoAWallOrASolidObject(t *testing.T) {
	rows := flatRows(20, 12)
	rows[10] = ".....W.............."
	w := specWorld(t, GoalChest, rows, []SpecEntity{{Phrase: "chest", X: 12, Y: 10}}, 3, 10)
	before := playerOf(w).X
	if err := w.Act("right"); err != nil {
		t.Fatalf("a legal step should never be an error: %v", err)
	}
	if err := w.Act("right"); err != nil {
		// Walking into a wall is a no-op, not a bad request: the player is told
		// in the event log instead.
		t.Fatalf("walking into a wall should be refused gently, not as an API error: %v", err)
	}
	if after := playerOf(w).X; after != before+1 {
		t.Fatalf("the player should have stopped at the wall: %d -> %d", before, after)
	}
	if !anyEventContains(eventsOf(w), "solid") {
		t.Fatalf("the world should say why the step did not happen: %v", eventsOf(w))
	}

	star := mustWorld(t, "star-in-tree")
	mustSpawn(t, star, "box", 4, 8)
	if err := star.Act("right"); err != nil {
		t.Fatalf("walking into an object should be a gentle no-op: %v", err)
	}
	if got := playerOf(star).X; got != 3 {
		t.Fatalf("the player walked into a solid object: x=%d", got)
	}
	if !anyEventContains(eventsOf(star), "in the way") {
		t.Fatalf("the world should say what is in the way: %v", eventsOf(star))
	}
}

func TestJumpClimbsALadder(t *testing.T) {
	w := mustWorld(t, "star-in-tree")
	mustSpawn(t, w, "ladder", 4, 8)
	before := playerOf(w)
	if err := w.Act("jump"); err != nil {
		t.Fatalf("jump beside a ladder should work: %v", err)
	}
	after := playerOf(w)
	if after.Y >= before.Y {
		t.Fatalf("the player did not climb: y %d -> %d", before.Y, after.Y)
	}
	if !anyEventContains(eventsOf(w), "climb") {
		t.Fatalf("climbing should say so: %v", eventsOf(w))
	}
}

func TestTakeAndDropASmallObject(t *testing.T) {
	w := mustWorld(t, "star-in-tree")
	mustSpawn(t, w, "tiny box", 4, 8)
	if err := w.Act("take"); err != nil {
		t.Fatalf("taking a size-1 object beside the player should work: %v", err)
	}
	held := playerOf(w)
	if held.Holding == 0 {
		t.Fatalf("the player should be holding something")
	}
	taken, ok := findByKey(w, "box")
	if !ok || !taken.Held {
		t.Fatalf("the taken object should be marked held: %+v", taken)
	}
	if err := w.Act("drop"); err != nil {
		t.Fatalf("dropping should work: %v", err)
	}
	if playerOf(w).Holding != 0 {
		t.Fatalf("the player should have empty hands after dropping")
	}
	dropped := entitiesByKey(w, "box")[0]
	if dropped.Held {
		t.Fatalf("a dropped object must not still be held")
	}
	if err := w.Act("drop"); err != nil {
		t.Fatalf("dropping with empty hands should be a gentle no-op: %v", err)
	}
	if !anyEventContains(eventsOf(w), "hands are empty") {
		t.Fatalf("the world should say the hands are empty: %v", eventsOf(w))
	}
}

func TestPlayerIsSlowedInWaterButNeverDrowned(t *testing.T) {
	rows := flatRows(20, 12)
	rows[9] = "....~~~~~~.........."
	rows[10] = "....~~~~~~.........."
	w := specWorld(t, GoalChest, rows, []SpecEntity{{Phrase: "chest", X: 14, Y: 8}}, 2, 8)

	for i := 0; i < 2; i++ {
		if err := w.Act("right"); err != nil {
			t.Fatalf("walking on the bank failed: %v", err)
		}
	}
	w.Step(1) // gravity does the rest: the player drops into the stream
	state := w.Snapshot()
	if terrainAtState(state, state.Player.X, state.Player.Y) != TerrainWater {
		t.Fatalf("the player should be in the stream at (%d,%d), terrain code %d",
			state.Player.X, state.Player.Y, terrainAtState(state, state.Player.X, state.Player.Y))
	}

	// Slowed, not drowned: two wading steps move the player at most one cell,
	// and the world says so.
	before := playerOf(w).X
	waded := false
	for i := 0; i < 2; i++ {
		if err := w.Act("right"); err != nil {
			t.Fatalf("wading right in open water should be allowed: %v", err)
		}
		if anyEventContains(eventsOf(w), "wade") {
			waded = true
		}
	}
	if after := playerOf(w).X; after-before > 1 {
		t.Fatalf("water should halve the player's speed: x went %d -> %d over two steps", before, after)
	}
	if !waded {
		t.Fatalf("wading should say so: %v", eventsOf(w))
	}

	// And thirty more ticks in the water neither solves the puzzle nor hurts.
	w.Step(30)
	state = w.Snapshot()
	if state.Solved {
		t.Fatalf("paddling about is not opening the chest\n%s", describeState(state))
	}
	if state.Player.Holding != 0 {
		t.Fatalf("a player in water should not have picked anything up: %+v", state.Player)
	}
}

func TestBadActionsAreRejected(t *testing.T) {
	w := mustWorld(t, "star-in-tree")
	for _, action := range []string{"teleport", "", "fly", "Jump"} {
		if err := w.Act(action); err == nil {
			t.Fatalf("action %q should be rejected", action)
		}
	}
}

func TestSpawnRejectsOutOfRangeAndSolidGround(t *testing.T) {
	w := mustWorld(t, "star-in-tree")
	if _, _, _, err := w.Spawn("box", 999, 999); err == nil {
		t.Fatal("spawning outside the world should be an error")
	}
	if _, _, _, err := w.Spawn("box", -1, 4); err == nil {
		t.Fatal("spawning at a negative coordinate should be an error")
	}
	if _, _, _, err := w.Spawn("box", 7, 8); err == nil {
		t.Fatal("spawning inside the tree trunk should be an error")
	}
	if _, _, _, err := w.Spawn("box", 0, 9); err == nil {
		t.Fatal("spawning inside the ground should be an error")
	}
	if _, _, _, err := w.Spawn("   ", 4, 4); err == nil {
		t.Fatal("an empty phrase should be an error")
	}
	if _, _, _, err := w.Spawn("box", 3, 3); err != nil {
		t.Fatalf("spawning in open air should work: %v", err)
	}
}

// --- Reset, determinism, concurrency ---------------------------------------

func TestResetRestoresTheInitialStateExactly(t *testing.T) {
	for _, puzzle := range Puzzles() {
		w := mustWorld(t, puzzle.ID)
		before := w.Snapshot()

		mustSpawn(t, w, "box", 12, 5)
		mustSpawn(t, w, "cake", 14, 5)
		w.Step(7)
		_ = w.Act("right") // may or may not be possible, depending on the puzzle
		_ = w.Act("jump")
		w.Reset()
		after := w.Snapshot()

		if len(after.Entities) != len(before.Entities) {
			t.Fatalf("puzzle %s: reset left %d entities, want %d", puzzle.ID, len(after.Entities), len(before.Entities))
		}
		if len(after.Notebook) != 0 || len(before.Notebook) != 0 {
			t.Fatalf("puzzle %s: reset should clear the notebook", puzzle.ID)
		}
		if after.Player != before.Player {
			t.Fatalf("puzzle %s: reset moved the player: %+v want %+v", puzzle.ID, after.Player, before.Player)
		}
		if after.Solved || after.Goal.Met || after.Judged {
			t.Fatalf("puzzle %s: reset should clear the win", puzzle.ID)
		}
		if after.Width != before.Width || after.Height != before.Height {
			t.Fatalf("puzzle %s: reset changed the size", puzzle.ID)
		}
		for i := range before.Terrain {
			if after.Terrain[i] != before.Terrain[i] {
				t.Fatalf("puzzle %s: reset changed terrain at cell %d", puzzle.ID, i)
			}
		}
		if after.Ticks != 0 {
			t.Fatalf("puzzle %s: reset should wind the clock back, ticks=%d", puzzle.ID, after.Ticks)
		}
		// A reset world is solvable again: the whole point of reset.
		solution := solutionFor(puzzle.ID)
		for _, placement := range solution {
			mustSpawn(t, w, placement.Phrase, placement.X, placement.Y)
		}
		w.Step(30)
		if !w.Snapshot().Solved {
			t.Fatalf("puzzle %s should be solvable again after reset", puzzle.ID)
		}
	}
}

// solutionFor returns the first authored answer of a built-in puzzle.
func solutionFor(puzzleID string) []Placement {
	for _, template := range builtinPuzzles() {
		if template.puzzle.ID == puzzleID {
			return template.solutions[0].Phrases
		}
	}
	return nil
}

func TestSameSequenceGivesIdenticalJSON(t *testing.T) {
	run := func() string {
		w := mustWorld(t, "star-in-tree")
		mustSpawn(t, w, "ladder", 5, 8)
		mustSpawn(t, w, "cake", 12, 5)
		mustSpawn(t, w, "match", 14, 8)
		w.Step(30)
		state := w.Snapshot()
		state.ID = "" // ids are unique per world on purpose
		raw, err := json.Marshal(state)
		if err != nil {
			t.Fatalf("snapshot is not JSON: %v", err)
		}
		return string(raw)
	}
	first, second := run(), run()
	if first != second {
		t.Fatalf("the same sequence produced different worlds:\n%s\n%s", first, second)
	}
}

func TestStepsAreIdempotentAtRest(t *testing.T) {
	w := mustWorld(t, "cross-the-river")
	w.Step(50)
	settled := w.Snapshot()
	w.Step(30)
	again := w.Snapshot()
	settled.Ticks, again.Ticks = 0, 0 // the clock is supposed to advance
	first, err := json.Marshal(settled)
	if err != nil {
		t.Fatalf("snapshot is not JSON: %v", err)
	}
	second, err := json.Marshal(again)
	if err != nil {
		t.Fatalf("snapshot is not JSON: %v", err)
	}
	if string(first) != string(second) {
		t.Fatalf("a settled world changed when stepped again:\n%s\n%s", first, second)
	}
}

func TestConcurrentStepSnapshotAndSpawn(t *testing.T) {
	w := mustWorld(t, "star-in-tree")
	var wg sync.WaitGroup
	for i := 0; i < 4; i++ {
		wg.Add(3)
		go func() {
			defer wg.Done()
			for k := 0; k < 40; k++ {
				w.Step(2)
			}
		}()
		go func() {
			defer wg.Done()
			for k := 0; k < 40; k++ {
				state := w.Snapshot()
				if state.Width != 20 {
					t.Errorf("snapshot came back malformed: %dx%d", state.Width, state.Height)
					return
				}
				_ = w.Events()
			}
		}()
		go func() {
			defer wg.Done()
			for k := 0; k < 20; k++ {
				if _, _, _, err := w.Spawn("cake", 12+int(k%6), 4); err != nil {
					// A full column is a legitimate refusal, not a failure.
					continue
				}
				_, _ = w.Trial("ladder", 5, 8)
			}
		}()
	}
	wg.Wait()
	if !w.Snapshot().Solved {
		// The walker only solves it if a ladder actually landed; the spawns above
		// are cakes, so an unsolved world is the expected outcome. This assertion
		// exists to make sure nothing panicked.
		t.Log("world stayed unsolved under concurrent access, as expected")
	}
}

func TestWorldIDsAreUnique(t *testing.T) {
	seen := map[string]bool{}
	for i := 0; i < 20; i++ {
		w := mustWorld(t, "star-in-tree")
		if seen[w.ID()] {
			t.Fatalf("world id %q was handed out twice", w.ID())
		}
		seen[w.ID()] = true
	}
	if _, err := New("no-such-puzzle"); err == nil {
		t.Fatal("New should refuse an unknown puzzle id")
	}
}

func TestDescriptorIsStable(t *testing.T) {
	w := mustWorld(t, "light-the-candles")
	first := w.Describe()
	if !strings.Contains(first, "puzzle light-the-candles") {
		t.Fatalf("Describe should name the puzzle:\n%s", first)
	}
	if again := w.Describe(); again != first {
		t.Fatalf("Describe is not stable")
	}
}
