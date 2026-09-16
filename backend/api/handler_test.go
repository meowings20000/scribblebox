package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"backend/lexicon"
	"backend/world"
)

const allowedOrigin = "http://localhost:3000"

func newTestHandler() http.Handler {
	return New(NewWorldStore(20), []string{allowedOrigin, "http://127.0.0.1:3000"})
}

func request(t *testing.T, handler http.Handler, method, path, body string, headers map[string]string) *httptest.ResponseRecorder {
	t.Helper()
	var reader io.Reader
	if body != "" {
		reader = strings.NewReader(body)
	}
	req := httptest.NewRequest(method, path, reader)
	for key, value := range headers {
		req.Header.Set(key, value)
	}
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, req)
	return recorder
}

func decodeBody(t *testing.T, recorder *httptest.ResponseRecorder, target any) {
	t.Helper()
	if err := json.Unmarshal(recorder.Body.Bytes(), target); err != nil {
		t.Fatalf("response is not JSON (%v): %s", err, recorder.Body.String())
	}
}

func firstPuzzleID(t *testing.T, handler http.Handler) string {
	t.Helper()
	recorder := request(t, handler, http.MethodGet, "/api/puzzles", "", nil)
	if recorder.Code != http.StatusOK {
		t.Fatalf("GET /api/puzzles = %d", recorder.Code)
	}
	var payload struct {
		Puzzles []world.Puzzle `json:"puzzles"`
	}
	decodeBody(t, recorder, &payload)
	if len(payload.Puzzles) == 0 {
		t.Fatal("no puzzles")
	}
	return payload.Puzzles[0].ID
}

func newWorld(t *testing.T, handler http.Handler, puzzleID string) string {
	t.Helper()
	recorder := request(t, handler, http.MethodPost, "/api/worlds", `{"puzzleId":"`+puzzleID+`"}`, nil)
	if recorder.Code != http.StatusCreated {
		t.Fatalf("create world = %d (%s)", recorder.Code, recorder.Body.String())
	}
	var payload struct {
		State world.State `json:"state"`
	}
	decodeBody(t, recorder, &payload)
	if payload.State.ID == "" {
		t.Fatalf("state has no id: %+v", payload.State)
	}
	return payload.State.ID
}

func TestHealthAndPuzzles(t *testing.T) {
	handler := newTestHandler()

	health := request(t, handler, http.MethodGet, "/api/health", "", nil)
	if health.Code != http.StatusOK {
		t.Fatalf("health = %d", health.Code)
	}
	var healthPayload struct {
		Status  string `json:"status"`
		Words   int    `json:"words"`
		Puzzles int    `json:"puzzles"`
	}
	decodeBody(t, health, &healthPayload)
	if healthPayload.Status != "ok" || healthPayload.Words < 100 || healthPayload.Puzzles == 0 {
		t.Fatalf("health payload = %+v", healthPayload)
	}

	puzzles := request(t, handler, http.MethodGet, "/api/puzzles", "", nil)
	if puzzles.Code != http.StatusOK {
		t.Fatalf("puzzles = %d", puzzles.Code)
	}
	// The answers must never leave the server.
	for _, forbidden := range []string{"solutions", "Solutions", "Phrases"} {
		if strings.Contains(puzzles.Body.String(), forbidden) {
			t.Fatalf("the puzzles response leaks %q: %s", forbidden, puzzles.Body.String())
		}
	}
	var payload struct {
		Puzzles []world.Puzzle `json:"puzzles"`
	}
	decodeBody(t, puzzles, &payload)
	if len(payload.Puzzles) < 4 {
		t.Fatalf("expected at least four puzzles, got %d", len(payload.Puzzles))
	}
	for _, puzzle := range payload.Puzzles {
		if puzzle.ID == "" || puzzle.Title == "" || puzzle.Brief == "" || puzzle.Hint == "" || puzzle.Width <= 0 || puzzle.Height <= 0 {
			t.Fatalf("puzzle is incomplete: %+v", puzzle)
		}
	}
}

