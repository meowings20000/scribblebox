package lexicon

// foodAndDrink: things a player will type because they are hungry, or because
// the puzzle needed bait.
var foodAndDrink = []entry{
	{
		key: "apple", name: "apple", cat: CatFood, tags: []string{TagEdible},
		mass: 1, size: 1, color: "#e0394a", shape: ShapeCircle, glyph: "APL",
		note:  "gravity's favourite demonstration",
		alias: []string{"fruit"},
	},
	{
		key: "banana", name: "banana", cat: CatFood, tags: []string{TagEdible},
		mass: 1, size: 1, color: "#f0d94a", shape: ShapeBlob, glyph: "BAN",
		note: "a snack with a built-in wrapper and a bad reputation",
	},
	{
		key: "bread", name: "bread", cat: CatFood, tags: []string{TagEdible, TagFlammable},
		mass: 2, size: 2, color: "#c9a24a", shape: ShapeBox, glyph: "BRE",
		note:  "the foundation every sandwich has ever been built on",
		alias: []string{"loaf"},
	},
	{
		key: "cake", name: "cake", cat: CatFood, tags: []string{TagEdible, TagFragile},
		mass: 2, size: 2, color: "#f0c9e0", shape: ShapeBox, glyph: "CAK",
		note:  "lies about how many people it will actually feed",
		alias: []string{"birthday cake"},
	},
	{
		key: "cheese", name: "cheese", cat: CatFood, tags: []string{TagEdible},
		mass: 2, size: 1, color: "#f0d94a", shape: ShapeBox, glyph: "CHS",
		note: "smelly, beloved, structurally unreliable",
	},
	{
		key: "sandwich", name: "sandwich", cat: CatFood, tags: []string{TagEdible},
		mass: 1, size: 1, color: "#d9c08a", shape: ShapeBox, glyph: "SWH",
		note:  "two slices of bread making a solemn promise to some cheese",
		alias: []string{"sarnie"},
	},
	{
		key: "pizza", name: "pizza", cat: CatFood, tags: []string{TagEdible},
		mass: 2, size: 2, color: "#e07a3a", shape: ShapeCircle, glyph: "PZA",
		note:  "a disc of dinner that will end up under the sofa",
		alias: []string{"slice"},
	},
	{
		key: "milk", name: "milk", cat: CatFood, tags: []string{TagEdible},
		mass: 2, size: 1, color: "#f7fbff", shape: ShapeBottle, glyph: "MLK",
		note: "white, honest, and spoiling as we speak",
	},
	{
		key: "coffee", name: "coffee", cat: CatFood, tags: []string{TagEdible},
		mass: 1, size: 1, color: "#4a3020", shape: ShapeBottle, glyph: "COF",
		note: "the reason the morning gets attempted at all",
	},
	{
		key: "tea", name: "tea", cat: CatFood, tags: []string{TagEdible},
		mass: 1, size: 1, color: "#a8c98a", shape: ShapeBottle, glyph: "TEA",
		note: "hot leaf water, served with ceremony and judgement",
	},
	{
		key: "juice", name: "juice", cat: CatFood, tags: []string{TagEdible},
		mass: 1, size: 1, color: "#e0a43a", shape: ShapeBottle, glyph: "JUC",
		note: "fruit, but in a considerable hurry",
	},
	{
		key: "egg", name: "egg", cat: CatFood, tags: []string{TagEdible, TagFragile},
		mass: 1, size: 1, color: "#f5f0e6", shape: ShapeCircle, glyph: "EGG",
		note: "a shell, a secret, and almost no structural confidence",
	},
	{
		key: "bacon", name: "bacon", cat: CatFood, tags: []string{TagEdible, TagFlammable},
		mass: 1, size: 1, color: "#c94f4f", shape: ShapeFlag, glyph: "BCN",
		note: "the only food that improves everything, including itself",
	},
	{
		key: "chocolate", name: "chocolate", cat: CatFood, tags: []string{TagEdible},
		mass: 1, size: 1, color: "#5a3a24", shape: ShapeBox, glyph: "CHO",
		note:  "happiness with a melting point",
		alias: []string{"choc"},
	},
	{
		key: "cookie", name: "cookie", cat: CatFood, tags: []string{TagEdible, TagFragile},
		mass: 1, size: 1, color: "#c9934a", shape: ShapeCircle, glyph: "COO",
		note:  "crumbs everywhere and absolutely no regrets",
		alias: []string{"biscuit"},
	},
	{
		key: "donut", name: "donut", cat: CatFood, tags: []string{TagEdible, TagWheeled},
		mass: 1, size: 1, color: "#e08a2f", shape: ShapeCircle, glyph: "DNT",
		note:  "a circle with a hole and a firm rolling agenda",
		alias: []string{"doughnut"},
	},
	{
		key: "honey", name: "honey", cat: CatFood, tags: []string{TagEdible, TagSticky},
		mass: 1, size: 1, color: "#e0a43a", shape: ShapeBottle, glyph: "HNY",
		note: "bee effort, carefully jarred and slowly weaponised",
	},
	{
		key: "ice cream", name: "ice cream", cat: CatFood, tags: []string{TagEdible, TagCold, TagFrozen},
		mass: 1, size: 1, color: "#f5d9e0", shape: ShapeCircle, glyph: "ICR",
		note:  "a race against physics, and physics is winning",
		alias: []string{"icecream", "gelato"},
	},
	{
		key: "soup", name: "soup", cat: CatFood, tags: []string{TagEdible, TagWater},
		mass: 2, size: 1, color: "#d9a24a", shape: ShapeBlob, glyph: "SUP",
		note: "dinner, or a warm bath for vegetables",
	},
	{
		key: "soda", name: "soda", cat: CatFood, tags: []string{TagEdible},
		mass: 1, size: 1, color: "#c94a4a", shape: ShapeBottle, glyph: "SDA",
		note:  "sugar and bubbles in a can of bad decisions",
		alias: []string{"cola", "fizzy drink"},
	},
	{
		key: "salt", name: "salt", cat: CatMaterial, tags: []string{TagEdible, TagFragile},
		mass: 1, size: 1, color: "#f5f8ff", shape: ShapeBottle, glyph: "SLT",
		note: "a mineral that improves food and ruins snails",
	},
	{
		key: "pepper", name: "pepper", cat: CatMaterial, tags: []string{TagEdible, TagFragile},
		mass: 1, size: 1, color: "#3a3a3a", shape: ShapeBottle, glyph: "PEP",
		note:  "sneeze powder with an excellent reputation",
		alias: []string{"black pepper"},
	},
	{
		key: "carrot", name: "carrot", cat: CatFood, tags: []string{TagEdible, TagPlant},
		mass: 1, size: 1, color: "#e08a2f", shape: ShapeTool, glyph: "CRT",
		note: "orange, crunchy, and falsely accused of improving eyesight",
	},
	{
		key: "potato", name: "potato", cat: CatFood, tags: []string{TagEdible, TagPlant},
		mass: 1, size: 1, color: "#c9a24a", shape: ShapeBlob, glyph: "PTT",
		note:  "the most flexible food in the drawer",
		alias: []string{"spud"},
	},
	{
		key: "tomato", name: "tomato", cat: CatFood, tags: []string{TagEdible, TagPlant, TagFragile},
		mass: 1, size: 1, color: "#d94f2f", shape: ShapeCircle, glyph: "TMT",
		note: "a fruit, but do not start that argument in here",
	},
	{
		key: "onion", name: "onion", cat: CatFood, tags: []string{TagEdible, TagPlant},
		mass: 1, size: 1, color: "#d9c08a", shape: ShapeCircle, glyph: "ONN",
		note: "layers, tears, and no apologies offered",
	},
	{
		key: "grape", name: "grape", cat: CatFood, tags: []string{TagEdible, TagPlant},
		mass: 1, size: 1, color: "#7a4a8a", shape: ShapeCircle, glyph: "GRA",
		note: "wine, waiting patiently in a small purple ball",
	},
	{
		key: "orange", name: "orange", cat: CatFood, tags: []string{TagEdible, TagPlant},
		mass: 1, size: 1, color: "#f97316", shape: ShapeCircle, glyph: "ORG",
		note: "a colour that had a fruit named after it",
	},
	{
		key: "lemon", name: "lemon", cat: CatFood, tags: []string{TagEdible, TagPlant},
		mass: 1, size: 1, color: "#facc15", shape: ShapeCircle, glyph: "LEM",
		note: "sour enough to reset your entire face",
	},
	{
		key: "cherry", name: "cherry", cat: CatFood, tags: []string{TagEdible, TagPlant},
		mass: 1, size: 1, color: "#c9284a", shape: ShapeCircle, glyph: "CHY",
		note: "small, red, and never ever alone",
	},
	{
		key: "strawberry", name: "strawberry", cat: CatFood, tags: []string{TagEdible, TagPlant, TagFragile},
		mass: 1, size: 1, color: "#e0394a", shape: ShapeCircle, glyph: "STB",
		note: "sweet, seedy, and the entire reason summer exists",
	},
	{
		key: "watermelon", name: "watermelon", cat: CatFood, tags: []string{TagEdible, TagPlant, TagHeavy, TagWheeled},
		mass: 4, size: 2, color: "#4a8a4a", shape: ShapeCircle, glyph: "WML",
		note: "a fruit with its own gravity, and it rolls",
	},
}
