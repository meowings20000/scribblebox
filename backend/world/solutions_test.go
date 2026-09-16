package world

import (
	"strings"
	"testing"
)

// The most important test in the package: every authored answer is replayed
// against the real rule engine. If a claim of "you can solve this four ways" is
// not true, this test says so.

func TestPuzzlesAreComplete(t *testing.T) {
	puzzles := Puzzles()
	if len(puzzles) != 4 {
		t.Fatalf("expected the four built-in puzzles, got %d", len(puzzles))
	}
	seen := map[string]bool{}
	for _, p := range puzzles {
		if p.ID == "" || p.Title == "" || p.Brief == "" || p.Hint == "" || p.GoalKind == "" {
			t.Fatalf("puzzle is missing text: %+v", p)
		}
		if seen[p.ID] {
			t.Fatalf("duplicate puzzle id %q", p.ID)
		}
		seen[p.ID] = true
		if !knownGoalKind(p.GoalKind) {
			t.Fatalf("puzzle %s has a goal kind the engine cannot referee: %q", p.ID, p.GoalKind)
		}
		if p.Width < 12 || p.Width > 40 || p.Height < 8 || p.Height > 20 {
			t.Fatalf("puzzle %s is %dx%d, outside the easy size range", p.ID, p.Width, p.Height)
		}
		if len(p.Solutions) != 0 {
			t.Fatalf("Puzzles() leaked %d answers for %s", len(p.Solutions), p.ID)
		}
	}
	for _, want := range []string{"star-in-tree", "cross-the-river", "light-the-candles", "open-the-chest"} {
		if !seen[want] {
			t.Fatalf("missing built-in puzzle %q", want)
		}
	}
}

func TestEveryPuzzleHasManyAnswers(t *testing.T) {
	for _, template := range builtinPuzzles() {
		if len(template.solutions) < 4 {
			t.Fatalf("puzzle %s has only %d answers; the game promises several ways", template.puzzle.ID, len(template.solutions))
		}
		descriptions := map[string]bool{}
		for _, solution := range template.solutions {
			if solution.Description == "" {
				t.Fatalf("puzzle %s has an answer with no description", template.puzzle.ID)
			}
			if len(solution.Phrases) == 0 {
				t.Fatalf("puzzle %s has an answer with no objects", template.puzzle.ID)
			}
			if descriptions[solution.Description] {
				t.Fatalf("puzzle %s repeats an answer description: %q", template.puzzle.ID, solution.Description)
			}
			descriptions[solution.Description] = true
		}
	}
}

// TestSolutionPlacementsAreLegal enforces the placement rules: inside the grid,
// not inside solid terrain, and a phrase the notebook can read.
func TestSolutionPlacementsAreLegal(t *testing.T) {
	for _, template := range builtinPuzzles() {
		w := mustWorld(t, template.puzzle.ID)
		bounds := w.Snapshot()
		for _, solution := range template.solutions {
			if len(solution.Phrases) > 3 {
				t.Fatalf("puzzle %s answer %q needs %d objects: keep it to three", template.puzzle.ID, solution.Description, len(solution.Phrases))
			}
			for _, placement := range solution.Phrases {
				if placement.X < 0 || placement.Y < 0 || placement.X >= bounds.Width || placement.Y >= bounds.Height {
					t.Fatalf("puzzle %s answer %q places %s outside the %dx%d world at (%d, %d)",
						template.puzzle.ID, solution.Description, placement.Phrase, bounds.Width, bounds.Height, placement.X, placement.Y)
				}
				if isSolid(terrainAtState(bounds, placement.X, placement.Y)) {
					t.Fatalf("puzzle %s answer %q places %s inside solid terrain at (%d, %d)",
						template.puzzle.ID, solution.Description, placement.Phrase, placement.X, placement.Y)
				}
			}
		}
	}
}

func TestEveryAuthoredSolutionSolvesItsPuzzle(t *testing.T) {
	checked := 0
	for _, template := range builtinPuzzles() {
		for _, solution := range template.solutions {
			solution := solution
			name := template.puzzle.ID + "/" + solution.Description
			t.Run(name, func(t *testing.T) {
				w := mustWorld(t, template.puzzle.ID)
				for _, placement := range solution.Phrases {
					mustSpawn(t, w, placement.Phrase, placement.X, placement.Y)
				}
				w.Step(30)
				state := w.Snapshot()
				if !state.Solved || !state.Goal.Met {
					t.Fatalf("this answer does not solve %s\n%s", template.puzzle.ID, describeState(state))
				}
			})
			checked++
		}
	}
	if checked < 16 {
		t.Fatalf("only %d answers were replayed; the game claims at least four per puzzle", checked)
	}
	t.Logf("replayed %d authored answers against the rule engine", checked)
}

// TestEveryAuthoredSolutionIsEasy enforces the difficulty target: one object
// (three at the very most), solved inside 25 ticks, with the player a short walk
// from the goal.
func TestEveryAuthoredSolutionIsEasy(t *testing.T) {
	for _, template := range builtinPuzzles() {
		for _, solution := range template.solutions {
			if len(solution.Phrases) > 3 {
				t.Fatalf("puzzle %s answer %q spends %d objects", template.puzzle.ID, solution.Description, len(solution.Phrases))
			}
			w := mustWorld(t, template.puzzle.ID)
			for _, placement := range solution.Phrases {
				mustSpawn(t, w, placement.Phrase, placement.X, placement.Y)
			}
			solvedAt := -1
			for tick := 1; tick <= 25; tick++ {
				w.Step(1)
				if w.Snapshot().Solved {
					solvedAt = tick
					break
				}
			}
			if solvedAt == -1 {
				t.Fatalf("puzzle %s answer %q did not finish inside 25 ticks\n%s",
					template.puzzle.ID, solution.Description, describeState(w.Snapshot()))
			}
		}
	}
}

