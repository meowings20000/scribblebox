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
- Small and readable: width between 12 and 40, height between 8 and 20, at most 25 entities.
- Short distances: the player starts within 12 cells of whatever must be reached. Water is at most 4 cells wide.
- The OBVIOUS object must work in one spawn: a ladder/rope/box for a star, a bridge/boat/plank (or freezing the water) for a crossing, a match/torch/lighter for candles, a key or crowbar for a chest.
- Never require two objects in a precise order, never require a precision jump, nothing is deadly and nothing is timed.
- Still allow MANY ways to win, all of them obvious; never bake in a single intended answer and never include an answer key.
- Put ground under everything the player stands on. The player start (playerX, playerY) must be an empty cell above ground, within 12 cells of the goal object. Entities must be inside the grid and not inside solid terrain.
- The engine rejects puzzles that break these limits, so keep them: a puzzle that is too hard, too big or too far will be refused.

Answer with STRICT JSON and nothing else, exactly this shape:
{"id":"a-short-slug","title":"short title","brief":"what the player must do, one line","hint":"a nudge that does not give the answer away","goalKind":"the requested kind","width":28,"height":16,"terrain":[...],"entities":[{"phrase":"ladder","x":3,"y":4}],"playerX":2,"playerY":9}`

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
