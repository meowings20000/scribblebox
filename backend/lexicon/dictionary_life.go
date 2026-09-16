package lexicon

// animalsAndPlants: everything alive that is not a person and not on a plate.
var animalsAndPlants = []entry{
	{
		key: "dog", name: "dog", cat: CatAnimal, tags: []string{TagAnimal},
		mass: 2, size: 2, color: "#b58a52", shape: ShapeAnimal, glyph: "DOG",
		note:  "loyal, loud, and quietly convinced it is a small person",
		alias: []string{"puppy", "hound"},
	},
	{
		key: "cat", name: "cat", cat: CatAnimal, tags: []string{TagAnimal, TagClimbable},
		mass: 1, size: 1, color: "#8a7a6a", shape: ShapeAnimal, glyph: "CAT",
		note:  "climbs everything and apologises for absolutely none of it",
		alias: []string{"kitten"},
	},
	{
		key: "bird", name: "bird", cat: CatAnimal, tags: []string{TagAnimal, TagBuoyant},
		mass: 1, size: 1, color: "#4a8ad9", shape: ShapeAnimal, glyph: "BRD",
		note:  "opinions, delivered at dawn, at volume",
		alias: []string{"sparrow"},
	},
	{
		key: "rabbit", name: "rabbit", cat: CatAnimal, tags: []string{TagAnimal},
		mass: 1, size: 1, color: "#d9d2c0", shape: ShapeAnimal, glyph: "RAB",
		note:  "fast, quiet, and perpetually late for something",
		alias: []string{"bunny"},
	},
	{
		key: "horse", name: "horse", cat: CatAnimal, tags: []string{TagAnimal, TagHeavy},
		mass: 4, size: 3, color: "#7a5230", shape: ShapeAnimal, glyph: "HRS",
		note: "a vehicle that would take great offence at the word vehicle",
	},
	{
		key: "elephant", name: "elephant", cat: CatAnimal, tags: []string{TagAnimal, TagHeavy},
		mass: 5, size: 3, color: "#9aa4ad", shape: ShapeAnimal, glyph: "ELP",
		note: "remembers everything, including this sentence",
	},
	{
		key: "monkey", name: "monkey", cat: CatAnimal, tags: []string{TagAnimal, TagClimbable},
		mass: 2, size: 2, color: "#8a6a45", shape: ShapeAnimal, glyph: "MNK",
		note: "hands where hands have absolutely no business being",
	},
	{
		key: "fish", name: "fish", cat: CatAnimal, tags: []string{TagAnimal, TagWater, TagEdible},
		mass: 1, size: 1, color: "#5aa8c9", shape: ShapeAnimal, glyph: "FSH",
		note: "breathes the one thing that would ruin your whole day",
	},
	{
		key: "bee", name: "bee", cat: CatAnimal, tags: []string{TagAnimal, TagSharp},
		mass: 1, size: 1, color: "#e0b843", shape: ShapeAnimal, glyph: "BEE",
		note:  "small, striped, and holding a very old grudge",
		alias: []string{"bumblebee"},
	},
	{
		key: "ant", name: "ant", cat: CatAnimal, tags: []string{TagAnimal, TagClimbable},
		mass: 1, size: 1, color: "#4a3f2f", shape: ShapeAnimal, glyph: "ANT",
		note: "carries ten times its weight and still makes it look easy",
	},
	{
		key: "worm", name: "worm", cat: CatAnimal, tags: []string{TagAnimal},
		mass: 1, size: 1, color: "#c98a8a", shape: ShapeAnimal, glyph: "WRM",
		note: "an honest creature with no other plans for the afternoon",
	},
	{
		key: "spider", name: "spider", cat: CatAnimal, tags: []string{TagAnimal, TagRope, TagClimbable},
		mass: 1, size: 1, color: "#2b2f36", shape: ShapeAnimal, glyph: "SPD",
		note: "eight legs, one web, and zero invitations",
	},
	{
		key: "snake", name: "snake", cat: CatAnimal, tags: []string{TagAnimal, TagRope},
		mass: 2, size: 2, color: "#6b8a4a", shape: ShapeAnimal, glyph: "SNE",
		note: "a rope with a personality and a plan",
	},
	{
		key: "lion", name: "lion", cat: CatAnimal, tags: []string{TagAnimal, TagHeavy, TagSharp},
		mass: 4, size: 3, color: "#c98a3a", shape: ShapeAnimal, glyph: "LIO",
		note: "the loudest possible answer to a quiet afternoon",
	},
	{
		key: "tiger", name: "tiger", cat: CatAnimal, tags: []string{TagAnimal, TagSharp},
		mass: 4, size: 3, color: "#e0a43a", shape: ShapeAnimal, glyph: "TGR",
		note: "stripes, patience, and a very short list of friends",
	},
	{
		key: "bear", name: "bear", cat: CatAnimal, tags: []string{TagAnimal, TagHeavy, TagSharp},
		mass: 5, size: 3, color: "#6b4a2f", shape: ShapeAnimal, glyph: "BER",
		note: "would like your sandwich and is willing to discuss terms",
	},
	{
		key: "wolf", name: "wolf", cat: CatAnimal, tags: []string{TagAnimal, TagSharp},
		mass: 3, size: 2, color: "#7d7d85", shape: ShapeAnimal, glyph: "WOL",
		note: "travels in a group and fully expects you to keep up",
	},
	{
		key: "turtle", name: "turtle", cat: CatAnimal, tags: []string{TagAnimal, TagSolid},
		mass: 2, size: 2, color: "#5a8a5a", shape: ShapeAnimal, glyph: "TRT",
		note: "armour with an extremely relaxed schedule",
	},
	{
		key: "penguin", name: "penguin", cat: CatAnimal, tags: []string{TagAnimal, TagCold, TagBuoyant},
		mass: 2, size: 2, color: "#2b2f36", shape: ShapeAnimal, glyph: "PNG",
		note: "formal wear, deeply informal behaviour",
	},
	{
		key: "shark", name: "shark", cat: CatAnimal, tags: []string{TagAnimal, TagWater, TagSharp, TagCutting},
		mass: 4, size: 3, color: "#6a8a9a", shape: ShapeAnimal, glyph: "SHK",
		note: "the ocean's least ambiguous hello",
	},
	{
		key: "whale", name: "whale", cat: CatAnimal, tags: []string{TagAnimal, TagWater, TagBuoyant, TagHeavy},
		mass: 5, size: 3, color: "#4a6a8a", shape: ShapeAnimal, glyph: "WHL",
		note: "enormous, musical, and completely unaware of your plans",
	},
	{
		key: "octopus", name: "octopus", cat: CatAnimal, tags: []string{TagAnimal, TagWater},
		mass: 3, size: 2, color: "#c94a8a", shape: ShapeAnimal, glyph: "OCT",
		note: "eight arms and no interest whatsoever in your rules",
	},
	{
		key: "crab", name: "crab", cat: CatAnimal, tags: []string{TagAnimal, TagWater, TagSharp, TagSolid},
		mass: 2, size: 1, color: "#d94f4f", shape: ShapeAnimal, glyph: "CRB",
		note: "sideways, and extremely proud of it",
	},
	{
		key: "butterfly", name: "butterfly", cat: CatAnimal, tags: []string{TagAnimal, TagBuoyant, TagFragile},
		mass: 1, size: 1, color: "#e07ac9", shape: ShapeAnimal, glyph: "BTF",
		note: "a very short career in a very pretty costume",
	},
	{
		key: "chicken", name: "chicken", cat: CatAnimal, tags: []string{TagAnimal, TagEdible},
		mass: 1, size: 1, color: "#f0e6d2", shape: ShapeAnimal, glyph: "CKN",
		note: "brave in groups, delicious in sandwiches",
	},
	{
		key: "cow", name: "cow", cat: CatAnimal, tags: []string{TagAnimal, TagHeavy, TagEdible},
		mass: 4, size: 3, color: "#f5f0e6", shape: ShapeAnimal, glyph: "COW",
		note: "milk, mostly, plus a great deal of standing around",
	},
	{
		key: "pig", name: "pig", cat: CatAnimal, tags: []string{TagAnimal, TagEdible},
		mass: 3, size: 2, color: "#f0c9c9", shape: ShapeAnimal, glyph: "PIG",
		note: "food, and also surprisingly good at being one",
	},
	{
		key: "sheep", name: "sheep", cat: CatAnimal, tags: []string{TagAnimal, TagEdible},
		mass: 2, size: 2, color: "#f2f5f8", shape: ShapeAnimal, glyph: "SHP",
		note: "fluffy, agreeable, and counted when you cannot sleep",
	},
	{
		key: "duck", name: "duck", cat: CatAnimal, tags: []string{TagAnimal, TagBuoyant, TagEdible},
		mass: 1, size: 1, color: "#e0c04a", shape: ShapeAnimal, glyph: "DUC",
		note: "floats, waddles, and quacks with total authority",
	},
	{
		key: "mouse", name: "mouse", cat: CatAnimal, tags: []string{TagAnimal},
		mass: 1, size: 1, color: "#9aa4ad", shape: ShapeAnimal, glyph: "MSE",
		note:  "small, quick, and no longer remotely afraid of you",
		alias: []string{"mice"},
	},
	{
		key: "frog", name: "frog", cat: CatAnimal, tags: []string{TagAnimal, TagBuoyant},
		mass: 1, size: 1, color: "#5aa85a", shape: ShapeAnimal, glyph: "FRG",
		note: "jumps with no plan and lands with no fuss",
	},
	{
		key: "dinosaur", name: "dinosaur", cat: CatAnimal, tags: []string{TagAnimal, TagHeavy, TagSharp},
		mass: 5, size: 3, color: "#6b8a4a", shape: ShapeAnimal, glyph: "DNO",
		note:  "extinct, unbothered, and still considerably bigger than you",
		alias: []string{"t-rex", "trex"},
	},
	{
		key: "tree", name: "tree", cat: CatPlant, tags: []string{TagPlant, TagClimbable, TagFlammable, TagSolid, TagHeavy},
		mass: 4, size: 3, color: "#3f7d3f", shape: ShapeTree, glyph: "TRE",
		note: "shade, fruit, and a very long-term plan",
	},
	{
		key: "bush", name: "bush", cat: CatPlant, tags: []string{TagPlant, TagFlammable, TagSolid},
		mass: 2, size: 2, color: "#4a8a4a", shape: ShapeTree, glyph: "BSH",
		note:  "a tree that gave up early but kept all the leaves",
		alias: []string{"shrub"},
	},
	{
		key: "flower", name: "flower", cat: CatPlant, tags: []string{TagPlant, TagFragile, TagCollectible},
		mass: 1, size: 1, color: "#e05a8c", shape: ShapeStar, glyph: "FLW",
		note:  "pretty, temporary, and the entire point of spring",
		alias: []string{"blossom"},
	},
	{
		key: "grass", name: "grass", cat: CatPlant, tags: []string{TagPlant, TagFlammable},
		mass: 1, size: 2, color: "#6aa84f", shape: ShapeRope, glyph: "GRS",
		note:  "green, patient, and quietly taking over everything",
		alias: []string{"lawn"},
	},
	{
		key: "cactus", name: "cactus", cat: CatPlant, tags: []string{TagPlant, TagSharp},
		mass: 2, size: 2, color: "#4a8a5a", shape: ShapeTree, glyph: "CAC",
		note: "a plant that has clearly chosen violence",
	},
	{
		key: "mushroom", name: "mushroom", cat: CatPlant, tags: []string{TagPlant, TagEdible, TagFragile},
		mass: 1, size: 1, color: "#c94a4a", shape: ShapeBlob, glyph: "MSH",
		note:  "dinner or a lesson, entirely depending on the mushroom",
		alias: []string{"toadstool"},
	},
	{
		key: "fern", name: "fern", cat: CatPlant, tags: []string{TagPlant, TagFlammable},
		mass: 1, size: 1, color: "#3f7d5f", shape: ShapeTree, glyph: "FRN",
		note: "older than trees and considerably better at waiting",
	},
	{
		key: "seaweed", name: "seaweed", cat: CatPlant, tags: []string{TagPlant, TagWater, TagRope, TagEdible},
		mass: 1, size: 2, color: "#3f6b5a", shape: ShapeRope, glyph: "SEA",
		note: "the sea's hair, and it is well overdue a cut",
	},
	{
		key: "pumpkin", name: "pumpkin", cat: CatPlant, tags: []string{TagPlant, TagEdible, TagWheeled},
		mass: 3, size: 2, color: "#e08a2f", shape: ShapeCircle, glyph: "PKP",
		note: "a vegetable that rolls, and on one night a year, glows",
	},
}
