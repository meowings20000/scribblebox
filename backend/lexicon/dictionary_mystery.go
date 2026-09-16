package lexicon

// treasureAndMystery: prizes, sky things, and the objects that only make sense
// in a game where the book itself is a character.
var treasureAndMystery = []entry{
	{
		key: "coin", name: "coin", cat: CatTool, tags: []string{TagMoney, TagTreasure, TagCollectible},
		mass: 1, size: 1, color: "#ffd23f", shape: ShapeCircle, glyph: "COI",
		note:  "small change, enormous plans",
		alias: []string{"money", "gold coin"},
	},
	{
		key: "gold", name: "gold", cat: CatMaterial, tags: []string{TagMoney, TagTreasure, TagHeavy, TagConductive},
		mass: 3, size: 1, color: "#ffd23f", shape: ShapeBox, glyph: "GLD",
		note:  "heavy, shiny, and the cause of most of the maps ever drawn",
		alias: []string{"gold bar", "goldbar"},
	},
	{
		key: "diamond", name: "diamond", cat: CatMaterial, tags: []string{TagTreasure, TagMoney, TagCollectible, TagSharp},
		mass: 1, size: 1, color: "#a8e8f0", shape: ShapeStar, glyph: "DIA",
		note:  "pressed carbon with a great deal of self-esteem",
		alias: []string{"gem", "jewel"},
	},
	{
		key: "chest", name: "chest", cat: CatContainer, tags: []string{TagContainer, TagSolid, TagTreasure},
		mass: 3, size: 2, color: "#8a5a2b", shape: ShapeChest, glyph: "CHT",
		note:  "locked, mostly, and honest about it",
		alias: []string{"treasure chest"},
	},
	{
		key: "treasure", name: "treasure", cat: CatMystery, tags: []string{TagTreasure, TagMoney, TagCollectible},
		mass: 3, size: 2, color: "#ffd23f", shape: ShapeChest, glyph: "TRS",
		note:  "everything the map promised and nothing it mentioned",
		alias: []string{"loot", "prize"},
	},
	{
		key: "crown", name: "crown", cat: CatTool, tags: []string{TagTreasure, TagMoney, TagCollectible},
		mass: 1, size: 1, color: "#ffd23f", shape: ShapeStar, glyph: "CRN",
		note:  "a hat that ends arguments before they start",
		alias: []string{"tiara"},
	},
	{
		key: "ball", name: "ball", cat: CatTool, tags: []string{TagWheeled},
		mass: 1, size: 1, color: "#f2f5f8", shape: ShapeCircle, glyph: "BLL",
		note:  "rolls away at exactly the moment it is needed most",
		alias: []string{"soccer ball", "football"},
	},
	{
		key: "kite", name: "kite", cat: CatTool, tags: []string{TagBuoyant, TagRope, TagFragile},
		mass: 1, size: 2, color: "#e05a8c", shape: ShapeStar, glyph: "KIT",
		note: "flies only when the wind is in a good mood",
	},
	{
		key: "star", name: "star", cat: CatMystery, tags: []string{TagLightSource, TagCollectible, TagMagical},
		mass: 1, size: 1, color: "#fff176", shape: ShapeStar, glyph: "STR",
		note: "everybody wishes on it, nobody ever visits",
	},
	{
		key: "moon", name: "moon", cat: CatMystery, tags: []string{TagLightSource, TagMagical},
		mass: 5, size: 3, color: "#e6e9f2", shape: ShapeCircle, glyph: "MOO",
		note: "borrowed light, delivered at no charge",
	},
	{
		key: "sun", name: "sun", cat: CatMystery, tags: []string{TagFire, TagLightSource, TagMagical},
		mass: 5, size: 3, color: "#ffd93b", shape: ShapeCircle, glyph: "SUN",
		note: "do not look directly at it, however tempting the puzzle makes it",
	},
	{
		key: "rainbow", name: "rainbow", cat: CatMystery, tags: []string{TagMagical, TagLightSource, TagWater},
		mass: 3, size: 3, color: "#ff8fd0", shape: ShapeBlob, glyph: "RBW",
		note: "a promise made entirely out of weather",
	},
	{
		key: "cloud", name: "cloud", cat: CatMaterial, tags: []string{TagWater, TagBuoyant},
		mass: 3, size: 3, color: "#dfe7ee", shape: ShapeBlob, glyph: "CLD",
		note:  "weather you can argue with, and you will lose",
		alias: []string{"rain cloud", "raincloud"},
	},
	{
		key: "rain", name: "rain", cat: CatMaterial, tags: []string{TagWater, TagCold},
		mass: 2, size: 3, color: "#6fa8dc", shape: ShapeBlob, glyph: "RAN",
		note:  "an entire sky deciding to come and visit",
		alias: []string{"rainfall", "raindrops"},
	},
	{
		key: "wind", name: "wind", cat: CatMystery, tags: []string{TagMagical},
		mass: 1, size: 3, color: "#dfe7ee", shape: ShapeBlob, glyph: "WND",
		note: "invisible, opinionated, and impossible to put in a box",
	},
	{
		key: "mystery box", name: "mystery box", cat: CatMystery, tags: []string{TagContainer, TagSolid},
		mass: 2, size: 2, color: "#8a5ab0", shape: ShapeBox, glyph: "???",
		note:  "the book refuses to say what is inside, which is the fun part",
		alias: []string{"mysterybox"},
	},
	{
		key: "treasure map", name: "treasure map", cat: CatTool, tags: []string{TagFlammable, TagFragile, TagTool, TagTreasure},
		mass: 1, size: 1, color: "#e0d2a8", shape: ShapeBook, glyph: "MAP",
		note:  "an X, a dotted line, and no sense of scale whatsoever",
		alias: []string{"map"},
	},
	{
		key: "gravestone", name: "gravestone", cat: CatStructure, tags: []string{TagSolid, TagHeavy, TagPlatform},
		mass: 4, size: 2, color: "#9aa4ad", shape: ShapeBox, glyph: "RIP",
		note:  "the heaviest thing in the world with a sense of humour",
		alias: []string{"tombstone"},
	},
	{
		key: "time machine", name: "time machine", cat: CatMachine, tags: []string{TagMachine, TagMagical, TagConductive},
		mass: 4, size: 2, color: "#6b4a8a", shape: ShapeMachine, glyph: "TIM",
		note:  "probably works, which is the worrying part",
		alias: []string{"timemachine"},
	},
}
