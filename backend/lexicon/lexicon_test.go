package lexicon

import (
	"fmt"
	"reflect"
	"regexp"
	"sort"
	"strings"
	"sync"
	"testing"
	"unicode/utf8"
)

var hexColor = regexp.MustCompile(`^#[0-9a-f]{6}$`)

func inList(list []string, want string) bool {
	for _, s := range list {
		if s == want {
			return true
		}
	}
	return false
}

// requiredVocabulary is the word list the game design asks the bookshelf to
// cover. Every one of these must be known, either directly or as an alias.
var requiredVocabulary = []string{
	// climbing and getting up
	"ladder", "rope", "grapple", "grappling hook", "vine", "chain", "stairs",
	"elevator", "jetpack", "balloon", "spring", "trampoline", "box", "crate",
	"barrel", "plank", "bridge", "raft", "boat", "canoe", "surfboard", "log",
	"stool", "chair", "table", "steps", "scaffolding",
	// fire and light
	"match", "lighter", "torch", "candle", "campfire", "bonfire", "fireplace",
	"lantern", "lamp", "flashlight", "flamethrower", "fire extinguisher",
	"water", "bucket", "hose", "rain cloud", "ice", "ice cube", "freezer",
	"fridge", "snow", "fan", "air conditioner", "magnifying glass", "sun",
	// cutting
	"knife", "sword", "axe", "saw", "scissors", "chainsaw", "drill", "crowbar",
	"hammer", "nail", "screwdriver", "wire", "tape", "glue", "glue stick",
	// unlocking and blowing up
	"key", "master key", "lockpick", "bomb", "dynamite", "tnt", "grenade",
	"rocket", "cannon", "magic wand", "wizard", "spell",
	// living things and food
	"dog", "cat", "bird", "rabbit", "horse", "elephant", "monkey", "fish",
	"bee", "ant", "worm", "tree", "bush", "flower", "grass", "apple", "banana",
	"bread", "cake", "cheese", "sandwich", "milk", "coffee", "tea", "pizza",
	// school, house and garage
	"desk", "computer", "phone", "television", "door", "gate", "window",
	"mirror", "pillow", "blanket", "bed", "sofa", "toilet", "sink", "shower",
	"bathtub", "fork", "spoon", "plate", "cup", "bottle", "bag", "backpack",
	"umbrella", "clock", "radio", "camera", "telescope", "magnet", "battery",
	"generator", "brick", "stone", "rock", "boulder", "sand", "mud",
	"snowball", "kite", "car", "bicycle", "skateboard", "truck", "helicopter",
	"plane", "crown", "coin", "gold", "diamond", "chest", "treasure", "star",
	"moon", "cloud", "rainbow", "ghost", "dragon", "dinosaur", "robot",
	"alien", "ninja", "pirate", "king", "queen", "teacher", "student", "chef",
	"doctor",
}

func TestCountIsBigEnough(t *testing.T) {
	if Count() < 180 {
		t.Fatalf("dictionary has %d nouns, need at least 180", Count())
	}
	t.Logf("dictionary holds %d nouns and %d adjectives", Count(), len(modifierTable))
}

func TestDictionarySortedAndUnique(t *testing.T) {
	objects := Dictionary()
	if len(objects) != Count() {
		t.Fatalf("Dictionary returned %d objects, Count says %d", len(objects), Count())
	}
	seen := map[string]bool{}
	for i, obj := range objects {
		if seen[obj.Key] {
			t.Errorf("duplicate key %q", obj.Key)
		}
		seen[obj.Key] = true
		if i > 0 && objects[i-1].Key >= obj.Key {
			t.Errorf("Dictionary not sorted by key: %q before %q", objects[i-1].Key, obj.Key)
		}
	}
	if len(seen) != len(objects) {
		t.Fatalf("Dictionary has %d keys for %d objects", len(seen), len(objects))
	}
}

