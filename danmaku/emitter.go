package danmaku

import (
	"th10/entity/bullet"
	"math"
)

// Emitter 弹幕发射器
type Emitter struct {
	X, Y     float64     // 发射位置
	Pool     *bullet.Pool // 共享子弹池
	Active   bool
}

func NewEmitter(pool *bullet.Pool) *Emitter {
	return &Emitter{Pool: pool, Active: true}
}

// SetPos 更新发射器位置（跟随敌人）
func (e *Emitter) SetPos(x, y float64) {
	e.X = x
	e.Y = y
}

// Ring 环形弹：count 颗子弹均匀分布，baseAngle 为起始角度
func (e *Emitter) Ring(count int, speed, baseAngle float64, t bullet.Type, color int) {
	if !e.Active {
		return
	}
	step := 2 * math.Pi / float64(count)
	for i := 0; i < count; i++ {
		angle := baseAngle + step*float64(i)
		e.Pool.Fire(e.X, e.Y, speed, angle, t, color, false)
	}
}

// Aimed 自机狙：向 (tx,ty) 发射 count 颗子弹，spread 为扇形展开角
func (e *Emitter) Aimed(tx, ty float64, count int, speed, spread float64, t bullet.Type, color int) {
	if !e.Active {
		return
	}
	baseAngle := math.Atan2(ty-e.Y, tx-e.X)
	if count == 1 {
		e.Pool.Fire(e.X, e.Y, speed, baseAngle, t, color, false)
		return
	}
	start := baseAngle - spread/2
	step := spread / float64(count-1)
	for i := 0; i < count; i++ {
		angle := start + step*float64(i)
		e.Pool.Fire(e.X, e.Y, speed, angle, t, color, false)
	}
}

// Spiral 螺旋弹：每帧调用一次，angle 每次递增 deltaAngle
func (e *Emitter) Spiral(arms int, speed, angle float64, t bullet.Type, color int) {
	if !e.Active {
		return
	}
	step := 2 * math.Pi / float64(arms)
	for i := 0; i < arms; i++ {
		a := angle + step*float64(i)
		e.Pool.Fire(e.X, e.Y, speed, a, t, color, false)
	}
}

// Random 随机散射
func (e *Emitter) Random(count int, speedMin, speedMax float64, t bullet.Type, color int, rng func() float64) {
	if !e.Active {
		return
	}
	for i := 0; i < count; i++ {
		angle := rng() * 2 * math.Pi
		speed := speedMin + rng()*(speedMax-speedMin)
		e.Pool.Fire(e.X, e.Y, speed, angle, t, color, false)
	}
}
