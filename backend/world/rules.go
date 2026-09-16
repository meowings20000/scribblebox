package world

import (
	"fmt"

	"backend/lexicon"
)

// The rule order, which never changes between ticks. It is written down here
// once and followed exactly, because "deterministic" means the same world plus
// the same calls gives the same numbers, forever:
//
//  1. cold      (rule 3)  a cold or frozen thing freezes the water beside it
//  2. fire      (rule 4)  flames spread, burning things burn down, water puts
//     fires out
//  3. cutting   (rule 5)  a sharp thing cuts a rope or a plant
//  4. unlocking (rule 6)  a key opens a container beside it
//  5. blasts    (rule 7)  an explosive next to fire goes off
//  6. power     (rule 8)  batteries light up machines through wires, lifts rise
//  7. settling  (rules 1-2, 13)  gravity, buoyancy, riders on rising lifts
//  8. hovering  (rule 10) magical things ignore one step of gravity
//  9. player    (rule 9, 16) the player falls, walks, climbs
//  10. goal      the win condition is read
//
// Freezing runs before gravity on purpose: an ice cube dropped at the river's
// edge freezes the surface and then rests on it, instead of sinking first.
// Burning runs before gravity so a rope cut free of its branch is already gone
// before the box above it starts to fall.
func (w *World) tickLocked() {
	w.state.ticks++
	w.freezeLocked()
	w.burnLocked()
	w.cutLocked()
	w.unlockLocked()
	w.blastLocked()
	w.powerLocked()
	w.settleLocked()
	w.hoverLocked()
	w.playerLocked()
	w.checkGoalLocked()
}

// freezeLocked is rule 3: an entity with a cold or frozen tag turns every water
// cell it touches into solid ice. This is how freezing a stream builds a bridge.
func (w *World) freezeLocked() {
	for i := range w.state.entities {
		freezer := w.state.entities[i]
		if !hasAnyTag(freezer, lexicon.TagCold, lexicon.TagFrozen) {
			continue
		}
		frozen := 0
		for _, p := range freezer.nearCells(1) {
			if w.terrainAt(p) == TerrainWater {
				w.setTerrain(p, TerrainGround)
				frozen++
			}
		}
		if frozen > 0 {
			w.sayLocked(fmt.Sprintf("the %s freezes the water solid", freezer.Object.Name))
		}
	}
}

// burnLocked is rule 4, in three parts:
//
//	a. a flame sets every flammable thing beside it alight;
//	b. anything burning beside water goes out; anything else that has burned for
//	   three ticks is ash, and its cells become walkable ash terrain;
//	c. a flame standing in water is put out and removed.
func (w *World) burnLocked() {
	// a. spread. A flame sets its neighbours alight, and so does a thing that is
	// already burning: a candle in flames is a fire, which is what lets fire hop
	// along a row of candles. A burning thing that is sitting next to water
	// spreads nothing: it is already going out.
	for i := range w.state.entities {
		flame := w.state.entities[i]
		if !hasTag(flame, lexicon.TagFire) && !flame.Burning {
			continue
		}
		if flame.Burning && w.touchesWater(flame) {
			continue
		}
		for j := range w.state.entities {
			if i == j {
				continue
			}
			target := &w.state.entities[j]
			if target.Burning || !hasTag(*target, lexicon.TagFlammable) || hasTag(*target, lexicon.TagFire) {
				continue
			}
			if !adjacent(flame, *target, 1) {
				continue
			}
			target.Burning = true
			w.sayLocked(fmt.Sprintf("the %s catches fire from the %s", target.Object.Name, flame.Object.Name))
		}
	}

	// b. burn down, or go out in the water. Iterating over ids, not indexes:
	// removing an entity shifts the slice underneath us.
	for _, id := range w.entityIDsLocked() {
		index, ok := w.entityByID(id)
		if !ok {
			continue
		}
		if !w.state.entities[index].Burning {
			continue
		}
		entity := &w.state.entities[index]
		if w.touchesWater(*entity) {
			entity.Burning = false
			delete(w.state.burn, id)
			w.sayLocked(fmt.Sprintf("the fire on the %s goes out in the water", entity.Object.Name))
			continue
		}
		w.state.burn[id]++
		if w.state.burn[id] < burnTicksToAsh {
			continue
		}
		name := entity.Object.Name
		w.turnToAshLocked(*entity)
		w.removeEntity(id)
		w.sayLocked(fmt.Sprintf("the %s burns down to ash", name))
	}

	// c. a flame in the water is only steam.
	putOut := []int{}
	for _, entity := range w.state.entities {
		if hasTag(entity, lexicon.TagFire) && w.touchesWater(entity) {
			putOut = append(putOut, entity.ID)
		}
	}
	for _, id := range putOut {
		index, ok := w.entityByID(id)
		if !ok {
			continue
		}
		name := w.state.entities[index].Object.Name
		w.removeEntity(id)
		w.sayLocked(fmt.Sprintf("the %s is put out by the water", name))
	}
}

