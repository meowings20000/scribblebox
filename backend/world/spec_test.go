package world

import (
	"strings"
	"testing"
)

// Externally authored puzzles: the AI path. A spec world must behave exactly like
// a built-in one, and every rule that keeps an AI puzzle playable and easy is
// tested by trying to break it.

// baseSpec returns a valid, easy spec for a goal kind, so each negative test can
// break exactly one thing.
func baseSpec(kind string) PuzzleSpec {
	rows := flatRows(20, 12)
	spec := PuzzleSpec{
		ID:       "generated-" + kind,
		Title:    "A generated puzzle",
		Brief:    "Something to do.",
		Hint:     "Try the obvious thing.",
		GoalKind: kind,
		Width:    20,
		Height:   12,
		Terrain:  terrainFromRowsQuiet(rows),
		PlayerX:  6,
		PlayerY:  10,
	}
	switch kind {
	case GoalStar:
		spec.Entities = []SpecEntity{{Phrase: "star", X: 10, Y: 8}}
	case GoalCandles:
		spec.Entities = []SpecEntity{
			{Phrase: "candle", X: 9, Y: 10},
			{Phrase: "candle", X: 10, Y: 10},
		}
	case GoalChest:
		spec.Entities = []SpecEntity{{Phrase: "chest", X: 10, Y: 10}}
	case GoalCross:
		rows[9] = "########~~##########"
		rows[10] = "########~~##########"
		rows[11] = "########~~##########"
		spec.Terrain = terrainFromRowsQuiet(rows)
		spec.PlayerY = 8
		spec.Entities = nil
	}
	return spec
}

// terrainFromRows for a spec needs a testing.T only in the test helper, so this
// local version takes the error path out of the picture for base specs.
func terrainFromRowsQuiet(rows []string) []int {
	out := []int{}
	for _, row := range rows {
		for _, r := range []rune(row) {
			out = append(out, tileSymbols[r])
		}
	}
	return out
}

func TestSpecWorldIsPlayableForEveryGoalKind(t *testing.T) {
	recipes := map[string][]Placement{
		GoalStar:    {{Phrase: "ladder", X: 9, Y: 10}},
		GoalCross:   {{Phrase: "canoe", X: 7, Y: 8}},
		GoalCandles: {{Phrase: "match", X: 9, Y: 9}},
		GoalChest:   {{Phrase: "key", X: 9, Y: 10}},
	}
	for kind, recipe := range recipes {
		spec := baseSpec(kind)
		w, err := NewFromSpec(spec)
		if err != nil {
			t.Fatalf("the %s spec should load: %v", kind, err)
		}
		if got := w.Snapshot().Puzzle.ID; got != spec.ID {
			t.Fatalf("a spec world should carry the spec's id, got %q want %q", got, spec.ID)
		}
		for _, placement := range recipe {
			mustSpawn(t, w, placement.Phrase, placement.X, placement.Y)
		}
		w.Step(30)
		state := w.Snapshot()
		if !state.Solved || !state.Goal.Met {
			t.Fatalf("the %s spec should be solvable with its obvious one-object answer\n%s", kind, describeState(state))
		}
	}
}

// rowsFor gives the same layouts baseSpec builds, as rows.
func rowsFor(kind string) []string {
	rows := flatRows(20, 12)
	switch kind {
	case GoalStar:
		return rows
	case GoalCandles:
		return rows
	case GoalChest:
		return rows
	case GoalCross:
		rows[9] = "########~~##########"
		rows[10] = "########~~##########"
		rows[11] = "########~~##########"
		return rows
	}
	return rows
}

func TestSpecWorldBehavesLikeABuiltin(t *testing.T) {
	spec := baseSpec(GoalChest)
	spec.Terrain = terrainFromRowsQuiet(rowsFor(GoalChest))
	w, err := NewFromSpec(spec)
	if err != nil {
		t.Fatalf("spec should load: %v", err)
	}

	// Spawn, notebook, step, act, reset: the same surface a built-in has.
	obj, note, approximate, err := w.Spawn("cake", 12, 6)
	if err != nil {
		t.Fatalf("spawn in a spec world failed: %v", err)
	}
	if obj.Key != "cake" || note == "" || approximate {
		t.Fatalf("spawn returned odd values: %+v %q %v", obj, note, approximate)
	}
	if _, _, approximate, err = w.Spawn("zorble", 14, 6); err != nil || !approximate {
		t.Fatalf("an unknown word should still spawn and say it was improvised: %v %v", err, approximate)
	}
	w.Step(3)
	if w.Snapshot().Ticks != 3 {
		t.Fatalf("stepping a spec world did not advance the clock")
	}
	if err := w.Act("teleport"); err == nil {
		t.Fatal("a spec world should reject a nonsense action too")
	}
	if err := w.Act("left"); err != nil {
		t.Fatalf("walking in a spec world failed: %v", err)
	}
	state := w.Snapshot()
	if len(state.Notebook) != 2 {
		t.Fatalf("notebook has %d entries, want 2", len(state.Notebook))
	}
	if len(state.Puzzle.Solutions) != 0 {
		t.Fatalf("a spec world must have no answer key")
	}
	raw, err := jsonMarshal(state)
	if err != nil {
		t.Fatalf("state is not JSON: %v", err)
	}
	if strings.Contains(string(raw), "solutions") {
		t.Fatalf("a spec snapshot leaks the word solutions: %s", raw)
	}
	w.Reset()
	if w.Snapshot().Ticks != 0 || len(w.Snapshot().Notebook) != 0 {
		t.Fatalf("reset should clear a spec world too")
	}

	// A hint needs authored answers, and a spec has none: it must refuse.
	if _, err := w.Hint(); err == nil {
		t.Fatalf("a spec world has no authored answers, so Hint must refuse rather than guess")
	}
}

