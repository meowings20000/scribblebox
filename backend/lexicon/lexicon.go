// Package lexicon is the authored word book behind Scribblebox: a finite
// dictionary of spawnable objects, an adjective modifier system ("big flaming
// ladder"), and a parser that never refuses a word.
//
// The book is honest about being finite. A noun it has never seen still spawns
// an object, marked Approximate, built deterministically from the typed word,
// so the player is never told "no" by the notebook.
package lexicon

import (
	"sort"
	"strings"
	"unicode"
)

// Object is one spawnable thing: everything the world engine needs in order to
// decide what it can do, and everything the frontend needs in order to draw it.
type Object struct {
	Key         string   `json:"key"`         // canonical id, lowercase, e.g. "ladder"
	Name        string   `json:"name"`        // display name as typed, e.g. "big flaming ladder"
	Noun        string   `json:"noun"`        // the resolved base noun, e.g. "ladder"
	Category    string   `json:"category"`    // structure|tool|material|animal|plant|vehicle|container|machine|food|mystery
	Approximate bool     `json:"approximate"` // true when the noun was not in the dictionary
	Modifiers   []string `json:"modifiers"`   // applied adjectives, in order
	Tags        []string `json:"tags"`        // pinned tag vocabulary, merged base-first
	Mass        int      `json:"mass"`        // 1 (feather) .. 5 (anvil)
	Size        int      `json:"size"`        // 1 (fits in a hand) .. 3 (takes many cells)
	Color       string   `json:"color"`       // hex, used to draw it
	Shape       string   `json:"shape"`       // pinned shape vocabulary
	Glyph       string   `json:"glyph"`       // 1-3 characters drawn on the object
	Note        string   `json:"note"`        // one flavouring line shown when it appears
}

// HasTag reports whether the object carries tag. The world engine reads the
// pinned tag strings directly; this is only a convenience.
func (o Object) HasTag(tag string) bool {
	for _, t := range o.Tags {
		if t == tag {
			return true
		}
	}
	return false
}

// Categories. The world engine and the frontend both switch on these; there are
// no others. People (wizard, king, teacher) live under CatMystery and carry the
// "person" tag, because the pinned category list has no category for them.
const (
	CatStructure = "structure"
	CatTool      = "tool"
	CatMaterial  = "material"
	CatAnimal    = "animal"
	CatPlant     = "plant"
	CatVehicle   = "vehicle"
	CatContainer = "container"
	CatMachine   = "machine"
	CatFood      = "food"
	CatMystery   = "mystery"
)

// Pinned tags the world engine reads.
const (
	TagFlammable   = "flammable"    // catches fire
	TagFire        = "fire"         // is a flame / starts fires
	TagLightSource = "light-source" // lights candles and darkness
	TagWater       = "water"        // is water
	TagCold        = "cold"         // freezes water it touches
	TagFrozen      = "frozen"       // is made of ice, solid
	TagSharp       = "sharp"        // can cut
	TagCutting     = "cutting"      // cuts ropes and plants when adjacent
	TagClimbable   = "climbable"    // a player can climb it
	TagPlatform    = "platform"     // solid, walkable surface
	TagSolid       = "solid"        // blocks movement
	TagRope        = "rope"         // can be cut and climbed
	TagBuoyant     = "buoyant"      // floats on water
	TagHeavy       = "heavy"        // sinks and crushes fragile things
	TagFragile     = "fragile"      // breaks under heavy things
	TagConductive  = "conductive"   // carries power
	TagPowerSource = "power-source" // supplies power
	TagMachine     = "machine"      // does something when powered
	TagLift        = "lift"         // a powered machine that rises
	TagUnlock      = "unlock"       // opens locked things
	TagExplosive   = "explosive"    // blows up near fire
	TagEdible      = "edible"       // food
	TagContainer   = "container"    // can hold a small object
	TagCollectible = "collectible"  // a prize
	TagWheeled     = "wheeled"      // rolls
	TagMagical     = "magical"      // does something impossible in a rule-bending way
	TagAnimal      = "animal"
	TagPlant       = "plant"
	TagPerson      = "person"
	TagFurniture   = "furniture"
	TagWeapon      = "weapon"
	TagTool        = "tool"
	TagMoney       = "money"
	TagTreasure    = "treasure"

	// TagSticky is named by the modifier table (the "sticky" adjective is
	// specified as "+sticky") but is absent from the pinned list above. It is
	// deliberately the only tag outside that list: it is applied by modifiers
	// alone and no authored object is required to carry it.
	TagSticky = "sticky"
)

