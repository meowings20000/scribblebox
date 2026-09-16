package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"backend/ai"
	"backend/world"
)

const (
	maxQuestionChars = 500
	maxThemeChars    = 120
)

// goalKinds the engine can referee. An AI-generated puzzle is only accepted if
// it uses one of these, so every puzzle stays winnable and checkable.
var refereeableGoalKinds = []string{"star", "cross", "candles", "chest"}

// aiCredentials arrive from the player's browser on every AI call: they are used
// for that request only, never stored on the server and never logged.
type aiCredentials struct {
	BaseURL        string `json:"baseUrl"`
	APIKey         string `json:"apiKey"`
	Model          string `json:"model"`
	TimeoutSeconds int    `json:"timeoutSeconds"`
}

func (c aiCredentials) config() ai.Config {
	return ai.Config{
		BaseURL:        c.BaseURL,
		APIKey:         c.APIKey,
		Model:          c.Model,
		TimeoutSeconds: c.TimeoutSeconds,
	}
}

// judge lets a model rule on an idea the rule engine cannot referee.
func (h *Handler) judge(w http.ResponseWriter, r *http.Request) {
	var body struct {
		aiCredentials
		WorldID  string `json:"worldId"`
		Question string `json:"question"`
	}
	if !decodeJSON(w, r, &body) {
		return
	}
	if h.ai == nil {
		writeError(w, http.StatusServiceUnavailable, "this server has no AI client configured")
		return
	}
	session, ok := h.store.get(strings.TrimSpace(body.WorldID))
	if !ok {
		writeError(w, http.StatusNotFound, "unknown world")
		return
	}
	question := strings.TrimSpace(body.Question)
	if question == "" {
		writeError(w, http.StatusBadRequest, "describe your idea before asking the judge")
		return
	}
	if len([]rune(question)) > maxQuestionChars {
		writeError(w, http.StatusBadRequest, fmt.Sprintf("keep the question under %d characters", maxQuestionChars))
		return
	}
	config := body.aiCredentials.config()
	if err := config.Validate(); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	state := session.Snapshot()
	verdict, err := h.ai.Judge(r.Context(), config, ai.JudgeInput{
		PuzzleTitle: state.Puzzle.Title,
		Brief:       state.Puzzle.Brief,
		GoalKind:    state.Puzzle.GoalKind,
		Legend:      propertyLegend(),
		Objects:     objectSummaries(state),
		Events:      lastEvents(state.Events, 6),
		Question:    question,
	})
	if err != nil {
		status, message := aiFailure(err)
		writeJSON(w, status, map[string]any{"error": message, "usedAi": true, "model": config.Model})
		return
	}
	if verdict.Approved {
		// A judge may accept a solution the engine cannot simulate. If the world
		// is already solved the verdict still stands, it just changes nothing.
		if err := session.AcceptJudge(verdict.Reason); err != nil {
			logJudgeFailure(err)
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"verdict": verdict,
		"state":   session.Snapshot(),
		"usedAi":  true,
		"model":   config.Model,
	})
}