func TestSpecValidationRejectsBrokenPuzzles(t *testing.T) {
	cases := []struct {
		name    string
		mutate  func(*PuzzleSpec)
		wantSub string
	}{
		{"bad-terrain-code", func(s *PuzzleSpec) { s.Terrain[5] = 9 }, "terrain code 9"},
		{"short-terrain", func(s *PuzzleSpec) { s.Terrain = s.Terrain[:10] }, "needs"},
		{"unknown-goal-kind", func(s *PuzzleSpec) { s.GoalKind = "dragon" }, "cannot referee"},
		{"player-in-a-wall", func(s *PuzzleSpec) { s.Terrain[10*s.Width+6] = TerrainWall }, "starts inside solid"},
		{"player-outside", func(s *PuzzleSpec) { s.PlayerX = 99 }, "outside"},
		{"entity-outside", func(s *PuzzleSpec) {
			s.Entities = []SpecEntity{{Phrase: "star", X: 99, Y: 99}}
		}, "does not fit"},
		{"entity-in-solid-ground", func(s *PuzzleSpec) {
			s.Entities = []SpecEntity{{Phrase: "star", X: 6, Y: 8}}
			s.Terrain[8*s.Width+6] = TerrainWall
		}, "buried"},
		{"unnamed-entity", func(s *PuzzleSpec) {
			s.Entities = []SpecEntity{{Phrase: "  ", X: 10, Y: 8}}
		}, "has no name"},
		{"no-ground-at-all", func(s *PuzzleSpec) {
			for i := range s.Terrain {
				s.Terrain[i] = TerrainAir
			}
		}, "no ground"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			spec := baseSpec(GoalStar)
			spec.Terrain = terrainFromRowsQuiet(rowsFor(GoalStar))
			tc.mutate(&spec)
			_, err := NewFromSpec(spec)
			if err == nil {
				t.Fatalf("this spec should have been refused")
			}
			if !strings.Contains(err.Error(), tc.wantSub) {
				t.Fatalf("the refusal should explain itself: got %q, want it to mention %q", err.Error(), tc.wantSub)
			}
		})
	}
}

func TestSpecValidationNeedsTheRightGoalObject(t *testing.T) {
	cases := []struct {
		name string
		spec PuzzleSpec
		want string
	}{
		{"candles-without-a-candle", baseSpecWith(GoalCandles, nil), "candle"},
		{"chest-without-a-container", baseSpecWith(GoalChest, nil), "container"},
		{"star-without-a-prize", baseSpecWith(GoalStar, []SpecEntity{{Phrase: "cake", X: 10, Y: 8}}), "collectible"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := NewFromSpec(tc.spec)
			if err == nil {
				t.Fatalf("this spec should have been refused")
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("the refusal should name the missing thing: got %q, want %q", err.Error(), tc.want)
			}
		})
	}

	// A crossing puzzle with no water at all is not a crossing puzzle.
	bridge := baseSpec(GoalCross)
	bridge.Terrain = terrainFromRowsQuiet(flatRows(20, 12))
	_, err := NewFromSpec(bridge)
	if err == nil || !strings.Contains(err.Error(), "water") {
		t.Fatalf("a crossing puzzle with no water should be refused, got %v", err)
	}
}

func baseSpecWith(kind string, items []SpecEntity) PuzzleSpec {
	spec := baseSpec(kind)
	spec.Terrain = terrainFromRowsQuiet(rowsFor(kind))
	spec.Entities = items
	return spec
}

