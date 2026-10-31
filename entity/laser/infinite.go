package laser

import (
	"th10/entity/bullet"
	"th10/zun"
)

// InfiniteTemplate is an infinite laser's template (0x7e dwords, laser
// +0x424): what ECL 413/429/432/434 build.
type InfiniteTemplate struct {
	Pos    [3]float32                // +0
	Vel    [3]float32                // +0xc, the drift per frame (ECL 415)
	Angle  float32                   // +0x18
	Omega  float32                   // +0x1c, the turn per frame (ECL 419)
	Max    float32                   // +0x20, the length the beam grows to
	Init   float32                   // +0x24, the length it starts at
	Width  float32                   // +0x28, the full width
	Speed  float32                   // +0x2c, how fast it grows
	Delay  int32                     // +0x30, frames as a thin warning line
	Expand int32                     // +0x34, frames widening to Width
	Hold   int32                     // +0x38, frames at Width
	Shrink int32                     // +0x3c, frames narrowing to nothing
	Type   int16                     // +0x40
	Colour int16                     // +0x42
	Flags  uint32                    // +0x44: bit 0 follows boss 0, bit 1 draws the beam additively
	Ex     [bullet.ExCount]bullet.Ex // +0x48, copied by ECL 432/434 but never run
}

// Infinite is an infinite laser: a beam fixed at its tail that warns, widens,
// holds and narrows.
type Infinite struct {
	Common
	Tpl InfiniteTemplate // +0x424
}

func newInfinite() *Infinite { l := &Infinite{}; l.T.Reset(); return l }

// SetTpl430 writes the template's drift velocity.
func (l *Infinite) SetTpl430(v [3]float32) { l.Tpl.Vel = v }

// SetTpl440 writes the template's turn rate.
func (l *Infinite) SetTpl440(v float32) { l.Tpl.Omega = v }

// init is FUN_0041e5c0.
func (l *Infinite) init(t *InfiniteTemplate, env *Env) {
	l.Tpl = *t
	c := &l.Common
	c.State = StateDelay
	c.startVMs(bullet.ScriptID(t.Type, int32(t.Colour)), int32(t.Colour), t.Flags&2 != 0, env)
	c.Pos = t.Pos
	c.Angle = t.Angle
	c.SE = 0x18
	c.Length = t.Init
	c.Width = 2
	c.Speed = t.Speed
}

// update is FUN_0041e700. It reports whether the laser is done.
func (l *Infinite) update(m *Manager, env *Env, gs float32) bool {
	c := &l.Common
	t := &l.Tpl
	if c.Length < t.Max {
		c.Length = float32(gs*c.Speed) + c.Length
		if t.Max < c.Length {
			c.Length = t.Max
		}
	}
	c.Angle = zun.AddAngle(c.Angle, float32(gs*t.Omega))
	if t.Flags&1 != 0 && env.Boss != nil {
		if p, ok := env.Boss(); ok {
			c.Pos = p
		}
	}
	for k := range 3 {
		c.Pos[k] = float32(gs*t.Vel[k]) + c.Pos[k]
	}

	switch c.State {
	case StateDelay:
		if t.Delay <= c.T.Cur {
			c.T.Set(0)
			c.State = StateExpand
		}
	case StateExpand:
		if c.T.Cur < t.Expand {
			c.Width = float32(float32(t.Width*c.T.F) / float32(t.Expand))
			break
		}
		c.T.Set(0)
		c.State = StateLine
		c.Width = t.Width
		fallthrough
	case StateLine:
		if c.T.Cur < t.Hold {
			break
		}
		c.T.Set(0)
		c.State = StateShrink
		fallthrough
	case StateShrink:
		if t.Shrink <= c.T.Cur {
			return true
		}
		c.Width = t.Width - float32(float32(t.Width*c.T.F)/float32(t.Shrink))
	}

	if (c.State == StateExpand || c.State == StateLine) && 16 < c.Length {
		var hw float32
		if 32 <= c.Width {
			hw = c.Width - float32(float32(c.Width+16)*float32(1.0/3))
		} else {
			hw = float32(c.Width * 0.5)
		}
		c.hitTest(l, m, hw, env)
	}
	c.runVMs(env)
	return false
}

// cut is FUN_0041eb00: cut the segments inside the size box centred on p.
func (l *Infinite) cut(m *Manager, p, size [3]float32, items bool, env *Env) int {
	return l.split(m, items, env, inBox(p, size))
}

// circle is FUN_0041efa0: cut the segments within r of p.
func (l *Infinite) circle(m *Manager, p [3]float32, r float32, items bool, env *Env) int {
	return l.split(m, items, env, inCircle(p, r))
}

// split is the body FUN_0041eb00 and FUN_0041efa0 share. The beam is cut
// back to its first marked segment (to nothing if that is the first; it
// grows back), and every later unmarked run flies off as a plain line laser
// at speed 8.
func (l *Infinite) split(m *Manager, items bool, env *Env, inside func([3]float32) bool) int {
	c := &l.Common
	if !(12 < c.Length) {
		return 0
	}
	orig := c.Pos
	marks, count := c.mark(int32(l.Tpl.Colour), false, items, env, inside)
	n := len(marks)
	if count == 0 {
		return 0
	}
	j := 0
	for j < n && !marks[j] {
		j++
	}
	if j >= n {
		return count
	}
	c.Length = float32(float32(j) * 12)
	ux, uy := zun.Polar(c.Angle, 6)
	step := [3]float32{ux + ux, uy + uy, 0}
	for {
		for j < n && marks[j] {
			j++
		}
		if n <= j {
			return count
		}
		start := j
		for j < n && !marks[j] {
			j++
		}
		fs := float32(start)
		size := float32(float32(j-start) * 12)
		t := LineTemplate{
			Angle:   c.Angle,
			Full:    size,
			Init:    size,
			MaxDist: l.Tpl.Max - float32(fs*12),
			Width:   c.Width,
			Speed:   8,
			Type:    l.Tpl.Type,
			Colour:  l.Tpl.Colour,
		}
		for k := range 3 {
			t.Pos[k] = orig[k] + float32(step[k]*fs)
		}
		m.FireLine(&t, env)
		if n <= j {
			return count
		}
	}
}

// clear is FUN_0041f400: a spark (and with items, an item) on every segment
// on screen, then the laser is dead.
func (l *Infinite) clear(items bool, env *Env) {
	c := &l.Common
	if 12 < c.Length {
		id := int32(l.Tpl.Colour)*2 + 0x11
		c.walk(c.Pos[2], false, func(seg [3]float32) {
			if outside7a0(seg, 16, 16) {
				return
			}
			env.effect(id, seg)
			if items && !outside7a0(seg, 32, 32) {
				env.cancelItem(seg)
			}
		})
	}
	c.State = StateDead
}