// generatePuzzle asks a model for a new puzzle and refuses to hand the player
// anything the engine cannot referee.
func (h *Handler) generatePuzzle(w http.ResponseWriter, r *http.Request) {
	var body struct {
		aiCredentials
		Theme    string `json:"theme"`
		GoalKind string `json:"goalKind"`
	}
	if !decodeJSON(w, r, &body) {
		return
	}
	if h.ai == nil {
		writeError(w, http.StatusServiceUnavailable, "this server has no AI client configured")
		return
	}
	goalKind := strings.TrimSpace(body.GoalKind)
	if goalKind == "" {
		goalKind = "star"
	}
	if !refereeable(goalKind) {
		writeError(w, http.StatusBadRequest, "goalKind must be one of "+strings.Join(refereeableGoalKinds, ", "))
		return
	}
	theme := strings.TrimSpace(body.Theme)
	if len([]rune(theme)) > maxThemeChars {
		writeError(w, http.StatusBadRequest, fmt.Sprintf("keep the theme under %d characters", maxThemeChars))
		return
	}
	config := body.aiCredentials.config()
	if err := config.Validate(); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	generated, err := h.ai.GeneratePuzzle(r.Context(), config, ai.PuzzleRequest{Theme: theme, GoalKind: goalKind})
	if err != nil {
		status, message := aiFailure(err)
		writeJSON(w, status, map[string]any{"error": message, "usedAi": true, "model": config.Model})
		return
	}

	var spec world.PuzzleSpec
	if err := json.Unmarshal(generated.Spec, &spec); err != nil {
		writeJSON(w, http.StatusBadGateway, map[string]any{
			"error":  "the AI's puzzle could not be read as a puzzle: " + err.Error(),
			"usedAi": true, "model": config.Model,
		})
		return
	}
	if spec.GoalKind != goalKind {
		writeJSON(w, http.StatusBadGateway, map[string]any{
			"error":  fmt.Sprintf("the AI asked for goal kind %q but sent a %q puzzle", goalKind, spec.GoalKind),
			"usedAi": true, "model": config.Model,
		})
		return
	}
	// The world package is the referee: if it will not build the puzzle, the
	// player never sees a broken one.
	if _, err := world.NewFromSpec(spec); err != nil {
		writeJSON(w, http.StatusBadGateway, map[string]any{
			"error":  "the AI's puzzle was rejected: " + err.Error(),
			"usedAi": true, "model": config.Model,
		})
		return
	}

	writeJSON(w, http.StatusCreated, map[string]any{
		"spec":   spec,
		"notes":  generated.Notes,
		"usedAi": true,
		"model":  generated.Model,
	})
}

// ping proves the player's settings work before they rely on them. It is the one
// place the game talks to the endpoint just to see whether it answers.
func (h *Handler) ping(w http.ResponseWriter, r *http.Request) {
	var body struct {
		aiCredentials
	}
	if !decodeJSON(w, r, &body) {
		return
	}
	if h.ai == nil {
		writeError(w, http.StatusServiceUnavailable, "this server has no AI client configured")
		return
	}
	config := body.aiCredentials.config()
	if err := config.Validate(); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	started := time.Now()
	answer, err := h.ai.Chat(r.Context(), config,
		"You are a connection test for a game's settings panel. Answer with one short word.",
		"Reply with the single word: ready", 0)
	if err != nil {
		status, message := aiFailure(err)
		writeJSON(w, status, map[string]any{"error": message, "usedAi": true, "model": config.Model})
		return
	}
	reply := strings.TrimSpace(answer)
	if runes := []rune(reply); len(runes) > 80 {
		reply = string(runes[:80]) + "…"
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"ok":        true,
		"usedAi":    true,
		"model":     config.Model,
		"reply":     reply,
		"latencyMs": time.Since(started).Milliseconds(),
	})
}

