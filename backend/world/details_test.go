package world

import (
	"strings"
	"testing"

	"backend/lexicon"
)

// Details of the rule list that need their own small, precise test: hands, a
// plain jump, a machine that is not a lift, and light sources.

func TestHandsHoldOneThingAndItFollowsThePlayer(t *testing.T) {
	rows := flatRows(20, 12)
	w := specWorld(t, GoalChest, rows, []SpecEntity{{Phrase: "chest", X: 16, Y: 10}}, 6, 10)

	mustSpawn(t, w, "coin", 7, 10)
	if err := w.Act("take"); err != nil {
		t.Fatalf("taking a size-1 coin beside the player failed: %v", err)
	}
	if held := playerOf(w).Holding; held == 0 {
		t.Fatalf("the player should be holding the coin")
	}
	// The held thing follows the player.
	if err := w.Act("left"); err != nil {
		t.Fatalf("walking while holding something failed: %v", err)
	}
	player := playerOf(w)
	coin := entitiesByKey(w, "coin")[0]
	if coin.X != player.X || coin.Y != player.Y {
		t.Fatalf("the held coin did not follow the player: player %+v coin (%d,%d)", player, coin.X, coin.Y)
	}

	// Hands hold one thing: taking something new drops the old one.
	mustSpawn(t, w, "key", player.X-1, player.Y)
	if err := w.Act("take"); err != nil {
		t.Fatalf("taking a second size-1 object failed: %v", err)
	}
	if coin = entitiesByKey(w, "coin")[0]; coin.Held {
		t.Fatalf("the old object should have been dropped when the hands took something new")
	}
	if !entitiesByKey(w, "key")[0].Held {
		t.Fatalf("the new object should be the one being held")
	}
}

func TestPlainJumpGoesUpExactlyOneCell(t *testing.T) {
	rows := flatRows(20, 12)
	w := specWorld(t, GoalChest, rows, []SpecEntity{{Phrase: "chest", X: 16, Y: 10}}, 6, 10)
	before := playerOf(w)
	if err := w.Act("jump"); err != nil {
		t.Fatalf("jumping on flat ground failed: %v", err)
	}
	after := playerOf(w)
	if before.Y-after.Y != 1 {
		t.Fatalf("a plain jump should rise one cell: %d -> %d", before.Y, after.Y)
	}
	if after.X != before.X {
		t.Fatalf("a plain jump should not move the player sideways")
	}
}

func TestPoweredMachineWithoutALiftJustHums(t *testing.T) {
	rows := flatRows(20, 12)
	rows[8] = "..........#........."
	w := specWorld(t, GoalChest, rows, []SpecEntity{{Phrase: "chest", X: 10, Y: 7}}, 10, 10)

	// A fan is a machine with nothing to lift.
	mustSpawn(t, w, "battery", 11, 10)
	mustSpawn(t, w, "wire", 12, 10)
	fan := mustSpawn(t, w, "fan", 13, 10)
	w.Step(1)
	if !entitiesByKey(w, "fan")[0].Powered {
		t.Fatalf("a machine on a wire should be powered\n%s", describeState(w.Snapshot()))
	}
	if !anyEventContains(eventsOf(w), "hums to life") {
		t.Fatalf("a powered machine should announce itself: %v", eventsOf(w))
	}
	w.Step(5)
	if after := entitiesByKey(w, "fan")[0]; after.Y != fan.Y {
		t.Fatalf("a machine that is not a lift should not rise: y %d -> %d", fan.Y, after.Y)
	}
}

func TestALightSourceShinesWithoutBurningAnything(t *testing.T) {
	w := mustWorld(t, "star-in-tree")
	mustSpawn(t, w, "rope", 12, 8)
	mustSpawn(t, w, "lamp", 13, 8)
	w.Step(5)
	rope, ok := findByKey(w, "rope")
	if !ok {
		t.Fatalf("a lamp should not burn a rope away\n%s", describeState(w.Snapshot()))
	}
	if rope.Burning {
		t.Fatalf("a light source does not set things alight, only fire does")
	}
}