// Pinned shapes the frontend knows how to draw.
const (
	ShapeBox     = "box"
	ShapeLadder  = "ladder"
	ShapeRope    = "rope"
	ShapePlank   = "plank"
	ShapeBlob    = "blob"
	ShapeCircle  = "circle"
	ShapeStar    = "star"
	ShapeTree    = "tree"
	ShapeFlame   = "flame"
	ShapeKey     = "key"
	ShapeTool    = "tool"
	ShapeAnimal  = "animal"
	ShapePerson  = "person"
	ShapeBottle  = "bottle"
	ShapeBook    = "book"
	ShapeVehicle = "vehicle"
	ShapeChest   = "chest"
	ShapeCandle  = "candle"
	ShapeFlag    = "flag"
	ShapeMachine = "machine"
)

// Mass and size bounds. Every object and every modified object stays inside
// these, which is what keeps the world's physics sane.
const (
	MinMass, MaxMass = 1, 5
	MinSize, MaxSize = 1, 3
)

// maxTokens bounds the work a single typed phrase can cause. The HTTP layer
// already caps the phrase at 60 characters; this is a second, cheap ceiling.
const maxTokens = 24

var pinnedTags = []string{
	TagFlammable, TagFire, TagLightSource, TagWater, TagCold, TagFrozen, TagSharp,
	TagCutting, TagClimbable, TagPlatform, TagSolid, TagRope, TagBuoyant, TagHeavy,
	TagFragile, TagConductive, TagPowerSource, TagMachine, TagLift, TagUnlock,
	TagExplosive, TagEdible, TagContainer, TagCollectible, TagWheeled, TagMagical,
	TagAnimal, TagPlant, TagPerson, TagFurniture, TagWeapon, TagTool, TagMoney,
	TagTreasure,
}

// validTags is the pinned vocabulary plus the one modifier-only addition.
var validTags = append(append([]string{}, pinnedTags...), TagSticky)

var pinnedShapes = []string{
	ShapeBox, ShapeLadder, ShapeRope, ShapePlank, ShapeBlob, ShapeCircle,
	ShapeStar, ShapeTree, ShapeFlame, ShapeKey, ShapeTool, ShapeAnimal,
	ShapePerson, ShapeBottle, ShapeBook, ShapeVehicle, ShapeChest,
	ShapeCandle, ShapeFlag, ShapeMachine,
}

var pinnedCategories = []string{
	CatStructure, CatTool, CatMaterial, CatAnimal, CatPlant, CatVehicle,
	CatContainer, CatMachine, CatFood, CatMystery,
}

// entry is one authored dictionary row. dictionary_*.go files hold the rows;
// init() below turns them into Objects and lookup indexes.
type entry struct {
	key   string
	name  string
	cat   string
	tags  []string
	mass  int
	size  int
	color string
	shape string
	glyph string
	note  string
	alias []string
}

// term is one autocomplete candidate: a key, a display name, or an alias.
type term struct {
	display string // what the player sees
	word    string // normalised form used for matching
	key     string // the canonical key this term belongs to
}

var (
	entryList   []entry           // every authored row, in file order
	byWord      map[string]Object // normalised key, name or alias -> object template
	allNouns    []Object          // every authored object, sorted by key
	searchTerms []term            // every key, name and alias, for Suggest
)

func init() {
	entries := make([]entry, 0, 256)
	for _, group := range dictionaryGroups {
		entries = append(entries, group...)
	}
	entryList = entries

	byWord = make(map[string]Object, len(entries)*2)
	seenKeys := make(map[string]bool, len(entries))
	for _, e := range entries {
		if seenKeys[e.key] {
			// A duplicate key would make the book ambiguous; the first entry
			// wins so the package stays usable, and the tests fail loudly.
			continue
		}
		seenKeys[e.key] = true

		obj := Object{
			Key:         e.key,
			Name:        e.name,
			Noun:        e.name,
			Category:    e.cat,
			Approximate: false,
			Modifiers:   []string{},
			Tags:        append([]string{}, e.tags...),
			Mass:        e.mass,
			Size:        e.size,
			Color:       e.color,
			Shape:       e.shape,
			Glyph:       e.glyph,
			Note:        e.note,
		}
		allNouns = append(allNouns, obj)

		addWord := func(word string) {
			normalised := normalizeWord(word)
			if normalised == "" {
				return
			}
			if _, taken := byWord[normalised]; !taken {
				byWord[normalised] = obj
			}
		}
		addWord(e.key)
		addWord(e.name)
		for _, a := range e.alias {
			addWord(a)
		}

		searchTerms = append(searchTerms, term{display: e.key, word: normalizeWord(e.key), key: e.key})
		if name := normalizeWord(e.name); name != normalizeWord(e.key) {
			searchTerms = append(searchTerms, term{display: e.name, word: name, key: e.key})
		}
		for _, a := range e.alias {
			searchTerms = append(searchTerms, term{display: a, word: normalizeWord(a), key: e.key})
		}
	}

	sort.Slice(allNouns, func(i, j int) bool { return allNouns[i].Key < allNouns[j].Key })
}

