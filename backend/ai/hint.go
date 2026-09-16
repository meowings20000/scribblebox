package ai

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

// HintInput is what the model is told when asked for four choices.
type HintInput struct {
	PuzzleTitle string
	Brief       string
	GoalKind    string
	Objects     []ObjectSummary
}

// SuggestedHint is the model's proposal. Nothing here is trusted: the engine
// simulates every option afterwards and keeps exactly one that really works.
// Options arrive raw so that a stray number or object in the array can be
// skipped instead of failing the whole answer.
type SuggestedHint struct {
	Options   []json.RawMessage `json:"options"`
	Rationale string            `json:"rationale"`
}

const hintSystemPrompt = `You suggest four object choices for a hint inside "Scribblebox", a small
Scribblenauts-style puzzle game for young children. The player types an object's name into a notebook
and it appears in the world; objects have properties and the engine handles gravity, stacking, water
and ice, fire, cutting, unlocking, explosions, batteries and climbing.

The game is deliberately EASY: the obvious object should be one of your four, and the other three should
be clearly wrong but funny and child-friendly (a cake, a pillow, a rubber duck). Use one or two words
each, ordinary nouns a child would type, never a sentence, never an explanation, never a made-up word.

Answer with STRICT JSON and nothing else, exactly this shape:
{"options": ["ladder", "cake", "pillow", "umbrella"], "rationale": "one short friendly line that does not give the answer away"}`

// SuggestHint asks the model for four options and a friendly line.
func (c *Client) SuggestHint(ctx context.Context, cfg Config, input HintInput) ([]string, string, error) {
	answer, err := c.Chat(ctx, cfg, hintSystemPrompt, hintUserPrompt(input), 0.7)
	if err != nil {
		return nil, "", err
	}
	raw, err := extractJSON(answer)
	if err != nil {
		return nil, "", newError("protocol", 0, "the AI did not send choices we could read: %v", err)
	}
	var suggested SuggestedHint
	if err := json.Unmarshal([]byte(raw), &suggested); err != nil {
		return nil, "", newError("protocol", 0, "the AI's choices were not in the expected shape: %v", err)
	}

	options := []string{}
	seen := map[string]bool{}
	for _, raw := range suggested.Options {
		var option string
		if err := json.Unmarshal(raw, &option); err != nil {
			continue // a number or an object is not an object name
		}
		trimmed := strings.TrimSpace(option)
		if trimmed == "" || len([]rune(trimmed)) > maxHintWordRunes {
			continue
		}
		lowered := strings.ToLower(trimmed)
		if seen[lowered] {
			continue
		}
		seen[lowered] = true
		options = append(options, trimmed)
	}
	if len(options) == 0 {
		return nil, "", errors.New("the AI sent no usable object names")
	}
	if len(options) > maxHintOptions {
		options = options[:maxHintOptions]
	}
	return options, strings.TrimSpace(suggested.Rationale), nil
}

const (
	// maxHintWordRunes keeps an answer from smuggling a paragraph into a button.
	maxHintWordRunes = 30
	// maxHintOptions bounds how many candidates the engine has to simulate.
	maxHintOptions = 8
)

func hintUserPrompt(input HintInput) string {
	var builder strings.Builder
	fmt.Fprintf(&builder, "Puzzle: %s\n", input.PuzzleTitle)
	fmt.Fprintf(&builder, "Goal: %s\n", input.Brief)
	if input.GoalKind != "" {
		fmt.Fprintf(&builder, "Goal type: %s\n", input.GoalKind)
	}
	if len(input.Objects) > 0 {
		names := make([]string, 0, len(input.Objects))
		for index, object := range input.Objects {
			if index == 12 {
				break
			}
			names = append(names, object.Name)
		}
		fmt.Fprintf(&builder, "The player has already made: %s\n", strings.Join(names, ", "))
	}
	builder.WriteString("\nSuggest four object choices: one that would really work, and three that clearly would not. Reply with the strict JSON only.")
	return builder.String()
}
