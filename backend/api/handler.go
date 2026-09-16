// Package api exposes the Scribblebox game over HTTP.
package api

import (
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
	"sync"

	"backend/ai"
	"backend/lexicon"
	"backend/world"
)

const (
	// maxBodyBytes has to fit an AI-generated puzzle spec: terrain for a 60x30
	// grid plus up to 40 entities.
	maxBodyBytes  = 64 << 10
	maxPhraseChar = 60
	maxTicks      = 40
)

// WorldStore keeps bounded, concurrently safe game sessions.
type WorldStore struct {
	mu       sync.Mutex
	worlds   map[string]*world.World
	order    []string
	capacity int
}

// NewWorldStore creates a store that keeps at most capacity worlds alive.
func NewWorldStore(capacity int) *WorldStore {
	if capacity <= 0 {
		capacity = 500
	}
	return &WorldStore{worlds: map[string]*world.World{}, capacity: capacity}
}

func (s *WorldStore) add(w *world.World) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.worlds[w.ID()] = w
	s.order = append(s.order, w.ID())
	for len(s.order) > s.capacity {
		oldest := s.order[0]
		s.order = s.order[1:]
		delete(s.worlds, oldest)
	}
}

func (s *WorldStore) get(id string) (*world.World, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	w, ok := s.worlds[id]
	return w, ok
}

func (s *WorldStore) len() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.worlds)
}

type Handler struct {
	store          *WorldStore
	ai             *ai.Client
	allowedOrigins map[string]bool
}

// New builds the HTTP handler. Origins outside the allowlist are rejected, so a
// page on another site cannot drive a player's world.
func New(store *WorldStore, allowedOrigins []string) http.Handler {
	return NewWithAI(store, allowedOrigins, ai.NewClient())
}

// NewWithAI lets a caller inject the AI client (tests point it at a local fake).
func NewWithAI(store *WorldStore, allowedOrigins []string, client *ai.Client) http.Handler {
	handler := &Handler{store: store, ai: client, allowedOrigins: map[string]bool{}}
	for _, origin := range allowedOrigins {
		if trimmed := strings.TrimSpace(origin); trimmed != "" {
			handler.allowedOrigins[trimmed] = true
		}
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/health", handler.health)
	mux.HandleFunc("GET /api/puzzles", handler.puzzles)
	mux.HandleFunc("GET /api/words", handler.words)
	mux.HandleFunc("POST /api/worlds", handler.createWorld)
	mux.HandleFunc("GET /api/worlds/{id}", handler.getWorld)
	mux.HandleFunc("POST /api/worlds/{id}/spawn", handler.spawn)
	mux.HandleFunc("POST /api/worlds/{id}/step", handler.step)
	mux.HandleFunc("POST /api/worlds/{id}/act", handler.act)
	mux.HandleFunc("POST /api/worlds/{id}/reset", handler.reset)
	mux.HandleFunc("POST /api/worlds/{id}/hint", handler.hint)
	mux.HandleFunc("POST /api/worlds/{id}/hint/choose", handler.chooseHint)
	mux.HandleFunc("POST /api/ai/judge", handler.judge)
	mux.HandleFunc("POST /api/ai/puzzles", handler.generatePuzzle)
	mux.HandleFunc("POST /api/ai/ping", handler.ping)
	mux.HandleFunc("POST /api/ai/hint", handler.hintFromAI)
	mux.HandleFunc("/", handler.notFound)

	return handler.cors(mux)
}

func (h *Handler) cors(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if origin != "" {
			if !h.allowedOrigins[origin] {
				writeError(w, http.StatusForbidden, "origin not allowed")
				return
			}
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
			w.Header().Add("Vary", "Origin")
		}
		if r.Method == http.MethodOptions {
			if origin == "" {
				writeError(w, http.StatusBadRequest, "preflight requests must include an Origin header")
				return
			}
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (h *Handler) health(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"status":  "ok",
		"words":   lexicon.Count(),
		"puzzles": len(world.Puzzles()),
	})
}

func (h *Handler) puzzles(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"puzzles": world.Puzzles()})
}

// words powers the notebook's autocomplete, and deliberately reports that the
// book is finite: a word it does not know still spawns something. When the exact
// word is in the dictionary, `match` carries the shape and colour it is drawn
// with, so the four-choice hint can show the real icon instead of a placeholder.
func (h *Handler) words(w http.ResponseWriter, r *http.Request) {
	query := strings.TrimSpace(r.URL.Query().Get("q"))
	limit := 8
	if raw := r.URL.Query().Get("limit"); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil && parsed > 0 && parsed <= 40 {
			limit = parsed
		}
	}
	matches := lexicon.Suggest(query, limit)
	payload := map[string]any{
		"words":       matches,
		"exact":       lexicon.Knows(query),
		"dictionary":  lexicon.Count(),
		"alwaysWorks": true,
	}
	if object, ok := lexicon.Lookup(strings.ToLower(query)); ok {
		payload["match"] = map[string]any{
			"key":   object.Key,
			"shape": object.Shape,
			"color": object.Color,
			"glyph": object.Glyph,
			"note":  object.Note,
		}
	}
	writeJSON(w, http.StatusOK, payload)
}

