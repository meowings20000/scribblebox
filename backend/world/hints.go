package world

import (
	"errors"
	"fmt"
	"strings"
)

// The four-choice hint. The promise is that exactly one of the four phrases
// solves the puzzle at the offered spot, and the promise is kept by simulation:
// every option is dry-run in a sandbox copy before it is offered, so a hint can
// never lie and the answer never has to be marked.
//
// Nothing here tells the client which option is right. The options are shuffled
// into a slot that depends on the world's id and state, so the same world asked
// twice gets the same four options in the same order, and two different worlds
// rarely agree about where the answer sits.

// hintOptions is the pinned four-option shape.
const hintOptions = 4

// Hint builds a four-choice hint for the world's current state. It tries the
// authored spot first and then a widening ring of nearby cells, so a player who
// has already drawn something at the best place still gets help instead of a
// refusal. It returns a clear error rather than a hint it cannot stand behind.
func (w *World) Hint() (HintSet, error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.state.solved || w.state.goalMet {
		return HintSet{}, errors.New("this puzzle is already solved: there is nothing left to hint at")
	}
	if len(w.hint.decoys) == 0 || (w.hint.placement.X == 0 && w.hint.placement.Y == 0) {
		return HintSet{}, errors.New("this puzzle arrived without hints: ask the notebook to draw something and see what happens")
	}

	var lastErr error
	for _, placement := range w.hintSpotsLocked() {
		set, err := w.hintAtLocked(placement)
		if err == nil {
			return set, nil
		}
		lastErr = err
	}
	return HintSet{}, fmt.Errorf("no spot in this puzzle can be verified for a hint right now (%v): start again with Reset, or keep drawing", lastErr)
}

// hintSpotsLocked lists the places a hint object may be dropped, best first: the
// authored spot, then a widening ring of nearby cells, and finally the cell the
// player is standing in (objects come to rest on top of whatever is there). The
// order is fixed, so the same world always gets the same help.
func (w *World) hintSpotsLocked() []HintPlacement {
	spots := []HintPlacement{}
	seen := map[HintPlacement]bool{}
	add := func(x, y int) {
		if x < 0 || y < 0 || x >= w.puzzle.Width || y >= w.puzzle.Height {
			return
		}
		placement := HintPlacement{X: x, Y: y}
		if seen[placement] {
			return
		}
		seen[placement] = true
		spots = append(spots, placement)
	}

	authored := w.hint.placement
	add(authored.X, authored.Y)
	for radius := 1; radius <= 4; radius++ {
		add(authored.X-radius, authored.Y)
		add(authored.X+radius, authored.Y)
		add(authored.X, authored.Y-radius)
		add(authored.X, authored.Y+radius)
		add(authored.X-radius, authored.Y-radius)
		add(authored.X+radius, authored.Y-radius)
		add(authored.X-radius, authored.Y+radius)
		add(authored.X+radius, authored.Y+radius)
	}
	add(w.state.player.X, w.state.player.Y)
	return spots
}

// hintAtLocked builds the hint at one particular spot, or reports why it cannot.
func (w *World) hintAtLocked(placement HintPlacement) (HintSet, error) {
	correct, err := w.verifiedAnswerLocked(placement)
	if err != nil {
		return HintSet{}, err
	}

	verifiedWrong := []string{}
	for _, decoy := range w.hint.decoys {
		if len(verifiedWrong) == hintOptions-1 {
			break
		}
		solved, err := w.trialLocked(decoy, placement)
		if err != nil {
			continue // something that will not even spawn here is not an option
		}
		if solved {
			continue // it solves the puzzle: never offer two correct options
		}
		verifiedWrong = append(verifiedWrong, decoy)
	}
	if len(verifiedWrong) != hintOptions-1 {
		return HintSet{}, fmt.Errorf("only %d of the %d decoys could be verified at (%d, %d)", len(verifiedWrong), hintOptions-1, placement.X, placement.Y)
	}

	options := append([]string{correct}, verifiedWrong...)
	// Self-check: exactly one option may solve the puzzle. This is the promise,
	// proven rather than asserted.
	solving, err := w.countSolvingLocked(options, placement)
	if err != nil {
		return HintSet{}, err
	}
	if solving != 1 {
		return HintSet{}, fmt.Errorf("the hint would be dishonest (%d of the options solve it), so there is no hint", solving)
	}

	return HintSet{
		Placement: placement,
		Options:   rotateIntoPlace(options, w.slotSeedLocked(placement)),
		Rationale: w.hint.rationale,
	}, nil
}

// verifiedAnswerLocked finds the first authored answer whose opening phrase
// provably solves the puzzle from the offered spot, in the current state. The
// first phrase is used because a hint is meant to be one object, not a recipe.
func (w *World) verifiedAnswerLocked(placement HintPlacement) (string, error) {
	for _, solution := range w.puzzle.Solutions {
		if len(solution.Phrases) == 0 {
			continue
		}
		phrase := strings.TrimSpace(solution.Phrases[0].Phrase)
		if phrase == "" {
			continue
		}
		solved, err := w.trialLocked(phrase, placement)
		if err != nil {
			continue
		}
		if solved {
			return phrase, nil
		}
	}
	return "", errors.New("no verified answer settles this puzzle from the hint spot as it stands: start again with Reset, or keep drawing")
}

// countSolvingLocked counts how many of the options solve the puzzle.
func (w *World) countSolvingLocked(options []string, placement HintPlacement) (int, error) {
	solving := 0
	for _, phrase := range options {
		solved, err := w.trialLocked(phrase, placement)
		if err != nil {
			return 0, err
		}
		if solved {
			solving++
		}
	}
	return solving, nil
}

// trialLocked dry-runs a phrase at a placement in a sandbox, without touching
// the live world.
func (w *World) trialLocked(phrase string, placement HintPlacement) (bool, error) {
	sandbox := w.sandboxLocked()
	if _, _, _, err := sandbox.spawnLocked(phrase, placement.X, placement.Y, false); err != nil {
		return false, err
	}
	sandbox.Step(trialTicks)
	return sandbox.Snapshot().Solved, nil
}

// slotSeedLocked is a small deterministic number derived from the world's id, its
// current state and the spot being hinted at. The same state always yields the
// same slot; a changed state or a different spot may move the answer somewhere
// else, which is fine and even helpful.
func (w *World) slotSeedLocked(placement HintPlacement) uint32 {
	seed := hashString(w.id)
	seed = seed*31 + hashString(w.puzzle.ID)
	seed = seed*31 + uint32(w.state.nextID)
	seed = seed*31 + uint32(w.state.ticks)
	seed = seed*31 + uint32(len(w.state.entities))
	seed = seed*31 + uint32(placement.X)
	seed = seed*31 + uint32(placement.Y)
	return seed
}

// rotateIntoPlace moves the answer into a slot decided by seed, so that it is
// not always option 0.
func rotateIntoPlace(options []string, seed uint32) []string {
	if len(options) == 0 {
		return options
	}
	rotation := int(seed % uint32(len(options)))
	out := make([]string, 0, len(options))
	out = append(out, options[rotation:]...)
	out = append(out, options[:rotation]...)
	return out
}

// hashString is FNV-1a: stable across runs and platforms, which matters because
// a world's hint must be the same before and after a reload.
func hashString(s string) uint32 {
	var h uint32 = 2166136261
	for i := 0; i < len(s); i++ {
		h ^= uint32(s[i])
		h *= 16777619
	}
	return h
}
