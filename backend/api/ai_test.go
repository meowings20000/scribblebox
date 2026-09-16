package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"backend/ai"
	"backend/world"
)

const aiTestKey = "sk-api-test-key-never-echo"

// fakeAI stands in for the player's OpenAI-compatible endpoint.
func fakeAI(t *testing.T, status int, content string) *httptest.Server {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer "+aiTestKey {
			t.Errorf("Authorization = %q", got)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		if status >= 400 {
			_, _ = w.Write([]byte(`{"error":{"message":"invalid api key"}}`))
			return
		}
		encoded, _ := json.Marshal(content)
		_, _ = fmt.Fprintf(w, `{"model":"fake","choices":[{"message":{"content":%s}}]}`, encoded)
	}))
	t.Cleanup(server.Close)
	return server
}

func aiHandler(t *testing.T, endpoint string) http.Handler {
	t.Helper()
	return NewWithAI(NewWorldStore(20), []string{allowedOrigin}, ai.NewClient())
}

func errorMessage(t *testing.T, recorder *httptest.ResponseRecorder) string {
	t.Helper()
	var payload struct {
		Error  string `json:"error"`
		Model  string `json:"model"`
		UsedAI bool   `json:"usedAi"`
	}
	decodeBody(t, recorder, &payload)
	return payload.Error
}

// specJSON builds a spec the world package accepts: goal kind star, one star
// entity, flat ground, the player standing on it, and the prize close enough to
// satisfy the game's easiness rules.
func specJSON(id, goalKind string) string {
	width, height := 20, 10
	terrain := make([]int, width*height)
	for x := 0; x < width; x++ {
		terrain[(height-1)*width+x] = 1
	}
	entity := `{"phrase":"star","x":10,"y":2}`
	switch goalKind {
	case "candles":
		entity = `{"phrase":"candle","x":10,"y":7},{"phrase":"candle","x":12,"y":7}`
	case "chest":
		entity = `{"phrase":"chest","x":10,"y":7}`
	case "cross":
		for x := 6; x < 10; x++ {
			terrain[(height-1)*width+x] = 2
		}
		entity = `{"phrase":"plank","x":3,"y":7}`
	}
	encodedTerrain, _ := json.Marshal(terrain)
	return fmt.Sprintf(
		`{"id":%q,"title":"An invented puzzle","brief":"Reach the goal.","hint":"try something silly","goalKind":%q,"width":%d,"height":%d,"terrain":%s,"entities":[%s],"playerX":2,"playerY":8}`,
		id, goalKind, width, height, encodedTerrain, entity,
	)
}

func judgeBody(worldID, question, endpoint string) string {
	return fmt.Sprintf(
		`{"baseUrl":%q,"apiKey":%q,"model":"fake","worldId":%q,"question":%q}`,
		endpoint+"/v1", aiTestKey, worldID, question,
	)
}

func TestJudgeRequiresSettingsAndAWorld(t *testing.T) {
	handler := aiHandler(t, "")
	id := newWorld(t, handler, firstPuzzleID(t, handler))

	cases := []struct {
		name string
		body string
		want int
	}{
		{"no settings", fmt.Sprintf(`{"worldId":%q,"question":"would a ladder work?"}`, id), http.StatusBadRequest},
		{"no key", fmt.Sprintf(`{"baseUrl":"http://127.0.0.1:1/v1","worldId":%q,"question":"x"}`, id), http.StatusBadRequest},
		{"no question", judgeBody(id, "  ", "http://127.0.0.1:1"), http.StatusBadRequest},
		{"unknown world", judgeBody("nope", "would a ladder work?", "http://127.0.0.1:1"), http.StatusNotFound},
		{"question too long", judgeBody(id, strings.Repeat("why ", 200), "http://127.0.0.1:1"), http.StatusBadRequest},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			recorder := request(t, handler, http.MethodPost, "/api/ai/judge", tc.body, nil)
			if recorder.Code != tc.want {
				t.Fatalf("status = %d, want %d (%s)", recorder.Code, tc.want, recorder.Body.String())
			}
			if strings.Contains(recorder.Body.String(), aiTestKey) {
				t.Fatal("the key leaked into the response")
			}
		})
	}
}

