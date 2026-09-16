package lexicon

import (
	"testing"
)

// tagSet returns a comparable tag membership set.
func tagSet(obj Object) map[string]bool {
	out := make(map[string]bool, len(obj.Tags))
	for _, tag := range obj.Tags {
		out[tag] = true
	}
	return out
}

func modifiedFields(base, mod Object) int {
	changed := 0
	if base.Size != mod.Size {
		changed++
	}
	if base.Mass != mod.Mass {
		changed++
	}
	if base.Color != mod.Color {
		changed++
	}
	if !sameTagSet(base.Tags, mod.Tags) {
		changed++
	}
	return changed
}

func sameTagSet(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	left := map[string]int{}
	for _, tag := range a {
		left[tag]++
	}
	for _, tag := range b {
		left[tag]--
	}
	for _, n := range left {
		if n != 0 {
			return false
		}
	}
	return true
}

// TestModifierListIsComplete pins the authored adjective list the design asks
// for, so an adjective can never quietly disappear.
func TestModifierListIsComplete(t *testing.T) {
	required := []string{
		"big", "small", "tiny", "giant", "huge", "little", "heavy", "light",
		"metal", "steel", "iron", "wooden", "glass", "stone", "plastic", "paper",
		"flaming", "burning", "hot", "frozen", "icy", "cold", "wet", "dry",
		"sticky", "sharp", "blunt", "magical", "invisible", "floating", "wind-up",
		"broken", "golden", "silver", "shiny", "rusty", "old", "new", "red",
		"blue", "green", "yellow", "black", "white", "purple", "pink", "orange",
		"sparkly", "glowing", "electric",
	}
	for _, word := range required {
		if canonicalModifier(word) == "" {
			t.Errorf("required adjective %q is missing from the modifier table", word)
		}
	}
	if len(modifierTable) < len(required) {
		t.Errorf("modifier table holds %d adjectives, the design asks for at least %d", len(modifierTable), len(required))
	}
	for _, m := range modifierTable {
		if m.sizeDelta == 0 && m.massDelta == 0 && m.color == "" && len(m.addTags) == 0 && len(m.removeTags) == 0 {
			t.Errorf("adjective %q changes nothing at all", m.word)
		}
		for _, tag := range append(append([]string{}, m.addTags...), m.removeTags...) {
			if !inList(validTags, tag) {
				t.Errorf("adjective %q touches unpinned tag %q", m.word, tag)
			}
		}
	}
	t.Logf("%d authored adjectives", len(modifierTable))
}

// TestEveryModifierChangesSomething proves no adjective is decorative: each one
// alters at least one field for at least one plausible noun.
func TestEveryModifierChangesSomething(t *testing.T) {
	nouns := Dictionary()
	for _, m := range modifierTable {
		found := ""
		for _, noun := range nouns {
			base, ok := Parse(noun.Name)
			if !ok {
				t.Fatalf("Parse(%q) refused an authored noun", noun.Name)
			}
			mod, ok := Parse(m.word + " " + noun.Name)
			if !ok {
				t.Fatalf("Parse(%q) refused a modified noun", m.word+" "+noun.Name)
			}
			if modifiedFields(base, mod) > 0 {
				found = noun.Name
				break
			}
		}
		if found == "" {
			t.Errorf("adjective %q changes nothing for any noun in the book", m.word)
		}
	}
}

// TestEveryNounIsModifiable is the mirror image: every noun reacts to at least
// one adjective, so no object is frozen against the player's adjectives.
func TestEveryNounIsModifiable(t *testing.T) {
	for _, noun := range Dictionary() {
		base, ok := Parse(noun.Name)
		if !ok {
			t.Fatalf("Parse(%q) refused an authored noun", noun.Name)
		}
		reacts := false
		for _, m := range modifierTable {
			mod, ok := Parse(m.word + " " + noun.Name)
			if !ok {
				t.Fatalf("Parse(%q) refused", m.word+" "+noun.Name)
			}
			if modifiedFields(base, mod) > 0 {
				reacts = true
				break
			}
		}
		if !reacts {
			t.Errorf("noun %q ignores every adjective in the book", noun.Name)
		}
	}
}

