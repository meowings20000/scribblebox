package ai

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

// PuzzleRequest is what the player asks for when they want a new puzzle.
type PuzzleRequest struct {
	Theme    string
	GoalKind string
}

// GeneratedPuzzle holds the raw spec the model produced. Validation belongs to
// the world package, which is the only thing that can referee a puzzle, so the
// spec travels as JSON and is checked there before it can ever be played.
type GeneratedPuzzle struct {
	Spec  json.RawMessage `json:"spec"`
	Notes string          `json:"notes"`
	Model string          `json:"model"`
}

const puzzleSystemPrompt = `You author puzzles for "Scribblebox", a small Scribblenauts-style game where the player
types object names into a notebook and the objects appear in a 2D grid world. The game's engine handles:
gravity and stacking, floating and sinking in water, freezing water into ice, fire spreading to flammable
objects, cutting ropes, unlocking containers, explosions, batteries powering machines, and climbing.

Terrain codes for the "terrain" array (row-major, length must be width*height):
0 air, 1 ground (solid), 2 water, 3 tree trunk (solid), 4 wall (solid, indestructible), 5 ash (walkable).

Goal kinds the engine can referee — you MUST use the requested one:
- "star": the player must reach a collectible star (spawn a "star" entity).
- "cross": the player must reach the far bank across water (terrain 2) between two banks.
- "candles": every "candle" entity must end up lit (place two or three of them).
- "chest": a "chest" entity (a locked container) must be opened.

Rules for a good puzzle — this game is deliberately EASY, aimed at a primary-school pupil who should
solve it on the first or second try:
- Keep it TINY: exactly 12 wide by 8 tall unless the theme really needs more (the engine accepts 12x8 up
  to 40x20). A tiny world means a short "terrain" array, which you can write accurately and quickly.
- The terrain array must have EXACTLY width*height numbers, row by row, top row first: mostly 0 with 1
  for the ground row at the bottom (and 2 where you want water).
- At most 6 entities: the goal object, whatever scenery the theme needs, and the player.
- Short distances: the player starts within 12 cells of whatever must be reached. Water is at most 4 cells wide.
- The OBVIOUS object must work in one spawn: a ladder/rope/box for a star, a bridge/boat/plank (or freezing the water) for a crossing, a match/torch/lighter for candles, a key or crowbar for a chest.
- Never require two objects in a precise order, never require a precision jump, nothing is deadly and nothing is timed.
- Still allow MANY ways to win, all of them obvious; never bake in a single intended answer and never include an answer key.
- Put ground under everything the player stands on. The player start (playerX, playerY) must be an empty cell above ground, within 12 cells of the goal object.
- At most 6 entities. The engine rejects puzzles that break these limits, so keep them.

Answer with STRICT JSON and nothing else, exactly this shape:
{"id":"a-short-slug","title":"short title","brief":"what the player must do, one line","hint":"a nudge that does not give the answer away","goalKind":"the requested kind","width":12,"height":8,"terrain":[...],"entities":[{"phrase":"ladder","x":3,"y":4}],"playerX":2,"playerY":6}`

// GeneratePuzzle asks the model for a new puzzle spec.
func (c *Client) GeneratePuzzle(ctx context.Context, cfg Config, request PuzzleRequest) (GeneratedPuzzle, error) {
	answer, err := c.Chat(ctx, cfg, puzzleSystemPrompt, puzzleUserPrompt(request), 0.8)
	if err != nil {
		return GeneratedPuzzle{}, err
	}
	raw, err := extractJSON(answer)
	if err != nil {
		return GeneratedPuzzle{}, newError("protocol", 0, "the AI did not send a puzzle we could read: %v", err)
	}
	// Tolerate a wrapper object such as {"spec": {...}, "notes": "..."}.
	var wrapper struct {
		Spec  json.RawMessage `json:"spec"`
		Notes string          `json:"notes"`
	}
	if err := json.Unmarshal([]byte(raw), &wrapper); err == nil && len(wrapper.Spec) > 0 && wrapper.Spec[0] == '{' {
		notes := strings.TrimSpace(wrapper.Notes)
		if notes == "" {
			notes = "the AI described the puzzle without extra notes"
		}
		return GeneratedPuzzle{Spec: wrapper.Spec, Notes: notes, Model: cfg.Model}, nil
	}
	if !json.Valid([]byte(raw)) {
		return GeneratedPuzzle{}, newError("protocol", 0, "the AI's puzzle was not valid JSON")
	}
	return GeneratedPuzzle{Spec: json.RawMessage(raw), Notes: "the AI sent the puzzle spec directly", Model: cfg.Model}, nil
}

// RepairRequest asks the model to fix a spec the engine refused. The engine is the
// one that knows what is wrong, so its exact reason is handed back.
type RepairRequest struct {
	Theme    string
	GoalKind string
	Previous json.RawMessage
	Problem  string
}

const repairSystemPrompt = `You fix puzzle specs for "Scribblebox", a small Scribblenauts-style game. A
puzzle engine rejected the spec you are given and explained exactly why. Correct only what the engine
complained about, keep everything else, and return the whole corrected spec.

Rules the engine checks:
- width 12..40, height 8..20, and the "terrain" array must have EXACTLY width*height entries.
- terrain codes: 0 air, 1 ground, 2 water, 3 tree trunk, 4 wall, 5 ash.
- the four goal kinds and what they need: "star" needs a collectible "star" entity; "candles" needs
  "candle" entities; "chest" needs a "chest" entity; "cross" needs water with a bank on each side.
- the player start must be inside the grid, not inside solid terrain, and within 12 cells of whatever
  must be reached; water is at most 4 cells wide; at most 25 entities.

Answer with STRICT JSON and nothing else: the corrected spec object.`

// RepairPuzzle sends the engine's own complaint back to the model.
func (c *Client) RepairPuzzle(ctx context.Context, cfg Config, request RepairRequest) (GeneratedPuzzle, error) {
	var builder strings.Builder
	fmt.Fprintf(&builder, "Goal kind: %s\nTheme: %s\n", strings.TrimSpace(request.GoalKind), strings.TrimSpace(request.Theme))
	fmt.Fprintf(&builder, "The engine rejected this spec:\n%s\n", string(request.Previous))
	fmt.Fprintf(&builder, "The engine said: %s\n\nReturn the corrected spec as strict JSON only.", request.Problem)

	answer, err := c.Chat(ctx, cfg, repairSystemPrompt, builder.String(), 0.3)
	if err != nil {
		return GeneratedPuzzle{}, err
	}
	raw, err := extractJSON(answer)
	if err != nil {
		return GeneratedPuzzle{}, newError("protocol", 0, "the AI's repair was not JSON we could read: %v", err)
	}
	if !json.Valid([]byte(raw)) {
		return GeneratedPuzzle{}, newError("protocol", 0, "the AI's repaired puzzle was not valid JSON")
	}
	return GeneratedPuzzle{Spec: json.RawMessage(raw), Notes: "the engine asked for a fix and the AI sent a corrected puzzle", Model: cfg.Model}, nil
}

func puzzleUserPrompt(request PuzzleRequest) string {
	kind := strings.TrimSpace(request.GoalKind)
	if kind == "" {
		kind = "star"
	}
	theme := strings.TrimSpace(request.Theme)
	if theme == "" {
		theme = "any theme you like"
	}
	return fmt.Sprintf("Goal kind: %s\nTheme: %s\n\nInvent the puzzle and reply with the strict JSON spec only.", kind, theme)
}

// ErrNoSpec means the model answered but the answer carried no puzzle object.
var ErrNoSpec = errors.New("the AI's answer contained no puzzle spec")
