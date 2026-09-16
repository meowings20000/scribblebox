package world

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
)

// Shared test helpers. Everything here talks to the engine the way the HTTP
// layer does: New / NewFromSpec, Spawn, Step, Act, Reset, Snapshot.

func mustWorld(t *testing.T, puzzleID string) *World {
	t.Helper()
	w, err := New(puzzleID)
	if err != nil {
		t.Fatalf("New(%q) failed: %v", puzzleID, err)
	}
	return w
}

func mustSpawn(t *testing.T, w *World, phrase string, x, y int) Entity {
	t.Helper()
	obj, _, _, err := w.Spawn(phrase, x, y)
	if err != nil {
		t.Fatalf("Spawn(%q, %d, %d) failed: %v\n%s", phrase, x, y, err, w.Describe())
	}
	found, ok := findByKey(w, obj.Key)
	if !ok {
		t.Fatalf("Spawn(%q) reported success but no %s is in the world", phrase, obj.Key)
	}
	return found
}

func findByKey(w *World, key string) (Entity, bool) {
	w.mu.Lock()
	defer w.mu.Unlock()
	for _, e := range w.state.entities {
		if e.Object.Key == key {
			return e, true
		}
	}
	return Entity{}, false
}

// entitiesByKey returns every entity with a given key, in world order.
func entitiesByKey(w *World, key string) []Entity {
	w.mu.Lock()
	defer w.mu.Unlock()
	out := []Entity{}
	for _, e := range w.state.entities {
		if e.Object.Key == key {
			out = append(out, e)
		}
	}
	return out
}

func stateJSON(t *testing.T, w *World) string {
	t.Helper()
	raw, err := json.Marshal(w.Snapshot())
	if err != nil {
		t.Fatalf("snapshot is not JSON: %v", err)
	}
	return string(raw)
}

// jsonMarshal is a tiny wrapper so tests can serialise anything the way the HTTP
// layer does.
func jsonMarshal(v any) ([]byte, error) { return json.Marshal(v) }

func terrainAtState(st State, x, y int) int {
	return st.Terrain[y*st.Width+x]
}

func playerOf(w *World) Player { return w.Snapshot().Player }

func eventsOf(w *World) []string { return w.Events() }

func anyEventContains(events []string, needle string) bool {
	for _, e := range events {
		if strings.Contains(e, needle) {
			return true
		}
	}
	return false
}

// tileSymbols mirrors the puzzle drawings, so a test can build its own world.
var tileSymbols = map[rune]int{
	'.': TerrainAir,
	'#': TerrainGround,
	'~': TerrainWater,
	'T': TerrainTrunk,
	'W': TerrainWall,
	'a': TerrainAsh,
}

func terrainFromRows(t *testing.T, rows []string) []int {
	t.Helper()
	out := []int{}
	for y, row := range rows {
		for _, r := range []rune(row) {
			code, ok := tileSymbols[r]
			if !ok {
				t.Fatalf("row %d has an unknown symbol %q", y, string(r))
			}
			out = append(out, code)
		}
	}
	return out
}

// specWorld builds a spec world for a rule test, the same way the AI path does.
func specWorld(t *testing.T, kind string, rows []string, items []SpecEntity, px, py int) *World {
	t.Helper()
	if len(rows) == 0 {
		t.Fatal("specWorld needs at least one row")
	}
	spec := PuzzleSpec{
		ID:       "test-" + kind,
		Title:    "A test puzzle",
		Brief:    "For the test suite.",
		Hint:     "For the test suite.",
		GoalKind: kind,
		Width:    len([]rune(rows[0])),
		Height:   len(rows),
		Terrain:  terrainFromRows(t, rows),
		Entities: items,
		PlayerX:  px,
		PlayerY:  py,
	}
	w, err := NewFromSpec(spec)
	if err != nil {
		t.Fatalf("NewFromSpec failed: %v\nspec: %+v", err, spec)
	}
	return w
}

// flatRows is a wide open field with a floor, for rule tests that need room.
func flatRows(width, height int) []string {
	rows := make([]string, 0, height)
	for y := 0; y < height; y++ {
		row := make([]rune, width)
		for x := range row {
			row[x] = '.'
		}
		if y == height-1 {
			for x := range row {
				row[x] = '#'
			}
		}
		rows = append(rows, string(row))
	}
	return rows
}

// chestSpec puts a locked chest on a flat field, far from the player so nothing
// is solved by accident.
func chestSpec(t *testing.T, width, height int, chestX, chestY, px, py int) *World {
	t.Helper()
	rows := flatRows(width, height)
	return specWorld(t, GoalChest, rows, []SpecEntity{{Phrase: "chest", X: chestX, Y: chestY}}, px, py)
}

func describeState(st State) string {
	var b strings.Builder
	fmt.Fprintf(&b, "solved=%v judged=%v goal=%q met=%v ticks=%d player=(%d,%d) holding=%d\n",
		st.Solved, st.Judged, st.Goal.Progress, st.Goal.Met, st.Ticks, st.Player.X, st.Player.Y, st.Player.Holding)
	for _, e := range st.Entities {
		fmt.Fprintf(&b, "  %d %s at (%d,%d) %dx%d burning=%v opened=%v cut=%v powered=%v held=%v\n",
			e.ID, e.Object.Key, e.X, e.Y, e.W, e.H, e.Burning, e.Opened, e.Cut, e.Powered, e.Held)
	}
	fmt.Fprintf(&b, "  notebook=%d events=%v\n", len(st.Notebook), st.Events)
	return b.String()
}
