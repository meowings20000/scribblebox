// Package world is the Scribblebox rule engine: a small, fully deterministic
// grid world where the player types an object's name and it appears, then uses
// it to solve a puzzle.
//
// There is no randomness anywhere in this package and no physics library: every
// rule is a local, countable test evaluated in a fixed order, once per tick, so
// the same world plus the same sequence of calls always produces the same
// result. That is what lets the tests replay an authored answer and require the
// puzzle to fall over.
//
// The package is also the game's referee. It owns the four goal kinds the
// engine can judge honestly (star, cross, candles, chest), validates
// externally authored puzzles down to the same goal kinds, dry-runs hint
// candidates in a sandbox before offering them, and can accept an outside
// judge's verdict when a player's idea is real but unsimulated.
package world

import "backend/lexicon"

// Terrain codes. They are pinned: the HTTP layer and the frontend both read
// State.Terrain as a flat row-major slice of these numbers.
//
//	0 air    - empty, walkable, fall-through
//	1 ground - solid, walkable on top, stops falling
//	2 water  - swimmable, freezable, slows the player
//	3 trunk  - a tree trunk: solid like ground, but not climbable by itself
//	4 wall   - solid and unbreakable, survives explosions untouched
//	5 ash    - what is left after a fire: walkable and empty, like air
const (
	TerrainAir    = 0
	TerrainGround = 1
	TerrainWater  = 2
	TerrainTrunk  = 3
	TerrainWall   = 4
	TerrainAsh    = 5
)

// Airborne terrain codes: nothing here holds an entity up or blocks movement.
// Ash is deliberately in this set: a burnt thing leaves a walkable floor.
func isSolid(code int) bool {
	return code == TerrainGround || code == TerrainTrunk || code == TerrainWall
}

// isWalkable reports whether the player may stand in this terrain.
func isWalkable(code int) bool { return !isSolid(code) }

// Goal kinds the engine can referee. An externally authored puzzle is only
// accepted if it uses one of these, which is what keeps every puzzle winnable
// and checkable by the same code path as the built-in four.
const (
	GoalStar    = "star"
	GoalCross   = "cross"
	GoalCandles = "candles"
	GoalChest   = "chest"
)

// GoalKinds lists the four refereeable kinds in a stable order, for validation
// messages and the HTTP layer's error text.
var GoalKinds = []string{GoalStar, GoalCross, GoalCandles, GoalChest}

// Puzzle is a playable scenario. Solutions is the answer key: it is tagged
// json:"-" and must never reach a client, which the tests assert against the
// serialised JSON of both Puzzles() and Snapshot().
type Puzzle struct {
	ID        string     `json:"id"`
	Title     string     `json:"title"`
	Brief     string     `json:"brief"`
	Hint      string     `json:"hint"`
	GoalKind  string     `json:"goalKind"`
	Width     int        `json:"width"`
	Height    int        `json:"height"`
	Solutions []Solution `json:"-"` // NEVER serialised: these are answers
}

// Solution is one authored way to solve a puzzle: an order-free list of objects
// the player spawns. Every solution in this package is verified by the test
// suite by actually replaying it and requiring the goal to be met.
type Solution struct {
	Description string      `json:"description"`
	Phrases     []Placement `json:"phrases"`
}

// Placement is one spawned phrase and where it goes.
type Placement struct {
	Phrase string `json:"phrase"`
	X      int    `json:"x"`
	Y      int    `json:"y"`
}

// Entity is one spawned object living in the world.
type Entity struct {
	ID      int            `json:"id"`
	Object  lexicon.Object `json:"object"`
	X       int            `json:"x"`
	Y       int            `json:"y"`
	W       int            `json:"w"`
	H       int            `json:"h"`
	Burning bool           `json:"burning"`
	Opened  bool           `json:"opened"`
	Cut     bool           `json:"cut"`
	Powered bool           `json:"powered"`
	Held    bool           `json:"held"`
}

// Player is the avatar. OnGround is recomputed every tick and after every move.
type Player struct {
	X        int  `json:"x"`
	Y        int  `json:"y"`
	OnGround bool `json:"onGround"`
	Holding  int  `json:"holding"` // entity id, 0 = empty hands
}

// Goal is the human-readable progress of the puzzle's win condition.
type Goal struct {
	Title    string `json:"title"`
	Met      bool   `json:"met"`
	Progress string `json:"progress"`
}

// NotebookEntry remembers every phrase the player has typed, in order, so the
// frontend can show the notebook and the AI layer can describe what was tried.
type NotebookEntry struct {
	Phrase      string `json:"phrase"`
	Key         string `json:"key"`
	Approximate bool   `json:"approximate"`
	X           int    `json:"x"`
	Y           int    `json:"y"`
}

// State is the complete wire form of a world. Solutions are absent by
// construction: the puzzle is copied without its answer key.
type State struct {
	ID        string          `json:"id"`
	Puzzle    Puzzle          `json:"puzzle"`
	Width     int             `json:"width"`
	Height    int             `json:"height"`
	Terrain   []int           `json:"terrain"`
	Entities  []Entity        `json:"entities"`
	Player    Player          `json:"player"`
	Solved    bool            `json:"solved"`
	Judged    bool            `json:"judged"`
	JudgeNote string          `json:"judgeNote"`
	Ticks     int             `json:"ticks"`
	Events    []string        `json:"events"`
	Notebook  []NotebookEntry `json:"notebook"`
	Goal      Goal            `json:"goal"`
}

// PuzzleSpec is an externally authored puzzle (e.g. generated by an AI) reduced
// to the pieces the engine can referee. Terrain uses the same codes as
// State.Terrain, row-major, len(Terrain) == Width*Height.
type PuzzleSpec struct {
	ID       string       `json:"id"`
	Title    string       `json:"title"`
	Brief    string       `json:"brief"`
	Hint     string       `json:"hint"`
	GoalKind string       `json:"goalKind"`
	Width    int          `json:"width"`
	Height   int          `json:"height"`
	Terrain  []int        `json:"terrain"`
	Entities []SpecEntity `json:"entities"`
	PlayerX  int          `json:"playerX"`
	PlayerY  int          `json:"playerY"`
}

// SpecEntity is one object an external spec asks to place.
type SpecEntity struct {
	Phrase string `json:"phrase"`
	X      int    `json:"x"`
	Y      int    `json:"y"`
}

// HintPlacement is where a hint object is meant to be spawned. The frontend
// spawns a chosen hint option here, so the "one of these works" promise is true
// exactly where the player will put it.
type HintPlacement struct {
	X int `json:"x"`
	Y int `json:"y"`
}

// HintSet is a four-choice hint: Option 0..3, exactly one of which solves the
// puzzle when spawned at Placement. The correct option is NOT marked, so the
// answer never travels to the client.
type HintSet struct {
	Placement HintPlacement `json:"placement"`
	Options   []string      `json:"options"`   // exactly four phrases
	Rationale string        `json:"rationale"` // one friendly line, no answer
}