func TestJudgeApprovesAndMarksTheWorldSolved(t *testing.T) {
	server := fakeAI(t, http.StatusOK, `{"approved": true, "reason": "A ladder reaches the branch.", "confidence": "high"}`)
	handler := aiHandler(t, server.URL)
	id := newWorld(t, handler, firstPuzzleID(t, handler))

	recorder := request(t, handler, http.MethodPost, "/api/ai/judge", judgeBody(id, "I lean a ladder against the tree — does that count?", server.URL), nil)
	if recorder.Code != http.StatusOK {
		t.Fatalf("judge = %d (%s)", recorder.Code, recorder.Body.String())
	}
	var payload struct {
		Verdict ai.Verdict  `json:"verdict"`
		State   world.State `json:"state"`
		UsedAI  bool        `json:"usedAi"`
		Model   string      `json:"model"`
	}
	decodeBody(t, recorder, &payload)
	if !payload.Verdict.Approved || payload.Verdict.Confidence != "high" || payload.Verdict.Reason == "" {
		t.Fatalf("verdict = %+v", payload.Verdict)
	}
	if !payload.UsedAI || payload.Model != "fake" {
		t.Fatalf("payload = %+v", payload)
	}
	if !payload.State.Solved || !payload.State.Judged || !strings.Contains(payload.State.JudgeNote, "ladder") {
		t.Fatalf("the accepted judgement should mark the world solved: solved=%v judged=%v note=%q",
			payload.State.Solved, payload.State.Judged, payload.State.JudgeNote)
	}
	if !payload.State.Goal.Met || !strings.Contains(payload.State.Goal.Progress, "judge") {
		t.Fatalf("goal = %+v", payload.State.Goal)
	}
	if strings.Contains(recorder.Body.String(), aiTestKey) {
		t.Fatal("the key leaked into the response")
	}
}

func TestJudgeDenialLeavesTheWorldAlone(t *testing.T) {
	server := fakeAI(t, http.StatusOK, `{"approved": false, "reason": "A cake does not reach the star.", "confidence": "high"}`)
	handler := aiHandler(t, server.URL)
	id := newWorld(t, handler, firstPuzzleID(t, handler))

	recorder := request(t, handler, http.MethodPost, "/api/ai/judge", judgeBody(id, "cake?", server.URL), nil)
	if recorder.Code != http.StatusOK {
		t.Fatalf("judge = %d", recorder.Code)
	}
	var payload struct {
		Verdict ai.Verdict  `json:"verdict"`
		State   world.State `json:"state"`
	}
	decodeBody(t, recorder, &payload)
	if payload.Verdict.Approved {
		t.Fatal("expected a denial")
	}
	if payload.State.Solved || payload.State.Judged {
		t.Fatalf("a denial must not solve anything: %+v", payload.State.Goal)
	}
}

func TestJudgeReportsUpstreamFailures(t *testing.T) {
	handler := aiHandler(t, "")
	id := newWorld(t, handler, firstPuzzleID(t, handler))

	t.Run("bad key", func(t *testing.T) {
		server := fakeAI(t, http.StatusUnauthorized, "")
		recorder := request(t, handler, http.MethodPost, "/api/ai/judge", judgeBody(id, "does this work?", server.URL), nil)
		if recorder.Code != http.StatusBadGateway {
			t.Fatalf("status = %d (%s)", recorder.Code, recorder.Body.String())
		}
		var payload struct {
			Error string `json:"error"`
		}
		decodeBody(t, recorder, &payload)
		if !strings.Contains(payload.Error, "refused the key") {
			t.Fatalf("error = %q", payload.Error)
		}
		if strings.Contains(payload.Error, aiTestKey) {
			t.Fatal("the key leaked into the error")
		}
	})

	t.Run("prose instead of json", func(t *testing.T) {
		server := fakeAI(t, http.StatusOK, "Sure thing, that sounds good!")
		recorder := request(t, handler, http.MethodPost, "/api/ai/judge", judgeBody(id, "does this work?", server.URL), nil)
		if recorder.Code != http.StatusBadGateway {
			t.Fatalf("status = %d (%s)", recorder.Code, recorder.Body.String())
		}
	})

	t.Run("unreachable", func(t *testing.T) {
		body := fmt.Sprintf(`{"baseUrl":"http://127.0.0.1:1/v1","apiKey":%q,"model":"fake","worldId":%q,"question":"x"}`, aiTestKey, id)
		recorder := request(t, handler, http.MethodPost, "/api/ai/judge", body, nil)
		if recorder.Code != http.StatusBadGateway {
			t.Fatalf("status = %d (%s)", recorder.Code, recorder.Body.String())
		}
		var payload struct {
			Error string `json:"error"`
		}
		decodeBody(t, recorder, &payload)
		if !strings.Contains(payload.Error, "could not reach") {
			t.Fatalf("error = %q", payload.Error)
		}
	})

	t.Run("timeout", func(t *testing.T) {
		// The handler sleeps longer than the configured timeout and then returns,
		// so the client gives up (proving the timeout path) while the server can
		// still shut down cleanly.
		slow := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			time.Sleep(1500 * time.Millisecond)
		}))
		defer slow.Close()
		body := fmt.Sprintf(`{"baseUrl":%q,"apiKey":%q,"model":"fake","timeoutSeconds":1,"worldId":%q,"question":"x"}`,
			slow.URL+"/v1", aiTestKey, id)
		recorder := request(t, handler, http.MethodPost, "/api/ai/judge", body, nil)
		if recorder.Code != http.StatusGatewayTimeout {
			t.Fatalf("status = %d, want 504 (%s)", recorder.Code, recorder.Body.String())
		}
	})
}