// TestDictionaryRoundTrip is the core guarantee: every authored object can be
// found by its own display name and parsed back to exactly itself.
func TestDictionaryRoundTrip(t *testing.T) {
	for _, want := range Dictionary() {
		got, ok := Lookup(want.Name)
		if !ok {
			t.Fatalf("Lookup(%q) failed for an object the dictionary claims to have", want.Name)
		}
		compareObjects(t, "Lookup("+want.Name+")", got, want)

		if _, ok := Lookup(want.Key); !ok {
			t.Errorf("Lookup(%q) by key failed", want.Key)
		}

		parsed, ok := Parse(want.Name)
		if !ok {
			t.Fatalf("Parse(%q) refused an authored noun", want.Name)
		}
		compareObjects(t, "Parse("+want.Name+")", parsed, want)
		if parsed.Approximate {
			t.Errorf("Parse(%q) marked an authored noun approximate", want.Name)
		}
		if len(parsed.Modifiers) != 0 {
			t.Errorf("Parse(%q) invented modifiers %v", want.Name, parsed.Modifiers)
		}
		if parsed.Name != want.Name {
			t.Errorf("Parse(%q).Name = %q, want the name as typed", want.Name, parsed.Name)
		}
	}
}

// TestAliasesResolveToTheirObject checks each authored alias still lands on the
// same canonical key.
func TestAliasesResolveToTheirObject(t *testing.T) {
	if len(entryList) == 0 {
		t.Fatal("no entries were loaded")
	}
	keys := map[string]bool{}
	for _, e := range entryList {
		if keys[e.key] {
			t.Errorf("duplicate authored key %q", e.key)
		}
		keys[e.key] = true
	}
	for _, e := range entryList {
		for _, alias := range e.alias {
			obj, ok := Lookup(alias)
			if !ok {
				t.Errorf("alias %q of %q does not resolve", alias, e.key)
				continue
			}
			if obj.Key != e.key {
				t.Errorf("alias %q resolved to %q, want %q", alias, obj.Key, e.key)
			}
		}
	}

	// The synonyms the design brief calls out by name.
	for _, pair := range [][2]string{
		{"grappling hook", "grapple"},
		{"crate", "box"},
		{"soccer ball", "ball"},
		{"steps", "stairs"},
		{"tnt", "dynamite"},
		{"glue stick", "glue"},
		{"rain cloud", "cloud"},
	} {
		left, okLeft := Lookup(pair[0])
		right, okRight := Lookup(pair[1])
		if !okLeft || !okRight {
			t.Fatalf("expected %q and %q to be in the dictionary", pair[0], pair[1])
		}
		if left.Key != right.Key {
			t.Errorf("%q and %q should share a key, got %q and %q", pair[0], pair[1], left.Key, right.Key)
		}
	}
}

func TestRequiredVocabularyIsKnown(t *testing.T) {
	missing := []string{}
	for _, word := range requiredVocabulary {
		if !Knows(word) {
			missing = append(missing, word)
		}
	}
	if len(missing) > 0 {
		t.Fatalf("%d required words missing from the book: %v", len(missing), missing)
	}
}

// TestObjectFieldValidity checks every authored object is drawable and sane.
func TestObjectFieldValidity(t *testing.T) {
	for _, obj := range Dictionary() {
		name := obj.Key
		if obj.Name == "" {
			t.Errorf("%s: empty name", name)
		}
		if obj.Noun != obj.Name {
			t.Errorf("%s: noun %q does not match name %q", name, obj.Noun, obj.Name)
		}
		if obj.Key != strings.ToLower(obj.Key) || obj.Key != normalizeWord(obj.Key) {
			t.Errorf("%s: key is not a normalised lowercase id", name)
		}
		if !inList(pinnedCategories, obj.Category) {
			t.Errorf("%s: unknown category %q", name, obj.Category)
		}
		if !inList(pinnedShapes, obj.Shape) {
			t.Errorf("%s: unknown shape %q", name, obj.Shape)
		}
		if !hexColor.MatchString(obj.Color) {
			t.Errorf("%s: colour %q is not a hex colour", name, obj.Color)
		}
		if obj.Mass < MinMass || obj.Mass > MaxMass {
			t.Errorf("%s: mass %d out of range", name, obj.Mass)
		}
		if obj.Size < MinSize || obj.Size > MaxSize {
			t.Errorf("%s: size %d out of range", name, obj.Size)
		}
		glyphRunes := utf8.RuneCountInString(obj.Glyph)
		if glyphRunes < 1 || glyphRunes > 3 {
			t.Errorf("%s: glyph %q must be 1-3 characters", name, obj.Glyph)
		}
		if strings.TrimSpace(obj.Note) == "" {
			t.Errorf("%s: empty note", name)
		}
		if strings.ContainsAny(obj.Note, "\n\r") {
			t.Errorf("%s: note must be one line", name)
		}
		if len(obj.Tags) == 0 {
			t.Errorf("%s: no tags at all", name)
		}
		if !inList(pinnedCategories, obj.Category) {
			continue
		}
		for _, tag := range obj.Tags {
			if !inList(validTags, tag) {
				t.Errorf("%s: unpinned tag %q", name, tag)
			}
		}
		if hasDuplicates(obj.Tags) {
			t.Errorf("%s: duplicate tags in %v", name, obj.Tags)
		}
	}
}

