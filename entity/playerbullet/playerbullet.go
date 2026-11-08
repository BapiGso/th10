package playerbullet

import (
	"th10/anm"
	"th10/collision"
	"th10/entity/enemy"
	"th10/sht"
	"th10/zun"
)

// Bullet 自机子弹
type Bullet struct {
	X, Y         float64
	VX           float64
	VY           float64 // 向上飞，通常为负值
	Damage       int
	Width        float64 // 判定宽
	Height       float64 // 判定高
	Active       bool
	Age          int
	Angle, Speed float64
	ScriptID     int // -1 for legacy bullets; original pl0N.anm script for SHT shots
	Homing       bool
	fromSHT      bool
	target       *enemy.Enemy
	state        int32   // R+0x40: 0 free, 1 shot, 2 hit animation
	VM, ExtraVM  *anm.VM // R+0x44/+0x48, World handles
	Z            float32 // R+0x1c; hit sparks use 0.1 (0x428824)
	shot         sht.Shot
	pos          [3]float32 // R+0x14: game coordinates, before the diagnostic screen offset
	timer        zun.Timer  // R+0x00
	hitThisFrame bool       // R+0x50
	bright       bool       // R+0x54
}

// Pool 自机子弹对象池
type Pool struct {
	bullets []Bullet
	world   *anm.World
	env     *anm.Env
	file    *anm.File
	// FUN_00427e90/FUN_00428280/FUN_00428c20 read these from the owning
	// player and managers; nil providers leave standalone pool tests usable.
	GameSpeed      func() float32
	SourcePosition func(source int) [3]float32
	OptionCount    func() int
	ShotClock      func() int32
	NoAttack       func() bool
	ManagerPresent func() bool
	HitArea        func(x, y, radius, growth float32, frames, damage int32)
	PlaySE         func(id int, x float32)
	laserSources   [5]bool // P+0x42f4, one gate per SHT source
}

func NewPool(capacity int) *Pool {
	return &Pool{bullets: make([]Bullet, capacity)}
}

// Fire 发射一颗自机子弹
func (p *Pool) Fire(x, y, vx, vy float64, damage int) {
	for i := range p.bullets {
		if !p.bullets[i].Active && p.bullets[i].state == 0 {
			p.bullets[i] = Bullet{
				X: x, Y: y, VX: vx, VY: vy,
				Damage: damage,
				Width:  6, Height: 12,
				Active: true, ScriptID: -1,
			}
			return
		}
	}
}

// FireSHT is FUN_00427e90 for all six SHTs. The source gate is checked
// before allocation; the initial velocity is subtracted before callbacks.
func (p *Pool) FireSHT(x, y float64, s sht.Shot, target *enemy.Enemy) {
	if s.Source < 0 || s.Source >= len(p.laserSources) || (s.Kind == 3 && p.laserSources[s.Source]) {
		return // 0x427e9d-0x427eb8
	}
	for i := range p.bullets {
		if p.bullets[i].Active || p.bullets[i].state != 0 {
			continue
		}
		angle := zun.NormAngle(float32(s.Angle))
		vx, vy := zun.Polar(angle, float32(s.Speed)) // FUN_0044c5d0
		pos := [3]float32{
			float32(float32(float32(s.X)-vx) + float32(x-gameOriginX)),
			float32(float32(float32(s.Y)-vy) + float32(y-gameOriginY)), 0,
		} // 0x427ff2-0x428007, x87 24-bit operation order
		b := &p.bullets[i]
		*b = Bullet{X: gameOriginX + float64(pos[0]), Y: gameOriginY + float64(pos[1]), VX: float64(vx), VY: float64(vy),
			Damage: s.Damage, Width: s.Width, Height: s.Height, Active: true, Angle: float64(angle), Speed: s.Speed,
			ScriptID: s.Script, Homing: s.Callbacks[1] == 1, fromSHT: true, state: 1, shot: s, pos: pos}
		b.timer.Reset() // 0x427f05-0x427f0b
		if s.Kind == 3 {
			p.laserSources[s.Source] = true // 0x427f9b
		}
		p.spawnShot(b)
		if s.Callbacks[0] == 1 {
			b.initHoming(target) // 0x428120-0x428133 -> FUN_00428ad0
		}
		if s.Sound >= 0 && p.PlaySE != nil {
			p.PlaySE(s.Sound, b.pos[0]) // 0x428135-0x42814d, FUN_0043dd10
		}
		return
	}
}

// HomingTargetable mirrors the original target's 0x11/0xc0000 flag rejection.
func HomingTargetable(e *enemy.Enemy) bool {
	return e != nil && e.Active && e.ECLFlags&0xc0011 == 0
}

func f32(v float64) float64 { return float64(float32(v)) }

func normAngle(a float64) float64 {
	return float64(zun.NormAngle(float32(a)))
}

// Update 更新所有自机子弹
func (p *Pool) Update() {
	for i := range p.bullets {
		b := &p.bullets[i]
		if !b.Active && b.state != 2 {
			continue
		}
		if b.fromSHT {
			p.updateSHT(b) // FUN_00428280
			continue
		} else {
			b.X += b.VX
			b.Y += b.VY
			if b.Y < -20 {
				b.Active = false
			}
		}
		b.Age++
	}
}

// HitEnemy is FUN_00428630's shot damage query. Ordinary records become hit
// animations; beams stay active and damage only on their four-tick gate.
func (p *Pool) HitEnemy(ex, ey, ew, eh float64) int {
	totalDamage := 0
	for i := range p.bullets {
		b := &p.bullets[i]
		if !b.Active {
			continue
		}
		if b.fromSHT {
			if !shotOverlap(b, ex, ey, ew, eh) {
				continue
			}
			if !b.bright {
				p.world.Interrupt(b.VM, 2) // FUN_0040c480, 0x428797
				b.bright = true
			}
			b.hitThisFrame = true // R+0x50, 0x4287a6
			if b.shot.Kind == 3 {
				if b.timer.Cur != b.timer.Prev && b.timer.Cur%4 == 0 { // FUN_00427c70 at 0x4287bb
					totalDamage += b.Damage
				}
				continue // 0x4287cf: a laser never becomes a hit spark
			}
			totalDamage += b.Damage
			p.hit(b)
			if b.shot.Kind == 2 && p.HitArea != nil {
				p.HitArea(b.pos[0], b.pos[1], 32, 1.4, 13, int32(b.Damage/3)) // 0x428851-0x428863
			}
		} else if collision.RectHit(b.X, b.Y, b.Width, b.Height, ex, ey, ew, eh) {
			totalDamage += b.Damage
			p.release(b)
		}
	}
	return totalDamage
}

// Each 遍历活跃子弹
func (p *Pool) Each(fn func(b *Bullet)) {
	for i := range p.bullets {
		if p.bullets[i].Active {
			fn(&p.bullets[i])
		}
	}
}

// ActiveCount 当前活跃自机子弹数（供无头模拟的 trace 采样）。
func (p *Pool) ActiveCount() int {
	n := 0
	for i := range p.bullets {
		if p.bullets[i].Active {
			n++
		}
	}
	return n
}