// burnTicksToAsh is the documented "3 burning ticks" before a burning thing is
// gone.
const burnTicksToAsh = 3

// touchesWater reports whether any cell within one of the entity's cells is
// water.
func (w *World) touchesWater(e Entity) bool {
	for _, p := range e.nearCells(1) {
		if w.terrainAt(p) == TerrainWater {
			return true
		}
	}
	return false
}

// turnToAshLocked leaves ash where an entity used to be. Walls stay walls:
// nothing in this world breaks a wall, not even a blast.
func (w *World) turnToAshLocked(e Entity) {
	for _, p := range e.cells() {
		if w.inside(p) && !isSolid(w.terrainAt(p)) && w.terrainAt(p) != TerrainWater {
			w.setTerrain(p, TerrainAsh)
		}
	}
}

// cutLocked is rule 5: a cutting or sharp thing severs a rope or a plant beside
// it. A cut thing stops being climbable, a platform and solid, so whatever it
// was holding up starts falling on the next tick without any special case.
func (w *World) cutLocked() {
	for i := range w.state.entities {
		cutter := w.state.entities[i]
		if cutter.Held || !hasAnyTag(cutter, lexicon.TagCutting, lexicon.TagSharp) {
			continue
		}
		for j := range w.state.entities {
			if i == j {
				continue
			}
			target := &w.state.entities[j]
			if target.Cut || !hasAnyTag(*target, lexicon.TagRope, lexicon.TagPlant) {
				continue
			}
			if !adjacent(cutter, *target, 1) {
				continue
			}
			target.Cut = true
			w.sayLocked(fmt.Sprintf("the %s cuts the %s", cutter.Object.Name, target.Object.Name))
		}
	}
}

// unlockLocked is rule 6: a key beside a closed container opens it.
func (w *World) unlockLocked() {
	for i := range w.state.entities {
		key := w.state.entities[i]
		if key.Held || !hasTag(key, lexicon.TagUnlock) {
			continue
		}
		for j := range w.state.entities {
			if i == j {
				continue
			}
			container := &w.state.entities[j]
			if container.Opened || !hasTag(*container, lexicon.TagContainer) {
				continue
			}
			if !adjacent(key, *container, 1) {
				continue
			}
			container.Opened = true
			w.sayLocked(fmt.Sprintf("the %s opens the %s", key.Object.Name, container.Object.Name))
		}
	}
}

// blastLocked is rule 7: an explosive next to a flame goes off, taking out
// everything within two cells except its own kind. Containers are not destroyed
// by a blast, they are blown open, and walls are never touched.
func (w *World) blastLocked() {
	type blast struct {
		index int
		name  string
		key   string
	}
	blasts := []blast{}
	for i, entity := range w.state.entities {
		if entity.Held || !hasTag(entity, lexicon.TagExplosive) {
			continue
		}
		if !entity.Burning && !w.adjacentFlame(i) {
			continue
		}
		blasts = append(blasts, blast{index: i, name: entity.Object.Name, key: entity.Object.Key})
	}
	if len(blasts) == 0 {
		return
	}

	for _, b := range blasts {
		source := w.state.entities[b.index]
		doomed := []int{}
		for j, other := range w.state.entities {
			if other.ID == source.ID || other.Object.Key == b.key {
				continue // the explosion source's own kind is spared
			}
			if !adjacent(source, other, blastRadius) {
				continue
			}
			if hasTag(other, lexicon.TagContainer) {
				if !w.state.entities[j].Opened {
					w.state.entities[j].Opened = true
					w.sayLocked(fmt.Sprintf("the blast blows the %s open", other.Object.Name))
				}
				continue
			}
			doomed = append(doomed, other.ID)
		}
		w.sayLocked(fmt.Sprintf("the %s goes off with a bang", b.name))
		for _, id := range doomed {
			if idx, ok := w.entityByID(id); ok {
				w.turnToAshLocked(w.state.entities[idx])
				w.removeEntity(id)
			}
		}
		w.removeEntity(source.ID)
	}
}

