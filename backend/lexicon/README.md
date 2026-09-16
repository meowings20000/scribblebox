# lexicon — the book the notebook writes in

`backend/lexicon` is Scribblebox's authored dictionary. It has three jobs:

1. **Say what a word is.** `Parse("big flaming ladder")` returns the object the
   world should spawn: category, pinned tags, mass, size, colour, shape, glyph
   and one line of flavour.
2. **Bend a word with adjectives.** Adjectives in front of the noun change
   properties and merge with the base object's own tags.
3. **Never refuse a word.** A noun the book has never seen still spawns an
   object, marked `Approximate`, built deterministically from a hash of the
   typed text.

## API

```go
func Parse(phrase string) (Object, bool)          // ok=false ONLY for untypable or adjective-only phrases
func Lookup(word string) (Object, bool)           // dictionary only: no modifiers, never Approximate
func Knows(phrase string) bool                    // is the base noun in the book?
func Suggest(prefix string, limit int) []string   // autocomplete: prefix first, substring second
func Count() int                                  // how many nouns are authored
func Dictionary() []Object                        // every authored object, sorted by key
```

`Object` also has `HasTag(tag string) bool` as a convenience.

## Behaviour worth knowing

- **Name vs Noun vs Key.** `Name` is the phrase as typed (`"big flaming ladder"`,
  case preserved, whitespace collapsed). `Noun` is the resolved base noun
  (`"ladder"`, or the canonical display name such as `"grappling hook"`). `Key`
  is the canonical lowercase id (`"ladder"`, `"grapple"`, `"ice cube"`) and is
  the same whatever aliases or adjectives were used.
- **Adjectives are read front to back, last one wins** when two conflict:
  `"metal wooden box"` burns, `"wooden metal box"` does not. Adjectives after
  the noun (`"ladder big"`) are also applied.
- **A single word that is a real noun beats the adjective reading.** `"orange"`
  is the fruit, `"orange rope"` is a repainted rope. `"giant"`, `"stone"` and
  `"orange"` are all dictionary nouns as well as adjectives.
- **A lone adjective is refused** (`Parse("flaming")` → `ok=false`), as are
  empty, whitespace-only and punctuation-only phrases (`"!!!"`). Digits, other
  scripts and symbols are typable, so `"999"`, `"梯子"` and `"🚀"` all spawn
  something.
- **Unknown nouns are stable.** `"zorble"` is the same shape, colour, mass,
  size, glyph and note in every call and every process, because everything comes
  from FNV-1a over the normalised noun, never from a random source.
- **`Suggest` never dead-ends.** It matches keys, display names and aliases
  (prefix first, then substring) and, if it found fewer than `limit` words,
  offers the typed text itself.

## Pinned vocabularies

The world engine switches on **tags**; the frontend draws **shapes**.

Tags: `flammable, fire, light-source, water, cold, frozen, sharp, cutting,
climbable, platform, solid, rope, buoyant, heavy, fragile, conductive,
power-source, machine, lift, unlock, explosive, edible, container, collectible,
wheeled, magical, animal, plant, person, furniture, weapon, tool, money,
treasure` — plus `sticky`, which the modifier table names (`sticky → +sticky`)
and which is the only tag outside the pinned list. It is applied by adjectives
and also appears on glue, tape, honey and mud.

Shapes: `box, ladder, rope, plank, blob, circle, star, tree, flame, key, tool,
animal, person, bottle, book, vehicle, chest, candle, flag, machine`.

Categories: `structure, tool, material, animal, plant, vehicle, container,
machine, food, mystery`. People (wizard, king, teacher, …) are `mystery` with
the `person` tag, because the pinned category list has no category for them.

## Adjectives

53 authored adjectives. Size and mass are deltas summed across the phrase and
clamped to `1..3` and `1..5`. Materials, state adjectives and colours are
documented in `modifiers.go`; two examples:

| adjective      | effect |
| -------------- | ------ |
| `metal/steel/iron` | `+conductive +heavy −flammable` |
| `frozen/icy/cold`  | `+frozen +cold −fire` |
| `wooden/paper` | `+flammable −conductive` |
| `sharp`        | `+sharp +cutting` |
| `broken`       | `−platform −climbable` |

Tag merging never replaces base tags and never produces duplicates.

## Files

| file | contents |
| ---- | -------- |
| `lexicon.go` | `Object`, pinned vocabularies, `Parse`/`Lookup`/`Knows`/`Suggest`/`Count`/`Dictionary`, tokenizer, index build |
| `modifiers.go` | the adjective table and the merge/clamp rules |
| `improvise.go` | hash-based improvised objects and their spelling rules |
| `dictionary.go` | the group list the index is built from |
| `dictionary_*.go` | the authored nouns, grouped for review |
| `*_test.go` | round trips, field validity, vocabulary coverage, modifier semantics, improvised stability |

## Tests

```sh
cd backend && go test -race -count=1 ./lexicon/ && go vet ./lexicon/
```

The suite proves: every authored object round-trips through `Lookup` and
`Parse`; at least 180 nouns and every pinned tag, shape and category are in use;
every field is valid (hex colour, known shape, 1–3 character glyph, mass and
size in range, one-line note); every adjective changes something for at least
one noun and every noun reacts to at least one adjective; merging never
duplicates or drops a base tag; improvised objects are stable, deterministic and
always inside the pinned vocabularies.