func TestWordsAutocomplete(t *testing.T) {
	handler := newTestHandler()
	recorder := request(t, handler, http.MethodGet, "/api/words?q=lad&limit=5", "", nil)
	if recorder.Code != http.StatusOK {
		t.Fatalf("words = %d", recorder.Code)
	}
	var payload struct {
		Words       []string `json:"words"`
		Exact       bool     `json:"exact"`
		Dictionary  int      `json:"dictionary"`
		AlwaysWorks bool     `json:"alwaysWorks"`
	}
	decodeBody(t, recorder, &payload)
	if len(payload.Words) == 0 || payload.Dictionary < 100 || !payload.AlwaysWorks {
		t.Fatalf("words payload = %+v", payload)
	}
	if !strings.Contains(strings.Join(payload.Words, ","), "ladder") {
		t.Fatalf("expected ladder in %v", payload.Words)
	}

	empty := request(t, handler, http.MethodGet, "/api/words", "", nil)
	var emptyPayload struct {
		Words []string `json:"words"`
	}
	decodeBody(t, empty, &emptyPayload)
	if len(emptyPayload.Words) == 0 {
		t.Fatal("an empty query should still suggest something")
	}
}

// The four-choice hint shows the icon a word will have in the world, which it can
// only do if the words endpoint hands back the shape. An unknown word must not
// pretend to have one.
func TestWordsReportsTheShapeOfAWordItKnows(t *testing.T) {
	handler := newTestHandler()

	recorder := request(t, handler, http.MethodGet, "/api/words?q=pillow", "", nil)
	if recorder.Code != http.StatusOK {
		t.Fatalf("words = %d", recorder.Code)
	}
	var payload struct {
		Exact bool `json:"exact"`
		Match *struct {
			Key   string `json:"key"`
			Shape string `json:"shape"`
			Color string `json:"color"`
			Glyph string `json:"glyph"`
			Note  string `json:"note"`
		} `json:"match"`
	}
	decodeBody(t, recorder, &payload)
	if !payload.Exact || payload.Match == nil {
		t.Fatalf("a known word should report a match: %s", recorder.Body.String())
	}
	if payload.Match.Key == "" || payload.Match.Shape == "" || !strings.HasPrefix(payload.Match.Color, "#") {
		t.Fatalf("match is incomplete: %+v", payload.Match)
	}

	odd := request(t, handler, http.MethodGet, "/api/words?q=zorble", "", nil)
	var oddPayload struct {
		Exact bool `json:"exact"`
		Match *struct {
			Key string `json:"key"`
		} `json:"match"`
	}
	decodeBody(t, odd, &oddPayload)
	if oddPayload.Exact || oddPayload.Match != nil {
		t.Fatalf("an unknown word must not claim a shape: %s", odd.Body.String())
	}
}

