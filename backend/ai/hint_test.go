package ai

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"
)

func TestSuggestHintParsesChoices(t *testing.T) {
	answer := "```json\n{\"options\": [\"ladder\", \" cake \", \"ladder\", \"pillow\", \"rubber duck\", 7, \"\"], \"rationale\": \"One of these will do it.\"}\n```"
	fake := newFakeEndpoint(t, http.StatusOK, completion(answer))

	options, rationale, err := NewClient().SuggestHint(context.Background(), testConfig(fake.URL), HintInput{
		PuzzleTitle: "The star in the tree",
		Brief:       "Get the star out of the tree.",
		GoalKind:    "star",
	})
	if err != nil {
		t.Fatalf("SuggestHint = %v", err)
	}
	want := []string{"ladder", "cake", "pillow", "rubber duck"}
	if len(options) != len(want) {
		t.Fatalf("options = %v, want %v", options, want)
	}
	for index := range want {
		if options[index] != want[index] {
			t.Fatalf("options = %v, want %v", options, want)
		}
	}
	if rationale != "One of these will do it." {
		t.Fatalf("rationale = %q", rationale)
	}

	system := fake.requests[0].Messages[0].Content
	if !strings.Contains(system, "STRICT JSON") || !strings.Contains(system, "EASY") {
		t.Fatalf("the system prompt should ask for easy, strict JSON: %q", system)
	}
	user := fake.requests[0].Messages[1].Content
	if !strings.Contains(user, "The star in the tree") || !strings.Contains(user, "star") {
		t.Fatalf("user prompt = %q", user)
	}
}

func TestSuggestHintRejectsUnusableAnswers(t *testing.T) {
	cases := map[string]string{
		"prose":        "Try a ladder, a cake, a pillow and a duck.",
		"empty list":   `{"options": [], "rationale": "sorry"}`,
		"blank words":  `{"options": ["   ", ""], "rationale": "sorry"}`,
		"wrong shape":  `{"options": "ladder", "rationale": "sorry"}`,
		"paragraph":    `{"options": ["` + strings.Repeat("a", 40) + `"], "rationale": "too long to be a button"}`,
		"never closed": `{"options": ["ladder"]`,
	}
	for name, answer := range cases {
		t.Run(name, func(t *testing.T) {
			fake := newFakeEndpoint(t, http.StatusOK, completion(answer))
			if _, _, err := NewClient().SuggestHint(context.Background(), testConfig(fake.URL), HintInput{PuzzleTitle: "x"}); err == nil {
				t.Fatal("expected an error")
			}
		})
	}

	// A very long list is trimmed rather than rejected: the engine only needs a few.
	long := map[string]any{"options": []string{"a", "b", "c", "d", "e", "f", "g", "h", "i", "j", "k", "l"}, "rationale": "plenty"}
	encoded, _ := json.Marshal(long)
	fake := newFakeEndpoint(t, http.StatusOK, completion(string(encoded)))
	options, _, err := NewClient().SuggestHint(context.Background(), testConfig(fake.URL), HintInput{PuzzleTitle: "x"})
	if err != nil {
		t.Fatalf("SuggestHint = %v", err)
	}
	if len(options) != maxHintOptions {
		t.Fatalf("options = %d, want %d", len(options), maxHintOptions)
	}
}

func TestSuggestHintSurfacesEndpointFailures(t *testing.T) {
	fake := newFakeEndpoint(t, http.StatusUnauthorized, `{"error":{"message":"invalid api key"}}`)
	if _, _, err := NewClient().SuggestHint(context.Background(), testConfig(fake.URL), HintInput{PuzzleTitle: "x"}); err == nil {
		t.Fatal("expected the auth failure to be surfaced")
	} else if !strings.Contains(err.Error(), "refused the key") {
		t.Fatalf("error = %v", err)
	}
}
