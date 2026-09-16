package world

import "backend/lexicon"

// This file is the world's reading of an object's properties: which tags block
// the player, which hold them up, how big a spawned thing is, and the small
// authored overlay that supplies affordances the word book leaves out.
//
// Division of labour, stated plainly because it is a real dependency:
//
//   - the lexicon owns what an object IS (its name, look, mass, size, tags),
//   - the world owns what an object DOES (every rule in rules.go).
//
// The overlay below only ever ADDS tags, only for a handful of bare nouns, and
// only when the player typed the noun with no adjectives. That last condition is
// what keeps the book's own modifier semantics intact: "broken ladder" must not
// come back climbable just because a plain ladder is.

// designNote records one overlay entry: extra tags the world guarantees for a
// noun, and the sentence explaining why.
type designNote struct {
	tags []string
	why  string
}

// affordanceOverlay is the world's authored reading of nouns the book describes
// loosely. Each entry is a promise the puzzles rely on, and every promise is
// covered by a test.
var affordanceOverlay = map[string]designNote{
	// A rope is hemp: it burns. (The book tags it rope and climbable only, but
	// the fire rules promise "a match next to a rope burns it to ash".)
	"rope": {tags: []string{lexicon.TagFlammable}, why: "hemp burns"},
	// A spring under your feet is a lift: a trampoline that cannot lift is
	// just a rug.
	"trampoline":  {tags: []string{lexicon.TagMachine, lexicon.TagLift, lexicon.TagPowerSource}, why: "a trampoline IS a lift"},
	"springboard": {tags: []string{lexicon.TagMachine, lexicon.TagLift, lexicon.TagPowerSource}, why: "a springboard IS a lift"},
	"spring":      {tags: []string{lexicon.TagMachine, lexicon.TagLift, lexicon.TagPowerSource}, why: "a spring IS a lift"},
	// A jetpack carries its own fuel and you stand on it.
	"jetpack": {tags: []string{lexicon.TagMachine, lexicon.TagLift, lexicon.TagPowerSource, lexicon.TagPlatform}, why: "it carries its own fuel"},
	"rocket":  {tags: []string{lexicon.TagMachine, lexicon.TagPowerSource}, why: "it carries its own fuel"},
	"glider":  {tags: []string{lexicon.TagMachine}, why: "a glider is still a machine"},
	// Blunt tools that open a lock: the world says a hammer, a drill and a wand
	// defeat a lock, so a child typing one is not punished for the book's
	// narrower tag list.
	"hammer": {tags: []string{lexicon.TagUnlock}, why: "a hammer breaks the hasp"},
	"drill":  {tags: []string{lexicon.TagUnlock}, why: "a drill goes through the lock"},
	"wand":   {tags: []string{lexicon.TagUnlock}, why: "magic does not check for a key"},
}

// applyAffordances merges the overlay into a freshly parsed object. Bare nouns
// only: adjectives are the book's business.
func applyAffordances(obj lexicon.Object) lexicon.Object {
	if len(obj.Modifiers) > 0 {
		return obj
	}
	note, ok := affordanceOverlay[obj.Key]
	if !ok {
		return obj
	}
	tags := append([]string{}, obj.Tags...)
	for _, tag := range note.tags {
		if !containsTag(tags, tag) {
			tags = append(tags, tag)
		}
	}
	obj.Tags = tags
	return obj
}

// hasTag reports whether an entity's object carries tag.
func hasTag(e Entity, tag string) bool {
	for _, t := range e.Object.Tags {
		if t == tag {
			return true
		}
	}
	return false
}

// hasAnyTag reports whether an entity carries at least one of the tags.
func hasAnyTag(e Entity, tags ...string) bool {
	for _, tag := range tags {
		if hasTag(e, tag) {
			return true
		}
	}
	return false
}

func containsTag(tags []string, want string) bool {
	for _, tag := range tags {
		if tag == want {
			return true
		}
	}
	return false
}

