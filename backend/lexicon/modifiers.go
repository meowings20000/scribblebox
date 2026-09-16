package lexicon

// modifier is one authored adjective. It changes a property rather than
// describing one: size and mass shift, tags are added or removed, and some
// materials and colours repaint the object.
//
// Deltas are summed across every adjective in the phrase and then clamped, so
// "big giant tiny box" is legal and lands inside 1..3 / 1..5 rather than
// exploding the physics.
type modifier struct {
	word       string   // the authored adjective, as it appears in Modifiers
	addTags    []string // tags the adjective grants
	removeTags []string // tags the adjective takes away
	sizeDelta  int      // added to Size
	massDelta  int      // added to Mass
	color      string   // overrides Color when set
}

// modifierTable is the authored adjective list. Every entry must change at
// least one field for at least one plausible noun; TestModifierCoverage proves
// that against the real dictionary.
var modifierTable = []modifier{
	// Size and weight.
	{word: "big", sizeDelta: 1},
	{word: "huge", sizeDelta: 2},
	{word: "giant", sizeDelta: 2},
	{word: "little", sizeDelta: -1},
	{word: "small", sizeDelta: -1, massDelta: -1},
	{word: "tiny", sizeDelta: -2, massDelta: -1},
	{word: "heavy", massDelta: 2, addTags: []string{TagHeavy}},
	{word: "light", massDelta: -2},

	// Materials.
	{word: "metal", addTags: []string{TagConductive, TagHeavy}, removeTags: []string{TagFlammable}, massDelta: 1, color: "#9aa4ad"},
	{word: "steel", addTags: []string{TagConductive, TagHeavy}, removeTags: []string{TagFlammable}, massDelta: 1, color: "#b9c2cb"},
	{word: "iron", addTags: []string{TagConductive, TagHeavy}, removeTags: []string{TagFlammable}, massDelta: 1, color: "#6f7378"},
	{word: "wooden", addTags: []string{TagFlammable}, removeTags: []string{TagConductive}, color: "#8b5a2b"},
	{word: "glass", addTags: []string{TagFragile}, removeTags: []string{TagFlammable}, color: "#bfe3ee"},
	{word: "stone", addTags: []string{TagHeavy}, removeTags: []string{TagFlammable, TagConductive}, massDelta: 2, color: "#8d8d8d"},
	{word: "plastic", removeTags: []string{TagConductive}, color: "#e6e6e6"},
	{word: "paper", addTags: []string{TagFlammable}, removeTags: []string{TagConductive}, color: "#f2ecdc"},

	// Fire, cold and weather.
	{word: "flaming", addTags: []string{TagFire, TagLightSource, TagFlammable}, color: "#ff6a00"},
	{word: "burning", addTags: []string{TagFire, TagLightSource, TagFlammable}, color: "#ff3d00"},
	{word: "hot", addTags: []string{TagFire, TagLightSource}, color: "#ff7b3d"},
	{word: "frozen", addTags: []string{TagFrozen, TagCold}, removeTags: []string{TagFire}, color: "#bfe9ff"},
	{word: "icy", addTags: []string{TagFrozen, TagCold}, removeTags: []string{TagFire}, color: "#cdeffd"},
	{word: "cold", addTags: []string{TagCold}, removeTags: []string{TagFire}, color: "#a9d8ef"},
	{word: "wet", removeTags: []string{TagFlammable}, color: "#7fb3d5"},
	{word: "dry", addTags: []string{TagFlammable}},

	// Condition.
	{word: "sticky", addTags: []string{TagSticky}},
	{word: "sharp", addTags: []string{TagSharp, TagCutting}},
	{word: "blunt", removeTags: []string{TagSharp, TagCutting}},
	{word: "magical", addTags: []string{TagMagical}, color: "#c77dff"},
	{word: "invisible", addTags: []string{TagMagical}, color: "#eaf2ff"},
	{word: "floating", addTags: []string{TagBuoyant}, massDelta: -1},
	{word: "wind-up", addTags: []string{TagMachine}},
	{word: "broken", removeTags: []string{TagPlatform, TagClimbable}},
	{word: "rusted", removeTags: []string{TagSharp}, color: "#7a4a2a"},
	{word: "rusty", color: "#a0522d"},
	{word: "old", addTags: []string{TagFragile}, color: "#cbb994"},
	{word: "new", color: "#fdfdfd"},

	// Value and shine.
	{word: "golden", addTags: []string{TagTreasure}, color: "#ffd23f"},
	{word: "silver", addTags: []string{TagTreasure}, color: "#c9ccd1"},
	{word: "shiny", addTags: []string{TagTreasure}, color: "#f6f7f9"},
	{word: "sparkly", addTags: []string{TagTreasure}, color: "#ffe066"},
	{word: "glowing", addTags: []string{TagLightSource}, color: "#fff59d"},

	// Power.
	{word: "electric", addTags: []string{TagConductive, TagPowerSource}, color: "#7ef9ff"},

	// Colours.
	{word: "red", color: "#e63946"},
	{word: "blue", color: "#3b82f6"},
	{word: "green", color: "#22c55e"},
	{word: "yellow", color: "#facc15"},
	{word: "black", color: "#1f2937"},
	{word: "white", color: "#f8fafc"},
	{word: "purple", color: "#a855f7"},
	{word: "pink", color: "#ec4899"},
	{word: "orange", color: "#f97316"},
}

// modifierIndex maps the normalised adjective to its row. Normalising means
// "Wind-Up", "wind-up" and "wind up" all reach the same adjective.
var modifierIndex = map[string]modifier{}

func init() {
	for _, m := range modifierTable {
		modifierIndex[normalizeWord(m.word)] = m
	}
}

// canonicalModifier returns the authored spelling of an adjective, or "" when
// the token is not an adjective at all.
func canonicalModifier(token string) string {
	m, ok := modifierIndex[normalizeWord(token)]
	if !ok {
		return ""
	}
	return m.word
}

// withModifiers merges every adjective into a copy of base. Base tags are never
// replaced, tag order is base-first, duplicates are impossible, and both size
// and mass land clamped inside their bounds.
func withModifiers(base Object, mods []string) Object {
	obj := cloneObject(base)
	if len(mods) == 0 {
		return obj
	}

	tags := append([]string{}, obj.Tags...)
	size, mass := obj.Size, obj.Mass
	for _, raw := range mods {
		m, ok := modifierIndex[normalizeWord(raw)]
		if !ok {
			continue
		}
		for _, drop := range m.removeTags {
			tags = removeTag(tags, drop)
		}
		for _, add := range m.addTags {
			tags = appendTag(tags, add)
		}
		size += m.sizeDelta
		mass += m.massDelta
		if m.color != "" {
			obj.Color = m.color
		}
	}

	obj.Tags = dedupeTags(tags)
	obj.Size = clamp(size, MinSize, MaxSize)
	obj.Mass = clamp(mass, MinMass, MaxMass)
	return obj
}
