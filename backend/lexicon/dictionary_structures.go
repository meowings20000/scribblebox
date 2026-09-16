package lexicon

// climbingAndStructures: the things a player builds, stacks, climbs and stands
// on. This is the vocabulary most climbing puzzles reach for.
var climbingAndStructures = []entry{
	{
		key: "ladder", name: "ladder", cat: CatStructure, tags: []string{TagClimbable},
		mass: 2, size: 1, color: "#a1662f", shape: ShapeLadder, glyph: "LAD",
		note: "two rungs from the top is where all the good hiding places are",
	},
	{
		key: "rope", name: "rope", cat: CatTool, tags: []string{TagRope, TagClimbable},
		mass: 1, size: 1, color: "#c8a165", shape: ShapeRope, glyph: "ROP",
		note: "cuttable, climbable, and always about one length too short",
	},
	{
		key: "chain", name: "chain", cat: CatMaterial, tags: []string{TagRope, TagClimbable, TagHeavy, TagSolid},
		mass: 3, size: 2, color: "#9aa4ad", shape: ShapeRope, glyph: "CHN",
		note: "heavier than rope and much less willing to be tied",
	},
	{
		key: "grapple", name: "grappling hook", cat: CatTool, tags: []string{TagTool, TagClimbable, TagRope, TagSharp},
		mass: 2, size: 1, color: "#7d8a97", shape: ShapeTool, glyph: "GRP",
		note:  "throw it up, pull it down, hope it holds",
		alias: []string{"grappling hook", "grapple hook", "grappling iron"},
	},
	{
		key: "vine", name: "vine", cat: CatPlant, tags: []string{TagPlant, TagClimbable, TagRope, TagFlammable},
		mass: 1, size: 2, color: "#3f7d3f", shape: ShapeRope, glyph: "VIN",
		note: "the jungle's rope, and it snaps if you look at it wrong",
	},
	{
		key: "stairs", name: "stairs", cat: CatStructure, tags: []string{TagClimbable, TagPlatform, TagSolid},
		mass: 5, size: 3, color: "#b9b3a7", shape: ShapePlank, glyph: "STA",
		note:  "the slowest way up and by far the fastest way down",
		alias: []string{"steps", "staircase", "stairway"},
	},
	{
		key: "elevator", name: "elevator", cat: CatMachine, tags: []string{TagMachine, TagPlatform, TagLift, TagSolid, TagHeavy},
		mass: 5, size: 3, color: "#cfd6dd", shape: ShapeMachine, glyph: "ELV",
		note:  "needs power, and it will not take you sideways",
		alias: []string{"lift", "elevator shaft"},
	},
	{
		key: "jetpack", name: "jetpack", cat: CatMachine, tags: []string{TagMachine, TagLift, TagFlammable},
		mass: 2, size: 2, color: "#d94f4f", shape: ShapeMachine, glyph: "JET",
		note:  "briefly the best idea anybody has ever had",
		alias: []string{"jet pack", "jetpack"},
	},
	{
		key: "balloon", name: "balloon", cat: CatTool, tags: []string{TagBuoyant, TagFragile},
		mass: 1, size: 1, color: "#e05a8c", shape: ShapeCircle, glyph: "BAL",
		note: "one sharp thing away from a disappointing afternoon",
	},
	{
		key: "spring", name: "spring", cat: CatTool, tags: []string{TagTool, TagSolid},
		mass: 1, size: 1, color: "#b8c4cf", shape: ShapeTool, glyph: "SPR",
		note:  "stores a bounce and forgets to warn anybody",
		alias: []string{"coil"},
	},
	{
		key: "trampoline", name: "trampoline", cat: CatStructure, tags: []string{TagPlatform, TagFragile, TagBuoyant},
		mass: 3, size: 3, color: "#4a6fa5", shape: ShapeCircle, glyph: "TRM",
		note: "physics, but this time in your favour",
	},
	{
		key: "box", name: "box", cat: CatContainer, tags: []string{TagContainer, TagSolid, TagPlatform, TagFlammable},
		mass: 2, size: 2, color: "#c08a4a", shape: ShapeBox, glyph: "BOX",
		note:  "a crate by any other name, and it still holds exactly one thing",
		alias: []string{"crate", "wooden box", "cardboard box"},
	},
	{
		key: "barrel", name: "barrel", cat: CatContainer, tags: []string{TagContainer, TagSolid, TagWheeled, TagFlammable},
		mass: 3, size: 2, color: "#8a5a2b", shape: ShapeBottle, glyph: "BAR",
		note:  "rolls when you least want it to and holds whatever you feed it",
		alias: []string{"keg"},
	},
	{
		key: "plank", name: "plank", cat: CatMaterial, tags: []string{TagPlatform, TagSolid, TagBuoyant, TagFlammable},
		mass: 2, size: 2, color: "#b58a52", shape: ShapePlank, glyph: "PLK",
		note:  "the bridge you build when there is no bridge",
		alias: []string{"board", "timber", "wooden board"},
	},
	{
		key: "bridge", name: "bridge", cat: CatStructure, tags: []string{TagPlatform, TagSolid, TagHeavy},
		mass: 4, size: 3, color: "#8d8d8d", shape: ShapePlank, glyph: "BRG",
		note:  "the polite way across water, and the only one with railings",
		alias: []string{"footbridge"},
	},
	{
		key: "raft", name: "raft", cat: CatVehicle, tags: []string{TagBuoyant, TagPlatform, TagSolid, TagFlammable},
		mass: 3, size: 2, color: "#b58a52", shape: ShapePlank, glyph: "RFT",
		note: "a floating argument against drowning",
	},
	{
		key: "boat", name: "boat", cat: CatVehicle, tags: []string{TagBuoyant, TagPlatform, TagSolid, TagContainer},
		mass: 4, size: 2, color: "#e8e2d4", shape: ShapeVehicle, glyph: "BOA",
		note:  "floats, and that is the whole personality",
		alias: []string{"ship", "rowboat"},
	},
	{
		key: "canoe", name: "canoe", cat: CatVehicle, tags: []string{TagBuoyant, TagPlatform, TagSolid},
		mass: 3, size: 2, color: "#c1744a", shape: ShapeVehicle, glyph: "CAN",
		note:  "a boat that fully intends you to do the work",
		alias: []string{"kayak"},
	},
	{
		key: "surfboard", name: "surfboard", cat: CatVehicle, tags: []string{TagBuoyant, TagPlatform},
		mass: 2, size: 2, color: "#4ec9d9", shape: ShapePlank, glyph: "SUR",
		note:  "waves, balance, and a large unimpressed audience",
		alias: []string{"surf board"},
	},
	{
		key: "log", name: "log", cat: CatMaterial, tags: []string{TagPlatform, TagBuoyant, TagWheeled, TagHeavy, TagFlammable},
		mass: 3, size: 2, color: "#7a5230", shape: ShapePlank, glyph: "LOG",
		note:  "rolls downhill with the confidence of something that cannot be blamed",
		alias: []string{"tree trunk"},
	},
	{
		key: "stool", name: "stool", cat: CatStructure, tags: []string{TagFurniture, TagPlatform, TagSolid, TagFlammable},
		mass: 2, size: 1, color: "#a9763f", shape: ShapeBox, glyph: "STL",
		note: "a chair that skipped the back and went straight to the point",
	},
	{
		key: "chair", name: "chair", cat: CatStructure, tags: []string{TagFurniture, TagPlatform, TagSolid, TagFlammable},
		mass: 2, size: 2, color: "#8e6b45", shape: ShapeBox, glyph: "CHR",
		note:  "stand on it and the engineering gets personal very quickly",
		alias: []string{"seat"},
	},
	{
		key: "table", name: "table", cat: CatStructure, tags: []string{TagFurniture, TagPlatform, TagSolid, TagFlammable},
		mass: 3, size: 2, color: "#a9825c", shape: ShapePlank, glyph: "TBL",
		note: "a flat apology for the fact that there are no more chairs",
	},
	{
		key: "scaffolding", name: "scaffolding", cat: CatStructure, tags: []string{TagClimbable, TagPlatform, TagSolid, TagHeavy},
		mass: 4, size: 3, color: "#9aa4ad", shape: ShapeLadder, glyph: "SCF",
		note:  "the temporary truth holding up the permanent one",
		alias: []string{"scaffold"},
	},
	{
		key: "fence", name: "fence", cat: CatStructure, tags: []string{TagSolid, TagClimbable},
		mass: 2, size: 2, color: "#cbb994", shape: ShapePlank, glyph: "FEN",
		note:  "a wall with strong opinions about the view",
		alias: []string{"picket fence"},
	},
	{
		key: "wall", name: "wall", cat: CatStructure, tags: []string{TagSolid, TagHeavy, TagPlatform},
		mass: 5, size: 3, color: "#b0a08a", shape: ShapeBox, glyph: "WAL",
		note:  "the world's most honest barrier",
		alias: []string{"brick wall"},
	},
	{
		key: "pillar", name: "pillar", cat: CatStructure, tags: []string{TagSolid, TagHeavy, TagPlatform},
		mass: 5, size: 3, color: "#cfc8bb", shape: ShapeBox, glyph: "PLR",
		note:  "holds up ceilings and takes absolutely no questions",
		alias: []string{"column"},
	},
	{
		key: "hook", name: "hook", cat: CatTool, tags: []string{TagTool, TagSharp},
		mass: 1, size: 1, color: "#8a8f98", shape: ShapeTool, glyph: "HOK",
		note: "curved specifically to catch on things you cannot reach",
	},
	{
		key: "net", name: "net", cat: CatTool, tags: []string{TagRope, TagContainer},
		mass: 1, size: 2, color: "#d9d2c0", shape: ShapeRope, glyph: "NET",
		note:  "catches whatever you were too slow to grab by hand",
		alias: []string{"netting"},
	},
	{
		key: "flag", name: "flag", cat: CatStructure, tags: []string{TagSolid, TagClimbable},
		mass: 1, size: 2, color: "#d94f4f", shape: ShapeFlag, glyph: "FLG",
		note:  "a piece of cloth with territorial ambitions",
		alias: []string{"banner"},
	},
	{
		key: "window", name: "window", cat: CatStructure, tags: []string{TagFragile, TagSolid},
		mass: 2, size: 2, color: "#bfe3ee", shape: ShapeBox, glyph: "WIN",
		note:  "transparent, breakable, and the only exit on the third floor",
		alias: []string{"pane"},
	},
	{
		key: "door", name: "door", cat: CatStructure, tags: []string{TagSolid, TagFurniture},
		mass: 3, size: 3, color: "#8a5a2b", shape: ShapeBox, glyph: "DOR",
		note: "a wall with a handle and considerably better manners",
	},
	{
		key: "gate", name: "gate", cat: CatStructure, tags: []string{TagSolid, TagClimbable},
		mass: 3, size: 2, color: "#7d7d7d", shape: ShapeBox, glyph: "GAT",
		note: "a door that has stopped pretending to be friendly",
	},
	{
		key: "lock", name: "lock", cat: CatMachine, tags: []string{TagSolid},
		mass: 2, size: 1, color: "#b8a44a", shape: ShapeBox, glyph: "LCK",
		note:  "the smallest object with the strongest opinions",
		alias: []string{"padlock"},
	},
}