// TestKeyOpensAContainerBesideItOnly is rule 6 in isolation: adjacency, not
// possession.
func TestKeyOpensAContainerBesideItOnly(t *testing.T) {
	rows := flatRows(20, 12)
	w := specWorld(t, GoalChest, rows, []SpecEntity{{Phrase: "chest", X: 10, Y: 10}}, 8, 10)

	// A key far from the chest does nothing.
	mustSpawn(t, w, "key", 3, 10)
	w.Step(3)
	chest, ok := findByKey(w, "chest")
	if !ok {
		t.Fatalf("the chest should still be there")
	}
	if chest.Opened {
		t.Fatalf("a key on the far side of the room should not open the chest")
	}
	if !anyEventContains(eventsOf(w), "still locked") && !strings.Contains(w.Snapshot().Goal.Progress, "locked") {
		t.Fatalf("the goal should still read as locked: %q", w.Snapshot().Goal.Progress)
	}

	// A key beside it does.
	mustSpawn(t, w, "key", 9, 10)
	w.Step(1)
	chest = entitiesByKey(w, "chest")[0]
	if !chest.Opened {
		t.Fatalf("a key beside the chest should open it\n%s", describeState(w.Snapshot()))
	}
	if !anyEventContains(eventsOf(w), "opens the") {
		t.Fatalf("unlocking should be logged: %v", eventsOf(w))
	}
	if !w.Snapshot().Solved {
		t.Fatalf("an opened chest solves the chest puzzle")
	}
}

// TestAffordanceOverlayIsDocumented checks the one place the world adds tags to
// the word book: every entry must say why, and must promise only tags that the
// engine actually reads.
func TestAffordanceOverlayIsDocumented(t *testing.T) {
	if len(affordanceOverlay) == 0 {
		t.Fatal("the overlay should not be empty")
	}
	for noun, note := range affordanceOverlay {
		if strings.TrimSpace(note.why) == "" {
			t.Fatalf("the overlay entry for %q does not explain itself", noun)
		}
		if len(note.tags) == 0 {
			t.Fatalf("the overlay entry for %q promises nothing", noun)
		}
		obj, ok := lexicon.Parse(noun)
		if !ok {
			t.Fatalf("the overlay mentions %q, which the book cannot read", noun)
		}
		if obj.Key != noun {
			t.Fatalf("the overlay key %q does not match the book's key %q", noun, obj.Key)
		}
		got := applyAffordances(obj)
		for _, tag := range note.tags {
			if !hasObjectTag(got, tag) {
				t.Fatalf("applyAffordances(%q) should promise %q", noun, tag)
			}
		}
		// Adjectives are the book's business: "broken ladder" must not come back
		// climbable because a plain ladder is.
		modified, ok := lexicon.Parse("broken " + noun)
		if ok {
			withMods := applyAffordances(modified)
			if len(withMods.Modifiers) == 0 && len(modified.Modifiers) > 0 {
				t.Fatalf("applyAffordances should leave modified phrases alone")
			}
			for _, tag := range note.tags {
				if !containsTag(modified.Tags, tag) && containsTag(withMods.Tags, tag) {
					t.Fatalf("a modified %q should keep the book's tags, the overlay added %q", noun, tag)
				}
			}
		}
	}
}

// TestOverlayPromisesHoldInTheWorld plays each overlay noun for real and checks
// the promised tag is what the engine actually sees.
func TestOverlayPromisesHoldInTheWorld(t *testing.T) {
	w := mustWorld(t, "star-in-tree")
	mustSpawn(t, w, "rope", 12, 8)
	spawned, ok := findByKey(w, "rope")
	if !ok {
		t.Fatal("the rope is missing")
	}
	if !hasTag(spawned, lexicon.TagFlammable) {
		t.Fatalf("a rope is hemp and should burn: %v", spawned.Object.Tags)
	}

	trampoline := mustSpawn(t, w, "trampoline", 14, 3)
	if !hasTag(trampoline, lexicon.TagLift) || !hasTag(trampoline, lexicon.TagMachine) {
		t.Fatalf("a trampoline should be a lift the engine can run: %v", trampoline.Object.Tags)
	}
	hammer := mustSpawn(t, w, "hammer", 16, 6)
	if !hasTag(hammer, lexicon.TagUnlock) {
		t.Fatalf("a hammer should defeat a lock: %v", hammer.Object.Tags)
	}
}
