package playerbullet

import (
	"math"
	"th10/collision"
	"th10/entity/enemy"
	"th10/game"
	"th10/sht"
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
	ScriptID     int // -1 for legacy bullets; original pl00.anm script for SHT shots
	Homing       bool
	fromSHT      bool
	target       *enemy.Enemy
}

// Pool 自机子弹对象池
type Pool struct {
	bullets []Bullet
}

func NewPool(capacity int) *Pool {
	return &Pool{bullets: make([]Bullet, capacity)}
}

// Fire 发射一颗自机子弹
func (p *Pool) Fire(x, y, vx, vy float64, damage int) {
	for i := range p.bullets {
		if !p.bullets[i].Active {
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

// FireSHT emits the supported Reimu A record. FUN_00427e90 subtracts the
// initial velocity so the same-tick bullet update starts at the muzzle.
func (p *Pool) FireSHT(x, y float64, s sht.Shot, target *enemy.Enemy) {
	for i := range p.bullets {
		if p.bullets[i].Active {
			continue
		}
		angle := normAngle(s.Angle)
		vx, vy := f32(math.Cos(angle)*s.Speed), f32(math.Sin(angle)*s.Speed)
		if target != nil && math.Abs(target.X-float64(game.FieldLeft+game.FieldRight)/2) > 224 {
			target = nil
		}
		p.bullets[i] = Bullet{X: f32(f32(s.X-vx) + x), Y: f32(f32(s.Y-vy) + y), VX: vx, VY: vy,
			Damage: s.Damage, Width: s.Width, Height: s.Height, Active: true, Angle: angle, Speed: s.Speed,
			ScriptID: s.Script, Homing: s.Callbacks[1] == 1, fromSHT: true, target: target}
		return
	}
}

// HomingTargetable mirrors the original target's 0x11/0xc0000 flag rejection.
func HomingTargetable(e *enemy.Enemy) bool {
	return e != nil && e.Active && e.ECLFlags&0xc0011 == 0
}

func f32(v float64) float64 { return float64(float32(v)) }

func normAngle(a float64) float64 {
	pi := float64(float32(math.Pi))
	tau := float64(float32(2 * math.Pi))
	if a > pi {
		a -= tau
	}
	if a < -pi {
		a += tau
	}
	return f32(a)
}

// tickHoming ports 0x428b10. The target is fixed at birth: losing it does not
// reacquire, and after age 120 the shot accelerates without further steering.
func (b *Bullet) tickHoming() {
	if !HomingTargetable(b.target) {
		b.target = nil
	}
	if b.target == nil {
		b.Speed = f32(math.Min(b.Speed+f32(0.1), 16))
		return
	}
	if b.Age >= 120 {
		b.Speed = f32(b.Speed + f32(0.2))
		return
	}
	aim := f32(math.Atan2(b.target.Y-b.Y, b.target.X-b.X))
	delta := normAngle(aim - b.Angle)
	if math.Abs(delta) >= f32(math.Pi/4) {
		b.Speed = f32(math.Max(b.Speed-f32(0.3), 4))
	} else if math.Abs(delta) <= f32(math.Pi/12) {
		b.Speed = f32(math.Min(b.Speed+f32(0.1), 16))
	}
	b.Angle = normAngle(f32(delta*f32(0.2)) + b.Angle)
}

// Update 更新所有自机子弹
func (p *Pool) Update() {
	for i := range p.bullets {
		b := &p.bullets[i]
		if !b.Active {
			continue
		}
		if b.fromSHT {
			if b.Homing {
				b.tickHoming()
			}
			b.VX = f32(math.Cos(b.Angle) * b.Speed)
			b.VY = f32(math.Sin(b.Angle) * b.Speed)
			b.X = f32(b.X + b.VX)
			b.Y = f32(b.Y + b.VY)
			// Original culling starts at age 10 and uses ANM sprite bounds.
			// Until full ANM instances are wired, use the SHT collision extent.
			if b.Age >= 10 && (b.X+b.Width < game.FieldLeft || b.X-b.Width > game.FieldRight || b.Y+b.Height < game.FieldTop || b.Y-b.Height > game.FieldBottom) {
				b.Active = false
				b.target = nil
			}
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

// HitEnemy 检测是否命中敌人，命中则返回伤害值并销毁子弹
func (p *Pool) HitEnemy(ex, ey, ew, eh float64) int {
	totalDamage := 0
	for i := range p.bullets {
		b := &p.bullets[i]
		if !b.Active {
			continue
		}
		if b.fromSHT && b.Y-b.Height/2 < game.FieldTop {
			continue
		}
		if collision.RectHit(b.X, b.Y, b.Width, b.Height, ex, ey, ew, eh) {
			totalDamage += b.Damage
			b.Active = false
			b.target = nil
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