func TestSpawnStepActResetOverHTTP(t *testing.T) {
	handler := newTestHandler()
	id := newWorld(t, handler, firstPuzzleID(t, handler))

	spawn := request(t, handler, http.MethodPost, "/api/worlds/"+id+"/spawn", `{"phrase":"ladder","x":3,"y":3}`, nil)
	if spawn.Code != http.StatusOK {
		t.Fatalf("spawn = %d (%s)", spawn.Code, spawn.Body.String())
	}
	var spawnPayload struct {
		Spawned     lexicon.Object `json:"spawned"`
		Note        string         `json:"note"`
		Approximate bool           `json:"approximate"`
		Events      []string       `json:"events"`
		State       world.State    `json:"state"`
	}
	decodeBody(t, spawn, &spawnPayload)
	if spawnPayload.Spawned.Key != "ladder" || spawnPayload.Spawned.Noun == "" || spawnPayload.Spawned.Shape == "" {
		t.Fatalf("spawned object = %+v", spawnPayload.Spawned)
	}
	if spawnPayload.Note == "" || len(spawnPayload.Events) == 0 {
		t.Fatalf("spawn payload = %+v", spawnPayload)
	}
	if len(spawnPayload.State.Notebook) != 1 || spawnPayload.State.Notebook[0].Key != "ladder" {
		t.Fatalf("notebook = %+v", spawnPayload.State.Notebook)
	}
	if len(spawnPayload.State.Entities) == 0 {
		t.Fatal("the world should contain the spawned object")
	}

	// An unknown word still spawns something, and says so.
	odd := request(t, handler, http.MethodPost, "/api/worlds/"+id+"/spawn", `{"phrase":"big zorble","x":5,"y":3}`, nil)
	if odd.Code != http.StatusOK {
		t.Fatalf("unknown word spawn = %d (%s)", odd.Code, odd.Body.String())
	}
	var oddPayload struct {
		Spawned     lexicon.Object `json:"spawned"`
		Approximate bool           `json:"approximate"`
	}
	decodeBody(t, odd, &oddPayload)
	if !oddPayload.Approximate || oddPayload.Spawned.Approximate != true || oddPayload.Spawned.Noun == "" {
		t.Fatalf("improvised object = %+v", oddPayload)
	}

	step := request(t, handler, http.MethodPost, "/api/worlds/"+id+"/step", `{"ticks":4}`, nil)
	if step.Code != http.StatusOK {
		t.Fatalf("step = %d", step.Code)
	}
	var stepPayload struct {
		State world.State `json:"state"`
	}
	decodeBody(t, step, &stepPayload)
	if stepPayload.State.Ticks == 0 {
		t.Fatalf("ticks did not advance: %+v", stepPayload.State.Ticks)
	}

	act := request(t, handler, http.MethodPost, "/api/worlds/"+id+"/act", `{"action":"right"}`, nil)
	if act.Code != http.StatusOK {
		t.Fatalf("act = %d (%s)", act.Code, act.Body.String())
	}

	reset := request(t, handler, http.MethodPost, "/api/worlds/"+id+"/reset", "", nil)
	if reset.Code != http.StatusOK {
		t.Fatalf("reset = %d", reset.Code)
	}
	var resetPayload struct {
		State world.State `json:"state"`
	}
	decodeBody(t, reset, &resetPayload)
	if len(resetPayload.State.Notebook) != 0 {
		t.Fatalf("reset should clear the notebook: %+v", resetPayload.State.Notebook)
	}
}

// TestTheFourChoiceHintIsVerifiedEndToEnd is the important one: it takes the hint
// the HTTP layer hands the player and actually plays every option through the
// ordinary spawn/step routes, proving exactly one of the four solves the puzzle
// and that the answer is not marked anywhere in the response.
func TestTheFourChoiceHintIsVerifiedEndToEnd(t *testing.T) {
	probe := newTestHandler()
	var listed struct {
		Puzzles []world.Puzzle `json:"puzzles"`
	}
	decodeBody(t, request(t, probe, http.MethodGet, "/api/puzzles", "", nil), &listed)

	for _, puzzle := range listed.Puzzles {
		t.Run(puzzle.ID, func(t *testing.T) {
			handler := newTestHandler()
			id := newWorld(t, handler, puzzle.ID)

			recorder := request(t, handler, http.MethodPost, "/api/worlds/"+id+"/hint", "", nil)
			if recorder.Code != http.StatusOK {
				t.Fatalf("hint = %d (%s)", recorder.Code, recorder.Body.String())
			}
			var payload struct {
				Hint  world.HintSet `json:"hint"`
				State world.State   `json:"state"`
			}
			decodeBody(t, recorder, &payload)

			if len(payload.Hint.Options) != 4 {
				t.Fatalf("expected four options, got %v", payload.Hint.Options)
			}
			seen := map[string]bool{}
			for _, option := range payload.Hint.Options {
				if strings.TrimSpace(option) == "" || seen[option] {
					t.Fatalf("options must be non-empty and distinct: %v", payload.Hint.Options)
				}
				seen[option] = true
			}
			assertNoAnswerKey(t, recorder.Body.Bytes())

			solving := []string{}
			for _, option := range payload.Hint.Options {
				request(t, handler, http.MethodPost, "/api/worlds/"+id+"/reset", "", nil)
				spawned := request(t, handler, http.MethodPost, "/api/worlds/"+id+"/spawn",
					fmt.Sprintf(`{"phrase":%q,"x":%d,"y":%d}`, option, payload.Hint.Placement.X, payload.Hint.Placement.Y), nil)
				if spawned.Code != http.StatusOK {
					continue // it cannot even be placed there, so it is not an offer
				}
				request(t, handler, http.MethodPost, "/api/worlds/"+id+"/step", `{"ticks":40}`, nil)
				var after struct {
					State world.State `json:"state"`
				}
				decodeBody(t, request(t, handler, http.MethodGet, "/api/worlds/"+id, "", nil), &after)
				if after.State.Solved {
					solving = append(solving, option)
				}
			}
			if len(solving) != 1 {
				t.Fatalf("exactly one option may solve the puzzle; %d did: %v (options %v)", len(solving), solving, payload.Hint.Options)
			}
		})
	}
}

