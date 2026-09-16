package lexicon

import (
	"fmt"
	"reflect"
	"strings"
	"testing"
	"unicode/utf8"
)

// invented holds nonsense nouns the book has never seen: nonsense, schoolyard
// words, numbers, other scripts and symbols.
var invented = []string{
	"zorble", "flimbly", "krag", "wibbletron", "snorf", "blorptastic",
	"quazzle", "grumbly", "tibber", "plonker", "vrelp", "snazzlewump",
	"fjordle", "zorbletron", "bleddern", "chuffwump", "glorptangle",
	"999", "x", "qwertyuiop", "梯子", "火箭", "エアロ", "Ωμέγα", "мир",
	"phlogiston", "unobtainium", "thingamajig", "whatsit", "doohickey",
}

func TestUnknownNounsAreNeverRefused(t *testing.T) {
	for _, noun := range invented {
		obj, ok := Parse(noun)
		if !ok {
			t.Errorf("Parse(%q) refused an unknown noun; the book must never refuse", noun)
			continue
		}
		if !obj.Approximate {
			t.Errorf("Parse(%q).Approximate = false, want true", noun)
		}
		if Knows(noun) {
			t.Errorf("Knows(%q) = true for an improvised noun", noun)
		}
		if !inList(pinnedShapes, obj.Shape) {
			t.Errorf("Parse(%q) shape %q is not pinned", noun, obj.Shape)
		}
		if !inList(pinnedCategories, obj.Category) {
			t.Errorf("Parse(%q) category %q is not pinned", noun, obj.Category)
		}
		if !hexColor.MatchString(obj.Color) {
			t.Errorf("Parse(%q) colour %q is not hex", noun, obj.Color)
		}
		if obj.Mass < MinMass || obj.Mass > MaxMass || obj.Size < MinSize || obj.Size > MaxSize {
			t.Errorf("Parse(%q) mass/size out of range: %d/%d", noun, obj.Mass, obj.Size)
		}
		if n := utf8.RuneCountInString(obj.Glyph); n < 1 || n > 3 {
			t.Errorf("Parse(%q) glyph %q must be 1-3 characters", noun, obj.Glyph)
		}
		if !strings.Contains(obj.Note, obj.Noun) {
			t.Errorf("Parse(%q) note %q should name the improvised noun", noun, obj.Note)
		}
		if hasDuplicates(obj.Tags) {
			t.Errorf("Parse(%q) duplicate tags %v", noun, obj.Tags)
		}
		for _, tag := range obj.Tags {
			if !inList(validTags, tag) {
				t.Errorf("Parse(%q) unpinned tag %q", noun, tag)
			}
		}
	}
}

// TestImproviseIsStableAcrossCalls is the determinism guarantee: the same
// nonsense noun always produces the same object, in the same process and in any
// other run, because everything comes from a hash and not from a random source.
func TestImproviseIsStableAcrossCalls(t *testing.T) {
	for _, noun := range invented {
		first, ok := Parse(noun)
		if !ok {
			t.Fatalf("Parse(%q) refused", noun)
		}
		for i := 0; i < 5; i++ {
			again, ok := Parse(noun)
			if !ok {
				t.Fatalf("Parse(%q) refused on repeat", noun)
			}
			if !reflect.DeepEqual(first, again) {
				t.Fatalf("Parse(%q) is not stable:\n%v\n%v", noun, first, again)
			}
		}
	}

	// Case must not change the object, only the display name.
	lower, _ := Parse("zorble")
	upper, ok := Parse("ZORBLE")
	if !ok {
		t.Fatal("Parse(\"ZORBLE\") refused")
	}
	if upper.Key != lower.Key || upper.Noun != lower.Noun || upper.Shape != lower.Shape ||
		upper.Color != lower.Color || upper.Mass != lower.Mass || upper.Size != lower.Size ||
		upper.Glyph != lower.Glyph || upper.Note != lower.Note {
		t.Errorf("case changed the improvised object:\n%v\n%v", lower, upper)
	}
	if upper.Name != "ZORBLE" {
		t.Errorf("Name = %q, want the phrase as typed", upper.Name)
	}

	// A recorded snapshot: these exact objects were produced when the book was
	// written, so a future change to the hash, the palette or the note templates
	// cannot silently reshape every world that already improvised a noun.
	snapshot := map[string]struct {
		shape string
		color string
		mass  int
		size  int
		glyph string
	}{
		"zorble":  {ShapeKey, "#5a6b4a", 2, 3, "ZOR"},
		"flimbly": {ShapeStar, "#d94f4f", 3, 1, "FLI"},
		"krag":    {ShapePerson, "#5a8ad9", 3, 2, "KRA"},
		"zhorble": {ShapeMachine, "#5a6b4a", 1, 2, "ZHO"},
		"plimp":   {ShapeBottle, "#7d7d85", 1, 1, "PLI"},
	}
	for noun, want := range snapshot {
		obj, ok := Parse(noun)
		if !ok {
			t.Fatalf("Parse(%q) refused", noun)
		}
		if obj.Shape != want.shape || obj.Color != want.color || obj.Mass != want.mass ||
			obj.Size != want.size || obj.Glyph != want.glyph {
			t.Errorf("Parse(%q) = %s/%s/%d/%d/%s, want %s/%s/%d/%d/%s",
				noun, obj.Shape, obj.Color, obj.Mass, obj.Size, obj.Glyph,
				want.shape, want.color, want.mass, want.size, want.glyph)
		}
	}
}