func (h *Handler) createWorld(w http.ResponseWriter, r *http.Request) {
	var body struct {
		PuzzleID string          `json:"puzzleId"`
		AISpec   json.RawMessage `json:"aiSpec"`
	}
	if !decodeJSON(w, r, &body) {
		return
	}
	// An AI-authored puzzle arrives as a spec and is validated by the world
	// package before it can be played.
	if len(body.AISpec) > 0 && string(body.AISpec) != "null" {
		var spec world.PuzzleSpec
		if err := json.Unmarshal(body.AISpec, &spec); err != nil {
			writeError(w, http.StatusBadRequest, "aiSpec is not a puzzle: "+err.Error())
			return
		}
		session, err := world.NewFromSpec(spec)
		if err != nil {
			writeError(w, http.StatusBadRequest, "the puzzle was rejected: "+err.Error())
			return
		}
		h.store.add(session)
		writeJSON(w, http.StatusCreated, map[string]any{"state": session.Snapshot()})
		return
	}
	if strings.TrimSpace(body.PuzzleID) == "" {
		writeError(w, http.StatusBadRequest, "puzzleId is required")
		return
	}
	session, err := world.New(strings.TrimSpace(body.PuzzleID))
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	h.store.add(session)
	writeJSON(w, http.StatusCreated, map[string]any{"state": session.Snapshot()})
}

func (h *Handler) getWorld(w http.ResponseWriter, r *http.Request) {
	session, ok := h.store.get(r.PathValue("id"))
	if !ok {
		writeError(w, http.StatusNotFound, "unknown world")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"state": session.Snapshot()})
}

func (h *Handler) spawn(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Phrase string `json:"phrase"`
		X      int    `json:"x"`
		Y      int    `json:"y"`
	}
	if !decodeJSON(w, r, &body) {
		return
	}
	if len([]rune(strings.TrimSpace(body.Phrase))) > maxPhraseChar {
		writeError(w, http.StatusBadRequest, "that name is too long for the notebook (60 characters)")
		return
	}
	session, ok := h.store.get(r.PathValue("id"))
	if !ok {
		writeError(w, http.StatusNotFound, "unknown world")
		return
	}
	spawned, note, approximate, err := session.Spawn(body.Phrase, body.X, body.Y)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"spawned":     spawned,
		"note":        note,
		"approximate": approximate,
		"events":      session.Events(),
		"state":       session.Snapshot(),
	})
}

func (h *Handler) step(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Ticks int `json:"ticks"`
	}
	if !decodeJSON(w, r, &body) {
		return
	}
	if body.Ticks < 1 || body.Ticks > maxTicks {
		writeError(w, http.StatusBadRequest, "ticks must be between 1 and 40")
		return
	}
	session, ok := h.store.get(r.PathValue("id"))
	if !ok {
		writeError(w, http.StatusNotFound, "unknown world")
		return
	}
	session.Step(body.Ticks)
	writeJSON(w, http.StatusOK, map[string]any{"events": session.Events(), "state": session.Snapshot()})
}

func (h *Handler) act(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Action string `json:"action"`
	}
	if !decodeJSON(w, r, &body) {
		return
	}
	session, ok := h.store.get(r.PathValue("id"))
	if !ok {
		writeError(w, http.StatusNotFound, "unknown world")
		return
	}
	if err := session.Act(strings.TrimSpace(body.Action)); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"events": session.Events(), "state": session.Snapshot()})
}

// hint returns the four-choice hint. Exactly one option really solves the puzzle
// at the offered spot, and that promise is proven by simulation inside the world
// package before the options are sent. The correct option is never marked.
func (h *Handler) hint(w http.ResponseWriter, r *http.Request) {
	session, ok := h.store.get(r.PathValue("id"))
	if !ok {
		writeError(w, http.StatusNotFound, "unknown world")
		return
	}
	hint, err := session.Hint()
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"hint": hint, "state": session.Snapshot()})
}

func (h *Handler) reset(w http.ResponseWriter, r *http.Request) {
	session, ok := h.store.get(r.PathValue("id"))
	if !ok {
		writeError(w, http.StatusNotFound, "unknown world")
		return
	}
	session.Reset()
	writeJSON(w, http.StatusOK, map[string]any{"events": session.Events(), "state": session.Snapshot()})
}

func (h *Handler) notFound(w http.ResponseWriter, r *http.Request) {
	if allow := allowedMethodsFor(r.URL.Path); allow != "" {
		w.Header().Set("Allow", allow)
		writeError(w, http.StatusMethodNotAllowed, "method not allowed for this endpoint")
		return
	}
	writeError(w, http.StatusNotFound, "no such endpoint")
}

func allowedMethodsFor(path string) string {
	switch {
	case path == "/api/health", path == "/api/puzzles", path == "/api/words":
		return "GET, OPTIONS"
	case path == "/api/worlds", path == "/api/ai/judge", path == "/api/ai/puzzles", path == "/api/ai/ping", path == "/api/ai/hint":
		return "POST, OPTIONS"
	case strings.HasPrefix(path, "/api/worlds/"):
		if strings.HasSuffix(path, "/spawn") || strings.HasSuffix(path, "/step") || strings.HasSuffix(path, "/act") || strings.HasSuffix(path, "/reset") || strings.HasSuffix(path, "/hint") || strings.HasSuffix(path, "/hint/choose") {
			return "POST, OPTIONS"
		}
		return "GET, OPTIONS"
	default:
		return ""
	}
}

// decodeJSON accepts exactly one JSON object: a valid object followed by more
// JSON or garbage is rejected rather than half-applied.
func decodeJSON(w http.ResponseWriter, r *http.Request, target any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(target); err != nil {
		writeError(w, http.StatusBadRequest, "request body must be a single JSON object")
		return false
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		writeError(w, http.StatusBadRequest, "request body must contain exactly one JSON object")
		return false
	}
	return true
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		log.Printf("api: writing response failed: %v", err)
	}
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}

// logJudgeFailure records that an accepted verdict could not be applied. It never
// touches the player's credentials.
func logJudgeFailure(err error) {
	log.Printf("api: an accepted judge verdict could not be applied: %v", err)
}