// Parse turns a typed phrase into the object the notebook should spawn:
// base noun plus any adjectives in front of it, e.g. "big flaming ladder".
//
// ok is false only for a phrase with nothing typable in it (empty, whitespace,
// "!!!") or a phrase that is adjectives and no noun ("flaming"). An unknown
// noun is never refused: it comes back Approximate.
func Parse(phrase string) (Object, bool) {
	base, mods, tokens, ok := resolve(phrase)
	if !ok {
		return Object{}, false
	}
	return assemble(base, mods, tokens), true
}

// Lookup finds an authored object by its exact name, key or an alias. It takes
// no modifiers: Lookup("big ladder") is false, Lookup("ladder") is true.
// Approximate is always false here.
func Lookup(word string) (Object, bool) {
	return lookupNoun(word)
}

// Knows reports whether the phrase's base noun is in the dictionary.
func Knows(phrase string) bool {
	base, _, _, ok := resolve(phrase)
	return ok && !base.Approximate
}

// Suggest powers autocomplete. It matches keys, display names and aliases by
// prefix first and substring second, case-insensitively, sorted by relevance
// and then alphabetically. When it finds fewer than limit words it also offers
// the typed text itself, so the notebook never feels like a dead end.
func Suggest(prefix string, limit int) []string {
	if limit <= 0 {
		return nil
	}
	query := normalizeWord(prefix)

	type scored struct {
		display string
		key     string
		score   int
	}
	best := make(map[string]scored, len(searchTerms))
	for _, t := range searchTerms {
		score, matched := matchScore(t.word, query)
		if !matched {
			continue
		}
		current, seen := best[t.key]
		if !seen || score < current.score || (score == current.score && t.display < current.display) {
			best[t.key] = scored{display: t.display, key: t.key, score: score}
		}
	}

	ranked := make([]scored, 0, len(best))
	for _, s := range best {
		ranked = append(ranked, s)
	}
	sort.Slice(ranked, func(i, j int) bool {
		if ranked[i].score != ranked[j].score {
			return ranked[i].score < ranked[j].score
		}
		return ranked[i].display < ranked[j].display
	})

	out := make([]string, 0, limit)
	for _, s := range ranked {
		if len(out) == limit {
			break
		}
		out = append(out, s.display)
	}
	if len(out) < limit && query != "" && !containsString(out, query) {
		out = append(out, query)
	}
	return out
}

// Count reports how many nouns the book actually contains.
func Count() int { return len(allNouns) }

// Dictionary returns every authored object, sorted by key, for tests and
// tooling. Callers get their own copies and cannot corrupt the book.
func Dictionary() []Object {
	out := make([]Object, 0, len(allNouns))
	for _, obj := range allNouns {
		out = append(out, cloneObject(obj))
	}
	return out
}

// resolve splits a phrase into base noun (dictionary or improvised), the
// adjectives applied to it, the cleaned tokens, and ok.
func resolve(phrase string) (Object, []string, []string, bool) {
	tokens := tokenize(phrase)
	if len(tokens) == 0 {
		return Object{}, nil, nil, false
	}

	// A whole multi-word phrase can itself be a noun ("hot air balloon"), so
	// the full phrase gets first refusal before adjectives are peeled off.
	if len(tokens) > 1 {
		if obj, ok := lookupNoun(strings.Join(tokens, " ")); ok {
			return obj, []string{}, tokens, true
		}
	}

	if len(tokens) == 1 {
		if obj, ok := lookupNoun(tokens[0]); ok {
			return obj, []string{}, tokens, true
		}
		if isModifier(tokens[0]) {
			// adjectives and no noun: "flaming"
			return Object{}, nil, nil, false
		}
		return improviseObject(tokens[0]), []string{}, tokens, true
	}

	mods := make([]string, 0, len(tokens)-1)
	rest := tokens
	for len(rest) > 1 && isModifier(rest[0]) {
		mods = append(mods, canonicalModifier(rest[0]))
		rest = rest[1:]
	}

	if obj, ok := lookupNoun(strings.Join(rest, " ")); ok {
		return obj, mods, tokens, true
	}
	if len(rest) == 1 {
		if isModifier(rest[0]) {
			return Object{}, nil, nil, false
		}
		return improviseObject(rest[0]), mods, tokens, true
	}

	// Adjectives after the noun ("ladder big") still count.
	for k := len(rest) - 1; k >= 1; k-- {
		if !allModifiers(rest[k:]) {
			continue
		}
		if obj, ok := lookupNoun(strings.Join(rest[:k], " ")); ok {
			for _, t := range rest[k:] {
				mods = append(mods, canonicalModifier(t))
			}
			return obj, mods, tokens, true
		}
	}

	// Nothing matched: the last word is the noun. Unknown words before it are
	// not dictionary nouns and not adjectives, so they are dropped rather than
	// turned into a refusal.
	last := rest[len(rest)-1]
	if obj, ok := lookupNoun(last); ok {
		return obj, mods, tokens, true
	}
	return improviseObject(last), mods, tokens, true
}