func TestGeneratePuzzleAcceptsOnlyRefereeableSpecs(t *testing.T) {
	handler := aiHandler(t, "")

	t.Run("valid star puzzle", func(t *testing.T) {
		server := fakeAI(t, http.StatusOK, "```json\n"+specJSON("ai-star", "star")+"\n```")
		body := fmt.Sprintf(`{"baseUrl":%q,"apiKey":%q,"model":"fake","theme":"space station","goalKind":"star"}`, server.URL+"/v1", aiTestKey)
		recorder := request(t, handler, http.MethodPost, "/api/ai/puzzles", body, nil)
		if recorder.Code != http.StatusCreated {
			t.Fatalf("status = %d (%s)", recorder.Code, recorder.Body.String())
		}
		var payload struct {
			Spec   world.PuzzleSpec `json:"spec"`
			Notes  string           `json:"notes"`
			UsedAI bool             `json:"usedAi"`
			Model  string           `json:"model"`
		}
		decodeBody(t, recorder, &payload)
		if payload.Spec.ID != "ai-star" || payload.Spec.GoalKind != "star" || len(payload.Spec.Entities) == 0 || payload.Spec.Width == 0 {
			t.Fatalf("spec = %+v", payload.Spec)
		}
		if !payload.UsedAI || payload.Model != "fake" || payload.Notes == "" {
			t.Fatalf("payload = %+v", payload)
		}
		// The spec must also be playable straight away.
		playable := request(t, handler, http.MethodPost, "/api/worlds", `{"aiSpec":`+specJSON("ai-star", "star")+`}`, nil)
		if playable.Code != http.StatusCreated {
			t.Fatalf("playing an AI spec = %d (%s)", playable.Code, playable.Body.String())
		}
		var playPayload struct {
			State world.State `json:"state"`
		}
		decodeBody(t, playable, &playPayload)
		if playPayload.State.Puzzle.ID != "ai-star" || playPayload.State.Width == 0 {
			t.Fatalf("state = %+v", playPayload.State)
		}
	})

	t.Run("unknown goal kind", func(t *testing.T) {
		server := fakeAI(t, http.StatusOK, specJSON("ai-x", "star"))
		body := fmt.Sprintf(`{"baseUrl":%q,"apiKey":%q,"model":"fake","theme":"space","goalKind":"rescue"}`, server.URL+"/v1", aiTestKey)
		recorder := request(t, handler, http.MethodPost, "/api/ai/puzzles", body, nil)
		if recorder.Code != http.StatusBadRequest {
			t.Fatalf("status = %d (%s)", recorder.Code, recorder.Body.String())
		}
	})

	t.Run("mismatched goal kind", func(t *testing.T) {
		server := fakeAI(t, http.StatusOK, specJSON("ai-y", "star"))
		body := fmt.Sprintf(`{"baseUrl":%q,"apiKey":%q,"model":"fake","theme":"space","goalKind":"chest"}`, server.URL+"/v1", aiTestKey)
		recorder := request(t, handler, http.MethodPost, "/api/ai/puzzles", body, nil)
		if recorder.Code != http.StatusBadGateway {
			t.Fatalf("status = %d (%s)", recorder.Code, recorder.Body.String())
		}
		if !strings.Contains(recorder.Body.String(), "goal kind") {
			t.Fatalf("body = %s", recorder.Body.String())
		}
	})

	t.Run("structurally broken spec", func(t *testing.T) {
		broken := strings.Replace(specJSON("ai-broken", "chest"), `"terrain":[`, `"terrain":[9,9,`, 1)
		server := fakeAI(t, http.StatusOK, broken)
		body := fmt.Sprintf(`{"baseUrl":%q,"apiKey":%q,"model":"fake","theme":"space","goalKind":"chest"}`, server.URL+"/v1", aiTestKey)
		recorder := request(t, handler, http.MethodPost, "/api/ai/puzzles", body, nil)
		if recorder.Code != http.StatusBadGateway {
			t.Fatalf("status = %d (%s)", recorder.Code, recorder.Body.String())
		}
		if !strings.Contains(recorder.Body.String(), "rejected") {
			t.Fatalf("body = %s", recorder.Body.String())
		}
	})

	t.Run("no settings", func(t *testing.T) {
		recorder := request(t, handler, http.MethodPost, "/api/ai/puzzles", `{"theme":"space","goalKind":"star"}`, nil)
		if recorder.Code != http.StatusBadRequest {
			t.Fatalf("status = %d (%s)", recorder.Code, recorder.Body.String())
		}
	})
}

