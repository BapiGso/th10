package playerbullet

import (
	"th10/collision"
)

// Bullet 自机子弹
type Bullet struct {
	X, Y   float64
	VX     float64
	VY     float64 // 向上飞，通常为负值
	Damage int
	Width  float64 // 判定宽
	Height float64 // 判定高
	Active bool
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
				Active: true,
			}
			return
		}
	}
}

// Update 更新所有自机子弹
func (p *Pool) Update() {
	for i := range p.bullets {
		b := &p.bullets[i]
		if !b.Active {
			continue
		}
		b.X += b.VX
		b.Y += b.VY
		if b.Y < -20 {
			b.Active = false
		}
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
		if collision.RectHit(b.X, b.Y, b.Width, b.Height, ex, ey, ew, eh) {
			totalDamage += b.Damage
			b.Active = false
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