func TestImprovisedModifiersApply(t *testing.T) {
	plain, ok := Parse("zorble")
	if !ok {
		t.Fatal("Parse(\"zorble\") refused")
	}

	big, ok := Parse("big zorble")
	if !ok {
		t.Fatal("Parse(\"big zorble\") refused")
	}
	if !big.Approximate {
		t.Error("big zorble should be approximate")
	}
	if big.Noun != "zorble" || big.Key != "zorble" {
		t.Errorf("noun/key = %q/%q, want zorble", big.Noun, big.Key)
	}
	if big.Name != "big zorble" {
		t.Errorf("Name = %q, want exactly as typed", big.Name)
	}
	if len(big.Modifiers) != 1 || big.Modifiers[0] != "big" {
		t.Errorf("Modifiers = %v, want [big]", big.Modifiers)
	}
	wantSize := clamp(plain.Size+1, MinSize, MaxSize)
	if big.Size != wantSize {
		t.Errorf("big zorble size = %d, want %d", big.Size, wantSize)
	}
	if plain.Approximate == false {
		t.Error("the plain improvised noun should be approximate too")
	}

	flaming, ok := Parse("flaming zorble")
	if !ok {
		t.Fatal("Parse(\"flaming zorble\") refused")
	}
	if !flaming.Approximate {
		t.Error("flaming zorble should be approximate")
	}
	tags := tagSet(flaming)
	for _, want := range []string{TagFire, TagLightSource, TagFlammable} {
		if !tags[want] {
			t.Errorf("flaming zorble missing %q: %v", want, flaming.Tags)
		}
	}
	if flaming.Modifiers[0] != "flaming" {
		t.Errorf("Modifiers = %v", flaming.Modifiers)
	}

	tinyFrozen, ok := Parse("tiny frozen zorble")
	if !ok {
		t.Fatal("Parse(\"tiny frozen zorble\") refused")
	}
	if tinyFrozen.Size != MinSize {
		t.Errorf("tiny frozen zorble size = %d, want %d", tinyFrozen.Size, MinSize)
	}
	if !tagSet(tinyFrozen)[TagFrozen] {
		t.Errorf("tiny frozen zorble should be frozen: %v", tinyFrozen.Tags)
	}
	if len(tinyFrozen.Modifiers) != 2 || tinyFrozen.Modifiers[0] != "tiny" || tinyFrozen.Modifiers[1] != "frozen" {
		t.Errorf("Modifiers = %v, want [tiny frozen]", tinyFrozen.Modifiers)
	}
}

// TestImprovisedSpellingRules checks the suffix and shape hints.
func TestImprovisedSpellingRules(t *testing.T) {
	cases := []struct {
		noun string
		cat  string
	}{
		{"scooper", CatTool},
		{"zorper", CatTool},
		{"vibblor", CatTool},
		{"terrifier", CatTool},
		{"zorbletron", CatMystery},
	}
	for _, c := range cases {
		obj, ok := Parse(c.noun)
		if !ok {
			t.Fatalf("Parse(%q) refused", c.noun)
		}
		if obj.Category != c.cat {
			t.Errorf("Parse(%q).Category = %q, want %q", c.noun, obj.Category, c.cat)
		}
	}

	tool, _ := Parse("scooper")
	if !tagSet(tool)[TagTool] && !tagSet(tool)[TagMachine] {
		t.Errorf("a -er invention should at least be a tool: %v", tool.Tags)
	}

	for _, noun := range []string{"zorbbing", "flumping", "wibbling"} {
		obj, ok := Parse(noun)
		if !ok {
			t.Fatalf("Parse(%q) refused", noun)
		}
		if obj.Category != CatMachine && obj.Category != CatMystery {
			t.Errorf("Parse(%q) with -ing should be a machine or a mystery, got %q", noun, obj.Category)
		}
	}

	for _, noun := range []string{"aeioua", "ooeeoo", "aieeia"} {
		obj, ok := Parse(noun)
		if !ok {
			t.Fatalf("Parse(%q) refused", noun)
		}
		if obj.Shape != ShapeBlob {
			t.Errorf("a pile of vowels like %q should be drawn as a blob, got %q", noun, obj.Shape)
		}
	}
}