// blastRadius is the Chebyshev radius of rule 7.
const blastRadius = 2

// adjacentFlame reports whether any other entity carrying the fire tag is next
// to the entity at index i.
func (w *World) adjacentFlame(i int) bool {
	self := w.state.entities[i]
	for j, other := range w.state.entities {
		if i == j || !hasTag(other, lexicon.TagFire) {
			continue
		}
		if adjacent(self, other, 1) {
			return true
		}
	}
	return false
}

// powerLocked is rule 8. Power is recomputed from adjacency every tick, never
// remembered, so taking the battery away unpowers the machine it was feeding.
//
// A machine is powered when
//   - a power source is beside it, or
//   - a power source is beside a conductive thing (a wire) that is beside it;
//   - or the machine carries its own power, which is how a jetpack works.
//
// A powered lift rises one cell per tick until something stops it. A lift that
// supplies its own power only rises while it carries a rider: a jetpack that
// flew off on its own would be a very disappointing toy.
func (w *World) powerLocked() {
	for i := range w.state.entities {
		machine := &w.state.entities[i]
		if !hasTag(*machine, lexicon.TagMachine) {
			continue
		}
		powered := w.powerFor(i)
		switch {
		case powered && !machine.Powered:
			machine.Powered = true
			w.sayLocked(fmt.Sprintf("the %s hums to life", machine.Object.Name))
		case !powered && machine.Powered:
			machine.Powered = false
			w.sayLocked(fmt.Sprintf("the %s goes quiet", machine.Object.Name))
		}
		if !machine.Powered || !hasTag(*machine, lexicon.TagLift) {
			continue
		}
		selfPowered := hasTag(*machine, lexicon.TagPowerSource)
		rider, hasRider := w.riderOf(*machine)
		playerRiding := w.playerIsRider(*machine)
		if selfPowered && !hasRider && !playerRiding {
			// A lift that carries its own fuel only rises while it carries a
			// rider: a jetpack that flew off on its own would be a very
			// disappointing toy.
			continue
		}
		w.riseLocked(machine, rider, hasRider, playerRiding)
	}
}

// powerFor decides whether the machine at index i is powered right now.
func (w *World) powerFor(i int) bool {
	machine := w.state.entities[i]
	for j, other := range w.state.entities {
		if i == j || other.Held {
			continue
		}
		if !hasTag(other, lexicon.TagConductive) {
			continue
		}
		if !adjacent(machine, other, 1) {
			continue
		}
		for k, source := range w.state.entities {
			if k == i || k == j || source.Held {
				continue
			}
			if !hasTag(source, lexicon.TagPowerSource) {
				continue
			}
			if adjacent(other, source, 1) || adjacent(machine, source, 1) {
				return true
			}
		}
	}
	return false
}

// riderOf finds the entity or player standing directly on top of a lift.
func (w *World) riderOf(machine Entity) (Entity, bool) {
	for _, other := range w.state.entities {
		if other.ID == machine.ID || other.Held {
			continue
		}
		if other.Y+other.H == machine.Y && overlapsColumns(machine, other) {
			return other, true
		}
	}
	return Entity{}, false
}

// playerIsRider reports whether the player is standing on top of an entity.
func (w *World) playerIsRider(machine Entity) bool {
	return w.state.player.Y+1 == machine.Y && w.state.player.X >= machine.X && w.state.player.X < machine.X+machine.W
}

func overlapsColumns(a, b Entity) bool {
	return a.X < b.X+b.W && b.X < a.X+a.W
}

