package lexicon

// vehiclesAndMachines: things with wheels, rotors or engines.
var vehiclesAndMachines = []entry{
	{
		key: "car", name: "car", cat: CatVehicle, tags: []string{TagWheeled, TagMachine, TagSolid, TagContainer},
		mass: 4, size: 2, color: "#d94f4f", shape: ShapeVehicle, glyph: "CAR",
		note:  "freedom, four wheels, and a parking problem",
		alias: []string{"automobile"},
	},
	{
		key: "bicycle", name: "bicycle", cat: CatVehicle, tags: []string{TagWheeled, TagFragile},
		mass: 2, size: 2, color: "#4a8ad9", shape: ShapeVehicle, glyph: "BIK",
		note:  "two wheels powered entirely by optimism",
		alias: []string{"bike", "cycle"},
	},
	{
		key: "skateboard", name: "skateboard", cat: CatVehicle, tags: []string{TagWheeled, TagPlatform},
		mass: 1, size: 1, color: "#c9934a", shape: ShapePlank, glyph: "SKB",
		note: "gravity, but strictly on your own terms, briefly",
	},
	{
		key: "truck", name: "truck", cat: CatVehicle, tags: []string{TagWheeled, TagMachine, TagHeavy, TagContainer},
		mass: 5, size: 3, color: "#4a6fa5", shape: ShapeVehicle, glyph: "TRK",
		note:  "carries everything you forgot to plan for",
		alias: []string{"lorry"},
	},
	{
		key: "bus", name: "bus", cat: CatVehicle, tags: []string{TagWheeled, TagMachine, TagHeavy, TagContainer},
		mass: 5, size: 3, color: "#e0a43a", shape: ShapeVehicle, glyph: "BUS",
		note: "late, full, and somehow always the answer",
	},
	{
		key: "train", name: "train", cat: CatVehicle, tags: []string{TagWheeled, TagMachine, TagHeavy},
		mass: 5, size: 3, color: "#8a8f98", shape: ShapeVehicle, glyph: "TRN",
		note: "a bus that made a long-term commitment to a pair of rails",
	},
	{
		key: "motorcycle", name: "motorcycle", cat: CatVehicle, tags: []string{TagWheeled, TagMachine},
		mass: 3, size: 2, color: "#2b2f36", shape: ShapeVehicle, glyph: "MTC",
		note:  "speed, noise, and a complete disregard for weather",
		alias: []string{"motorbike"},
	},
	{
		key: "tractor", name: "tractor", cat: CatVehicle, tags: []string{TagWheeled, TagMachine, TagHeavy},
		mass: 4, size: 3, color: "#4a8a4a", shape: ShapeVehicle, glyph: "TRC",
		note: "slow, unstoppable, and better at mud than you are",
	},
	{
		key: "tank", name: "tank", cat: CatVehicle, tags: []string{TagWheeled, TagMachine, TagHeavy, TagSolid, TagWeapon, TagExplosive},
		mass: 5, size: 3, color: "#5a6b4a", shape: ShapeVehicle, glyph: "TNK",
		note: "a car that finally resolved its trust issues",
	},
	{
		key: "helicopter", name: "helicopter", cat: CatVehicle, tags: []string{TagLift, TagMachine, TagFragile},
		mass: 4, size: 3, color: "#4a6fa5", shape: ShapeVehicle, glyph: "HEL",
		note:  "a machine that argues with gravity and mostly wins",
		alias: []string{"chopper"},
	},
	{
		key: "plane", name: "plane", cat: CatVehicle, tags: []string{TagLift, TagMachine, TagFragile},
		mass: 5, size: 3, color: "#e8eef4", shape: ShapeVehicle, glyph: "PLN",
		note:  "a bus that got ambitious about the ceiling",
		alias: []string{"airplane", "aeroplane", "aircraft"},
	},
	{
		key: "submarine", name: "submarine", cat: CatVehicle, tags: []string{TagMachine, TagWater, TagContainer, TagSolid},
		mass: 5, size: 3, color: "#e0c04a", shape: ShapeVehicle, glyph: "SUB",
		note: "a boat with commitment issues about the surface",
	},
	{
		key: "airship", name: "hot air balloon", cat: CatVehicle, tags: []string{TagBuoyant, TagFragile, TagPlatform},
		mass: 3, size: 3, color: "#e05a8c", shape: ShapeCircle, glyph: "HAB",
		note:  "quiet, gentle, and entirely at the mercy of the wind",
		alias: []string{"hot air balloon", "hotairballoon"},
	},
	{
		key: "parachute", name: "parachute", cat: CatTool, tags: []string{TagBuoyant, TagFragile},
		mass: 1, size: 2, color: "#f0c9c9", shape: ShapeFlag, glyph: "PAR",
		note:  "a soft argument against the ground",
		alias: []string{"chute"},
	},
	{
		key: "glider", name: "glider", cat: CatVehicle, tags: []string{TagLift, TagFragile},
		mass: 3, size: 3, color: "#a8d8f0", shape: ShapeVehicle, glyph: "GLI",
		note: "silent flight and a very firm belief in updrafts",
	},
	{
		key: "crane", name: "crane", cat: CatMachine, tags: []string{TagMachine, TagHeavy, TagLift, TagWheeled},
		mass: 5, size: 3, color: "#e0a43a", shape: ShapeMachine, glyph: "CRN",
		note: "picks things up and has no interest whatsoever in letting go",
	},
	{
		key: "forklift", name: "forklift", cat: CatMachine, tags: []string{TagMachine, TagLift, TagWheeled},
		mass: 4, size: 3, color: "#e0a43a", shape: ShapeMachine, glyph: "FKL",
		note:  "lifts precisely one thing at a time, and about that it is religious",
		alias: []string{"fork lift"},
	},
	{
		key: "rocket", name: "rocket", cat: CatVehicle, tags: []string{TagExplosive, TagFire, TagLift},
		mass: 4, size: 2, color: "#e8eef4", shape: ShapeVehicle, glyph: "RKT",
		note:  "goes up, and the ceiling has some opinions about it",
		alias: []string{"rocket ship", "spaceship"},
	},
	{
		key: "robot", name: "robot", cat: CatMachine, tags: []string{TagMachine, TagConductive, TagTool},
		mass: 4, size: 2, color: "#9aa4ad", shape: ShapeMachine, glyph: "RBT",
		note:  "helpful, extremely literal, and out of warranty",
		alias: []string{"android"},
	},
	{
		key: "bubble", name: "bubble", cat: CatMystery, tags: []string{TagBuoyant, TagFragile},
		mass: 1, size: 1, color: "#bfe9ff", shape: ShapeCircle, glyph: "BUB",
		note:  "a sphere with a very short career plan",
		alias: []string{"bubbles"},
	},
	{
		key: "springboard", name: "springboard", cat: CatStructure, tags: []string{TagPlatform, TagTool, TagFragile},
		mass: 2, size: 2, color: "#4a6fa5", shape: ShapePlank, glyph: "SPG",
		note:  "a diving board with a very direct sense of humour",
		alias: []string{"diving board"},
	},
}
