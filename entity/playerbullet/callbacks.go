package playerbullet

import (
	"math"

	"th10/entity/enemy"
	"th10/zun"
)

// initHoming is FUN_00428ad0, init-table index 1 at 0x474770. It copies
// the player's cached target even for Marisa A's non-homing rockets.
func (b *Bullet) initHoming(target *enemy.Enemy) {
	b.target = target
	if target != nil && float32(math.Abs(float64(float32(target.X-gameOriginX)))) > 224 {
		b.target = nil // 0x428ae4-0x428af9
	}
}

// tickHoming is FUN_00428b10, update-table index 1 at 0x47477c. Speed
// changes are per invocation, not scaled by game speed. Age is R+4.
func (b *Bullet) tickHoming() {
	if b.state == 2 { // 0x428b14-0x428b18
		return
	}
	if !HomingTargetable(b.target) {
		b.target = nil
	}
	speed := float32(b.Speed)
	if b.target == nil {
		b.Speed = float64(min(float32(speed+0.1), 16)) // DAT_00470c18, 0x428b3b-0x428b59
		return
	}
	if b.Age >= 120 {
		b.Speed = float64(float32(speed + 0.2)) // DAT_00470c38, 0x428c0c; no clamp
		return
	}
	x, y := float32(b.X-gameOriginX), float32(b.Y-gameOriginY)
	if b.fromSHT {
		x, y = b.pos[0], b.pos[1]
	}
	dx := float32(b.target.X-gameOriginX) - x
	dy := float32(b.target.Y-gameOriginY) - y
	// 0x428b61-0x428b7d uses fpatan directly, not zun.Aim's coincident-point
	// half-pi special case. FUN_00428ce0 is the same short difference as AngleDiff.
	aim := float32(math.Atan2(float64(dy), float64(dx)))
	delta := zun.AngleDiff(aim, float32(b.Angle))
	abs := float32(math.Abs(float64(delta)))
	if abs >= float32(math.Pi/4) { // DAT_00470c48
		speed = max(float32(speed-0.3), 4) // DAT_00470c44/00470c40, 0x428ba6
	} else if abs <= float32(math.Pi/12) { // DAT_00470c3c
		speed = min(float32(speed+0.1), 16) // 0x428bd0-0x428be5
	}
	b.Angle = float64(zun.NormAngle(float32(delta*0.2) + float32(b.Angle))) // 0x428beb-0x428c04
	b.Speed = float64(speed)
}

// laserMuzzle is FUN_00428c20, update-table index 2 at 0x474780. It resets
// the laser's position to the current option, compensating the previous
// velocity before FUN_0044c2a0 adds the recomputed velocity later this tick.
func (p *Pool) laserMuzzle(b *Bullet) {
	if p.SourcePosition == nil {
		return
	}
	pos := p.SourcePosition(b.shot.Source)
	b.pos[0] = float32(float32(float32(b.shot.X)-float32(b.VX)) + pos[0])
	b.pos[1] = float32(float32(float32(b.shot.Y)-float32(b.VY)) + pos[1])
	b.pos[2], b.Z = 0, 0 // 0x428c6d-0x428c79
}