func TestHintRefusesWhenThereIsNothingToHintAt(t *testing.T) {
	handler := newTestHandler()
	id := newWorld(t, handler, firstPuzzleID(t, handler))

	unknown := request(t, handler, http.MethodPost, "/api/worlds/nope/hint", "", nil)
	if unknown.Code != http.StatusNotFound {
		t.Fatalf("unknown world = %d", unknown.Code)
	}

	// Solve it through the ordinary routes, then ask for a hint.
	recorder := request(t, handler, http.MethodPost, "/api/worlds/"+id+"/hint", "", nil)
	var payload struct {
		Hint world.HintSet `json:"hint"`
	}
	decodeBody(t, recorder, &payload)
	var solved bool
	for _, option := range payload.Hint.Options {
		request(t, handler, http.MethodPost, "/api/worlds/"+id+"/reset", "", nil)
		spawned := request(t, handler, http.MethodPost, "/api/worlds/"+id+"/spawn",
			fmt.Sprintf(`{"phrase":%q,"x":%d,"y":%d}`, option, payload.Hint.Placement.X, payload.Hint.Placement.Y), nil)
		if spawned.Code != http.StatusOK {
			continue
		}
		request(t, handler, http.MethodPost, "/api/worlds/"+id+"/step", `{"ticks":40}`, nil)
		var after struct {
			State world.State `json:"state"`
		}
		decodeBody(t, request(t, handler, http.MethodGet, "/api/worlds/"+id, "", nil), &after)
		if after.State.Solved {
			solved = true
			break
		}
	}
	if !solved {
		t.Fatal("the hint's correct option should solve the world through the ordinary routes")
	}
	after := request(t, handler, http.MethodPost, "/api/worlds/"+id+"/hint", "", nil)
	if after.Code != http.StatusBadRequest {
		t.Fatalf("a hint on a solved puzzle = %d (%s)", after.Code, after.Body.String())
	}
}

// assertNoAnswerKey walks a JSON response and fails if any field is named like an
// answer key. Prose that merely contains the word "answer" is fine; a field that
// tells the player which option is right is not.
func assertNoAnswerKey(t *testing.T, raw []byte) {
	t.Helper()
	var decoded any
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatalf("response is not JSON: %v", err)
	}
	forbidden := map[string]bool{"correct": true, "iscorrect": true, "solution": true, "solutions": true, "answer": true, "answers": true, "rightanswer": true}
	var walk func(value any, path string)
	walk = func(value any, path string) {
		switch typed := value.(type) {
		case map[string]any:
			for key, nested := range typed {
				if forbidden[strings.ToLower(key)] {
					t.Fatalf("the response marks the answer at %s.%s: %s", path, key, string(raw))
				}
				walk(nested, path+"."+key)
			}
		case []any:
			for index, nested := range typed {
				walk(nested, fmt.Sprintf("%s[%d]", path, index))
			}
		}
	}
	walk(decoded, "$")
}

