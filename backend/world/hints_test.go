package world

import (
	"strings"
	"testing"
)

// The four-choice hint, and the promise that exactly one of the four works. The
// promise is proven by simulating every option here, not by trusting the code
// that built the hint.

// correctIndex works out which option actually solves the puzzle, by playing it
// in a fresh world the way the frontend would.
func correctIndex(t *testing.T, puzzleID string, hint HintSet) int {
	t.Helper()
	solved := []int{}
	for i, option := range hint.Options {
		w := mustWorld(t, puzzleID)
		if _, _, _, err := w.Spawn(option, hint.Placement.X, hint.Placement.Y); err != nil {
			t.Fatalf("option %d (%q) would not even spawn at the offered spot: %v", i, option, err)
		}
		w.Step(40)
		if w.Snapshot().Solved {
			solved = append(solved, i)
		}
	}
	if len(solved) != 1 {
		t.Fatalf("a hint must have exactly one winning option, this one has %v of %d", solved, len(hint.Options))
	}
	return solved[0]
}

func TestHintOffersFourVerifiedOptionsForEveryPuzzle(t *testing.T) {
	for _, puzzle := range Puzzles() {
		w := mustWorld(t, puzzle.ID)
		hint, err := w.Hint()
		if err != nil {
			t.Fatalf("puzzle %s could not offer a hint: %v", puzzle.ID, err)
		}
		if len(hint.Options) != 4 {
			t.Fatalf("puzzle %s offered %d options, want exactly four", puzzle.ID, len(hint.Options))
		}
		if hint.Rationale == "" {
			t.Fatalf("puzzle %s offered a hint with no friendly line", puzzle.ID)
		}
		seen := map[string]bool{}
		for i, option := range hint.Options {
			if strings.TrimSpace(option) == "" {
				t.Fatalf("puzzle %s option %d is empty", puzzle.ID, i)
			}
			if seen[option] {
				t.Fatalf("puzzle %s offered %q twice", puzzle.ID, option)
			}
			seen[option] = true
		}
		// Exactly one of the four solves the puzzle, proven by playing it.
		correctIndex(t, puzzle.ID, hint)

		// And the correct choice works through the ordinary spawn path.
		w2 := mustWorld(t, puzzle.ID)
		hint2, err := w2.Hint()
		if err != nil {
			t.Fatalf("second hint failed: %v", err)
		}
		index := correctIndex(t, puzzle.ID, hint2)
		obj, _, _, err := w2.Spawn(hint2.Options[index], hint2.Placement.X, hint2.Placement.Y)
		if err != nil {
			t.Fatalf("the winning option should spawn normally: %v", err)
		}
		if obj.Key == "" {
			t.Fatalf("the winning option spawned nothing")
		}
		w2.Step(40)
		if !w2.Snapshot().Solved {
			t.Fatalf("the winning option should solve %s through the ordinary path\n%s",
				puzzle.ID, describeState(w2.Snapshot()))
		}
	}
}

func TestHintIsDeterministicForTheSameState(t *testing.T) {
	for _, puzzle := range Puzzles() {
		w := mustWorld(t, puzzle.ID)
		first, err := w.Hint()
		if err != nil {
			t.Fatalf("hint failed: %v", err)
		}
		second, err := w.Hint()
		if err != nil {
			t.Fatalf("second hint failed: %v", err)
		}
		if strings.Join(first.Options, "|") != strings.Join(second.Options, "|") {
			t.Fatalf("puzzle %s gave two different hints for the same state:\n%v\n%v", puzzle.ID, first.Options, second.Options)
		}
		if first.Placement != second.Placement || first.Rationale != second.Rationale {
			t.Fatalf("puzzle %s gave two different hint placements or lines", puzzle.ID)
		}
	}
}

func TestTheWinningOptionMovesBetweenWorlds(t *testing.T) {
	slots := map[int]bool{}
	for _, puzzle := range Puzzles() {
		for round := 0; round < 6; round++ {
			w := mustWorld(t, puzzle.ID)
			hint, err := w.Hint()
			if err != nil {
				t.Fatalf("hint failed for %s: %v", puzzle.ID, err)
			}
			slots[correctIndex(t, puzzle.ID, hint)] = true
		}
	}
	if len(slots) < 2 {
		t.Fatalf("the winning option is always in the same slot (%v): a hint that always sits in option 0 is not a choice", slots)
	}
}