// hintFromAI asks the model for four themed choices and then proves them by
// simulation. The model only supplies ideas: whether an option really solves the
// puzzle is decided by the engine, so exactly one option is correct no matter
// what was answered.
func (h *Handler) hintFromAI(w http.ResponseWriter, r *http.Request) {
	var body struct {
		aiCredentials
		WorldID string `json:"worldId"`
	}
	if !decodeJSON(w, r, &body) {
		return
	}
	if h.ai == nil {
		writeError(w, http.StatusServiceUnavailable, "this server has no AI client configured")
		return
	}
	session, ok := h.store.get(strings.TrimSpace(body.WorldID))
	if !ok {
		writeError(w, http.StatusNotFound, "unknown world")
		return
	}
	config := body.aiCredentials.config()
	if err := config.Validate(); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	// The engine's own hint gives us a placement and a pool of phrases whose
	// behaviour is already known: one solves the puzzle, three provably do not.
	base, err := session.Hint()
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	baseSolving, baseFailing, err := trialClassify(session, base.Options, base.Placement)
	if err != nil {
		writeError(w, http.StatusBadGateway, "the engine could not check the hint spot: "+err.Error())
		return
	}
	if len(baseSolving) == 0 {
		writeError(w, http.StatusBadGateway, "the engine found no answer that settles this puzzle right now: try Reset")
		return
	}

	suggested, rationale, err := h.ai.SuggestHint(r.Context(), config, ai.HintInput{
		PuzzleTitle: session.Snapshot().Puzzle.Title,
		Brief:       session.Snapshot().Puzzle.Brief,
		GoalKind:    session.Snapshot().Puzzle.GoalKind,
		Objects:     objectSummaries(session.Snapshot()),
	})
	if err != nil {
		status, message := aiFailure(err)
		writeJSON(w, status, map[string]any{"error": message, "usedAi": true, "model": config.Model})
		return
	}

	solving, failing, err := trialClassify(session, suggested, base.Placement)
	if err != nil {
		writeError(w, http.StatusBadGateway, "the engine could not check the AI's choices: "+err.Error())
		return
	}

	options := []string{}
	if len(solving) > 0 {
		options = append(options, solving[0]) // prefer the AI's own correct idea
		suggested = suggested[1:]
	} else {
		options = append(options, baseSolving[0])
	}
	// Fill the wrong slots from the AI's rejected ideas first, then from the
	// engine's verified decoys. Duplicates and the correct phrase never slip in.
	seen := map[string]bool{options[0]: true}
	for _, phrase := range append(append([]string{}, failing...), append(append([]string{}, suggested...), baseFailing...)...) {
		if len(options) == 4 {
			break
		}
		lowered := strings.ToLower(strings.TrimSpace(phrase))
		if lowered == "" || seen[lowered] {
			continue
		}
		seen[lowered] = true
		options = append(options, phrase)
	}
	if len(options) != 4 {
		writeError(w, http.StatusBadGateway, "the AI's choices could not be turned into four verified options: ask again")
		return
	}

	// Belt and braces: re-verify the final four exactly the way a player will use
	// them, and refuse to answer if more than one would solve the puzzle.
	finalSolving, _, err := trialClassify(session, options, base.Placement)
	if err != nil {
		writeError(w, http.StatusBadGateway, "the engine could not verify the final choices: "+err.Error())
		return
	}
	if len(finalSolving) != 1 {
		writeError(w, http.StatusBadGateway, "the engine refused to offer a hint where more than one choice would work")
		return
	}

	if strings.TrimSpace(rationale) == "" {
		rationale = base.Rationale
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"hint": world.HintSet{
			Placement: base.Placement,
			Options:   rotateForWorld(options, session.ID(), session.Snapshot().Ticks),
			Rationale: rationale,
		},
		"state":  session.Snapshot(),
		"usedAi": true,
		"model":  config.Model,
	})
}

// trialClassify dry-runs every phrase at the hint spot and splits them into the
// ones that solve the puzzle and the ones that do not. The live world is never
// touched.
func trialClassify(session *world.World, phrases []string, placement world.HintPlacement) (solving, failing []string, err error) {
	for _, phrase := range phrases {
		trimmed := strings.TrimSpace(phrase)
		if trimmed == "" {
			continue
		}
		solved, trialErr := session.Trial(trimmed, placement.X, placement.Y)
		if trialErr != nil {
			continue // something that will not even spawn there cannot be an option
		}
		if solved {
			solving = append(solving, trimmed)
		} else {
			failing = append(failing, trimmed)
		}
	}
	return solving, failing, nil
}

// rotateForWorld moves the answer out of the first slot deterministically, so the
// correct choice is not always option one and the same world stays stable.
func rotateForWorld(options []string, id string, ticks int) []string {
	if len(options) == 0 {
		return options
	}
	seed := uint32(0)
	for index := 0; index < len(id); index++ {
		seed = seed*31 + uint32(id[index])
	}
	seed = seed*31 + uint32(ticks)
	rotation := int(seed % uint32(len(options)))
	out := make([]string, 0, len(options))
	out = append(out, options[rotation:]...)
	out = append(out, options[:rotation]...)
	return out
}

// hintSettleTicks is how long the world is run after a choice is placed. The
// engine's own verification allows the same window (Trial steps forty ticks), so
// the state handed back already shows whether the choice worked — no more pressing
// "advance time" and guessing.
const hintSettleTicks = 40