// TestPlayerStartsNearTheGoal is the "never a long walk" rule.
func TestPlayerStartsNearTheGoal(t *testing.T) {
	for _, template := range builtinPuzzles() {
		w := mustWorld(t, template.puzzle.ID)
		state := w.Snapshot()
		goal, ok := goalPointForTest(w)
		if !ok {
			t.Fatalf("puzzle %s has no goal to measure against", template.puzzle.ID)
		}
		start := point{state.Player.X, state.Player.Y}
		if d := chebyshev(start, goal); d > 12 {
			t.Fatalf("puzzle %s starts the player %d cells from the goal", template.puzzle.ID, d)
		}
		if d := chebyshev(start, goal); d < 1 {
			t.Fatalf("puzzle %s starts the player on top of the goal", template.puzzle.ID)
		}
	}
}

// goalPointForTest reads the goal's location out of a live world.
func goalPointForTest(w *World) (point, bool) {
	w.mu.Lock()
	defer w.mu.Unlock()
	switch w.plan.kind {
	case GoalStar:
		if i, ok := w.entityByID(w.plan.starID); ok {
			return point{w.state.entities[i].X, w.state.entities[i].Y}, true
		}
	case GoalCandles:
		if len(w.plan.candleIDs) > 0 {
			if i, ok := w.entityByID(w.plan.candleIDs[0]); ok {
				return point{w.state.entities[i].X, w.state.entities[i].Y}, true
			}
		}
	case GoalChest:
		if i, ok := w.entityByID(w.plan.chestID); ok {
			return point{w.state.entities[i].X, w.state.entities[i].Y}, true
		}
	case GoalCross:
		return point{w.plan.waterMaxX + 1, w.plan.bankY}, true
	}
	return point{}, false
}

// TestNothingSolvesAPuzzleWithoutObjects: no objects means no win, whatever the
// player does. This is the negative test that keeps the referee honest.
func TestNothingSolvesAPuzzleWithoutObjects(t *testing.T) {
	for _, puzzle := range Puzzles() {
		w := mustWorld(t, puzzle.ID)
		w.Step(30)
		state := w.Snapshot()
		if state.Solved || state.Goal.Met {
			t.Fatalf("puzzle %s solved itself with no objects at all\n%s", puzzle.ID, describeState(state))
		}
		if len(state.Notebook) != 0 {
			t.Fatalf("puzzle %s recorded notebook entries it was never given", puzzle.ID)
		}
	}
}

// TestIrrelevantObjectsDoNotSolve: three cakes is not an answer to anything.
func TestIrrelevantObjectsDoNotSolve(t *testing.T) {
	for _, puzzle := range Puzzles() {
		w := mustWorld(t, puzzle.ID)
		for i := 0; i < 3; i++ {
			if _, _, _, err := w.Spawn("cake", 12+i*2, 5); err != nil {
				t.Fatalf("spawning a cake in %s failed: %v", puzzle.ID, err)
			}
		}
		w.Step(30)
		state := w.Snapshot()
		if state.Solved || state.Goal.Met {
			t.Fatalf("puzzle %s was solved by three cakes\n%s", puzzle.ID, describeState(state))
		}
		if len(state.Notebook) != 3 {
			t.Fatalf("puzzle %s notebook has %d entries, want 3", puzzle.ID, len(state.Notebook))
		}
	}
}

// TestResetAlwaysGivesASolvablePuzzle: a player cannot lose this game.
func TestResetAlwaysGivesASolvablePuzzle(t *testing.T) {
	for _, template := range builtinPuzzles() {
		w := mustWorld(t, template.puzzle.ID)
		// Make a mess: junk everywhere, ticks, actions, then check the answer
		// still works after a reset.
		if _, _, _, err := w.Spawn("cake", 12, 5); err != nil {
			t.Fatalf("spawn failed: %v", err)
		}
		w.Step(5)
		_ = w.Act("right")
		w.Reset()
		solution := template.solutions[0]
		for _, placement := range solution.Phrases {
			mustSpawn(t, w, placement.Phrase, placement.X, placement.Y)
		}
		w.Step(30)
		if !w.Snapshot().Solved {
			t.Fatalf("puzzle %s is unsolvable after a reset\n%s", template.puzzle.ID, describeState(w.Snapshot()))
		}
	}
}

// TestSolutionJSONNeverLeaks is the leak check: neither the puzzle list nor a
// snapshot may carry the words that would give an answer away.
func TestSolutionJSONNeverLeaks(t *testing.T) {
	raw, err := jsonMarshal(Puzzles())
	if err != nil {
		t.Fatalf("puzzles are not JSON: %v", err)
	}
	body := string(raw)
	for _, forbidden := range []string{"solutions", "Solutions", "Phrases", "Description"} {
		if strings.Contains(body, forbidden) {
			t.Fatalf("the puzzle list leaks %q: %s", forbidden, body)
		}
	}
	for _, puzzle := range Puzzles() {
		w := mustWorld(t, puzzle.ID)
		mustSpawn(t, w, "box", 12, 5)
		w.Step(3)
		snapshot := stateJSON(t, w)
		for _, forbidden := range []string{"solutions", "Solutions", "Phrases"} {
			if strings.Contains(snapshot, forbidden) {
				t.Fatalf("the snapshot of %s leaks %q", puzzle.ID, forbidden)
			}
		}
	}
}