// riseLocked lifts a machine one cell, carrying whoever is standing on it. It
// stops when something is in the way, which is the documented "rises until it
// hits something".
//
// A machine fed by a separate battery and wire also stops when it would rise out
// of its supply's reach: a lift only goes as far as its cable, so it does not
// climb out of contact and drop back, over and over.
func (w *World) riseLocked(machine *Entity, rider Entity, hasRider, playerRiding bool) {
	if machine.Y-1 < 0 || w.blockedAbove(*machine) {
		return
	}
	if hasTag(*machine, lexicon.TagMachine) && !hasTag(*machine, lexicon.TagPowerSource) && !w.wouldStayPowered(*machine) {
		return
	}
	switch {
	case hasRider:
		if !w.canMoveUp(rider) {
			return
		}
		if idx, ok := w.entityByID(rider.ID); ok {
			w.state.entities[idx].Y--
			w.sayLocked(fmt.Sprintf("the %s carries the %s up", machine.Object.Name, rider.Object.Name))
		}
	case playerRiding:
		if !w.canMoveUpPoint(w.state.player.X, w.state.player.Y) {
			return
		}
		w.state.player.Y--
	}
	machine.Y--
}

// wouldStayPowered reports whether the machine would still be powered one cell
// higher: the reach of its supply.
func (w *World) wouldStayPowered(machine Entity) bool {
	index, ok := w.entityByID(machine.ID)
	if !ok {
		return false
	}
	saved := w.state.entities[index].Y
	w.state.entities[index].Y--
	powered := w.powerFor(index)
	w.state.entities[index].Y = saved
	return powered
}

// blockedAbove reports whether the cell above the machine's top row is taken.
func (w *World) blockedAbove(machine Entity) bool {
	for dx := 0; dx < machine.W; dx++ {
		p := point{machine.X + dx, machine.Y - machine.H}
		if !w.inside(p) || isSolid(w.terrainAt(p)) {
			return true
		}
		if _, taken := w.entityAt(p); taken {
			return true
		}
		if w.state.player.X == p.X && w.state.player.Y == p.Y {
			return true
		}
	}
	return false
}

func (w *World) canMoveUp(e Entity) bool {
	moved := e
	moved.Y--
	for _, c := range moved.cells() {
		if !w.inside(c) || isSolid(w.terrainAt(c)) {
			return false
		}
		if _, taken := w.entityAt(c); taken {
			return false
		}
		if c.X == w.state.player.X && c.Y == w.state.player.Y {
			return false
		}
	}
	return true
}

func (w *World) canMoveUpPoint(x, y int) bool {
	p := point{x, y - 1}
	if !w.inside(p) || isSolid(w.terrainAt(p)) {
		return false
	}
	if _, taken := w.entityAt(p); taken {
		return false
	}
	return true
}

// settleLocked is rules 1 and 2 together: everything falls, floats or sinks one
// cell per tick, bottom of the pile first so a stack settles as a stack. A
// heavy thing crushes a fragile thing directly beneath it; a buoyant thing rests
// on water instead of falling through it.
func (w *World) settleLocked() {
	fell := map[int]bool{}
	for _, id := range w.settleOrderLocked() {
		index, ok := w.entityByID(id)
		if !ok {
			continue // crushed or burnt away earlier in this same tick
		}
		e := w.state.entities[index]
		if e.Held {
			continue
		}
		if w.crushedBy(e) {
			continue
		}
		if hasTag(e, lexicon.TagSticky) && w.touchesSolid(e) {
			continue // sticky things hold on to whatever solid is beside them
		}
		if w.overWater(e) {
			if floaty(e) {
				if w.tryMove(index, 0, -1) {
					fell[e.ID] = true
				}
				continue
			}
			if w.tryMove(index, 0, 1) {
				fell[e.ID] = true
			}
			continue
		}
		if w.supported(e) {
			continue
		}
		if w.tryMove(index, 0, 1) {
			fell[e.ID] = true
		}
	}
	w.state.fell = fell
}

// entityIDsLocked lists every living entity's id, in world order.
func (w *World) entityIDsLocked() []int {
	out := make([]int, 0, len(w.state.entities))
	for _, e := range w.state.entities {
		out = append(out, e.ID)
	}
	return out
}

// settleOrderLocked returns entity ids from the lowest on the grid to the
// highest, so a stack settles from underneath.
func (w *World) settleOrderLocked() []int {
	order := w.entityIDsLocked()
	byID := map[int]int{}
	for _, e := range w.state.entities {
		byID[e.ID] = e.Y
	}
	// insertion sort by bottom row, lowest first; the slice is tiny.
	for i := 1; i < len(order); i++ {
		for j := i; j > 0; j-- {
			if byID[order[j-1]] >= byID[order[j]] {
				break
			}
			order[j-1], order[j] = order[j], order[j-1]
		}
	}
	return order
}

