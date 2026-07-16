package bullet

import (
	"math"
	"th10/collision"
)

// Type 子弹外观类型
type Type int

const (
	TypeRice   Type = iota // 米弹
	TypeSmall              // 小玉
	TypeMiddle             // 中玉
	TypeLarge              // 大玉
	TypeLaser              // 激光
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
	Color    int  // 颜色索引
	Additive bool // 是否高光绘制
	Active   bool
	Grazed   bool    // 已被擦弹（每颗子弹只计一次）
	Age      int     // 存活帧数
	AccelS   float64 // 速度加速度（每帧）
	AccelA   float64 // 角度加速度（每帧）
	Actions  [8]Action
	Length   float64 // 激光长度；普通子弹为 0
	Width    float64 // 激光宽度；普通子弹为 0
	Lifetime int     // >0 时到期消失
}

type ActionKind int

const (
	ActionSetMotion ActionKind = iota + 1
	ActionSetAccel
	ActionSetAngleAccel
	ActionExpire
)

type Action struct {
	Active   bool
	Kind     ActionKind
	Start    int
	Angle    float64
	Speed    float64
	AccelS   float64
	AccelA   float64
	HasAngle bool
	HasSpeed bool
}

func (b *Bullet) AddAction(a Action) {
	a.Active = true
	for i := range b.Actions {
		if !b.Actions[i].Active {
			b.Actions[i] = a
			return
		}
	}
}

// Update 更新子弹位置
func (b *Bullet) Update(fieldLeft, fieldTop, fieldRight, fieldBottom float64) {
	if !b.Active {
		return
	}
	b.applyActions()
	if !b.Active {
		return
	}
	b.Speed += b.AccelS
	b.Angle += b.AccelA
	b.X += b.Speed * math.Cos(b.Angle)
	b.Y += b.Speed * math.Sin(b.Angle)
	b.Age++
	if b.Lifetime > 0 && b.Age >= b.Lifetime {
		b.Active = false
		return
	}
	if b.Type == TypeLaser {
		if !laserInBounds(b, fieldLeft, fieldTop, fieldRight, fieldBottom) {
			b.Active = false
		}
		return
	}
	// 出界销毁（留 64 像素余量）
	if !collision.InBounds(b.X, b.Y, 64, fieldLeft, fieldTop, fieldRight, fieldBottom) {
		b.Active = false
	}
}

func (b *Bullet) applyActions() {
	for i := range b.Actions {
		a := &b.Actions[i]
		if !a.Active || b.Age < a.Start {
			continue
		}
		switch a.Kind {
		case ActionSetMotion:
			if a.HasAngle {
				b.Angle = a.Angle
			}
			if a.HasSpeed {
				b.Speed = a.Speed
			}
			a.Active = false
		case ActionSetAccel:
			if a.HasAngle {
				b.Angle = a.Angle
			}
			b.AccelS = a.AccelS
			if a.AccelA != 0 {
				b.AccelA = a.AccelA
			}
			a.Active = false
		case ActionSetAngleAccel:
			b.AccelA = a.AccelA
			a.Active = false
		case ActionExpire:
			b.Active = false
			a.Active = false
			return
		}
	}
}

// HitPlayer 检测是否命中自机
func (b *Bullet) HitPlayer(px, py, pr float64) bool {
	if !b.Active {
		return false
	}
	if b.Type == TypeLaser {
		return pointSegmentDistance(px, py, b.X, b.Y, b.X+math.Cos(b.Angle)*b.Length, b.Y+math.Sin(b.Angle)*b.Length) <= pr+b.laserRadius()
	}
	return collision.CircleHit(b.X, b.Y, b.Radius, px, py, pr)
}

// GrazePlayer 检测擦弹，每颗子弹只擦一次
func (b *Bullet) GrazePlayer(px, py, pr float64) bool {
	if !b.Active || b.Grazed {
		return false
	}
	if b.Type == TypeLaser {
		if pointSegmentDistance(px, py, b.X, b.Y, b.X+math.Cos(b.Angle)*b.Length, b.Y+math.Sin(b.Angle)*b.Length) <= pr+b.laserRadius()*1.5 {
			b.Grazed = true
			return true
		}
		return false
	}
	if collision.CircleHit(b.X, b.Y, b.Radius*5, px, py, pr) {
		b.Grazed = true
		return true
	}
	return false
}

func (b *Bullet) laserRadius() float64 {
	if b.Width > 0 {
		return b.Width / 2
	}
	if b.Radius > 0 {
		return b.Radius
	}
	return 8
}

func pointSegmentDistance(px, py, ax, ay, bx, by float64) float64 {
	vx, vy := bx-ax, by-ay
	wx, wy := px-ax, py-ay
	len2 := vx*vx + vy*vy
	if len2 == 0 {
		return math.Hypot(px-ax, py-ay)
	}
	t := (wx*vx + wy*vy) / len2
	if t < 0 {
		t = 0
	} else if t > 1 {
		t = 1
	}
	cx, cy := ax+t*vx, ay+t*vy
	return math.Hypot(px-cx, py-cy)
}

func laserInBounds(b *Bullet, fieldLeft, fieldTop, fieldRight, fieldBottom float64) bool {
	margin := 96 + math.Max(b.Width, 0)
	x2 := b.X + math.Cos(b.Angle)*b.Length
	y2 := b.Y + math.Sin(b.Angle)*b.Length
	minX, maxX := math.Min(b.X, x2), math.Max(b.X, x2)
	minY, maxY := math.Min(b.Y, y2), math.Max(b.Y, y2)
	return maxX >= fieldLeft-margin && minX <= fieldRight+margin && maxY >= fieldTop-margin && minY <= fieldBottom+margin
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

func (p *Pool) FireLaser(x, y, speed, angle, length, width float64, color int, lifetime int) *Bullet {
	b := p.Fire(x, y, speed, angle, TypeLaser, color, true)
	if b == nil {
		return nil
	}
	if length <= 0 {
		length = 256
	}
	if width <= 0 {
		width = 12
	}
	b.Length = length
	b.Width = width
	b.Radius = width / 2
	b.Lifetime = lifetime
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