// TestPinnedVocabularyCoverage proves the dictionary actually exercises the
// whole pinned tag and shape vocabulary, so the world engine and the frontend
// both meet every case they claim to handle.
func TestPinnedVocabularyCoverage(t *testing.T) {
	objects := Dictionary()
	tagCounts := map[string]int{}
	shapeCounts := map[string]int{}
	categoryCounts := map[string]int{}
	for _, obj := range objects {
		for _, tag := range obj.Tags {
			tagCounts[tag]++
		}
		shapeCounts[obj.Shape]++
		categoryCounts[obj.Category]++
	}
	for _, tag := range validTags {
		if tagCounts[tag] == 0 {
			t.Errorf("pinned tag %q is never used by any object", tag)
		}
	}
	for _, shape := range pinnedShapes {
		if shapeCounts[shape] == 0 {
			t.Errorf("pinned shape %q is never used by any object", shape)
		}
	}
	for _, category := range pinnedCategories {
		if categoryCounts[category] == 0 {
			t.Errorf("pinned category %q is never used by any object", category)
		}
	}
	t.Logf("tags used: %d, shapes used: %d, categories used: %d", len(tagCounts), len(shapeCounts), len(categoryCounts))
}

func TestLookupIsDictionaryOnly(t *testing.T) {
	if _, ok := Lookup("big ladder"); ok {
		t.Error("Lookup must not apply modifiers")
	}
	if _, ok := Lookup("zorble"); ok {
		t.Error("Lookup must not improvise unknown nouns")
	}
	if _, ok := Lookup("  "); ok {
		t.Error("Lookup of whitespace should fail")
	}
	obj, ok := Lookup("LADDER")
	if !ok || obj.Key != "ladder" {
		t.Errorf("Lookup should be case-insensitive, got %+v ok=%v", obj, ok)
	}
	if obj.Approximate {
		t.Error("Lookup must never return Approximate")
	}
	hyphenated, ok := Lookup("ice-cube")
	if !ok || hyphenated.Key != "ice cube" {
		t.Errorf("Lookup should fold hyphens, got %q ok=%v", hyphenated.Key, ok)
	}
}

func TestKnows(t *testing.T) {
	cases := map[string]bool{
		"ladder":             true,
		"big flaming ladder": true,
		"LORRY":              true,
		"grappling hook":     true,
		"zorble":             false,
		"big zorble":         false,
		"flaming":            false,
		"":                   false,
		"!!!":                false,
	}
	for phrase, want := range cases {
		if got := Knows(phrase); got != want {
			t.Errorf("Knows(%q) = %v, want %v", phrase, got, want)
		}
	}
}

