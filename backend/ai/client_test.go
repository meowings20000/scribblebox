package ai

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

const testKey = "sk-test-key-do-not-log-me"

// fakeEndpoint stands in for an OpenAI-compatible server. It records what the
// client actually sent so tests can assert on the wire format.
type fakeEndpoint struct {
	*httptest.Server
	requests []chatRequest
	headers  []http.Header
	paths    []string
}

func newFakeEndpoint(t *testing.T, status int, body string) *fakeEndpoint {
	t.Helper()
	fake := &fakeEndpoint{}
	fake.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var request chatRequest
		_ = json.NewDecoder(r.Body).Decode(&request)
		fake.requests = append(fake.requests, request)
		fake.headers = append(fake.headers, r.Header.Clone())
		fake.paths = append(fake.paths, r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(fake.Close)
	return fake
}

func completion(content string) string {
	encoded, _ := json.Marshal(content)
	return `{"model":"fake-model","choices":[{"message":{"role":"assistant","content":` + string(encoded) + `}}]}`
}

func testConfig(endpoint string) Config {
	return Config{BaseURL: endpoint + "/v1", APIKey: testKey, Model: "fake-model"}
}

func TestConfigValidateAndEndpoint(t *testing.T) {
	cases := []struct {
		name    string
		config  Config
		wantErr string
		wantURL string
	}{
		{"bare base", Config{BaseURL: "http://127.0.0.1:3000/v1/", APIKey: testKey}, "", "http://127.0.0.1:3000/v1/chat/completions"},
		{"full url kept", Config{BaseURL: "https://api.example.com/v1/chat/completions", APIKey: testKey}, "", "https://api.example.com/v1/chat/completions"},
		{"no base", Config{APIKey: testKey}, "no AI base URL", ""},
		{"no key", Config{BaseURL: "http://x/v1"}, "no AI key", ""},
		{"bad scheme", Config{BaseURL: "ftp://x/v1", APIKey: testKey}, "must start with http", ""},
		{"no host", Config{BaseURL: "http:///v1", APIKey: testKey}, "needs a host", ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			config := tc.config
			err := config.Validate()
			if tc.wantErr == "" {
				if err != nil {
					t.Fatalf("Validate = %v", err)
				}
				if got := config.Endpoint(); got != tc.wantURL {
					t.Fatalf("Endpoint = %q, want %q", got, tc.wantURL)
				}
				if config.Model == "" {
					t.Fatal("model should be defaulted")
				}
				if config.TimeoutSeconds <= 0 || config.TimeoutSeconds > int(MaxTimeout/time.Second) {
					t.Fatalf("timeout = %d", config.TimeoutSeconds)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("Validate = %v, want an error containing %q", err, tc.wantErr)
			}
		})
	}
}

func TestChatSendsAProperRequestAndReturnsContent(t *testing.T) {
	fake := newFakeEndpoint(t, http.StatusOK, completion("hello from the model"))
	client := NewClient()

	answer, err := client.Chat(context.Background(), testConfig(fake.URL), "system prompt", "user prompt", 0.2)
	if err != nil {
		t.Fatalf("Chat = %v", err)
	}
	if answer != "hello from the model" {
		t.Fatalf("answer = %q", answer)
	}
	if len(fake.requests) != 1 {
		t.Fatalf("requests = %d", len(fake.requests))
	}
	request := fake.requests[0]
	if request.Model != "fake-model" || len(request.Messages) != 2 || request.Messages[0].Role != "system" || request.Messages[1].Role != "user" {
		t.Fatalf("request = %+v", request)
	}
	if fake.paths[0] != "/v1/chat/completions" {
		t.Fatalf("path = %q", fake.paths[0])
	}
	if got := fake.headers[0].Get("Authorization"); got != "Bearer "+testKey {
		t.Fatalf("Authorization = %q", got)
	}
	if got := fake.headers[0].Get("Content-Type"); got != "application/json" {
		t.Fatalf("Content-Type = %q", got)
	}
	if got := fake.headers[0].Get("User-Agent"); got == "" {
		t.Fatal("a User-Agent should be sent")
	}
}

func TestChatClassifiesFailuresHonestly(t *testing.T) {
	cases := []struct {
		name     string
		status   int
		body     string
		wantKind string
		wantIn   string
	}{
		{"bad key", http.StatusUnauthorized, `{"error":{"message":"invalid api key","type":"auth"}}`, "auth", "refused the key"},
		{"forbidden", http.StatusForbidden, `{"error":{"message":"no access to this model"}}`, "auth", "refused the key"},
		{"server error", http.StatusInternalServerError, `{"error":{"message":"upstream exploded"}}`, "server", "upstream exploded"},
		{"rate limited", http.StatusTooManyRequests, `{"error":{"message":"slow down"}}`, "server", "slow down"},
		{"not json", http.StatusOK, `<html>nope</html>`, "protocol", "did not return JSON"},
		{"no choices", http.StatusOK, `{"choices":[]}`, "protocol", "no choices"},
		{"empty answer", http.StatusOK, `{"choices":[{"message":{"content":"   "}}]}`, "protocol", "empty answer"},
		{"error field", http.StatusOK, `{"error":{"message":"model overloaded"}}`, "server", "model overloaded"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			fake := newFakeEndpoint(t, tc.status, tc.body)
			_, err := NewClient().Chat(context.Background(), testConfig(fake.URL), "s", "u", 0.2)
			if err == nil {
				t.Fatal("expected an error")
			}
			var apiError *Error
			if !errors.As(err, &apiError) {
				t.Fatalf("error = %T (%v)", err, err)
			}
			if apiError.Kind != tc.wantKind {
				t.Fatalf("kind = %q, want %q (%v)", apiError.Kind, tc.wantKind, err)
			}
			if !strings.Contains(err.Error(), tc.wantIn) {
				t.Fatalf("error %q should mention %q", err.Error(), tc.wantIn)
			}
			// The key must never leak into an error a user could see.
			if strings.Contains(err.Error(), testKey) {
				t.Fatalf("the key leaked into an error: %v", err)
			}
		})
	}
}

