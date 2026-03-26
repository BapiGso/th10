package bullet

import (
	"th10/collision"
	"th10/render"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
)

// Type 子弹外观类型
type Type int

const (
	TypeRice    Type = iota // 米弹
	TypeSmall               // 小玉
	TypeMiddle              // 中玉
	TypeLarge               // 大玉
	TypeLaser               // 激光
)

// RadiusOf 各弹型的判定半径（文章第15篇参考值）
func RadiusOf(t Type) float64 {
	switch t {
	case TypeRice:
		return 1
	case TypeSmall:
		return 2
	case TypeMiddle:
		return 6
	case TypeLarge:
		return 10
	default:
		return 2
	}
}

// Bullet 弹幕子弹
type Bullet struct {
	X, Y     float64
	Speed    float64
	Angle    float64 // 弧度，0=右，PI/2=下
	Radius   float64
	Type     Type
	Color    int // 颜色索引
	Additive bool // 是否高光绘制
	Active   bool
	Grazed   bool   // 已被擦弹（每颗子弹只计一次）
	Age      int    // 存活帧数
	AccelS   float64 // 速度加速度（每帧）
	AccelA   float64 // 角度加速度（每帧）
}

// Update 更新子弹位置
func (b *Bullet) Update(fieldLeft, fieldTop, fieldRight, fieldBottom float64) {
	if !b.Active {
		return
	}
	b.Speed += b.AccelS
	b.Angle += b.AccelA
	b.X += b.Speed * math.Cos(b.Angle)
	b.Y += b.Speed * math.Sin(b.Angle)
	b.Age++
	// 出界销毁（留 64 像素余量）
	if !collision.InBounds(b.X, b.Y, 64, fieldLeft, fieldTop, fieldRight, fieldBottom) {
		b.Active = false
	}
}

// HitPlayer 检测是否命中自机
func (b *Bullet) HitPlayer(px, py, pr float64) bool {
	if !b.Active {
		return false
	}
	return collision.CircleHit(b.X, b.Y, b.Radius, px, py, pr)
}

// GrazePlayer 检测擦弹，每颗子弹只擦一次
func (b *Bullet) GrazePlayer(px, py, pr float64) bool {
	if !b.Active || b.Grazed {
		return false
	}
	if collision.CircleHit(b.X, b.Y, b.Radius*5, px, py, pr) {
		b.Grazed = true
		return true
	}
	return false
}

// Pool 子弹对象池，预分配避免 GC
type Pool struct {
	bullets []Bullet
	cap     int
}

func NewPool(capacity int) *Pool {
	return &Pool{
		bullets: make([]Bullet, capacity),
		cap:     capacity,
	}
}

// Acquire 从池中获取一颗空闲子弹，返回指针；池满则返回 nil
func (p *Pool) Acquire() *Bullet {
	for i := range p.bullets {
		if !p.bullets[i].Active {
			p.bullets[i] = Bullet{} // 重置
			return &p.bullets[i]
		}
	}
	return nil
}

// Fire 便捷方法：发射一颗子弹
func (p *Pool) Fire(x, y, speed, angle float64, t Type, color int, additive bool) *Bullet {
	b := p.Acquire()
	if b == nil {
		return nil
	}
	b.X = x
	b.Y = y
	b.Speed = speed
	b.Angle = angle
	b.Radius = RadiusOf(t)
	b.Type = t
	b.Color = color
	b.Additive = additive
	b.Active = true
	return b
}

// Update 更新池中所有活跃子弹
func (p *Pool) Update(fieldLeft, fieldTop, fieldRight, fieldBottom float64) {
	for i := range p.bullets {
		if p.bullets[i].Active {
			p.bullets[i].Update(fieldLeft, fieldTop, fieldRight, fieldBottom)
		}
	}
}

// Each 遍历所有活跃子弹
func (p *Pool) Each(fn func(b *Bullet)) {
	for i := range p.bullets {
		if p.bullets[i].Active {
			fn(&p.bullets[i])
		}
	}
}

// ActiveCount 当前活跃子弹数
func (p *Pool) ActiveCount() int {
	n := 0
	for i := range p.bullets {
		if p.bullets[i].Active {
			n++
		}
	}
	return n
}

// ClearAll 清屏（Boss 击破时）
func (p *Pool) ClearAll() {
	for i := range p.bullets {
		p.bullets[i].Active = false
	}
}

// DrawLayer 实现 render.Drawable（常规弹）
func (b *Bullet) DrawLayer() render.Layer {
	if b.Additive {
		return render.LayerBulletAdditive
	}
	return render.LayerBullet
}

// Draw 绘制子弹（占位，需配合贴图）
func (b *Bullet) Draw(screen *ebiten.Image) {
	// TODO: 根据 Type/Color 从 spritesheet 切图绘制
}