func TestSuggest(t *testing.T) {
	if got := Suggest("lad", 8); !inList(got, "ladder") {
		t.Errorf("Suggest(\"lad\", 8) = %v, want it to contain ladder", got)
	}
	if got := Suggest("", 5); len(got) < 1 || len(got) > 5 {
		t.Errorf("Suggest(\"\", 5) = %v, want 1..5 words", got)
	}
	if got := Suggest("lad", 3); len(got) > 3 {
		t.Errorf("Suggest(\"lad\", 3) = %v, want at most 3", got)
	}
	if got := Suggest("ladder", 0); got != nil {
		t.Errorf("Suggest with limit 0 = %v, want nil", got)
	}
	if got := Suggest("LAD", 8); !reflect.DeepEqual(got, Suggest("lad", 8)) {
		t.Errorf("Suggest should be case-insensitive: %v vs %v", got, Suggest("lad", 8))
	}
	if got := Suggest("grapp", 8); !suggestsKey(got, "grapple") {
		t.Errorf("Suggest(\"grapp\") = %v, want something that resolves to the grapple", got)
	}
	for _, query := range []string{"", "a", "ice", "lad", "zorble", "!!!", "soc"} {
		got := Suggest(query, 8)
		if hasDuplicates(got) {
			t.Errorf("Suggest(%q) returned duplicates: %v", query, got)
		}
		for _, word := range got {
			if strings.TrimSpace(word) == "" {
				t.Errorf("Suggest(%q) returned a blank word: %v", query, got)
			}
		}
	}

	// Ordering: prefix matches must outrank substring matches.
	ranked := Suggest("lad", 8)
	ladderAt := indexOf(ranked, "ladder")
	if ladderAt < 0 {
		t.Fatalf("Suggest(\"lad\") lost ladder: %v", ranked)
	}
	for i, word := range ranked {
		if word == "ladder" {
			continue
		}
		if !strings.HasPrefix(word, "lad") && i < ladderAt {
			t.Errorf("Suggest(\"lad\") put substring match %q ahead of prefix match ladder", word)
		}
	}

	// The notebook never dead-ends: the typed word itself comes back when the
	// dictionary has nothing better to offer.
	if got := Suggest("zorble", 8); !inList(got, "zorble") {
		t.Errorf("Suggest(\"zorble\") = %v, want the typed word offered back", got)
	}
	if got := Suggest("ice", 8); !inList(got, "ice") || !inList(got, "ice cube") {
		t.Errorf("Suggest(\"ice\") = %v, want ice and ice cube", got)
	}

	// Substring matches are found too, just behind the prefix matches.
	if got := Suggest("adder", 8); !inList(got, "ladder") {
		t.Errorf("Suggest(\"adder\") = %v, want a substring match on ladder", got)
	}
	if got := Suggest("hainsaw", 8); !inList(got, "chainsaw") {
		t.Errorf("Suggest(\"hainsaw\") = %v, want a substring match on chainsaw", got)
	}
	if got := Suggest("balloon", 8); len(got) < 1 || got[0] != "balloon" {
		t.Errorf("Suggest(\"balloon\") = %v, want the exact match first", got)
	}

	// Dictionaries that are already full should not get the typed word bolted on.
	if got := Suggest("a", 2); len(got) != 2 {
		t.Errorf("Suggest(\"a\", 2) = %v, want exactly 2", got)
	}
}

func TestDictionaryIsNotMutableFromOutside(t *testing.T) {
	first := Dictionary()
	for i := range first {
		first[i].Tags = append(first[i].Tags, "bogus-tag")
		first[i].Modifiers = append(first[i].Modifiers, "bogus")
		first[i].Key = "vandalised"
	}
	second := Dictionary()
	for i := range second {
		if inList(second[i].Tags, "bogus-tag") {
			t.Fatalf("Dictionary() handed out shared state: %q", second[i].Key)
		}
	}

	looked, _ := Lookup("ladder")
	looked.Tags = append(looked.Tags, "bogus-tag")
	lookedAgain, _ := Lookup("ladder")
	if inList(lookedAgain.Tags, "bogus-tag") {
		t.Fatal("Lookup() handed out shared tag state")
	}

	parsed, _ := Parse("metal ladder")
	parsed.Tags = append(parsed.Tags, "bogus-tag")
	reparsed, _ := Parse("metal ladder")
	if inList(reparsed.Tags, "bogus-tag") {
		t.Fatal("Parse() handed out shared tag state")
	}
}

