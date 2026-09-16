package lexicon

import (
	"fmt"
	"strings"
	"unicode"
)

// The book is finite and admits it: an unknown noun is improvised, not refused.
// Everything below is hash-derived, so "zorble" is the same object every time
// it is typed, in every session, forever.

// improvisedColors is a fixed palette. Improvised objects pick from it by hash,
// which keeps the art consistent and stops the frontend seeing random mud.
var improvisedColors = []string{
	"#d94f4f", "#4a8ad9", "#4a8a4a", "#e0a43a", "#8a5ab0", "#c96a3a",
	"#3a8a8a", "#c94a8a", "#7d7d85", "#5a6b4a", "#c9a24a", "#5a8ad9",
}

// improvisedNotes are the book's confessions. %s is the noun.
var improvisedNotes = []string{
	"The book has never heard of a '%s', so it improvised",
	"'%s' is not written down anywhere, so the book guessed with confidence",
	"The book does not know what a '%s' is and has decided not to admit it",
	"No entry for '%s' - the notebook shrugged and drew something",
	"'%s' was never in the book, so this is the book making it up as it goes",
}

// improvisedGlyphMax is the pinned 1-3 character glyph budget.
const improvisedGlyphMax = 3

// improviseObject builds the deterministic stand-in for a noun the book does
// not know. Adjectives are applied afterwards by the caller.
func improviseObject(noun string) Object {
	word := normalizeWord(noun)
	if word == "" {
		word = noun
	}
	h := fnv1a64(word)

	shape := pinnedShapes[int(h%uint64(len(pinnedShapes)))]
	color := improvisedColors[int((h>>7)%uint64(len(improvisedColors)))]
	mass := 1 + int((h>>13)%3)
	size := 1 + int((h>>19)%3)

	category, tags := improviseCategory(word, h)
	if isVowelHeavy(word) {
		// A pile of vowels is not a thing with corners; draw it as a blob.
		shape = ShapeBlob
	}

	return Object{
		Key:         word,
		Name:        noun,
		Noun:        word,
		Category:    category,
		Approximate: true,
		Modifiers:   []string{},
		Tags:        tags,
		Mass:        clamp(mass, MinMass, MaxMass),
		Size:        clamp(size, MinSize, MaxSize),
		Color:       color,
		Shape:       shape,
		Glyph:       improviseGlyph(word),
		Note:        fmt.Sprintf(improvisedNotes[int((h>>29)%uint64(len(improvisedNotes)))], word),
	}
}

// improviseCategory reads what it can from the spelling: "-er" and "-or" are
// tools, "-ing" is a doing-thing, and the rest is a mystery the hash decides.
func improviseCategory(word string, h uint64) (string, []string) {
	runes := []rune(word)
	switch {
	case len(runes) >= 4 && (strings.HasSuffix(word, "er") || strings.HasSuffix(word, "or")):
		return CatTool, []string{TagTool}
	case len(runes) >= 5 && strings.HasSuffix(word, "ing"):
		if h%2 == 0 {
			return CatMachine, []string{TagMachine}
		}
		return CatMystery, []string{}
	default:
		return CatMystery, []string{}
	}
}

// improviseGlyph takes the first one to three letters or digits of the noun and
// shouts them, falling back to the leading runes so symbols and scripts the
// alphabet does not cover still get something drawn on them.
func improviseGlyph(word string) string {
	letters := make([]rune, 0, improvisedGlyphMax)
	for _, r := range word {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			letters = append(letters, unicode.ToUpper(r))
			if len(letters) == improvisedGlyphMax {
				break
			}
		}
	}
	if len(letters) > 0 {
		return string(letters)
	}
	runes := []rune(word)
	if len(runes) > improvisedGlyphMax {
		runes = runes[:improvisedGlyphMax]
	}
	return string(runes)
}

// isVowelHeavy spots the nonsense words a player invents out of vowels.
func isVowelHeavy(word string) bool {
	runes := []rune(word)
	if len(runes) < 4 {
		return false
	}
	vowels := 0
	letters := 0
	for _, r := range runes {
		if !unicode.IsLetter(r) {
			continue
		}
		letters++
		if strings.ContainsRune("aeiouy", unicode.ToLower(r)) {
			vowels++
		}
	}
	if letters == 0 {
		return false
	}
	return vowels*100/letters >= 60
}

// fnv1a64 is FNV-1a over the UTF-8 bytes: stable, small, and good enough to
// spread invented nouns across shapes and colours.
func fnv1a64(s string) uint64 {
	const (
		offset64 = 14695981039346656037
		prime64  = 1099511628211
	)
	var h uint64 = offset64
	for i := 0; i < len(s); i++ {
		h ^= uint64(s[i])
		h *= prime64
	}
	return h
}