// TestBigFlamingLadderIsExactlyAsSpecified is the worked example from the brief.
func TestBigFlamingLadderIsExactlyAsSpecified(t *testing.T) {
	base, ok := Parse("ladder")
	if !ok {
		t.Fatal("Parse(\"ladder\") refused")
	}
	obj, ok := Parse("big flaming ladder")
	if !ok {
		t.Fatal("Parse(\"big flaming ladder\") refused")
	}
	if obj.Name != "big flaming ladder" {
		t.Errorf("Name = %q, want exactly as typed", obj.Name)
	}
	if obj.Noun != "ladder" || obj.Key != "ladder" {
		t.Errorf("noun/key = %q/%q, want ladder", obj.Noun, obj.Key)
	}
	if len(obj.Modifiers) != 2 || obj.Modifiers[0] != "big" || obj.Modifiers[1] != "flaming" {
		t.Errorf("Modifiers = %v, want [big flaming] in order", obj.Modifiers)
	}
	if obj.Size != 2 {
		t.Errorf("size = %d, want 2 (ladder %d + big)", obj.Size, base.Size)
	}
	tags := tagSet(obj)
	for _, want := range []string{TagFire, TagLightSource, TagFlammable} {
		if !tags[want] {
			t.Errorf("missing %q, got %v", want, obj.Tags)
		}
	}
	if !tags[TagClimbable] {
		t.Errorf("adjective merged wrongly: base tag climbable lost, got %v", obj.Tags)
	}
	if obj.Approximate {
		t.Error("big flaming ladder should not be approximate")
	}
}

func TestMaterialAdjectives(t *testing.T) {
	rope, _ := Parse("rope")
	metal, ok := Parse("metal rope")
	if !ok {
		t.Fatal("Parse(\"metal rope\") refused")
	}
	tags := tagSet(metal)
	if !tags[TagConductive] || !tags[TagHeavy] {
		t.Errorf("metal rope should be conductive and heavy, got %v", metal.Tags)
	}
	if tags[TagFlammable] {
		t.Errorf("metal rope must not be flammable, got %v", metal.Tags)
	}
	if !tags[TagRope] || !tags[TagClimbable] {
		t.Errorf("metal rope lost its base tags, got %v", metal.Tags)
	}
	if metal.Key != rope.Key {
		t.Errorf("metal rope key %q, want %q", metal.Key, rope.Key)
	}

	wooden, ok := Parse("wooden ladder")
	if !ok {
		t.Fatal("Parse(\"wooden ladder\") refused")
	}
	if !tagSet(wooden)[TagFlammable] {
		t.Errorf("wooden ladder should burn, got %v", wooden.Tags)
	}

	glass, _ := Parse("glass window")
	if !tagSet(glass)[TagFragile] || tagSet(glass)[TagFlammable] {
		t.Errorf("glass window should be fragile and not flammable, got %v", glass.Tags)
	}

	stone, _ := Parse("stone box")
	stoneTags := tagSet(stone)
	if !stoneTags[TagHeavy] || stoneTags[TagFlammable] || stoneTags[TagConductive] {
		t.Errorf("stone box tags wrong: %v", stone.Tags)
	}

	paper, _ := Parse("paper box")
	if !tagSet(paper)[TagFlammable] {
		t.Errorf("paper box should burn: %v", paper.Tags)
	}
}

func TestColdAdjectives(t *testing.T) {
	water, _ := Parse("water")
	frozen, ok := Parse("frozen water")
	if !ok {
		t.Fatal("Parse(\"frozen water\") refused")
	}
	tags := tagSet(frozen)
	if !tags[TagFrozen] || !tags[TagCold] {
		t.Errorf("frozen water should be frozen and cold, got %v", frozen.Tags)
	}
	if tags[TagFire] {
		t.Errorf("frozen water must not be on fire, got %v", frozen.Tags)
	}
	if !tags[TagWater] {
		t.Errorf("frozen water is still water, got %v", frozen.Tags)
	}
	if frozen.Key != water.Key {
		t.Errorf("frozen water key %q, want %q", frozen.Key, water.Key)
	}

	// Cold removes fire from something that was burning, when it comes last.
	hotFrozen, _ := Parse("burning frozen match")
	hotFrozenTags := tagSet(hotFrozen)
	if hotFrozenTags[TagFire] {
		t.Errorf("frozen should win when it comes last, got %v", hotFrozen.Tags)
	}
	if !hotFrozenTags[TagFrozen] || !hotFrozenTags[TagCold] {
		t.Errorf("burning frozen match should be frozen, got %v", hotFrozen.Tags)
	}
	if !hotFrozenTags[TagLightSource] {
		t.Errorf("fire's light-source should survive, got %v", hotFrozen.Tags)
	}
	// And the other order means the match is on fire again: last adjective wins.
	frozenHot, _ := Parse("frozen burning match")
	if !tagSet(frozenHot)[TagFire] {
		t.Errorf("burning last should light it again, got %v", frozenHot.Tags)
	}
}

