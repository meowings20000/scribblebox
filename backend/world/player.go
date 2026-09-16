package world

import (
	"fmt"

	"backend/lexicon"
)

// Player rules: the five actions the frontend can send, the player's own
// gravity, the automatic walker that runs during Step, and the goal referee.
//
// The walker deserves an explanation, because it is the one rule that is not in
// the rule list. Step() has to be able to answer "if I draw this object, does
// the puzzle fall?" without a human at the keyboard: hints are dry-run that way,
// and the test suite replays authored answers that way. So during Step the
// player walks the way a helpful ten-year-old would - toward the goal, up
// anything climbable, and keeping going while in the air - and never does
// anything a player could not do with the five actions themselves.

// Act performs one player action: "left", "right", "jump", "take" or "drop".
func (w *World) Act(action string) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	switch action {
	case "left":
		return w.stepLocked(-1)
	case "right":
		return w.stepLocked(1)
	case "jump":
		return w.jumpLocked()
	case "take":
		return w.takeLocked()
	case "drop":
		return w.dropLocked()
	default:
		return fmt.Errorf("there is no such action as %q: try left, right, jump, take or drop", action)
	}
}

// stepLocked walks the player one cell sideways, into air or water. Solid
// terrain and solid objects are walls: walking into one is simply not a move,
// with an event saying so, rather than an error - a child bumping into a tree
// should not be told the request was bad. Water is slow going: one cell every two
// ticks, which is the documented "slowed but not drowned".
func (w *World) stepLocked(dx int) error {
	target := point{w.state.player.X + dx, w.state.player.Y}
	if !w.inside(target) {
		w.sayLocked("that way is the edge of the world")
		return nil
	}
	if isSolid(w.terrainAt(target)) {
		w.sayLocked(fmt.Sprintf("the %s is solid: you cannot walk into it", terrainName(w.terrainAt(target))))
		return nil
	}
	if idx, ok := w.entityAt(target); ok && blocking(w.state.entities[idx]) {
		w.sayLocked(fmt.Sprintf("the %s is in the way: climb it or go round", w.state.entities[idx].Object.Name))
		return nil
	}
	if w.playerInWaterLocked() && !w.wadeLocked() {
		w.sayLocked("you wade slowly through the water")
		return nil
	}
	w.movePlayerLocked(target, false)
	return nil
}

// movePlayerLocked puts the player in a cell and brings whatever they are
// holding along.
func (w *World) movePlayerLocked(target point, climbing bool) {
	w.state.player.X = target.X
	w.state.player.Y = target.Y
	w.state.player.OnGround = w.playerSupportedLocked()
	if held, ok := w.entityByID(w.state.player.Holding); ok {
		carried := w.state.entities[held]
		carried.X = target.X
		carried.Y = target.Y
		w.state.entities[held] = carried
	}
	if climbing {
		w.sayLocked("you climb up")
	}
}

// wadeLocked halves the player's speed in water, deterministically: every other
// tick they can move.
func (w *World) wadeLocked() bool {
	w.state.wade++
	return w.state.wade%2 == 1
}

// jumpLocked moves the player up one cell if they are standing on something.
// Beside something climbable it is a climb of up to two cells; beside a gap, the
// jump carries the player forward as well, which is how a leap over the corner
// of a stream works.
func (w *World) jumpLocked() error {
	if !w.playerSupportedLocked() {
		w.sayLocked("you are in the air: there is nothing to jump from")
		return nil
	}
	up := point{w.state.player.X, w.state.player.Y - 1}
	if !w.freeForPlayerLocked(up) {
		w.sayLocked("there is no room above you")
		return nil
	}

	// Climbing: something climbable beside or under the player gives a boost.
	if climb, ok := w.climbableNearbyLocked(); ok {
		best := w.state.player
		target := point{up.X, up.Y}
		for step := 0; step < climbHeight; step++ {
			candidate := point{w.state.player.X, w.state.player.Y - 1 - step}
			if !w.freeForPlayerLocked(candidate) {
				break
			}
			target = candidate
		}
		_ = best
		w.movePlayerLocked(target, false)
		w.sayLocked(fmt.Sprintf("you climb the %s", climb.Object.Name))
		return nil
	}

	// A leap across a gap: forward and up in one movement. On flat ground the jump
	// stays a jump; the leap only happens when there is nothing to stand on ahead.
	dx := w.facingLocked()
	forward := point{w.state.player.X + dx, w.state.player.Y}
	forwardUp := point{forward.X, forward.Y - 1}
	if dx != 0 && w.isAGapLocked(forward) && w.freeForPlayerLocked(forwardUp) {
		w.movePlayerLocked(forwardUp, false)
		w.sayLocked("you leap")
		return nil
	}

	w.movePlayerLocked(up, false)
	w.sayLocked("you jump")
	return nil
}