// chooseHint places a chosen option at a spot that is verified for the world as it
// stands right now. The four choices are proven when they are offered, but the
// player keeps playing — the walker moves, stacks grow — so the engine re-derives
// the spot before placing the object. That is what keeps "one of these works" true
// at the moment the player presses it, and it is why a choice is never stranded on
// a spot that has filled up in the meantime.
func (h *Handler) chooseHint(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Phrase string `json:"phrase"`
	}
	if !decodeJSON(w, r, &body) {
		return
	}
	phrase := strings.TrimSpace(body.Phrase)
	if phrase == "" {
		writeError(w, http.StatusBadRequest, "no choice was sent")
		return
	}
	if len([]rune(phrase)) > maxPhraseChar {
		writeError(w, http.StatusBadRequest, fmt.Sprintf("keep the choice under %d characters", maxPhraseChar))
		return
	}
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
	placed, note, approximate, err := session.Spawn(phrase, hint.Placement.X, hint.Placement.Y)
	movedSpot := false
	if err != nil {
		// The verified spot can be full as the world stands now. Rather than strand
		// the player, drop the choice above them and say that the spot moved.
		player := session.Snapshot().Player
		if placed, note, approximate, err = session.Spawn(phrase, player.X, player.Y); err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		movedSpot = true
	}

	// Let the world settle the way the verification did, so the answer is already
	// on screen when the player presses a choice.
	session.Step(hintSettleTicks)

	state := session.Snapshot()
	// Offer four fresh choices for the world as it stands after the choice, unless
	// the puzzle is won or the spot moved (in which case the old options no longer
	// describe anything).
	fresh := world.HintSet{}
	if !state.Solved && !movedSpot {
		if candidate, hintErr := session.Hint(); hintErr == nil {
			fresh = candidate
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"spawned":     placed,
		"note":        note,
		"approximate": approximate,
		"movedSpot":   movedSpot,
		"events":      session.Events(),
		"hint":        fresh,
		"state":       state,
	})
}

// aiFailure maps the AI client's error kinds onto honest HTTP answers.
func aiFailure(err error) (int, string) {
	var apiError *ai.Error
	if errors.As(err, &apiError) {
		switch apiError.Kind {
		case "config":
			return http.StatusBadRequest, apiError.Error()
		case "timeout":
			return http.StatusGatewayTimeout, apiError.Error()
		case "auth", "server", "protocol", "unreachable":
			return http.StatusBadGateway, apiError.Error()
		}
	}
	return http.StatusBadGateway, "the AI request failed: " + err.Error()
}

func refereeable(goalKind string) bool {
	for _, candidate := range refereeableGoalKinds {
		if candidate == goalKind {
			return true
		}
	}
	return false
}

// propertyLegend explains the tags the engine acts on, so the judge can reason
// about an object the same way the simulation does.
func propertyLegend() []string {
	return []string{
		"flammable = catches fire", "fire = is a flame", "light-source = lights candles",
		"cold/frozen = freezes water into ice", "sharp/cutting = cuts ropes and plants",
		"climbable = a player can climb it", "platform/solid = solid and walkable",
		"rope = can be cut and climbed", "buoyant = floats", "heavy = sinks and crushes fragile things",
		"conductive = carries power", "power-source = supplies power", "machine/lift = a powered machine that rises",
		"unlock = opens locked containers", "explosive = blows up near fire", "edible = food",
		"container = can hold a small object", "collectible = a prize", "magical = bends the rules",
	}
}

func objectSummaries(state world.State) []ai.ObjectSummary {
	summaries := []ai.ObjectSummary{}
	for _, entity := range state.Entities {
		summaries = append(summaries, ai.ObjectSummary{
			Name:        entity.Object.Name,
			Key:         entity.Object.Key,
			Tags:        entity.Object.Tags,
			Approximate: entity.Object.Approximate,
		})
		if len(summaries) >= 24 {
			break
		}
	}
	// Objects the player has already picked up or burned away are part of the
	// story too, so add anything the notebook remembers but the world no longer holds.
	seen := map[string]bool{}
	for _, summary := range summaries {
		seen[summary.Key] = true
	}
	for _, entry := range state.Notebook {
		if seen[entry.Key] || len(summaries) >= 32 {
			continue
		}
		seen[entry.Key] = true
		summaries = append(summaries, ai.ObjectSummary{Name: entry.Phrase, Key: entry.Key, Approximate: entry.Approximate})
	}
	return summaries
}

func lastEvents(events []string, limit int) []string {
	if len(events) <= limit {
		return events
	}
	return events[len(events)-limit:]
}
