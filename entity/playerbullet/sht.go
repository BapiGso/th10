package playerbullet

import (
	"math"

	"th10/game"
	"th10/zun"
)

const (
	gameOriginX = float64(game.FieldLeft+game.FieldRight) / 2
	gameOriginY = float64(game.FieldTop)
)

func (p *Pool) gameSpeed() float32 {
	if p.GameSpeed != nil {
		if gs := p.GameSpeed(); gs != 0 {
			return gs
		}
	}
	if p.env != nil && p.env.GameSpeed != 0 {
		return p.env.GameSpeed
	}
	return 1
}

// endLaser is FUN_00428280, 0x4282e4-0x428302/0x428330-0x42834e.
// Ending a beam releases its source immediately, while both VMs exit through
// label 1 and keep occupying the record until the main VM disappears.
func (p *Pool) endLaser(b *Bullet) {
	p.world.Interrupt(b.VM, 1)
	p.world.Interrupt(b.ExtraVM, 1)
	p.laserSources[b.shot.Source] = false
	b.state, b.Active = 2, false
	if !p.anmAttached() {
		p.release(b) // standalone tests have no animation lifetime to wait for
	}
}

// updateSHT is FUN_00428280: ending/brightness gates, update callback,
// velocity, grid movement, VM lookup/placement, then the record clock.
func (p *Pool) updateSHT(b *Bullet) {
	if b.shot.Kind == 3 && b.state == 1 {
		idle := p.ShotClock != nil && p.ShotClock() < 0
		lostSource := p.OptionCount != nil && b.shot.Source-1 >= p.OptionCount()
		blocked := p.NoAttack != nil && p.NoAttack()
		noManager := p.ManagerPresent != nil && !p.ManagerPresent()
		if idle || lostSource || blocked || noManager {
			p.endLaser(b)
		}
	}
	if b.state == 0 {
		return
	}
	if b.shot.Kind == 3 && b.state == 1 && !b.hitThisFrame && b.bright {
		p.world.Interrupt(b.VM, 3) // FUN_0040c4d0, 0x42836d-0x428375
		b.bright = false
	}
	b.hitThisFrame = false // 0x42837b
	switch b.shot.Callbacks[1] {
	case 1:
		b.tickHoming()
	case 2:
		p.laserMuzzle(b)
	}
	vx, vy := zun.Polar(float32(b.Angle), float32(b.Speed)) // 0x428393-0x42839d
	b.VX, b.VY = float64(vx), float64(vy)
	// FUN_0044c2a0 uses velocity without multiplying game speed; only the
	// record's Timer uses DAT_00476f78 (0x428577-0x4285c0).
	b.pos[0] = shotGrid(b.pos[0] + vx)
	b.pos[1] = shotGrid(b.pos[1] + vy)
	b.pos[2] = b.Z
	b.X, b.Y = gameOriginX+float64(b.pos[0]), gameOriginY+float64(b.pos[1])
	if p.anmAttached() {
		p.prep(b)
		if b.state == 0 {
			return
		}
	} else if b.shot.Kind != 3 && b.Age >= 10 &&
		(b.X+b.Width < game.FieldLeft || b.X-b.Width > game.FieldRight || b.Y+b.Height < game.FieldTop || b.Y-b.Height > game.FieldBottom) {
		p.release(b)
		return
	}
	b.timer.Tick(p.gameSpeed())
	b.Age = int(b.timer.Cur)
}

// shotGrid is FUN_0044c2a0 -> FUN_00452ff0: floor(float32(v*100))*0.01f.
func shotGrid(v float32) float32 {
	return float32(float32(math.Floor(float64(float32(v*100)))) * 0.01)
}

// shotOverlap is the rectangle/top-edge part of FUN_00428630. Ordinary
// shots must have their top at or below zero; beams test the enemy's bottom
// instead (0x428747-0x428778). All rectangle edge comparisons are inclusive.
func shotOverlap(b *Bullet, ex, ey, ew, eh float64) bool {
	x, y := float32(ex-gameOriginX), float32(ey-gameOriginY)
	w, h := float32(float32(ew)*0.5), float32(float32(eh)*0.5)
	left, top, right, bottom := x-w, y-h, x+w, y+h
	bw, bh := float32(float32(b.Width)*0.5), float32(float32(b.Height)*0.5)
	bx, by := b.pos[0], b.pos[1]
	bt := by - bh
	if bt > bottom || bx-bw > right || top > by+bh || left > bx+bw {
		return false
	}
	if b.shot.Kind == 3 {
		return bottom >= 0
	}
	return bt >= 0
}