// TestImprovisedWordsSpreadOut confirms the hash actually spreads invented
// nouns across the pinned shapes and the fixed palette instead of piling them
// all on one.
func TestImprovisedWordsSpreadOut(t *testing.T) {
	shapes := map[string]int{}
	colors := map[string]int{}
	glyphs := map[string]int{}
	notes := map[string]int{}
	for i := 0; i < 400; i++ {
		noun := fmt.Sprintf("%c%c%c%s", 'a'+(i%26), 'a'+(i/26)%26, 'a'+(i/676)%26, "shamble")
		obj, ok := Parse(noun)
		if !ok {
			t.Fatalf("Parse(%q) refused", noun)
		}
		if !inList(pinnedShapes, obj.Shape) {
			t.Fatalf("Parse(%q) shape %q is not pinned", noun, obj.Shape)
		}
		if !inList(improvisedColors, obj.Color) {
			t.Fatalf("Parse(%q) colour %q is not from the palette", noun, obj.Color)
		}
		if obj.Mass < 1 || obj.Mass > 3 || obj.Size < 1 || obj.Size > 3 {
			t.Fatalf("Parse(%q) mass/size should start inside 1..3, got %d/%d", noun, obj.Mass, obj.Size)
		}
		if obj.Category != CatTool && obj.Category != CatMachine && obj.Category != CatMystery {
			t.Fatalf("Parse(%q) category %q is not from the improvisation rules", noun, obj.Category)
		}
		shapes[obj.Shape]++
		colors[obj.Color]++
		glyphs[obj.Glyph]++
		notes[obj.Note]++
	}
	if len(shapes) < 8 {
		t.Errorf("improvised nouns only ever used %d shapes: %v", len(shapes), shapes)
	}
	if len(colors) < 6 {
		t.Errorf("improvised nouns only ever used %d colours: %v", len(colors), colors)
	}
	if len(notes) < 4 {
		t.Errorf("improvised notes are too repetitive: %d variants", len(notes))
	}
	if len(glyphs) < 50 {
		t.Errorf("improvised glyphs are too repetitive: %d variants", len(glyphs))
	}
}

func TestImproviseDirectlyFromTheHash(t *testing.T) {
	obj := improviseObject("Zorble")
	if obj.Key != "zorble" || obj.Noun != "zorble" {
		t.Errorf("key/noun = %q/%q, want zorble", obj.Key, obj.Noun)
	}
	if !obj.Approximate {
		t.Error("improvised objects are approximate by definition")
	}
	if obj.Glyph != "ZOR" {
		t.Errorf("glyph = %q, want ZOR", obj.Glyph)
	}
	if obj.Modifiers == nil {
		t.Error("Modifiers should be an empty slice, not nil")
	}
	if !strings.Contains(obj.Note, "zorble") {
		t.Errorf("note %q should mention the noun", obj.Note)
	}
	if obj.Size < 1 || obj.Size > 3 || obj.Mass < 1 || obj.Mass > 3 {
		t.Errorf("improvised mass/size must start inside 1..3, got %d/%d", obj.Mass, obj.Size)
	}
	if got := improviseGlyph("🚀"); got != "🚀" {
		t.Errorf("improvised glyph for a symbol = %q, want the symbol itself", got)
	}
	if got := improviseGlyph("ninety-nine"); utf8.RuneCountInString(got) > 3 {
		t.Errorf("glyph %q is more than three characters", got)
	}
}

func TestMultiWordUnknownPhrases(t *testing.T) {
	obj, ok := Parse("blimble florb")
	if !ok {
		t.Fatal("Parse(\"blimble florb\") refused")
	}
	if !obj.Approximate {
		t.Error("an unknown multi-word phrase should still spawn an approximate object")
	}
	if obj.Noun != "florb" {
		t.Errorf("noun = %q, want the last word as the noun", obj.Noun)
	}

	big, ok := Parse("big zorble tron")
	if !ok {
		t.Fatal("Parse(\"big zorble tron\") refused")
	}
	if big.Modifiers[0] != "big" {
		t.Errorf("leading adjective lost: %v", big.Modifiers)
	}
	if !big.Approximate {
		t.Error("big zorble tron should be approximate")
	}
}
