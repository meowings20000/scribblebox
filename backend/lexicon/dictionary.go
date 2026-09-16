package lexicon

// dictionaryGroups is the authored book, in reading order. Each group lives in
// its own file so the content stays reviewable; init() in lexicon.go flattens
// them into the lookup indexes.
//
// Every row carries a real category, honest tags, a sensible mass and size, a
// hex colour, one of the pinned shapes, a short glyph and one line of flavour.
var dictionaryGroups = [][]entry{
	climbingAndStructures,
	houseAndFurniture,
	toolsAndMaterials,
	animalsAndPlants,
	foodAndDrink,
	vehiclesAndMachines,
	treasureAndMystery,
	peopleAndCreatures,
}