// isAGapLocked reports whether a cell is empty air with nothing to stand on
// underneath: a ledge, a stream, or the edge of a platform.
func (w *World) isAGapLocked(p point) bool {
	if !w.walkableForPlayerLocked(p) || w.blockedForPlayerLocked(p) {
		return false
	}
	below := point{p.X, p.Y + 1}
	if !w.inside(below) {
		return false
	}
	if isSolid(w.terrainAt(below)) {
		return false
	}
	if idx, ok := w.entityAt(below); ok && standable(w.state.entities[idx]) {
		return false
	}
	return true
}

// climbHeight is how many cells a climbable thing can carry the player, per the
// rule "jump climbs up to two cells".
const climbHeight = 2

// climbableNearbyLocked finds a climbable entity the player is beside, standing
// on, or directly under, and which is not cut.
func (w *World) climbableNearbyLocked() (Entity, bool) {
	here := point{w.state.player.X, w.state.player.Y}
	for _, e := range w.state.entities {
		if e.Cut || e.Held || !hasTag(e, lexicon.TagClimbable) {
			continue
		}
		for _, c := range e.cells() {
			if chebyshev(c, here) <= 1 {
				return e, true
			}
		}
	}
	return Entity{}, false
}

// takeLocked picks up an adjacent size-1 object. Hands hold one thing: taking
// something new drops the old one beside the player, and the thing just put down
// is not immediately scooped back up.
func (w *World) takeLocked() error {
	droppedID := 0
	if idx, ok := w.entityByID(w.state.player.Holding); ok {
		dropped := w.state.entities[idx]
		dropped.Held = false
		w.state.entities[idx] = dropped
		w.state.player.Holding = 0
		droppedID = dropped.ID
		w.sayLocked(fmt.Sprintf("you put the %s down", dropped.Object.Name))
	}
	here := point{w.state.player.X, w.state.player.Y}
	// Two passes: everything else first, the thing just dropped only if nothing
	// else is within reach.
	for _, allowDropped := range []bool{false, true} {
		for i, e := range w.state.entities {
			if e.Held || e.Object.Size > 1 || !adjacentToPoint(e, here, 1) {
				continue
			}
			if !allowDropped && e.ID == droppedID {
				continue
			}
			w.state.entities[i].Held = true
			w.state.entities[i].X = here.X
			w.state.entities[i].Y = here.Y
			w.state.player.Holding = e.ID
			w.sayLocked(fmt.Sprintf("you take the %s", e.Object.Name))
			return nil
		}
	}
	w.sayLocked("there is nothing small enough to pick up next to you")
	return nil
}

// dropLocked puts the held object in the nearest free cell beside or above the
// player, or on top of a stack, and lets go.
func (w *World) dropLocked() error {
	index, ok := w.entityByID(w.state.player.Holding)
	if !ok {
		w.sayLocked("your hands are empty")
		return nil
	}
	held := w.state.entities[index]
	held.Held = false
	w.state.player.Holding = 0

	candidates := []point{
		{w.state.player.X, w.state.player.Y},
		{w.state.player.X, w.state.player.Y - 1},
		{w.state.player.X - 1, w.state.player.Y},
		{w.state.player.X + 1, w.state.player.Y},
	}
	for _, candidate := range candidates {
		trial := held
		trial.X, trial.Y = candidate.X, candidate.Y
		if w.footprintSolid(trial) || w.footprintTaken(trial) {
			continue
		}
		w.state.entities[index] = trial
		w.sayLocked(fmt.Sprintf("you drop the %s", held.Object.Name))
		return nil
	}
	// Nowhere free: keep holding it rather than losing it.
	held.Held = true
	w.state.entities[index] = held
	w.state.player.Holding = held.ID
	return fmt.Errorf("there is nowhere to put the %s down", held.Object.Name)
}

func adjacentToPoint(e Entity, p point, radius int) bool {
	for _, c := range e.cells() {
		if chebyshev(c, p) <= radius {
			return true
		}
	}
	return false
}

// facingLocked is the direction the player last walked; the walker faces the
// goal, so a leap goes the way the player was already going.
func (w *World) facingLocked() int {
	if w.state.facing == 0 {
		return 1
	}
	return w.state.facing
}

