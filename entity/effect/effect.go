package effect

import (
	"th10/render"

	"github.com/hajimehoshi/ebiten/v2"
)

// Type 特效类型
type Type int

const (
	TypeExplosion Type = iota // 爆炸
	TypeGraze                 // 擦弹火花
	TypePlayerDead            // 自机被弹
	TypeBulletCancel          // 子弹消除
	TypePowerUp               // 升级闪光
)

// Effect 视觉特效
type Effect struct {
	X, Y     float64
	Type     Type
	Age      int
	MaxAge   int
	Scale    float64
	Alpha    float64
	Active   bool
}

// Pool 特效对象池
type Pool struct {
	effects []Effect
}

func NewPool(capacity int) *Pool {
	return &Pool{effects: make([]Effect, capacity)}
}

// Spawn 生成一个特效
func (p *Pool) Spawn(x, y float64, t Type, maxAge int) {
	for i := range p.effects {
		if !p.effects[i].Active {
			p.effects[i] = Effect{
				X: x, Y: y, Type: t,
				MaxAge: maxAge, Scale: 1, Alpha: 1,
				Active: true,
			}
			return
		}
	}
}

// Update 更新所有特效
func (p *Pool) Update() {
	for i := range p.effects {
		e := &p.effects[i]
		if !e.Active {
			continue
		}
		e.Age++
		// 随时间淡出
		progress := float64(e.Age) / float64(e.MaxAge)
		e.Alpha = 1 - progress
		e.Scale = 1 + progress*0.5
		if e.Age >= e.MaxAge {
			e.Active = false
		}
	}
}

// Each 遍历活跃特效
func (p *Pool) Each(fn func(e *Effect)) {
	for i := range p.effects {
		if p.effects[i].Active {
			fn(&p.effects[i])
		}
	}
}

// DrawLayer 特效层
func (e *Effect) DrawLayer() render.Layer { return render.LayerEffect }

// Draw 绘制特效（占位）
func (e *Effect) Draw(screen *ebiten.Image) {
	// TODO: 根据 Type 和 Age/Alpha/Scale 绘制特效动画
}