func TestChatTimesOutAndSaysSo(t *testing.T) {
	slow := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(3 * time.Second)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(completion("too late")))
	}))
	defer slow.Close()

	config := testConfig(slow.URL)
	config.TimeoutSeconds = 1
	start := time.Now()
	_, err := NewClient().Chat(context.Background(), config, "s", "u", 0)
	if err == nil {
		t.Fatal("expected a timeout")
	}
	var apiError *Error
	if !errors.As(err, &apiError) || apiError.Kind != "timeout" {
		t.Fatalf("error = %v", err)
	}
	if elapsed := time.Since(start); elapsed > 3*time.Second {
		t.Fatalf("the timeout took too long: %v", elapsed)
	}
}

func TestChatRejectsAnOversizedResponse(t *testing.T) {
	big := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"` + strings.Repeat("a", maxResponseBytes+10) + `"}}]}`))
	}))
	defer big.Close()

	_, err := NewClient().Chat(context.Background(), testConfig(big.URL), "s", "u", 0)
	var apiError *Error
	if !errors.As(err, &apiError) || apiError.Kind != "protocol" {
		t.Fatalf("error = %v", err)
	}
	if !strings.Contains(err.Error(), "larger than") {
		t.Fatalf("error = %v", err)
	}
}

func TestChatReportsAnUnreachableEndpoint(t *testing.T) {
	// A port nothing listens on.
	_, err := NewClient().Chat(context.Background(), Config{BaseURL: "http://127.0.0.1:1/v1", APIKey: testKey}, "s", "u", 0)
	var apiError *Error
	if !errors.As(err, &apiError) || apiError.Kind != "unreachable" {
		t.Fatalf("error = %v", err)
	}
}

func TestJudgeParsesVerdicts(t *testing.T) {
	answer := "Here you go:\n```json\n{\"approved\": true, \"reason\": \"A ladder reaches the branch, so the star is yours.\", \"confidence\": \"high\"}\n```"
	fake := newFakeEndpoint(t, http.StatusOK, completion(answer))

	verdict, err := NewClient().Judge(context.Background(), testConfig(fake.URL), JudgeInput{
		PuzzleTitle: "The star in the tree",
		Brief:       "Get the star out of the tree.",
		GoalKind:    "star",
		Objects:     []ObjectSummary{{Name: "ladder", Key: "ladder", Tags: []string{"climbable"}}},
		Events:      []string{"a ladder appears"},
		Question:    "Does the ladder reach?",
	})
	if err != nil {
		t.Fatalf("Judge = %v", err)
	}
	if !verdict.Approved || verdict.Confidence != "high" || !strings.Contains(verdict.Reason, "ladder") {
		t.Fatalf("verdict = %+v", verdict)
	}

	// The system prompt must describe the game and demand JSON.
	sent := fake.requests[0].Messages[0].Content
	if !strings.Contains(sent, "STRICT JSON") || !strings.Contains(sent, "Scribblebox") {
		t.Fatalf("system prompt = %q", sent)
	}
	user := fake.requests[0].Messages[1].Content
	for _, want := range []string{"The star in the tree", "ladder", "Does the ladder reach?"} {
		if !strings.Contains(user, want) {
			t.Fatalf("user prompt %q is missing %q", user, want)
		}
	}
}

