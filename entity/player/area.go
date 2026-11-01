package player

import (
	"math"

	"th10/zun"
)

// damageArea is one of the player's 32 damage areas (+0x350c, 0x6c bytes
// each). Only the circle the respawn leaves is ported; the rectangles the
// bombs use come with M7.
type damageArea struct {
	radius, growth float32    // +0x00, +0x04
	pos            [2]float32 // +0x18
	timer          zun.Timer  // +0x44: frames left
	damage         int32      // +0x58: per hit, zeroed at the cap
	accum, cap     int32      // +0x5c, +0x60
	interval       int32      // +0x64
	active         bool       // +0x68 bit 0 (bit 1, circle, always set here)
}

// addArea starts a circle of radius r at (x, y), growing by growth a frame
// for n frames and dealing damage to each enemy it covers (FUN_00427b50).
func (p *Player) addArea(x, y, r, growth float32, n, damage int32) {
	for i := range p.areas {
		a := &p.areas[i]
		if a.active {
			continue
		}
		*a = damageArea{radius: r, growth: growth, pos: [2]float32{x, y},
			damage: damage, cap: 999999, interval: 4, active: true}
		a.timer.Set(n)
		return
	}
}

// updateAreas is the area loop in FUN_00425730's tail: move (no velocity
// here) and snap to the 0.01 grid (FUN_0044c2a0), grow, count down.
func (p *Player) updateAreas(gs float32) {
	for i := range p.areas {
		a := &p.areas[i]
		if !a.active {
			continue
		}
		a.pos[0], a.pos[1] = round01(a.pos[0]), round01(a.pos[1])
		a.radius += a.growth
		a.timer.Add(-1, gs)
		if a.timer.Cur <= 0 {
			a.active = false
		}
	}
}

// round01 snaps a coordinate down to the 0.01 grid: floor(v*100) * 0.01
// (FUN_00452ff0, as FUN_0044c2a0 applies it to the ECL positions).
func round01(v float32) float32 {
	return float32(float32(math.Floor(float64(float32(v*100)))) * 0.01)
}

// AreaDamage is the area half of FUN_00428630: the damage the areas deal to
// an enemy at (ex, ey), in game coordinates, this frame. An area skips the
// frames its timer lands on a multiple of its interval; each hit adds to its
// running total and the area stops dealing damage at its cap.
func (p *Player) AreaDamage(ex, ey float32) int {
	total := int32(0)
	for i := range p.areas {
		a := &p.areas[i]
		if !a.active || (a.timer.Cur != a.timer.Prev && a.timer.Cur%a.interval == 0) {
			continue
		}
		dx, dy := a.pos[0]-ex, a.pos[1]-ey
		if float32(dy*dy)+float32(dx*dx) <= float32(a.radius*a.radius) {
			total += a.damage
			a.accum += a.damage
			if a.cap <= a.accum {
				a.damage = 0
			}
		}
	}
	return int(total)
}