// The four choices are proven when they are handed out, and re-proven when a
// choice is pressed. This test changes the world between those two moments — the
// walker moves, stacks grow — which is exactly the case where a stale spot used to
// break the promise that one of the four works.
func TestChoosingAHintOptionIsVerifiedWhenItIsPressed(t *testing.T) {
	handler := newTestHandler()
	id := newWorld(t, handler, firstPuzzleID(t, handler))
	mess := func() {
		for _, phrase := range []string{"cake", "pillow"} {
			request(t, handler, http.MethodPost, "/api/worlds/"+id+"/spawn",
				fmt.Sprintf(`{"phrase":%q,"x":3,"y":2}`, phrase), nil)
			request(t, handler, http.MethodPost, "/api/worlds/"+id+"/step", `{"ticks":3}`, nil)
		}
	}

	recorder := request(t, handler, http.MethodPost, "/api/worlds/"+id+"/hint", "", nil)
	if recorder.Code != http.StatusOK {
		t.Fatalf("hint = %d (%s)", recorder.Code, recorder.Body.String())
	}
	var offered struct {
		Hint world.HintSet `json:"hint"`
	}
	decodeBody(t, recorder, &offered)

	won := []string{}
	for _, option := range offered.Hint.Options {
		request(t, handler, http.MethodPost, "/api/worlds/"+id+"/reset", "", nil)
		mess()
		chosen := request(t, handler, http.MethodPost, "/api/worlds/"+id+"/hint/choose",
			fmt.Sprintf(`{"phrase":%q}`, option), nil)
		if chosen.Code != http.StatusOK {
			t.Fatalf("choosing %q = %d (%s)", option, chosen.Code, chosen.Body.String())
		}
		var payload struct {
			Spawned   lexicon.Object `json:"spawned"`
			MovedSpot bool           `json:"movedSpot"`
			Events    []string       `json:"events"`
			State     world.State    `json:"state"`
			Hint      world.HintSet  `json:"hint"`
		}
		decodeBody(t, chosen, &payload)
		if payload.Spawned.Key == "" {
			t.Fatalf("choosing %q spawned nothing: %s", option, chosen.Body.String())
		}
		if len(payload.Events) == 0 {
			t.Fatalf("choosing %q said nothing happened", option)
		}
		assertNoAnswerKey(t, chosen.Body.Bytes())

		// The world is settled by the time the answer comes back, so the player sees
		// the outcome of the choice immediately.
		if payload.State.Ticks < hintSettleTicks {
			t.Fatalf("choosing %q should settle the world (%d ticks)", option, payload.State.Ticks)
		}
		if payload.State.Solved {
			won = append(won, option)
		}
	}
	if len(won) != 1 {
		t.Fatalf("exactly one of the four choices must win even when the world moved on; %d did: %v (%v)",
			len(won), won, offered.Hint.Options)
	}
}

func TestChoosingAHintOptionHandlesBadInput(t *testing.T) {
	handler := newTestHandler()
	id := newWorld(t, handler, firstPuzzleID(t, handler))

	cases := []struct {
		name string
		body string
		want int
	}{
		{"empty choice", `{"phrase":"  "}`, http.StatusBadRequest},
		{"no choice", `{}`, http.StatusBadRequest},
		{"unreadable choice", `{"phrase":"!!!"}`, http.StatusBadRequest},
		{"too long", fmt.Sprintf(`{"phrase":%q}`, strings.Repeat("a", 80)), http.StatusBadRequest},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			recorder := request(t, handler, http.MethodPost, "/api/worlds/"+id+"/hint/choose", tc.body, nil)
			if recorder.Code != tc.want {
				t.Fatalf("status = %d, want %d (%s)", recorder.Code, tc.want, recorder.Body.String())
			}
		})
	}

	unknown := request(t, handler, http.MethodPost, "/api/worlds/nope/hint/choose", `{"phrase":"ladder"}`, nil)
	if unknown.Code != http.StatusNotFound {
		t.Fatalf("unknown world = %d", unknown.Code)
	}
}