// TestSpecEasinessRulesRejectsHardPuzzles is the difficulty gate: this game is
// for primary-school players, so an AI may not hand over a marathon.
func TestSpecEasinessRulesRejectsHardPuzzles(t *testing.T) {
	cases := []struct {
		name   string
		mutate func(*PuzzleSpec)
		want   string
	}{
		{"too-wide", func(s *PuzzleSpec) {
			s.Width = 41
			s.Terrain = terrainFromRowsQuiet(flatRows(41, 12))
			s.Entities = []SpecEntity{{Phrase: "star", X: 10, Y: 8}}
		}, "between 12 and 40"},
		{"too-tall", func(s *PuzzleSpec) {
			s.Height = 21
			s.Terrain = terrainFromRowsQuiet(flatRows(20, 21))
			s.Entities = []SpecEntity{{Phrase: "star", X: 10, Y: 8}}
		}, "between 8 and 20"},
		{"too-small", func(s *PuzzleSpec) {
			s.Width = 8
			s.Terrain = terrainFromRowsQuiet(flatRows(8, 12))
			s.Entities = []SpecEntity{{Phrase: "star", X: 4, Y: 8}}
		}, "between 12 and 40"},
		{"too-many-entities", func(s *PuzzleSpec) {
			items := []SpecEntity{{Phrase: "star", X: 10, Y: 8}}
			for i := 0; i < 25; i++ {
				items = append(items, SpecEntity{Phrase: "cake", X: 1 + i, Y: 6})
			}
			s.Entities = items
		}, "at most 25"},
		{"goal-too-far", func(s *PuzzleSpec) {
			s.Width = 40
			s.Height = 20
			s.Terrain = terrainFromRowsQuiet(flatRows(40, 20))
			s.PlayerX, s.PlayerY = 39, 18
			s.Entities = []SpecEntity{{Phrase: "star", X: 1, Y: 1}}
		}, "within 12 cells"},
		{"goal-walled-in", func(s *PuzzleSpec) {
			rows := flatRows(20, 12)
			rows[7] = "....WWWWW..........."
			rows[8] = "....WW.WW..........."
			rows[9] = "....WWWWW..........."
			s.Terrain = terrainFromRowsQuiet(rows)
			s.Entities = []SpecEntity{{Phrase: "star", X: 6, Y: 8}}
			s.PlayerX, s.PlayerY = 12, 10
		}, "walled in"},
		{"water-too-wide", func(s *PuzzleSpec) {
			rows := flatRows(20, 12)
			rows[9] = "########~~~~~#######"
			rows[10] = "########~~~~~#######"
			rows[11] = "########~~~~~#######"
			s.Terrain = terrainFromRowsQuiet(rows)
			s.GoalKind = GoalCross
			s.Entities = nil
			s.PlayerX, s.PlayerY = 4, 8
		}, "cells wide"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			spec := baseSpec(GoalStar)
			spec.Terrain = terrainFromRowsQuiet(rowsFor(GoalStar))
			tc.mutate(&spec)
			_, err := NewFromSpec(spec)
			if err == nil {
				t.Fatalf("a too-hard puzzle should have been refused")
			}
			if !strings.Contains(err.Error(), "too hard") {
				t.Fatalf("a difficulty refusal should say so: %q", err.Error())
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("the refusal should say what to change: got %q, want %q", err.Error(), tc.want)
			}
		})
	}
}

func TestSpecWorldStaysUnsolvedWithoutObjects(t *testing.T) {
	for _, kind := range GoalKinds {
		spec := baseSpec(kind)
		spec.Terrain = terrainFromRowsQuiet(rowsFor(kind))
		w, err := NewFromSpec(spec)
		if err != nil {
			t.Fatalf("%s spec should load: %v", kind, err)
		}
		w.Step(30)
		if state := w.Snapshot(); state.Solved || state.Goal.Met {
			t.Fatalf("a %s spec solved itself with nothing drawn\n%s", kind, describeState(state))
		}
	}
}

func TestAcceptJudgeMarksSolvedAndResetClearsIt(t *testing.T) {
	w := mustWorld(t, "open-the-chest")
	if err := w.AcceptJudge("the dragon owed you a favour"); err != nil {
		t.Fatalf("the first judge verdict should be accepted: %v", err)
	}
	state := w.Snapshot()
	if !state.Solved || !state.Judged || !state.Goal.Met {
		t.Fatalf("an accepted verdict should solve the world: %+v", describeState(state))
	}
	if state.JudgeNote != "the dragon owed you a favour" {
		t.Fatalf("the verdict should be remembered: %q", state.JudgeNote)
	}
	if !strings.Contains(state.Goal.Progress, "the dragon owed you a favour") {
		t.Fatalf("the goal should quote the judge: %q", state.Goal.Progress)
	}
	if !anyEventContains(state.Events, "judge accepts") {
		t.Fatalf("an accepted verdict should be logged: %v", state.Events)
	}
	if err := w.AcceptJudge("and another thing"); err == nil {
		t.Fatal("a second verdict on an already solved world should be an error")
	}

	w.Reset()
	state = w.Snapshot()
	if state.Judged || state.JudgeNote != "" || state.Solved || state.Goal.Met {
		t.Fatalf("reset should clear the judge's verdict too: %+v", describeState(state))
	}
}