// blocking reports whether an entity is a physical obstruction: the player
// cannot walk through it, only onto it. A held object is never in the way (the
// player is carrying it), and cutting something removes whatever was holding it
// up (rule 5).
func blocking(e Entity) bool {
	if e.Held || e.Cut {
		return false
	}
	return hasAnyTag(e, lexicon.TagSolid, lexicon.TagPlatform)
}

// standable reports whether the player can stand on top of an entity, or climb
// it. Climbable things are steppable but not blocking: a ladder is something you
// go up, not something that stops you.
func standable(e Entity) bool {
	if e.Held || e.Cut {
		return false
	}
	return hasAnyTag(e, lexicon.TagSolid, lexicon.TagPlatform, lexicon.TagClimbable)
}

// floaty reports whether an entity rests on water instead of falling through it.
func floaty(e Entity) bool { return hasTag(e, lexicon.TagBuoyant) }

// tallShapes are drawn and treated as upright: one cell wide and as many tall as
// the object's size. Everything else lies flat: as many wide as its size, one
// cell tall. Shape is a pinned vocabulary, so this mapping is total.
var tallShapes = map[string]bool{
	lexicon.ShapeLadder:  true,
	lexicon.ShapeRope:    true,
	lexicon.ShapeTree:    true,
	lexicon.ShapePerson:  true,
	lexicon.ShapeAnimal:  true,
	lexicon.ShapeBottle:  true,
	lexicon.ShapeCandle:  true,
	lexicon.ShapeKey:     true,
	lexicon.ShapeMachine: true,
	lexicon.ShapeFlag:    true,
	lexicon.ShapeBook:    true,
}

// footprint returns the entity's width and height in cells from the object's
// pinned size (1..3) and shape. It is what makes a bridge span a stream and a
// ladder reach a branch.
func footprint(obj lexicon.Object) (int, int) {
	size := obj.Size
	if size < lexicon.MinSize {
		size = lexicon.MinSize
	}
	if size > lexicon.MaxSize {
		size = lexicon.MaxSize
	}
	if tallShapes[obj.Shape] {
		return 1, size
	}
	return size, 1
}

// cells lists every grid cell an entity occupies. X,Y is the bottom-left cell
// and the footprint extends right and up, so an entity always reads
// bottom-upwards the way the player sees it stand.
func (e Entity) cells() []point {
	out := make([]point, 0, e.W*e.H)
	for dy := 0; dy < e.H; dy++ {
		for dx := 0; dx < e.W; dx++ {
			out = append(out, point{e.X + dx, e.Y - dy})
		}
	}
	return out
}

// point is one grid cell. The origin is the top-left corner: y grows downwards,
// which is how the frontend draws and how State.Terrain is laid out.
type point struct {
	X int
	Y int
}

// chebyshev is the square-grid distance the rules use for "adjacent" (radius 1)
// and blast radius (radius 2).
func chebyshev(a, b point) int {
	dx, dy := a.X-b.X, a.Y-b.Y
	if dx < 0 {
		dx = -dx
	}
	if dy < 0 {
		dy = -dy
	}
	if dx > dy {
		return dx
	}
	return dy
}

// nearCells returns every cell within radius of the entity's own cells.
func (e Entity) nearCells(radius int) []point {
	seen := map[point]bool{}
	out := []point{}
	for _, c := range e.cells() {
		for dy := -radius; dy <= radius; dy++ {
			for dx := -radius; dx <= radius; dx++ {
				p := point{c.X + dx, c.Y + dy}
				if !seen[p] {
					seen[p] = true
					out = append(out, p)
				}
			}
		}
	}
	return out
}

// occupies reports whether the entity's footprint contains a cell.
func (e Entity) occupies(p point) bool {
	for _, c := range e.cells() {
		if c == p {
			return true
		}
	}
	return false
}

// adjacent reports whether two entities have cells within radius of each other
// (and are not the same entity).
func adjacent(a, b Entity, radius int) bool {
	for _, ca := range a.cells() {
		for _, cb := range b.cells() {
			if chebyshev(ca, cb) <= radius {
				return true
			}
		}
	}
	return false
}