func TestBadRequestsAreRejected(t *testing.T) {
	handler := newTestHandler()
	id := newWorld(t, handler, firstPuzzleID(t, handler))
	long := strings.Repeat("a", 80)

	cases := []struct {
		name   string
		method string
		path   string
		body   string
		want   int
	}{
		{"missing-puzzle", http.MethodPost, "/api/worlds", `{}`, http.StatusBadRequest},
		{"unknown-puzzle", http.MethodPost, "/api/worlds", `{"puzzleId":"nope"}`, http.StatusNotFound},
		{"trailing-json", http.MethodPost, "/api/worlds", `{"puzzleId":"a"}{"puzzleId":"b"}`, http.StatusBadRequest},
		{"not-json", http.MethodPost, "/api/worlds", `puzzleId=a`, http.StatusBadRequest},
		{"phrase-too-long", http.MethodPost, "/api/worlds/" + id + "/spawn", `{"phrase":"` + long + `","x":1,"y":1}`, http.StatusBadRequest},
		{"empty-phrase", http.MethodPost, "/api/worlds/" + id + "/spawn", `{"phrase":"   ","x":1,"y":1}`, http.StatusBadRequest},
		{"spawn-out-of-range", http.MethodPost, "/api/worlds/" + id + "/spawn", `{"phrase":"box","x":999,"y":999}`, http.StatusBadRequest},
		{"ticks-zero", http.MethodPost, "/api/worlds/" + id + "/step", `{"ticks":0}`, http.StatusBadRequest},
		{"ticks-huge", http.MethodPost, "/api/worlds/" + id + "/step", `{"ticks":9999}`, http.StatusBadRequest},
		{"bad-action", http.MethodPost, "/api/worlds/" + id + "/act", `{"action":"teleport"}`, http.StatusBadRequest},
		{"unknown-world", http.MethodGet, "/api/worlds/nope", "", http.StatusNotFound},
		{"unknown-world-spawn", http.MethodPost, "/api/worlds/nope/spawn", `{"phrase":"box","x":1,"y":1}`, http.StatusNotFound},
		{"method-not-allowed", http.MethodDelete, "/api/puzzles", "", http.StatusMethodNotAllowed},
		{"get-worlds-not-allowed", http.MethodGet, "/api/worlds", "", http.StatusMethodNotAllowed},
		{"unknown-endpoint", http.MethodGet, "/api/nothing", "", http.StatusNotFound},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			recorder := request(t, handler, tc.method, tc.path, tc.body, nil)
			if recorder.Code != tc.want {
				t.Fatalf("status = %d, want %d (%s)", recorder.Code, tc.want, recorder.Body.String())
			}
			var payload map[string]string
			decodeBody(t, recorder, &payload)
			if payload["error"] == "" {
				t.Fatalf("expected a JSON error body, got %s", recorder.Body.String())
			}
		})
	}
}

func TestCORSPolicy(t *testing.T) {
	handler := newTestHandler()

	blocked := request(t, handler, http.MethodPost, "/api/worlds", `{"puzzleId":"x"}`, map[string]string{"Origin": "http://evil.example"})
	if blocked.Code != http.StatusForbidden {
		t.Fatalf("hostile origin = %d", blocked.Code)
	}
	allowed := request(t, handler, http.MethodGet, "/api/puzzles", "", map[string]string{"Origin": allowedOrigin})
	if allowed.Header().Get("Access-Control-Allow-Origin") != allowedOrigin {
		t.Fatalf("allow header = %q", allowed.Header().Get("Access-Control-Allow-Origin"))
	}
	preflight := request(t, handler, http.MethodOptions, "/api/worlds", "", map[string]string{"Origin": allowedOrigin})
	if preflight.Code != http.StatusNoContent {
		t.Fatalf("preflight = %d", preflight.Code)
	}
	hostilePreflight := request(t, handler, http.MethodOptions, "/api/worlds", "", map[string]string{"Origin": "http://evil.example"})
	if hostilePreflight.Code != http.StatusForbidden {
		t.Fatalf("hostile preflight = %d", hostilePreflight.Code)
	}
	noOrigin := request(t, handler, http.MethodGet, "/api/health", "", nil)
	if noOrigin.Code != http.StatusOK {
		t.Fatalf("a request without Origin should be allowed: %d", noOrigin.Code)
	}
}

func TestWorldStoreIsBounded(t *testing.T) {
	store := NewWorldStore(2)
	handler := New(store, []string{allowedOrigin})
	puzzleID := firstPuzzleID(t, handler)

	first := newWorld(t, handler, puzzleID)
	second := newWorld(t, handler, puzzleID)
	third := newWorld(t, handler, puzzleID)

	if first == second || second == third {
		t.Fatal("world ids must be unique")
	}
	if store.len() != 2 {
		t.Fatalf("store length = %d, want 2", store.len())
	}
	if _, ok := store.get(first); ok {
		t.Fatal("the oldest world should have been evicted")
	}
	if _, ok := store.get(third); !ok {
		t.Fatal("the newest world should still exist")
	}
}