func TestJudgeDeniesAndNormalisesOddAnswers(t *testing.T) {
	denied := newFakeEndpoint(t, http.StatusOK, completion(`{"approved": false, "reason": "A puddle is not climbable.", "confidence": "wat"}`))
	verdict, err := NewClient().Judge(context.Background(), testConfig(denied.URL), JudgeInput{Question: "climb the puddle?"})
	if err != nil {
		t.Fatalf("Judge = %v", err)
	}
	if verdict.Approved {
		t.Fatal("a denial must stay a denial")
	}
	if verdict.Confidence != "medium" {
		t.Fatalf("confidence = %q, want the normalised default", verdict.Confidence)
	}
}

func TestJudgeRejectsAnswersItCannotRead(t *testing.T) {
	cases := map[string]string{
		"prose":      "Sure, that sounds fine to me!",
		"unclosed":   `{"approved": true, "reason": "because`,
		"no reason":  `{"approved": true, "reason": ""}`,
		"wrong type": `{"approved": "yes", "reason": "because"}`,
	}
	for name, answer := range cases {
		t.Run(name, func(t *testing.T) {
			fake := newFakeEndpoint(t, http.StatusOK, completion(answer))
			if _, err := NewClient().Judge(context.Background(), testConfig(fake.URL), JudgeInput{Question: "?"}); err == nil {
				t.Fatal("expected an error")
			}
		})
	}
}

func TestGeneratePuzzleExtractsSpecs(t *testing.T) {
	spec := `{"id":"ai-space","title":"Space station star","brief":"Reach the star.","hint":"climb something","goalKind":"star","width":20,"height":12,"terrain":[],"entities":[{"phrase":"star","x":5,"y":2}],"playerX":1,"playerY":9}`

	t.Run("direct spec", func(t *testing.T) {
		fake := newFakeEndpoint(t, http.StatusOK, completion("```json\n"+spec+"\n```"))
		generated, err := NewClient().GeneratePuzzle(context.Background(), testConfig(fake.URL), PuzzleRequest{Theme: "space", GoalKind: "star"})
		if err != nil {
			t.Fatalf("GeneratePuzzle = %v", err)
		}
		if !json.Valid(generated.Spec) {
			t.Fatalf("spec is not valid JSON: %s", generated.Spec)
		}
		var parsed struct {
			ID       string `json:"id"`
			GoalKind string `json:"goalKind"`
		}
		if err := json.Unmarshal(generated.Spec, &parsed); err != nil || parsed.ID != "ai-space" || parsed.GoalKind != "star" {
			t.Fatalf("spec = %s (%v)", generated.Spec, err)
		}
		if generated.Model != "fake-model" {
			t.Fatalf("model = %q", generated.Model)
		}
		if !strings.Contains(fake.requests[0].Messages[1].Content, "space") {
			t.Fatalf("the theme was not sent: %q", fake.requests[0].Messages[1].Content)
		}
	})

	t.Run("wrapped spec", func(t *testing.T) {
		wrapped := `{"spec": ` + spec + `, "notes": "made it roomy on purpose"}`
		fake := newFakeEndpoint(t, http.StatusOK, completion(wrapped))
		generated, err := NewClient().GeneratePuzzle(context.Background(), testConfig(fake.URL), PuzzleRequest{GoalKind: "star"})
		if err != nil {
			t.Fatalf("GeneratePuzzle = %v", err)
		}
		if !strings.Contains(string(generated.Spec), "ai-space") {
			t.Fatalf("spec = %s", generated.Spec)
		}
		if !strings.Contains(generated.Notes, "roomy") {
			t.Fatalf("notes = %q", generated.Notes)
		}
	})

	t.Run("no json", func(t *testing.T) {
		fake := newFakeEndpoint(t, http.StatusOK, completion("I am afraid I cannot do that."))
		if _, err := NewClient().GeneratePuzzle(context.Background(), testConfig(fake.URL), PuzzleRequest{GoalKind: "star"}); err == nil {
			t.Fatal("expected an error")
		}
	})
}

func TestExtractJSONIsStringAware(t *testing.T) {
	answer := `Sure: {"reason": "the brace } inside a string is fine", "approved": false, "confidence": "low"} trailing`
	raw, err := extractJSON(answer)
	if err != nil {
		t.Fatalf("extractJSON = %v", err)
	}
	var verdict Verdict
	if err := json.Unmarshal([]byte(raw), &verdict); err != nil {
		t.Fatalf("unmarshal = %v (%s)", err, raw)
	}
	if verdict.Approved || verdict.Reason == "" {
		t.Fatalf("verdict = %+v", verdict)
	}
}