// assemble applies the adjectives and stamps the display name the player typed.
func assemble(base Object, mods []string, tokens []string) Object {
	obj := withModifiers(base, mods)
	obj.Modifiers = append([]string{}, mods...)
	obj.Name = strings.Join(tokens, " ")
	return obj
}

// lookupNoun materialises a fresh copy of an authored object.
func lookupNoun(word string) (Object, bool) {
	obj, ok := byWord[normalizeWord(word)]
	if !ok {
		return Object{}, false
	}
	return cloneObject(obj), true
}

func cloneObject(obj Object) Object {
	out := obj
	out.Tags = append([]string{}, obj.Tags...)
	out.Modifiers = append([]string{}, obj.Modifiers...)
	return out
}

// matchScore ranks autocomplete candidates: exact, then prefix, then substring.
func matchScore(word, query string) (int, bool) {
	if query == "" {
		return 1, true
	}
	if word == query {
		return 0, true
	}
	if strings.HasPrefix(word, query) {
		return 1, true
	}
	if strings.Contains(word, query) {
		return 2, true
	}
	return 0, false
}

func containsString(list []string, want string) bool {
	for _, s := range list {
		if s == want {
			return true
		}
	}
	return false
}

func isModifier(token string) bool {
	return canonicalModifier(token) != ""
}

func allModifiers(tokens []string) bool {
	for _, t := range tokens {
		if !isModifier(t) {
			return false
		}
	}
	return len(tokens) > 0
}

// tokenize splits, trims surrounding punctuation and drops tokens with nothing
// typable in them, so "!!!" never becomes a noun but "zorble" always does.
//
// An absurdly long phrase is capped, but the last word is always kept: the noun
// must survive the cap, or the book would have to refuse.
func tokenize(phrase string) []string {
	fields := strings.Fields(phrase)
	tokens := make([]string, 0, len(fields))
	for _, f := range fields {
		t := strings.TrimFunc(f, func(r rune) bool {
			return unicode.IsPunct(r) && r != '-' && r != '\''
		})
		if t == "" || !hasContent(t) {
			continue
		}
		tokens = append(tokens, t)
	}
	if len(tokens) > maxTokens {
		trimmed := make([]string, 0, maxTokens)
		trimmed = append(trimmed, tokens[:maxTokens-1]...)
		trimmed = append(trimmed, tokens[len(tokens)-1])
		tokens = trimmed
	}
	return tokens
}

func hasContent(token string) bool {
	for _, r := range token {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || unicode.IsSymbol(r) {
			return true
		}
	}
	return false
}

// normalizeWord is the single shape every lookup goes through: lowercase, no
// hyphens, single spaces. "Ice-Cube", "ice cube" and "ice  cube" are one word.
func normalizeWord(s string) string {
	lowered := strings.ToLower(strings.TrimSpace(s))
	lowered = strings.ReplaceAll(lowered, "-", " ")
	return strings.Join(strings.Fields(lowered), " ")
}

func clamp(value, low, high int) int {
	if value < low {
		return low
	}
	if value > high {
		return high
	}
	return value
}

func appendTag(tags []string, tag string) []string {
	for _, t := range tags {
		if t == tag {
			return tags
		}
	}
	return append(tags, tag)
}

func removeTag(tags []string, tag string) []string {
	out := tags[:0]
	for _, t := range tags {
		if t != tag {
			out = append(out, t)
		}
	}
	return out
}

// dedupeTags keeps the first occurrence of each tag, so merging adjectives into
// a base object can never produce duplicates.
func dedupeTags(tags []string) []string {
	out := make([]string, 0, len(tags))
	seen := make(map[string]bool, len(tags))
	for _, t := range tags {
		if seen[t] {
			continue
		}
		seen[t] = true
		out = append(out, t)
	}
	return out
}
