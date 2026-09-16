package lexicon

// peopleAndCreatures: the humans and the not-quite-humans. They carry the
// "person" tag and live under CatMystery, because the pinned category list has
// no category of their own.
var peopleAndCreatures = []entry{
	{
		key: "wizard", name: "wizard", cat: CatMystery, tags: []string{TagPerson, TagMagical},
		mass: 2, size: 2, color: "#4a3f8a", shape: ShapePerson, glyph: "WIZ",
		note:  "allergic to sleeves and to being contradicted",
		alias: []string{"sorcerer", "mage"},
	},
	{
		key: "witch", name: "witch", cat: CatMystery, tags: []string{TagPerson, TagMagical},
		mass: 2, size: 2, color: "#3f6b4a", shape: ShapePerson, glyph: "WCH",
		note: "flies on a broom and holds a very long grudge",
	},
	{
		key: "fairy", name: "fairy", cat: CatMystery, tags: []string{TagPerson, TagMagical, TagBuoyant},
		mass: 1, size: 1, color: "#f5c9e0", shape: ShapePerson, glyph: "FRY",
		note: "small, delighted, and mildly dangerous when bored",
	},
	{
		key: "genie", name: "genie", cat: CatMystery, tags: []string{TagPerson, TagMagical},
		mass: 3, size: 3, color: "#6a8ad9", shape: ShapePerson, glyph: "GNI",
		note:  "three wishes, all of them a trap",
		alias: []string{"djinn"},
	},
	{
		key: "angel", name: "angel", cat: CatMystery, tags: []string{TagPerson, TagMagical, TagLightSource},
		mass: 2, size: 2, color: "#f5f8ff", shape: ShapePerson, glyph: "ANG",
		note: "reassuring, and quite impossible to get a straight answer from",
	},
	{
		key: "demon", name: "demon", cat: CatMystery, tags: []string{TagPerson, TagMagical, TagFire},
		mass: 3, size: 2, color: "#8a2f3f", shape: ShapePerson, glyph: "DMN",
		note: "negotiates badly, and enjoys it enormously",
	},
	{
		key: "vampire", name: "vampire", cat: CatMystery, tags: []string{TagPerson, TagMagical, TagSharp},
		mass: 2, size: 2, color: "#c9c2d9", shape: ShapePerson, glyph: "VMP",
		note: "will not come in unless you specifically invite it",
	},
	{
		key: "zombie", name: "zombie", cat: CatMystery, tags: []string{TagPerson},
		mass: 2, size: 2, color: "#7d8a6a", shape: ShapePerson, glyph: "ZMB",
		note: "motivated, but strictly in one direction",
	},
	{
		key: "skeleton", name: "skeleton", cat: CatMystery, tags: []string{TagPerson, TagFragile},
		mass: 1, size: 2, color: "#e8e2d4", shape: ShapePerson, glyph: "SKL",
		note: "the remains of a considerably better day",
	},
	{
		key: "mummy", name: "mummy", cat: CatMystery, tags: []string{TagPerson, TagFlammable, TagRope},
		mass: 2, size: 2, color: "#d9cbb0", shape: ShapePerson, glyph: "MUM",
		note: "bandaged, unhurried, and quietly cursed",
	},
	{
		key: "werewolf", name: "werewolf", cat: CatMystery, tags: []string{TagAnimal, TagPerson},
		mass: 3, size: 2, color: "#6b5b4a", shape: ShapeAnimal, glyph: "WWF",
		note: "loses his temper once every four weeks, like clockwork",
	},
	{
		key: "goblin", name: "goblin", cat: CatMystery, tags: []string{TagPerson},
		mass: 1, size: 1, color: "#6b8a4a", shape: ShapePerson, glyph: "GOB",
		note:  "small hands, large plans, no supervision",
		alias: []string{"gremlin"},
	},
	{
		key: "troll", name: "troll", cat: CatMystery, tags: []string{TagPerson, TagHeavy, TagSolid},
		mass: 4, size: 3, color: "#7d8a7a", shape: ShapePerson, glyph: "TRL",
		note: "lives under bridges and pays absolutely no rent",
	},
	{
		key: "ninja", name: "ninja", cat: CatMystery, tags: []string{TagPerson, TagSharp, TagWeapon},
		mass: 2, size: 2, color: "#2b2f36", shape: ShapePerson, glyph: "NIN",
		note: "you did not see this object and you cannot prove otherwise",
	},
	{
		key: "pirate", name: "pirate", cat: CatMystery, tags: []string{TagPerson, TagWeapon, TagTreasure},
		mass: 2, size: 2, color: "#4a3f4a", shape: ShapePerson, glyph: "PIR",
		note:  "arr, and several other legally binding statements",
		alias: []string{"buccaneer"},
	},
	{
		key: "king", name: "king", cat: CatMystery, tags: []string{TagPerson, TagMoney},
		mass: 2, size: 2, color: "#8a2f4a", shape: ShapePerson, glyph: "KNG",
		note:  "owns the crown and most of the arguments",
		alias: []string{"monarch"},
	},
	{
		key: "queen", name: "queen", cat: CatMystery, tags: []string{TagPerson, TagMoney},
		mass: 2, size: 2, color: "#6a2f8a", shape: ShapePerson, glyph: "QUN",
		note: "the crown, but considerably louder",
	},
	{
		key: "teacher", name: "teacher", cat: CatMystery, tags: []string{TagPerson, TagTool},
		mass: 2, size: 2, color: "#4a6fa5", shape: ShapePerson, glyph: "TCH",
		note: "knows the answer and would much rather you found it",
	},
	{
		key: "student", name: "student", cat: CatMystery, tags: []string{TagPerson, TagContainer},
		mass: 2, size: 2, color: "#4a8a6a", shape: ShapePerson, glyph: "STU",
		note:  "carrying homework and hope in exactly equal measure",
		alias: []string{"pupil"},
	},
	{
		key: "chef", name: "chef", cat: CatMystery, tags: []string{TagPerson, TagTool, TagSharp},
		mass: 2, size: 2, color: "#d9d2c0", shape: ShapePerson, glyph: "CHF",
		note:  "armed with a knife and an extremely strong opinion",
		alias: []string{"cook"},
	},
	{
		key: "doctor", name: "doctor", cat: CatMystery, tags: []string{TagPerson, TagTool},
		mass: 2, size: 2, color: "#dfe7ee", shape: ShapePerson, glyph: "DOC",
		note:  "will see you now, and invoice you later",
		alias: []string{"physician"},
	},
	{
		key: "alien", name: "alien", cat: CatMystery, tags: []string{TagPerson, TagMagical},
		mass: 2, size: 2, color: "#5aa86b", shape: ShapePerson, glyph: "ALN",
		note:  "visiting, judging, and taking small samples",
		alias: []string{"extra terrestrial"},
	},
	{
		key: "ghost", name: "ghost", cat: CatMystery, tags: []string{TagPerson, TagMagical, TagBuoyant},
		mass: 1, size: 2, color: "#dfe7f2", shape: ShapePerson, glyph: "GHO",
		note:  "unfinished business, impeccable manners",
		alias: []string{"spook"},
	},
	{
		key: "dragon", name: "dragon", cat: CatMystery, tags: []string{TagAnimal, TagMagical, TagFire, TagHeavy},
		mass: 5, size: 3, color: "#3f7d3f", shape: ShapeAnimal, glyph: "DRG",
		note: "hoards gold and is remarkably sensitive about it",
	},
	{
		key: "unicorn", name: "unicorn", cat: CatMystery, tags: []string{TagAnimal, TagMagical, TagSharp},
		mass: 3, size: 2, color: "#f0d9f5", shape: ShapeAnimal, glyph: "UNI",
		note: "a horse with a pricing problem and a sharp hat",
	},
	{
		key: "giant", name: "giant", cat: CatMystery, tags: []string{TagPerson, TagHeavy},
		mass: 5, size: 3, color: "#8a7a5a", shape: ShapePerson, glyph: "GNT",
		note:  "very tall, very slow, and surprisingly easy to annoy",
		alias: []string{"ogre"},
	},
}
