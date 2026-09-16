package lexicon

// toolsAndMaterials: cutting, unlocking, fire-starting, blowing up, and the
// raw stuff the world is made of.
var toolsAndMaterials = []entry{
	{
		key: "match", name: "match", cat: CatTool, tags: []string{TagFire, TagLightSource, TagFlammable},
		mass: 1, size: 1, color: "#d9c08a", shape: ShapeTool, glyph: "MAT",
		note:  "one use, one flame, no second chances",
		alias: []string{"matchstick"},
	},
	{
		key: "lighter", name: "lighter", cat: CatTool, tags: []string{TagFire, TagLightSource, TagFlammable, TagTool},
		mass: 1, size: 1, color: "#555a60", shape: ShapeTool, glyph: "LTR",
		note: "the match that keeps trying",
	},
	{
		key: "campfire", name: "campfire", cat: CatStructure, tags: []string{TagFire, TagLightSource, TagFlammable},
		mass: 3, size: 2, color: "#ff8c42", shape: ShapeFlame, glyph: "CMP",
		note:  "the oldest social network ever invented",
		alias: []string{"fire pit"},
	},
	{
		key: "bonfire", name: "bonfire", cat: CatStructure, tags: []string{TagFire, TagLightSource, TagFlammable, TagHeavy},
		mass: 5, size: 3, color: "#ff5a1f", shape: ShapeFlame, glyph: "BON",
		note: "a campfire that got completely and irreversibly out of hand",
	},
	{
		key: "fire", name: "fire", cat: CatMystery, tags: []string{TagFire, TagLightSource, TagFlammable},
		mass: 2, size: 2, color: "#ff6a00", shape: ShapeFlame, glyph: "FIR",
		note:  "not a thing so much as an ongoing decision",
		alias: []string{"flames"},
	},
	{
		key: "flamethrower", name: "flamethrower", cat: CatTool, tags: []string{TagFire, TagLightSource, TagWeapon, TagFlammable},
		mass: 3, size: 2, color: "#c94f2f", shape: ShapeMachine, glyph: "FLM",
		note:  "an answer to problems that were not on fire yet",
		alias: []string{"flame thrower"},
	},
	{
		key: "fire extinguisher", name: "fire extinguisher", cat: CatTool, tags: []string{TagWater, TagTool, TagContainer},
		mass: 2, size: 1, color: "#d94436", shape: ShapeBottle, glyph: "EXT",
		note:  "the least popular guest at any bonfire",
		alias: []string{"extinguisher"},
	},
	{
		key: "magnifying glass", name: "magnifying glass", cat: CatTool, tags: []string{TagFire, TagLightSource, TagTool},
		mass: 1, size: 1, color: "#a8c8e0", shape: ShapeTool, glyph: "MAG",
		note:  "reads small print and patiently starts small fires",
		alias: []string{"magnifier", "burning glass"},
	},
	{
		key: "water", name: "water", cat: CatMaterial, tags: []string{TagWater},
		mass: 3, size: 2, color: "#4aa3e0", shape: ShapeBlob, glyph: "WTR",
		note: "everybody's favourite solvent and nobody's friend",
	},
	{
		key: "bucket", name: "bucket", cat: CatContainer, tags: []string{TagContainer, TagSolid},
		mass: 2, size: 1, color: "#9aa4ad", shape: ShapeBottle, glyph: "BUC",
		note:  "holds water and, with effort, your dignity",
		alias: []string{"pail"},
	},
	{
		key: "ice", name: "ice", cat: CatMaterial, tags: []string{TagFrozen, TagCold, TagSolid},
		mass: 3, size: 2, color: "#bfe9ff", shape: ShapeBox, glyph: "ICE",
		note: "water that has made up its mind about something",
	},
	{
		key: "ice cube", name: "ice cube", cat: CatMaterial, tags: []string{TagFrozen, TagCold, TagSolid},
		mass: 1, size: 1, color: "#cdeffd", shape: ShapeBox, glyph: "ICB",
		note:  "the only thing that improves a drink by vanishing",
		alias: []string{"icecube", "cube"},
	},
	{
		key: "snow", name: "snow", cat: CatMaterial, tags: []string{TagCold, TagFrozen, TagFragile},
		mass: 1, size: 2, color: "#f2f7ff", shape: ShapeBlob, glyph: "SNW",
		note: "quiet, cold, and excellent at burying evidence",
	},
	{
		key: "snowball", name: "snowball", cat: CatTool, tags: []string{TagCold, TagFrozen, TagFragile},
		mass: 1, size: 1, color: "#f7fbff", shape: ShapeCircle, glyph: "SNB",
		note:  "a temporarily weaponised opinion about winter",
		alias: []string{"snow ball"},
	},
	{
		key: "sand", name: "sand", cat: CatMaterial, tags: []string{TagHeavy},
		mass: 3, size: 2, color: "#e3c88f", shape: ShapeBlob, glyph: "SND",
		note: "a thousand tiny rocks with no respect for socks",
	},
	{
		key: "mud", name: "mud", cat: CatMaterial, tags: []string{TagSticky, TagHeavy},
		mass: 3, size: 2, color: "#6b4f34", shape: ShapeBlob, glyph: "MUD",
		note: "water and dirt, in a conspiracy against your shoes",
	},
	{
		key: "brick", name: "brick", cat: CatMaterial, tags: []string{TagSolid, TagHeavy},
		mass: 2, size: 1, color: "#a8503a", shape: ShapeBox, glyph: "BRK",
		note: "the base unit of construction and of blunt opinions",
	},
	{
		key: "stone", name: "stone", cat: CatMaterial, tags: []string{TagSolid, TagHeavy},
		mass: 3, size: 1, color: "#8d8d8d", shape: ShapeCircle, glyph: "STN",
		note:  "old, patient, and considerably heavier than it looks",
		alias: []string{"rock", "pebble"},
	},
	{
		key: "boulder", name: "boulder", cat: CatMaterial, tags: []string{TagSolid, TagHeavy, TagWheeled},
		mass: 5, size: 3, color: "#7d7a72", shape: ShapeCircle, glyph: "BLD",
		note: "rolls downhill, and it has been waiting all day to",
	},
	{
		key: "knife", name: "knife", cat: CatTool, tags: []string{TagSharp, TagCutting, TagWeapon, TagTool},
		mass: 1, size: 1, color: "#cfd6dd", shape: ShapeTool, glyph: "KNF",
		note:  "cutting, slicing, and explaining yourself afterwards",
		alias: []string{"blade"},
	},
	{
		key: "sword", name: "sword", cat: CatTool, tags: []string{TagSharp, TagCutting, TagWeapon},
		mass: 2, size: 2, color: "#d0d7de", shape: ShapeTool, glyph: "SWD",
		note: "a knife that went away to knight school",
	},
	{
		key: "axe", name: "axe", cat: CatTool, tags: []string{TagSharp, TagCutting, TagWeapon, TagTool},
		mass: 2, size: 2, color: "#8a6a45", shape: ShapeTool, glyph: "AXE",
		note:  "firewood, doors, and anything else in the way",
		alias: []string{"ax"},
	},
	{
		key: "saw", name: "saw", cat: CatTool, tags: []string{TagSharp, TagCutting, TagTool},
		mass: 2, size: 1, color: "#b7c2cc", shape: ShapeTool, glyph: "SAW",
		note: "patient teeth for very impatient problems",
	},
	{
		key: "scissors", name: "scissors", cat: CatTool, tags: []string{TagSharp, TagCutting, TagTool},
		mass: 1, size: 1, color: "#b7c2cc", shape: ShapeTool, glyph: "SCI",
		note:  "two knives having a very organised argument",
		alias: []string{"shears"},
	},
	{
		key: "chainsaw", name: "chainsaw", cat: CatTool, tags: []string{TagSharp, TagCutting, TagMachine, TagTool},
		mass: 3, size: 2, color: "#d94f4f", shape: ShapeMachine, glyph: "CSW",
		note:  "subtle as a shout and about twice as effective",
		alias: []string{"chain saw"},
	},
	{
		key: "drill", name: "drill", cat: CatTool, tags: []string{TagSharp, TagCutting, TagMachine, TagTool},
		mass: 2, size: 1, color: "#e0a43a", shape: ShapeMachine, glyph: "DRL",
		note: "makes small holes and very large promises",
	},
	{
		key: "crowbar", name: "crowbar", cat: CatTool, tags: []string{TagTool, TagUnlock, TagHeavy},
		mass: 2, size: 2, color: "#c94f2f", shape: ShapeTool, glyph: "CRW",
		note:  "the skeleton key for doors that have forgotten their keyhole",
		alias: []string{"pry bar"},
	},
	{
		key: "hammer", name: "hammer", cat: CatTool, tags: []string{TagTool, TagHeavy, TagWeapon},
		mass: 2, size: 1, color: "#8a6a45", shape: ShapeTool, glyph: "HAM",
		note:  "if it does not fit, the hammer has a suggestion",
		alias: []string{"mallet"},
	},
	{
		key: "nail", name: "nail", cat: CatMaterial, tags: []string{TagSharp, TagSolid},
		mass: 1, size: 1, color: "#b7c2cc", shape: ShapeTool, glyph: "NAL",
		note: "small, sharp, and always the last one in the box",
	},
	{
		key: "screwdriver", name: "screwdriver", cat: CatTool, tags: []string{TagTool},
		mass: 1, size: 1, color: "#d94f4f", shape: ShapeTool, glyph: "SDV",
		note: "turns one kind of problem into a slightly different problem",
	},
	{
		key: "wire", name: "wire", cat: CatMaterial, tags: []string{TagConductive, TagRope},
		mass: 1, size: 1, color: "#c28a3a", shape: ShapeRope, glyph: "WIR",
		note: "carries power, and it would like you to know that",
	},
	{
		key: "tape", name: "tape", cat: CatTool, tags: []string{TagSticky, TagTool},
		mass: 1, size: 1, color: "#d9d2c0", shape: ShapeCircle, glyph: "TPE",
		note:  "fixes everything, including things that should not be fixed",
		alias: []string{"duct tape", "sticky tape"},
	},
	{
		key: "glue", name: "glue", cat: CatTool, tags: []string{TagSticky, TagTool},
		mass: 1, size: 1, color: "#f0e6d2", shape: ShapeBottle, glyph: "GLU",
		note:  "friendship, permanently and aggressively enforced",
		alias: []string{"glue stick", "superglue"},
	},
	{
		key: "catapult", name: "catapult", cat: CatMachine, tags: []string{TagMachine, TagWeapon, TagHeavy, TagWheeled},
		mass: 5, size: 3, color: "#8a6a45", shape: ShapeMachine, glyph: "CTP",
		note:  "delivers your opinion directly over the wall",
		alias: []string{"trebuchet"},
	},
	{
		key: "cannon", name: "cannon", cat: CatMachine, tags: []string{TagMachine, TagWeapon, TagExplosive, TagHeavy},
		mass: 5, size: 2, color: "#4a4f55", shape: ShapeMachine, glyph: "CNN",
		note: "an argument that arrives with a great deal of noise",
	},
	{
		key: "bomb", name: "bomb", cat: CatTool, tags: []string{TagExplosive, TagWeapon},
		mass: 2, size: 1, color: "#2b2f36", shape: ShapeCircle, glyph: "BMB",
		note: "a very short conversation with a very loud ending",
	},
	{
		key: "dynamite", name: "dynamite", cat: CatTool, tags: []string{TagExplosive, TagWeapon, TagFlammable},
		mass: 2, size: 1, color: "#d94f4f", shape: ShapeBox, glyph: "DYN",
		note:  "six sticks of optimism tied together with string",
		alias: []string{"tnt"},
	},
	{
		key: "grenade", name: "grenade", cat: CatTool, tags: []string{TagExplosive, TagWeapon},
		mass: 1, size: 1, color: "#4f6b3a", shape: ShapeCircle, glyph: "GRN",
		note: "the only gift in the world with a countdown",
	},
	{
		key: "bow", name: "bow", cat: CatTool, tags: []string{TagWeapon, TagTool, TagRope},
		mass: 2, size: 2, color: "#7a5230", shape: ShapeTool, glyph: "BOW",
		note: "stores a bad idea in a curved stick",
	},
	{
		key: "arrow", name: "arrow", cat: CatTool, tags: []string{TagSharp, TagWeapon, TagTool},
		mass: 1, size: 1, color: "#cbb994", shape: ShapeTool, glyph: "ARR",
		note: "a message delivered at speed and without nuance",
	},
	{
		key: "spear", name: "spear", cat: CatTool, tags: []string{TagSharp, TagWeapon, TagTool},
		mass: 2, size: 2, color: "#8a6a45", shape: ShapeTool, glyph: "SPE",
		note: "the original answer to the problem of distance",
	},
	{
		key: "shield", name: "shield", cat: CatTool, tags: []string{TagSolid, TagTool},
		mass: 2, size: 2, color: "#8a8f98", shape: ShapeCircle, glyph: "SHD",
		note: "the conversation-ending side of an argument",
	},
	{
		key: "armor", name: "armor", cat: CatTool, tags: []string{TagSolid, TagTool, TagHeavy},
		mass: 4, size: 2, color: "#9aa4ad", shape: ShapeBox, glyph: "ARM",
		note:  "dignity, cast in metal and impossible to sit down in",
		alias: []string{"armour"},
	},
	{
		key: "wand", name: "magic wand", cat: CatTool, tags: []string{TagMagical, TagTool},
		mass: 1, size: 1, color: "#6b4a8a", shape: ShapeTool, glyph: "WND",
		note:  "wood that learned one very good trick",
		alias: []string{"magic wand", "wand"},
	},
	{
		key: "spell", name: "spell", cat: CatMystery, tags: []string{TagMagical},
		mass: 1, size: 1, color: "#c77dff", shape: ShapeStar, glyph: "SPL",
		note:  "an instruction the world is not supposed to accept",
		alias: []string{"magic spell"},
	},
	{
		key: "potion", name: "potion", cat: CatFood, tags: []string{TagEdible, TagMagical, TagFragile},
		mass: 1, size: 1, color: "#b04ae0", shape: ShapeBottle, glyph: "PTN",
		note: "tastes terrible, works exactly once",
	},
	{
		key: "crystal ball", name: "crystal ball", cat: CatMystery, tags: []string{TagMagical, TagFragile, TagLightSource},
		mass: 2, size: 1, color: "#a8d8f0", shape: ShapeCircle, glyph: "ORB",
		note:  "shows the future, mostly the boring administrative parts",
		alias: []string{"crystalball", "orb"},
	},
	{
		key: "spell book", name: "spell book", cat: CatTool, tags: []string{TagMagical, TagFlammable, TagTool},
		mass: 2, size: 1, color: "#4a2f6b", shape: ShapeBook, glyph: "SPB",
		note:  "a book that answers back and has opinions",
		alias: []string{"spellbook", "grimoire"},
	},
}