// playerInWaterLocked reports whether the player is standing in water.
func (w *World) playerInWaterLocked() bool {
	return w.terrainAt(point{w.state.player.X, w.state.player.Y}) == TerrainWater
}

// playerSupportedLocked: the player rests on solid terrain, on a standable
// object, on the bottom of the world, or floats in water (slowed, never
// drowned).
func (w *World) playerSupportedLocked() bool {
	here := point{w.state.player.X, w.state.player.Y}
	if w.terrainAt(here) == TerrainWater {
		return true
	}
	below := point{here.X, here.Y + 1}
	if !w.inside(below) {
		return true
	}
	if isSolid(w.terrainAt(below)) {
		return true
	}
	if idx, ok := w.entityAt(below); ok && standable(w.state.entities[idx]) {
		return true
	}
	return false
}

func (w *World) freeForPlayerLocked(p point) bool {
	if !w.inside(p) || isSolid(w.terrainAt(p)) {
		return false
	}
	if idx, ok := w.entityAt(p); ok && blocking(w.state.entities[idx]) {
		return false
	}
	return true
}

func (w *World) walkableForPlayerLocked(p point) bool {
	return w.inside(p) && isWalkable(w.terrainAt(p))
}

func (w *World) blockedForPlayerLocked(p point) bool {
	if idx, ok := w.entityAt(p); ok && blocking(w.state.entities[idx]) {
		return true
	}
	return false
}

// playerLocked runs the player's own physics and then the walker.
func (w *World) playerLocked() {
	// Gravity: the player falls like anything else.
	if !w.playerSupportedLocked() {
		below := point{w.state.player.X, w.state.player.Y + 1}
		if w.inside(below) && !isSolid(w.terrainAt(below)) && !w.blockedForPlayerLocked(below) {
			w.movePlayerLocked(below, false)
		}
	}
	w.state.player.OnGround = w.playerSupportedLocked()
	w.walkerLocked()
}

// walkerLocked is the automatic play described at the top of this file: one step
// per tick toward the goal, up anything standable, and a drift while airborne so
// a leap lands where a person would expect it to.
func (w *World) walkerLocked() {
	if w.state.solved || w.state.goalMet {
		return
	}
	target, ok := w.walkTargetLocked()
	if !ok {
		return
	}
	dx := 0
	if target.X > w.state.player.X {
		dx = 1
	} else if target.X < w.state.player.X {
		dx = -1
	}
	if dx == 0 {
		return
	}
	w.state.facing = dx

	grounded := w.playerSupportedLocked()
	if grounded && w.playerInWaterLocked() {
		if !w.wadeLocked() {
			return
		}
	}

	forward := point{w.state.player.X + dx, w.state.player.Y}
	if !w.inside(forward) {
		return
	}
	if isSolid(w.terrainAt(forward)) {
		return // the player can climb objects, not cliffs
	}
	if idx, ok := w.entityAt(forward); ok {
		entity := w.state.entities[idx]
		switch {
		case standable(entity):
			if !grounded {
				return // in the air there is nothing to push off
			}
			destination := point{forward.X, entity.Y - entity.H}
			if !w.freeForPlayerLocked(destination) {
				return
			}
			w.movePlayerLocked(destination, false)
			if hasTag(entity, lexicon.TagClimbable) {
				w.sayLocked(fmt.Sprintf("you climb the %s", entity.Object.Name))
			} else {
				w.sayLocked(fmt.Sprintf("you climb onto the %s", entity.Object.Name))
			}
		case blocking(entity):
			return // a wall of a thing: go round it
		default:
			w.movePlayerLocked(forward, false) // walked over: a cake is not a step
		}
		return
	}
	w.movePlayerLocked(forward, false)
}

// walkTargetLocked is where the walker is heading: the goal object, or the far
// bank for a crossing.
func (w *World) walkTargetLocked() (point, bool) {
	switch w.plan.kind {
	case GoalStar:
		if idx, ok := w.entityByID(w.plan.starID); ok {
			return point{w.state.entities[idx].X, w.state.entities[idx].Y}, true
		}
	case GoalChest:
		if idx, ok := w.entityByID(w.plan.chestID); ok {
			return point{w.state.entities[idx].X, w.state.entities[idx].Y}, true
		}
	case GoalCandles:
		for _, id := range w.plan.candleIDs {
			if idx, ok := w.entityByID(id); ok {
				return point{w.state.entities[idx].X, w.state.entities[idx].Y}, true
			}
		}
	case GoalCross:
		return point{w.plan.waterMaxX + 1, w.plan.bankY}, true
	}
	return point{}, false
}

