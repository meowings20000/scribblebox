package ai

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

// Verdict is the model's ruling on a player's idea.
type Verdict struct {
	Approved   bool   `json:"approved"`
	Reason     string `json:"reason"`
	Confidence string `json:"confidence"`
}

// ObjectSummary describes one object the player has created, for the judge.
type ObjectSummary struct {
	Name        string   `json:"name"`
	Key         string   `json:"key"`
	Tags        []string `json:"tags"`
	Approximate bool     `json:"approximate"`
}

// JudgeInput is everything the referee is shown.
type JudgeInput struct {
	PuzzleTitle string
	Brief       string
	GoalKind    string
	Legend      []string
	Objects     []ObjectSummary
	Events      []string
	Question    string
}

const judgeSystemPrompt = `You are the referee inside "Scribblebox", a small Scribblenauts-style puzzle
game. The player types an object's name into a notebook and it appears in the world. Objects have
properties (tags) and the game's engine handles ordinary physics: gravity and stacking, floating and
sinking in water, freezing water into ice, fire spreading to flammable things, cutting ropes, unlocking
containers, explosions, batteries powering machines, and climbing.

The player will describe an idea in their own words and ask whether it counts. This game is meant for
young children, so be a kind, encouraging referee while staying honest:
- Approve when the idea would plausibly work in a cartoon-logic playground, even if the engine cannot
  simulate it. Unexpected but coherent ideas are exactly what this game rewards.
- Deny when it is incoherent, does not address the goal, or contradicts the object's stated properties
  (you cannot climb a puddle; a single candle does not light three candles across a river).
- Never approve merely to be encouraging, and never deny merely because you are surprised.
- Judge only the idea, not the player's spelling or grammar.
- Write the reason in short, simple words a child can read: say what works or what is missing, in one or
  two sentences, without jargon and without a lecture.

Answer with STRICT JSON and nothing else, exactly this shape:
{"approved": true, "reason": "one or two sentences a player would find useful", "confidence": "low|medium|high"}`

// Judge asks the model to referee one idea.
func (c *Client) Judge(ctx context.Context, cfg Config, input JudgeInput) (Verdict, error) {
	answer, err := c.Chat(ctx, cfg, judgeSystemPrompt, judgeUserPrompt(input), 0.2)
	if err != nil {
		return Verdict{}, err
	}
	raw, err := extractJSON(answer)
	if err != nil {
		return Verdict{}, newError("protocol", 0, "the AI did not answer with JSON we could read: %v", err)
	}
	var verdict Verdict
	if err := json.Unmarshal([]byte(raw), &verdict); err != nil {
		return Verdict{}, newError("protocol", 0, "the AI's verdict was not in the expected shape: %v", err)
	}
	verdict.Reason = strings.TrimSpace(verdict.Reason)
	if verdict.Reason == "" {
		return Verdict{}, errors.New("the AI returned a verdict with no reason")
	}
	switch verdict.Confidence {
	case "low", "medium", "high":
	default:
		verdict.Confidence = "medium"
	}
	return verdict, nil
}

func judgeUserPrompt(input JudgeInput) string {
	var builder strings.Builder
	fmt.Fprintf(&builder, "Puzzle: %s\n", input.PuzzleTitle)
	fmt.Fprintf(&builder, "Goal: %s\n", input.Brief)
	if input.GoalKind != "" {
		fmt.Fprintf(&builder, "Goal type: %s\n", input.GoalKind)
	}
	if len(input.Legend) > 0 {
		fmt.Fprintf(&builder, "Property legend: %s\n", strings.Join(input.Legend, ", "))
	}
	if len(input.Objects) > 0 {
		builder.WriteString("Objects the player has created:\n")
		for _, object := range input.Objects {
			fmt.Fprintf(&builder, "- %s (key %s, tags: %s)\n", object.Name, object.Key, strings.Join(object.Tags, ", "))
		}
	} else {
		builder.WriteString("Objects the player has created: none yet\n")
	}
	if len(input.Events) > 0 {
		fmt.Fprintf(&builder, "What just happened in the world: %s\n", strings.Join(input.Events, " | "))
	}
	fmt.Fprintf(&builder, "\nThe player asks: %s\n", input.Question)
	builder.WriteString("\nReply with the strict JSON verdict only.")
	return builder.String()
}

// extractJSON pulls the first complete JSON object out of a model answer that may
// be wrapped in prose or a markdown code fence. Brace matching is string-aware so
// a brace inside a sentence does not end it early.
func extractJSON(answer string) (string, error) {
	trimmed := strings.TrimSpace(answer)
	if trimmed == "" {
		return "", errors.New("the answer was empty")
	}
	start := strings.Index(trimmed, "{")
	if start < 0 {
		return "", errors.New("no JSON object was found in the answer")
	}
	depth := 0
	inString := false
	escaped := false
	for index := start; index < len(trimmed); index++ {
		character := trimmed[index]
		switch {
		case escaped:
			escaped = false
		case character == '\\' && inString:
			escaped = true
		case character == '"':
			inString = !inString
		case inString:
		case character == '{':
			depth++
		case character == '}':
			depth--
			if depth == 0 {
				return trimmed[start : index+1], nil
			}
		}
	}
	return "", errors.New("the JSON object in the answer was never closed")
}