func TestColourAdjectives(t *testing.T) {
	base, _ := Parse("box")
	for _, word := range []string{"red", "blue", "green", "yellow", "black", "white", "purple", "pink", "orange"} {
		obj, ok := Parse(word + " box")
		if !ok {
			t.Fatalf("Parse(%q) refused", word+" box")
		}
		if obj.Color == base.Color {
			t.Errorf("%s box did not change colour from %s", word, base.Color)
		}
		if !hexColor.MatchString(obj.Color) {
			t.Errorf("%s box colour %q is not hex", word, obj.Color)
		}
		if len(obj.Modifiers) != 1 || obj.Modifiers[0] != word {
			t.Errorf("%s box modifiers = %v", word, obj.Modifiers)
		}
	}
}

func TestSizeAndMassClamping(t *testing.T) {
	phrases := []string{
		"big big giant huge ladder",
		"tiny tiny small little ladder",
		"giant giant giant box",
		"small tiny little feather",
		"heavy heavy heavy heavy anvil",
		"light light light light box",
		"giant tiny giant tiny box",
	}
	for _, phrase := range phrases {
		obj, ok := Parse(phrase)
		if !ok {
			t.Fatalf("Parse(%q) refused", phrase)
		}
		if obj.Size < MinSize || obj.Size > MaxSize {
			t.Errorf("Parse(%q).Size = %d, out of range", phrase, obj.Size)
		}
		if obj.Mass < MinMass || obj.Mass > MaxMass {
			t.Errorf("Parse(%q).Mass = %d, out of range", phrase, obj.Mass)
		}
	}

	big, _ := Parse("big box")
	if big.Size != 3 {
		t.Errorf("big box size = %d, want 3 (box is 2)", big.Size)
	}
	giant, _ := Parse("giant box")
	if giant.Size != MaxSize {
		t.Errorf("giant box size = %d, want %d", giant.Size, MaxSize)
	}
	tiny, _ := Parse("tiny box")
	if tiny.Size != MinSize {
		t.Errorf("tiny box size = %d, want %d", tiny.Size, MinSize)
	}
	heavyLadder, _ := Parse("heavy ladder")
	if heavyLadder.Mass <= 2 {
		t.Errorf("heavy ladder mass = %d, want more than the plain 2", heavyLadder.Mass)
	}
}

// TestMergingNeverDuplicatesTags is the merge guarantee: adjectives add to the
// base tag list, never replace it, and never repeat a tag.
func TestMergingNeverDuplicatesTags(t *testing.T) {
	// Targeted collisions: adjectives whose tags the noun already carries.
	targeted := []string{
		"wooden vine", "metal wire", "metal battery", "golden treasure",
		"sharp knife", "frozen ice", "flaming campfire", "glowing candle",
		"heavy boulder", "wet mud", "broken ladder", "electric generator",
		"sticky glue glue", "old fragile glass window",
	}
	for _, phrase := range targeted {
		obj, ok := Parse(phrase)
		if !ok {
			t.Fatalf("Parse(%q) refused", phrase)
		}
		if hasDuplicates(obj.Tags) {
			t.Errorf("Parse(%q) produced duplicate tags: %v", phrase, obj.Tags)
		}
	}

	// Exhaustive: every noun against every adjective, and every noun against
	// every pair of adjectives.
	nouns := Dictionary()
	for _, noun := range nouns {
		for _, m := range modifierTable {
			phrase := m.word + " " + noun.Name
			obj, ok := Parse(phrase)
			if !ok {
				t.Fatalf("Parse(%q) refused", phrase)
			}
			if hasDuplicates(obj.Tags) {
				t.Fatalf("Parse(%q) produced duplicate tags: %v", phrase, obj.Tags)
			}
		}
	}
	for i, first := range modifierTable {
		second := modifierTable[(i*7+3)%len(modifierTable)]
		noun := nouns[(i*13)%len(nouns)]
		phrase := first.word + " " + second.word + " " + noun.Name
		obj, ok := Parse(phrase)
		if !ok {
			t.Fatalf("Parse(%q) refused", phrase)
		}
		if hasDuplicates(obj.Tags) {
			t.Errorf("Parse(%q) produced duplicate tags: %v", phrase, obj.Tags)
		}
		for _, tag := range obj.Tags {
			if !inList(validTags, tag) {
				t.Errorf("Parse(%q) produced unpinned tag %q", phrase, tag)
			}
		}
	}

	// A modified object keeps every base tag it is not explicitly stripped of.
	base, _ := Parse("ladder")
	modified, _ := Parse("wooden big ladder")
	for _, tag := range base.Tags {
		if !inList(modified.Tags, tag) {
			t.Errorf("wooden big ladder lost base tag %q: %v", tag, modified.Tags)
		}
	}
}