// evaluateGoalLocked is the referee: it answers whether the puzzle's win
// condition holds right now, and says so in words the player can read.
func (w *World) evaluateGoalLocked() (bool, string) {
	switch w.plan.kind {
	case GoalStar:
		return w.evaluateStarLocked()
	case GoalCross:
		return w.evaluateCrossLocked()
	case GoalCandles:
		return w.evaluateCandlesLocked()
	case GoalChest:
		return w.evaluateChestLocked()
	}
	if w.state.judged {
		return true, judgeProgress(w.state.judge)
	}
	return false, "nothing here can be won"
}

func (w *World) evaluateStarLocked() (bool, string) {
	index, ok := w.entityByID(w.plan.starID)
	if !ok {
		if w.state.judged {
			return true, judgeProgress(w.state.judge)
		}
		return false, "the star is gone"
	}
	star := w.state.entities[index]
	if star.Held || w.state.player.Holding == star.ID {
		return true, "you are holding the star"
	}
	here := point{w.state.player.X, w.state.player.Y}
	for _, c := range star.cells() {
		if chebyshev(c, here) <= 1 {
			return true, "the star is right beside you"
		}
	}
	return false, fmt.Sprintf("the star is %d cells away - get closer or bring it down", chebyshev(here, point{star.X, star.Y}))
}

func (w *World) evaluateCrossLocked() (bool, string) {
	here := point{w.state.player.X, w.state.player.Y}
	below := point{here.X, here.Y + 1}
	onGround := w.inside(below) && isSolid(w.terrainAt(below))
	if w.plan.farSide > 0 {
		if here.X > w.plan.waterMaxX && onGround {
			return true, "you are standing on the far bank"
		}
		return false, fmt.Sprintf("the far bank is %d cells away across the water", w.plan.waterMaxX+1-here.X)
	}
	if here.X < w.plan.waterMinX && onGround {
		return true, "you are standing on the far bank"
	}
	return false, "the far bank is across the water"
}

func (w *World) evaluateCandlesLocked() (bool, string) {
	lit := 0
	for _, id := range w.plan.candleIDs {
		index, ok := w.entityByID(id)
		if !ok {
			lit++ // burnt right down: that counts as lit
			continue
		}
		candle := w.state.entities[index]
		if candle.Burning || w.litBy(w.state.entities[index]) {
			lit++
		}
	}
	total := len(w.plan.candleIDs)
	if lit == total && total > 0 {
		return true, fmt.Sprintf("all %d candles are lit", total)
	}
	return false, fmt.Sprintf("%d of %d candles are lit", lit, total)
}

// litBy reports whether a flame or a light source is beside this entity.
func (w *World) litBy(e Entity) bool {
	for _, other := range w.state.entities {
		if other.ID == e.ID {
			continue
		}
		if !hasAnyTag(other, lexicon.TagFire, lexicon.TagLightSource) {
			continue
		}
		if adjacent(e, other, 1) {
			return true
		}
	}
	return false
}

func (w *World) evaluateChestLocked() (bool, string) {
	index, ok := w.entityByID(w.plan.chestID)
	if !ok {
		if w.state.judged {
			return true, judgeProgress(w.state.judge)
		}
		return false, "the chest is gone"
	}
	if w.state.entities[index].Opened {
		return true, "the chest is open"
	}
	return false, "the chest is still locked"
}

// judgeProgress is the wording for a solution the rule engine could not simulate
// but a judge accepted.
func judgeProgress(reason string) string {
	return fmt.Sprintf("you solved it and the judge agreed: %s", reason)
}

// goalTitle is the human-readable name of a goal kind.
func goalTitle(kind string) string {
	switch kind {
	case GoalStar:
		return "Get the star out of the tree"
	case GoalCross:
		return "Cross the river"
	case GoalCandles:
		return "Light the three candles"
	case GoalChest:
		return "Open the locked chest"
	}
	return "Finish the puzzle"
}

// terrainName turns a terrain code into words for error messages.
func terrainName(code int) string {
	switch code {
	case TerrainGround:
		return "ground"
	case TerrainTrunk:
		return "a tree trunk"
	case TerrainWall:
		return "wall"
	case TerrainWater:
		return "water"
	case TerrainAsh:
		return "ash"
	}
	return "air"
}