func TestWorldsFromASpecRejectBadInput(t *testing.T) {
	handler := aiHandler(t, "")

	cases := []struct {
		name string
		body string
		want int
	}{
		{"not json", `{"aiSpec":"ladder"}`, http.StatusBadRequest},
		{"unknown goal kind", `{"aiSpec":` + strings.Replace(specJSON("ai-z", "star"), `"goalKind":"star"`, `"goalKind":"teleport"`, 1) + `}`, http.StatusBadRequest},
		{"out of range player", `{"aiSpec":` + strings.Replace(specJSON("ai-p", "star"), `"playerX":2,"playerY":8`, `"playerX":99,"playerY":99`, 1) + `}`, http.StatusBadRequest},
		{"no star for a star puzzle", `{"aiSpec":` + strings.Replace(specJSON("ai-n", "star"), `{"phrase":"star","x":10,"y":2}`, `{"phrase":"cake","x":10,"y":2}`, 1) + `}`, http.StatusBadRequest},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			recorder := request(t, handler, http.MethodPost, "/api/worlds", tc.body, nil)
			if recorder.Code != tc.want {
				t.Fatalf("status = %d, want %d (%s)", recorder.Code, tc.want, recorder.Body.String())
			}
		})
	}
}

func TestAIWorldsStayPlayableThroughTheOrdinaryRoutes(t *testing.T) {
	handler := aiHandler(t, "")
	created := request(t, handler, http.MethodPost, "/api/worlds", `{"aiSpec":`+specJSON("ai-play", "star")+`}`, nil)
	if created.Code != http.StatusCreated {
		t.Fatalf("create = %d (%s)", created.Code, created.Body.String())
	}
	var payload struct {
		State world.State `json:"state"`
	}
	decodeBody(t, created, &payload)
	id := payload.State.ID

	spawn := request(t, handler, http.MethodPost, "/api/worlds/"+id+"/spawn", `{"phrase":"ladder","x":3,"y":3}`, nil)
	if spawn.Code != http.StatusOK {
		t.Fatalf("spawn = %d (%s)", spawn.Code, spawn.Body.String())
	}
	step := request(t, handler, http.MethodPost, "/api/worlds/"+id+"/step", `{"ticks":3}`, nil)
	if step.Code != http.StatusOK {
		t.Fatalf("step = %d", step.Code)
	}
	act := request(t, handler, http.MethodPost, "/api/worlds/"+id+"/act", `{"action":"right"}`, nil)
	if act.Code != http.StatusOK {
		t.Fatalf("act = %d (%s)", act.Code, act.Body.String())
	}
	reset := request(t, handler, http.MethodPost, "/api/worlds/"+id+"/reset", "", nil)
	if reset.Code != http.StatusOK {
		t.Fatalf("reset = %d", reset.Code)
	}
}