func TestTrialNeverTouchesTheLiveWorld(t *testing.T) {
	w := mustWorld(t, "star-in-tree")
	// From the starting position a ladder at the hint spot really does solve it.
	if solved, err := w.Trial("ladder", 5, 8); err != nil || !solved {
		t.Fatalf("Trial should report that a ladder solves this puzzle: %v %v", solved, err)
	}

	mustSpawn(t, w, "cake", 12, 5)
	w.Step(3)
	before := stateJSON(t, w)
	notebookBefore := len(w.Snapshot().Notebook)

	// Trials that fail, and a trial that cannot even happen: none of them may
	// change the real world.
	if solved, err := w.Trial("cake", 5, 8); err != nil || solved {
		t.Fatalf("Trial should report that a cake does not solve it: %v %v", solved, err)
	}
	if solved, err := w.Trial("pillow", 4, 5); err != nil || solved {
		t.Fatalf("Trial should report that a pillow does not solve it: %v %v", solved, err)
	}
	if _, err := w.Trial("ladder", 999, 999); err == nil {
		t.Fatalf("Trial should report a spawn that is impossible rather than pretending")
	}

	after := stateJSON(t, w)
	if before != after {
		t.Fatalf("Trial changed the live world:\n%s\n%s", before, after)
	}
	if len(w.Snapshot().Notebook) != notebookBefore {
		t.Fatalf("Trial wrote to the notebook")
	}
}

func TestHintRefusesOnASolvedWorld(t *testing.T) {
	w := mustWorld(t, "open-the-chest")
	mustSpawn(t, w, "key", 9, 8)
	w.Step(2)
	if !w.Snapshot().Solved {
		t.Fatalf("the key should have opened the chest\n%s", describeState(w.Snapshot()))
	}
	if _, err := w.Hint(); err == nil {
		t.Fatalf("a solved puzzle should not hand out hints")
	}
}

func TestHintJSONCarriesNoAnswerMarker(t *testing.T) {
	w := mustWorld(t, "light-the-candles")
	hint, err := w.Hint()
	if err != nil {
		t.Fatalf("hint failed: %v", err)
	}
	raw, err := jsonMarshal(hint)
	if err != nil {
		t.Fatalf("hint is not JSON: %v", err)
	}
	lower := strings.ToLower(string(raw))
	for _, forbidden := range []string{"correct", "solution", "answer", "winning"} {
		if strings.Contains(lower, forbidden) {
			t.Fatalf("the hint leaks %q: %s", forbidden, raw)
		}
	}
	if !strings.Contains(string(raw), "options") || !strings.Contains(string(raw), "placement") {
		t.Fatalf("the hint should carry options and a placement: %s", raw)
	}
}

// A player who has already drawn something at the best spot must not lose their
// hint: the engine moves to a nearby free cell and re-proves the promise there.
// A flat refusal while a good spot still exists would be the frustrating outcome.
func TestHintMovesToANearbySpotWhenTheUsualOneIsBlocked(t *testing.T) {
	w := mustWorld(t, "star-in-tree")
	first, err := w.Hint()
	if err != nil {
		t.Fatalf("the first hint should work: %v", err)
	}
	authored := first.Placement

	stacked := 0
	for i := 0; i < 14; i++ {
		if _, _, _, err := w.Spawn("box", authored.X, authored.Y); err != nil {
			break
		}
		stacked++
	}
	if stacked == 0 {
		t.Fatal("the test could not put anything on the authored spot")
	}

	// Work out honestly whether the authored spot is still usable.
	_, spotErr := w.Trial("ladder", authored.X, authored.Y)

	hint, err := w.Hint()
	if err != nil {
		t.Fatalf("a crowded spot must not cost the player their hint: %v", err)
	}
	if len(hint.Options) != 4 {
		t.Fatalf("options = %v", hint.Options)
	}
	if spotErr != nil && hint.Placement == authored {
		t.Fatalf("the authored spot %+v can no longer take an object, so the hint should have moved", authored)
	}

	// The promise still holds wherever the hint ended up.
	solving := []string{}
	for _, option := range hint.Options {
		fresh := mustWorld(t, "star-in-tree")
		if _, _, _, err := fresh.Spawn(option, hint.Placement.X, hint.Placement.Y); err != nil {
			continue
		}
		fresh.Step(40)
		if fresh.Snapshot().Solved {
			solving = append(solving, option)
		}
	}
	if len(solving) != 1 {
		t.Fatalf("exactly one option must solve the puzzle from %+v; %v did (%v)", hint.Placement, solving, hint.Options)
	}
}

func TestHintRefusesForASpecWorldWithNoAuthoredAnswers(t *testing.T) {
	spec := baseSpec(GoalChest)
	w, err := NewFromSpec(spec)
	if err != nil {
		t.Fatalf("spec should load: %v", err)
	}
	if _, err := w.Hint(); err == nil {
		t.Fatalf("a generated puzzle has no answer key, so a verified hint is impossible: Hint must refuse")
	}
}

func TestTrialAndHintAreSafeUnderConcurrency(t *testing.T) {
	w := mustWorld(t, "star-in-tree")
	done := make(chan bool)
	for i := 0; i < 3; i++ {
		go func() {
			for k := 0; k < 10; k++ {
				_, _ = w.Trial("ladder", 5, 8)
				_, _ = w.Hint()
				_ = w.Snapshot()
			}
			done <- true
		}()
	}
	for i := 0; i < 3; i++ {
		<-done
	}
	if !w.Snapshot().Solved == false {
		t.Fatalf("dry runs must not solve the real world")
	}
}