func TestParseIsConcurrentSafe(t *testing.T) {
	phrases := []string{
		"big flaming ladder", "metal rope", "frozen water", "golden crown",
		"tiny zorble", "giant wooden box", "sharp metal invisible dog",
	}
	var wg sync.WaitGroup
	for worker := 0; worker < 8; worker++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < 300; i++ {
				phrase := phrases[i%len(phrases)]
				obj, ok := Parse(phrase)
				if !ok {
					t.Errorf("Parse(%q) refused a phrase under load", phrase)
					return
				}
				if !inList(pinnedShapes, obj.Shape) || !hexColor.MatchString(obj.Color) {
					t.Errorf("Parse(%q) produced %+v", phrase, obj)
					return
				}
				Suggest("lad", 5)
				Dictionary()
			}
		}()
	}
	wg.Wait()
}

func TestArticlesAndPunctuationAreForgiven(t *testing.T) {
	cases := map[string]string{
		"ladder!":        "ladder",
		"  ladder  ":     "ladder",
		"the ladder.":    "ladder",
		"a big ladder":   "ladder",
		"Ice-Cube":       "ice cube",
		"grappling-hook": "grapple",
	}
	for phrase, wantKey := range cases {
		obj, ok := Parse(phrase)
		if !ok {
			t.Errorf("Parse(%q) refused a typable phrase", phrase)
			continue
		}
		if obj.Key != wantKey {
			t.Errorf("Parse(%q).Key = %q, want %q", phrase, obj.Key, wantKey)
		}
	}
}

func compareObjects(t *testing.T, label string, got, want Object) {
	t.Helper()
	if got.Key != want.Key {
		t.Errorf("%s: key %q, want %q", label, got.Key, want.Key)
	}
	if got.Noun != want.Noun {
		t.Errorf("%s: noun %q, want %q", label, got.Noun, want.Noun)
	}
	if got.Category != want.Category {
		t.Errorf("%s: category %q, want %q", label, got.Category, want.Category)
	}
	if got.Mass != want.Mass {
		t.Errorf("%s: mass %d, want %d", label, got.Mass, want.Mass)
	}
	if got.Size != want.Size {
		t.Errorf("%s: size %d, want %d", label, got.Size, want.Size)
	}
	if got.Color != want.Color {
		t.Errorf("%s: colour %q, want %q", label, got.Color, want.Color)
	}
	if got.Shape != want.Shape {
		t.Errorf("%s: shape %q, want %q", label, got.Shape, want.Shape)
	}
	if got.Glyph != want.Glyph {
		t.Errorf("%s: glyph %q, want %q", label, got.Glyph, want.Glyph)
	}
	if got.Note != want.Note {
		t.Errorf("%s: note %q, want %q", label, got.Note, want.Note)
	}
	if got.Approximate != want.Approximate {
		t.Errorf("%s: approximate %v, want %v", label, got.Approximate, want.Approximate)
	}
	if len(got.Tags) != len(want.Tags) {
		t.Errorf("%s: tags %v, want %v", label, got.Tags, want.Tags)
	} else {
		gotCopy := append([]string{}, got.Tags...)
		wantCopy := append([]string{}, want.Tags...)
		sort.Strings(gotCopy)
		sort.Strings(wantCopy)
		if !reflect.DeepEqual(gotCopy, wantCopy) {
			t.Errorf("%s: tags %v, want %v", label, got.Tags, want.Tags)
		}
	}
}

func hasDuplicates(list []string) bool {
	seen := map[string]bool{}
	for _, s := range list {
		if seen[s] {
			return true
		}
		seen[s] = true
	}
	return false
}

func indexOf(list []string, want string) int {
	for i, s := range list {
		if s == want {
			return i
		}
	}
	return -1
}

// suggestsKey reports whether any autocomplete suggestion resolves to key, so
// the test does not care whether the key or one of its aliases came back.
func suggestsKey(words []string, key string) bool {
	for _, word := range words {
		if obj, ok := Lookup(word); ok && obj.Key == key {
			return true
		}
	}
	return false
}

func describe(obj Object) string {
	return fmt.Sprintf("%s cat=%s mass=%d size=%d color=%s shape=%s tags=%v",
		obj.Name, obj.Category, obj.Mass, obj.Size, obj.Color, obj.Shape, obj.Tags)
}