func TestGeneratedPuzzlesNeverLeakAnAnswerKey(t *testing.T) {
	server := fakeAI(t, http.StatusOK, specJSON("ai-quiet", "star"))
	handler := aiHandler(t, server.URL)
	body := fmt.Sprintf(`{"baseUrl":%q,"apiKey":%q,"model":"fake","theme":"space","goalKind":"star"}`, server.URL+"/v1", aiTestKey)
	recorder := request(t, handler, http.MethodPost, "/api/ai/puzzles", body, nil)
	if strings.Contains(strings.ToLower(recorder.Body.String()), "solutions") {
		t.Fatalf("the generated puzzle leaks an answer key: %s", recorder.Body.String())
	}
	if strings.Contains(recorder.Body.String(), aiTestKey) {
		t.Fatal("the key leaked into the response")
	}
}

func TestPingProvesTheSettingsWork(t *testing.T) {
	handler := aiHandler(t, "")

	t.Run("working settings", func(t *testing.T) {
		server := fakeAI(t, http.StatusOK, "ready")
		body := fmt.Sprintf(`{"baseUrl":%q,"apiKey":%q,"model":"fake"}`, server.URL+"/v1", aiTestKey)
		recorder := request(t, handler, http.MethodPost, "/api/ai/ping", body, nil)
		if recorder.Code != http.StatusOK {
			t.Fatalf("ping = %d (%s)", recorder.Code, recorder.Body.String())
		}
		var payload struct {
			OK        bool   `json:"ok"`
			Model     string `json:"model"`
			Reply     string `json:"reply"`
			LatencyMs int64  `json:"latencyMs"`
		}
		decodeBody(t, recorder, &payload)
		if !payload.OK || payload.Model != "fake" || payload.Reply != "ready" || payload.LatencyMs < 0 {
			t.Fatalf("payload = %+v", payload)
		}
		if strings.Contains(recorder.Body.String(), aiTestKey) {
			t.Fatal("the key leaked into the response")
		}
	})

	t.Run("refused key", func(t *testing.T) {
		server := fakeAI(t, http.StatusUnauthorized, "")
		body := fmt.Sprintf(`{"baseUrl":%q,"apiKey":%q,"model":"fake"}`, server.URL+"/v1", aiTestKey)
		recorder := request(t, handler, http.MethodPost, "/api/ai/ping", body, nil)
		if recorder.Code != http.StatusBadGateway {
			t.Fatalf("ping = %d (%s)", recorder.Code, recorder.Body.String())
		}
		if !strings.Contains(recorder.Body.String(), "refused the key") {
			t.Fatalf("body = %s", recorder.Body.String())
		}
	})

	t.Run("nothing configured", func(t *testing.T) {
		recorder := request(t, handler, http.MethodPost, "/api/ai/ping", `{}`, nil)
		if recorder.Code != http.StatusBadRequest {
			t.Fatalf("ping = %d (%s)", recorder.Code, recorder.Body.String())
		}
	})
}

func TestAIFeaturesAreOffWithoutCredentials(t *testing.T) {
	handler := aiHandler(t, "")
	id := newWorld(t, handler, firstPuzzleID(t, handler))

	// Every ordinary route keeps working with no AI configured at all.
	for name, recorder := range map[string]*httptest.ResponseRecorder{
		"spawn": request(t, handler, http.MethodPost, "/api/worlds/"+id+"/spawn", `{"phrase":"box","x":3,"y":3}`, nil),
		"step":  request(t, handler, http.MethodPost, "/api/worlds/"+id+"/step", `{"ticks":2}`, nil),
		"act":   request(t, handler, http.MethodPost, "/api/worlds/"+id+"/act", `{"action":"jump"}`, nil),
	} {
		if recorder.Code != http.StatusOK {
			t.Fatalf("%s = %d (%s)", name, recorder.Code, recorder.Body.String())
		}
	}
	judge := request(t, handler, http.MethodPost, "/api/ai/judge", fmt.Sprintf(`{"worldId":%q,"question":"does this work?"}`, id), nil)
	if judge.Code != http.StatusBadRequest {
		t.Fatalf("judge without settings = %d (%s)", judge.Code, judge.Body.String())
	}
	if message := errorMessage(t, judge); !strings.Contains(message, "Settings") {
		t.Fatalf("the message should point at the settings panel: %q", message)
	}
}