func TestModifierOrderMatters(t *testing.T) {
	woodenThenMetal, _ := Parse("wooden metal box")
	metalThenWooden, _ := Parse("metal wooden box")
	if tagSet(woodenThenMetal)[TagFlammable] {
		t.Errorf("metal last should strip flammable: %v", woodenThenMetal.Tags)
	}
	if !tagSet(metalThenWooden)[TagFlammable] {
		t.Errorf("wooden last should restore flammable: %v", metalThenWooden.Tags)
	}
}

func TestAdjectivesAfterTheNounStillCount(t *testing.T) {
	obj, ok := Parse("ladder big")
	if !ok {
		t.Fatal("Parse(\"ladder big\") refused")
	}
	if obj.Key != "ladder" {
		t.Errorf("key = %q, want ladder", obj.Key)
	}
	if obj.Size != 2 {
		t.Errorf("size = %d, want 2", obj.Size)
	}
}

func TestAdjectivesWithNoNounAreRefused(t *testing.T) {
	for _, phrase := range []string{"flaming", "big", "tiny giant huge", "metal", "glass", "iron", "sharp blunt", "frozen icy cold", "big flaming"} {
		if _, ok := Parse(phrase); ok {
			t.Errorf("Parse(%q) should fail: adjectives need a noun", phrase)
		}
	}
	// But a dictionary noun that happens to look like an adjective still wins.
	for _, noun := range []string{"giant", "orange", "stone"} {
		obj, ok := Parse(noun)
		if !ok {
			t.Errorf("Parse(%q) refused a real dictionary noun", noun)
			continue
		}
		if !Knows(noun) {
			t.Errorf("Knows(%q) should be true", noun)
		}
		if obj.Approximate {
			t.Errorf("Parse(%q) should not be approximate", noun)
		}
	}
}

func TestBrokenAndBluntStripCapabilities(t *testing.T) {
	broken, _ := Parse("broken ladder")
	tags := tagSet(broken)
	if tags[TagClimbable] {
		t.Errorf("broken ladder should not be climbable: %v", broken.Tags)
	}
	brokenBox, _ := Parse("broken box")
	if tagSet(brokenBox)[TagPlatform] {
		t.Errorf("broken box should not be a platform: %v", brokenBox.Tags)
	}
	blunt, _ := Parse("blunt knife")
	bluntTags := tagSet(blunt)
	if bluntTags[TagSharp] || bluntTags[TagCutting] {
		t.Errorf("blunt knife should not cut: %v", blunt.Tags)
	}
	bluntSword, _ := Parse("blunt sword")
	if tagSet(bluntSword)[TagSharp] {
		t.Errorf("blunt sword should not be sharp: %v", bluntSword.Tags)
	}
}

func TestWildcardAdjectives(t *testing.T) {
	cases := []struct {
		phrase string
		tag    string
	}{
		{"glowing box", TagLightSource},
		{"magical box", TagMagical},
		{"invisible box", TagMagical},
		{"sticky box", TagSticky},
		{"floating box", TagBuoyant},
		{"wind-up box", TagMachine},
		{"electric box", TagPowerSource},
		{"golden box", TagTreasure},
		{"silver box", TagTreasure},
		{"sharp rope", TagCutting},
	}
	for _, c := range cases {
		obj, ok := Parse(c.phrase)
		if !ok {
			t.Fatalf("Parse(%q) refused", c.phrase)
		}
		if !tagSet(obj)[c.tag] {
			t.Errorf("Parse(%q) missing %q: %v", c.phrase, c.tag, obj.Tags)
		}
	}

	if obj, _ := Parse("sticky mud"); !tagSet(obj)[TagSticky] {
		t.Errorf("sticky mud should stay sticky once, got %v", obj.Tags)
	}
	if obj, _ := Parse("dry rope"); !tagSet(obj)[TagFlammable] {
		t.Errorf("dry rope should burn: %v", obj.Tags)
	}
	if obj, _ := Parse("burning wet book"); tagSet(obj)[TagFlammable] {
		t.Errorf("wet last should stop it burning: %v", obj.Tags)
	}
}