// TestParseRefusesUntypablePhrases pins the only case where the book says no:
// there is nothing typable to work with, or there are adjectives and no noun.
func TestParseRefusesUntypablePhrases(t *testing.T) {
	for _, phrase := range []string{"", " ", "   ", "	", "\n", "  	 \n ", "!!!", "???", "...", "?!.", "-", "--", "'", "''", "! ! !"} {
		if obj, ok := Parse(phrase); ok {
			t.Errorf("Parse(%q) = %v, want ok=false", phrase, describe(obj))
		}
		if Knows(phrase) {
			t.Errorf("Knows(%q) = true, want false", phrase)
		}
	}
}

func TestHasTag(t *testing.T) {
	ladder, ok := Parse("ladder")
	if !ok {
		t.Fatal("Parse(\"ladder\") refused")
	}
	if !ladder.HasTag(TagClimbable) {
		t.Error("ladder should be climbable")
	}
	if ladder.HasTag(TagFlammable) {
		t.Error("plain ladder should not be flammable")
	}
	flaming, _ := Parse("flaming ladder")
	if !flaming.HasTag(TagFire) {
		t.Error("flaming ladder should carry fire")
	}
	empty, _ := improviseObject("qqq"), true
	if empty.HasTag(TagMagical) {
		t.Error("an improvised mystery should not be magical")
	}
}

// TestVeryLongPhrasesAreBounded keeps a pathological phrase from doing
// pathological work: the tokenizer caps the phrase and the parse still succeeds.
func TestVeryLongPhrasesAreBounded(t *testing.T) {
	words := make([]string, 0, 60)
	for i := 0; i < 60; i++ {
		words = append(words, "big")
	}
	words = append(words, "ladder")
	phrase := strings.Join(words, " ")
	obj, ok := Parse(phrase)
	if !ok {
		t.Fatal("Parse refused an over-long phrase; it must still answer")
	}
	if obj.Key != "ladder" {
		t.Errorf("key = %q, want ladder", obj.Key)
	}
	if obj.Size != MaxSize {
		t.Errorf("size = %d, want %d after many big adjectives", obj.Size, MaxSize)
	}
	if tokens := len(strings.Fields(obj.Name)); tokens > maxTokens {
		t.Errorf("Name kept %d tokens, want at most %d", tokens, maxTokens)
	}
}

func TestNormalisationHelpers(t *testing.T) {
	cases := map[string]string{
		"  Ice  Cube ": "ice cube",
		"ICE-CUBE":     "ice cube",
		"Ice-Cube":     "ice cube",
		"ice	cube":     "ice cube",
		"":             "",
		"   ":          "",
	}
	for input, want := range cases {
		if got := normalizeWord(input); got != want {
			t.Errorf("normalizeWord(%q) = %q, want %q", input, got, want)
		}
	}
	if got := clamp(9, MinSize, MaxSize); got != MaxSize {
		t.Errorf("clamp high = %d", got)
	}
	if got := clamp(-9, MinSize, MaxSize); got != MinSize {
		t.Errorf("clamp low = %d", got)
	}
	if got := clamp(2, MinSize, MaxSize); got != 2 {
		t.Errorf("clamp middle = %d", got)
	}
	if !allModifiers([]string{"big", "hot"}) {
		t.Error("allModifiers should accept two adjectives")
	}
	if allModifiers([]string{"big", "ladder"}) {
		t.Error("allModifiers should reject a noun")
	}
	if allModifiers(nil) {
		t.Error("allModifiers of nothing is false")
	}
	if canonicalModifier("WIND-UP") != "wind-up" {
		t.Errorf("canonicalModifier(\"WIND-UP\") = %q", canonicalModifier("WIND-UP"))
	}
	if canonicalModifier("wind up") != "wind-up" {
		t.Errorf("canonicalModifier(\"wind up\") = %q", canonicalModifier("wind up"))
	}
	if canonicalModifier("ladder") != "" {
		t.Error("a noun is not an adjective")
	}
}