// tryMove shifts an entity by (dx, dy) if every destination cell is free.
func (w *World) tryMove(index, dx, dy int) bool {
	moved := w.state.entities[index]
	moved.X += dx
	moved.Y += dy
	for _, c := range moved.cells() {
		if !w.inside(c) || isSolid(w.terrainAt(c)) {
			return false
		}
		if c.X == w.state.player.X && c.Y == w.state.player.Y {
			return false
		}
		for j, other := range w.state.entities {
			if j == index {
				continue
			}
			if other.occupies(c) {
				return false
			}
		}
	}
	w.state.entities[index] = moved
	return true
}

// overWater reports whether any of the entity's cells is water.
func (w *World) overWater(e Entity) bool {
	for _, c := range e.cells() {
		if w.terrainAt(c) == TerrainWater {
			return true
		}
	}
	return false
}

// supported reports whether nothing should move: the entity has something solid
// (or a standable entity, or the player) directly under at least one of its
// bottom cells, or it is floating on water, or it is a powered lift holding its
// own weight.
func (w *World) supported(e Entity) bool {
	if e.Powered && hasAnyTag(e, lexicon.TagMachine, lexicon.TagLift) && hasTag(e, lexicon.TagLift) {
		// A powered lift is held up by its own motor: that is what makes it a
		// lift rather than a heavy box. Taking the battery away drops it.
		return true
	}
	for dx := 0; dx < e.W; dx++ {
		below := point{e.X + dx, e.Y + 1}
		if !w.inside(below) {
			return true
		}
		if isSolid(w.terrainAt(below)) {
			return true
		}
		if w.terrainAt(below) == TerrainWater && floaty(e) {
			return true
		}
		if below.X == w.state.player.X && below.Y == w.state.player.Y {
			return true
		}
		for j, other := range w.state.entities {
			if other.ID == e.ID {
				continue
			}
			if !other.occupies(below) {
				continue
			}
			if standable(other) {
				return true
			}
			_ = j
		}
	}
	return false
}

// crushedBy is the heavy-versus-fragile rule: a heavy thing destroys the fragile
// thing directly beneath it and keeps going.
func (w *World) crushedBy(e Entity) bool {
	if !hasTag(e, lexicon.TagHeavy) && e.Object.Mass < heavyMass {
		return false
	}
	for dx := 0; dx < e.W; dx++ {
		below := point{e.X + dx, e.Y + 1}
		idx, ok := w.entityAt(below)
		if !ok {
			continue
		}
		victim := w.state.entities[idx]
		if victim.ID == e.ID || !hasTag(victim, lexicon.TagFragile) {
			continue
		}
		w.sayLocked(fmt.Sprintf("the %s crushes the %s", e.Object.Name, victim.Object.Name))
		w.removeEntity(victim.ID)
		return true
	}
	return false
}

// heavyMass is the mass at which an object counts as heavy even without the tag.
const heavyMass = 4

// hoverLocked is rule 10: a magical thing ignores one step of gravity per tick,
// so it floats in place instead of falling. Kept deliberately simple: it undoes
// a fall rather than modelling lift.
func (w *World) hoverLocked() {
	for i := range w.state.entities {
		e := &w.state.entities[i]
		if e.Held || !hasTag(*e, lexicon.TagMagical) {
			continue
		}
		if !w.state.fell[e.ID] {
			continue
		}
		if _, taken := w.entityAt2(point{e.X, e.Y - 1}, e.ID); taken {
			continue
		}
		e.Y--
		delete(w.state.fell, e.ID)
	}
}

// touchesSolid reports whether any solid terrain is within one cell of the
// entity: what the sticky tag holds on to.
func (w *World) touchesSolid(e Entity) bool {
	for _, p := range e.nearCells(1) {
		if isSolid(w.terrainAt(p)) {
			return true
		}
	}
	return false
}

// entityAt2 finds an entity occupying a cell, ignoring one entity id.
func (w *World) entityAt2(p point, ignore int) (int, bool) {
	for i, e := range w.state.entities {
		if e.ID == ignore {
			continue
		}
		if e.occupies(p) {
			return i, true
		}
	}
	return -1, false
}
