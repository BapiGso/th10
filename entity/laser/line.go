package laser

import (
	"math"

	"th10/entity/bullet"
	"th10/zun"
)

// LineTemplate is a line laser's template (0x77 dwords, laser +0x424): what
// ECL 412/428/431/433 build, and what a bounce or a cut copies into the
// lasers it fires.
type LineTemplate struct {
	Pos          [3]float32                // +0
	Angle        float32                   // +0xc
	Full         float32                   // +0x10, the length the beam grows to
	Init         float32                   // +0x14, the length it starts at
	MaxDist      float32                   // +0x18, how far the tail may travel (0: no limit)
	Width        float32                   // +0x1c
	Speed        float32                   // +0x20
	Type, Colour int16                     // +0x24, +0x26
	Flags        uint32                    // +0x28: bit 0 draws the beam additively
	Ex           [bullet.ExCount]bullet.Ex // +0x2c
}

// Line is a line laser: a beam that grows from its tail to the full length,
// then travels.
type Line struct {
	Common
	Tpl LineTemplate // +0x424

	exIdx   int32     // +0x400, the next transform entry
	exFlags uint32    // +0x404, the active transforms
	grace   zun.Timer // +0x410, frames before the offscreen test applies

	accel    float32    // +0xa0
	accelT   zun.Timer  // +0x8c
	accelDur int32      // +0xb4
	accelVec [3]float32 // +0xa8

	stepSpeed, stepAngle    float32   // +0x108, +0x10c
	stepT                   zun.Timer // +0xf4
	stepDur, stepMax, stepN int32     // +0x11c, +0x120, +0x124

	bounceSpeed float32   // +0x13c
	waitT       zun.Timer // +0x15c
}

func newLine() *Line { l := &Line{}; l.T.Reset(); return l }

// SetTpl430 writes the template's angle, full length and starting length.
func (l *Line) SetTpl430(v [3]float32) { l.Tpl.Angle, l.Tpl.Full, l.Tpl.Init = v[0], v[1], v[2] }

// SetTpl440 writes the template's width.
func (l *Line) SetTpl440(v float32) { l.Tpl.Width = v }

// init is FUN_0041c8c0.
func (l *Line) init(t *LineTemplate, env *Env) {
	l.Tpl = *t
	c := &l.Common
	c.State = StateLine
	c.startVMs(bullet.ScriptID(t.Type, int32(t.Colour)), int32(t.Colour), t.Flags&1 != 0, env)
	l.grace.Set(30)
	c.Length = t.Init
	c.Pos = t.Pos
	c.Speed = t.Speed
	c.Angle = t.Angle
	c.SE = 0x18
	c.Width = t.Width
	if t.Init <= 0 {
		c.Dist = 0
	} else {
		c.Dist = 0.01
	}
	c.Vel[0], c.Vel[1] = zun.Polar(t.Angle, t.Speed)
}

// update is FUN_0041d3d0. It reports whether the laser is done.
func (l *Line) update(m *Manager, env *Env, gs float32) bool {
	c := &l.Common
	l.transforms(m, env)
	if l.exFlags != 0 {
		if l.exFlags&bullet.ExAccel != 0 {
			l.accelerate(gs)
		}
		if l.exFlags&bullet.ExStepRel != 0 {
			l.step(env, gs)
		}
		if l.exFlags&(bullet.ExBounce|bullet.ExBounceTop|bullet.ExBounceSides) != 0 {
			l.bounce(m, env)
		}
		if l.exFlags&bullet.ExWait != 0 {
			if l.waitT.Cur < 1 {
				l.exFlags ^= bullet.ExWait
			} else {
				l.waitT.Add(-1, gs)
			}
		}
	}

	if l.Tpl.Full <= c.Length {
		// Fully grown: the whole beam travels.
		c.Dist = float32(gs*c.Speed) + c.Dist
		for k := range 3 {
			c.Pos[k] = float32(gs*c.Vel[k]) + c.Pos[k]
		}
		if 0 < l.Tpl.MaxDist && l.Tpl.MaxDist < c.Dist+c.Length {
			c.Length = l.Tpl.MaxDist - c.Dist
			l.Tpl.Full = l.Tpl.MaxDist - c.Dist
			if c.Length <= 0 {
				return true
			}
		}
	} else {
		v := float32(gs*c.Speed) + c.Length
		c.Length = v
		if l.Tpl.Full < v {
			c.Length = l.Tpl.Full
		}
	}

	if l.grace.Cur < 1 {
		ex, ey := zun.Polar(c.Angle, c.Length)
		end := [3]float32{ex + c.Pos[0], ey + c.Pos[1], c.Pos[2]}
		if outside7a0(c.Pos, c.Width, c.Width) && outside7a0(end, c.Width, c.Width) {
			return true
		}
	} else {
		l.grace.Add(-1, gs)
	}

	if 16 < c.Length && 3 < c.Width {
		var hw float32
		if 32 <= c.Width {
			hw = c.Width - float32(float32(c.Width+16)*0.5)
		} else {
			hw = float32(c.Width * 0.5)
		}
		c.hitTest(l, m, hw, env)
	}
	c.runVMs(env)
	return false
}

// transforms is FUN_0041ca80, the line laser's transform chain. It reads and
// writes the template's entries, so a bounce's count carries into the lasers
// the bounce fires. Only the accel, step-relative, bounce and wait handlers
// do anything; the others' flags are set and ignored.
func (l *Line) transforms(m *Manager, env *Env) {
	c := &l.Common
	for l.exIdx <= 0x11 {
		if l.exIdx < 0 { // the binary reads before the table
			return
		}
		e := &l.Tpl.Ex[l.exIdx]
		if e.Type == 0 {
			return
		}
		// Async is compared as a float: -0.0 counts as 0.
		if uint32(e.Async)&0x7fffffff == 0 && l.exFlags != 0 {
			return
		}
		switch e.Type {
		case bullet.ExWait:
			l.exFlags |= bullet.ExWait
			l.waitT.Set(e.C)
		case bullet.ExStepRel, bullet.ExStepAim, bullet.ExStepAbs:
			l.exFlags |= e.Type
			l.stepAngle = e.A
			if e.B <= -999 {
				l.stepSpeed = c.Speed
			} else {
				l.stepSpeed = e.B
			}
			l.stepT.Reset()
			l.stepDur, l.stepMax, l.stepN = e.C, e.D, 0
		case bullet.ExSpawnBoost:
			l.exFlags |= bullet.ExSpawnBoost
		case bullet.ExAccel:
			l.exFlags |= bullet.ExAccel
			l.accel = e.A
			a := c.Angle
			if e.B > -990 {
				if e.B < 990 {
					a = e.B
				} else {
					a = zun.Aim(c.Pos[0], c.Pos[1], env.Player[0], env.Player[1])
				}
			}
			l.accelT.Set(0)
			l.accelDur = e.C
			l.accelVec[0], l.accelVec[1] = zun.Polar(a, e.A)
			if l.exIdx != 0 {
				env.se(c.SE, 0, false)
			}
		case bullet.ExAngular:
			l.exFlags |= bullet.ExAngular
			if l.exIdx != 0 {
				env.se(c.SE, 0, false)
			}
		case bullet.ExBounce, bullet.ExBounceTop, bullet.ExBounceSides:
			if e.C >= 1 {
				l.exFlags |= e.Type
				if e.A < 0 {
					l.bounceSpeed = c.Speed
				} else {
					l.bounceSpeed = e.A
				}
				e.C--
			}
		case bullet.ExScript:
			env.start(&c.beam, bullet.ScriptID(int16(e.C), e.D))
		case bullet.ExValue:
			c.ID = e.C
		case bullet.ExWrapX, bullet.ExWrapY, bullet.ExHoming:
			l.exFlags |= e.Type
		case bullet.ExCancel:
			c.State = StateDelay
		case bullet.ExSound:
			env.se(e.C, c.Pos[0], true)
		case bullet.ExGoto:
			l.exIdx = e.C
			continue
		}
		l.exIdx++
	}
}

// accelerate is FUN_0041d2c0.
func (l *Line) accelerate(gs float32) {
	c := &l.Common
	if l.accelT.Cur < l.accelDur {
		c.Speed = float32(gs*l.accel) + c.Speed
		for k := range 3 {
			c.Vel[k] = float32(gs*l.accelVec[k]) + c.Vel[k]
		}
		vx, vy := c.Vel[0], c.Vel[1]
		if abs32(vx) > 0.0001 || abs32(vy) > 0.0001 {
			c.Angle = float32(math.Atan2(float64(vy), float64(vx)))
		}
	} else {
		l.exFlags &^= bullet.ExAccel
	}
	l.accelT.Tick(gs)
}

// step is FUN_0041d170: slow the travel linearly to a stop over stepDur
// frames (the speed itself is kept), then turn by stepAngle, take stepSpeed
// and start over, stepMax times in all.
func (l *Line) step(env *Env, gs float32) {
	c := &l.Common
	var s float32
	if l.stepT.Cur < l.stepDur {
		s = c.Speed - float32(float32(c.Speed*l.stepT.F)/float32(l.stepDur))
	} else {
		env.se(c.SE, 0, false)
		l.stepN++
		if l.stepMax <= l.stepN {
			l.exFlags &^= bullet.ExStepRel
		}
		c.Angle = l.stepAngle + c.Angle
		s = l.stepSpeed
		c.Speed = s
		l.stepT.Reset()
	}
	c.Vel[0], c.Vel[1] = zun.Polar(c.Angle, s)
	l.stepT.Tick(gs)
}

// bounce is FUN_0041cfd0: once the head reaches a wall, fire the reflected
// beam from it as a new line laser (the sides always; the top for
// ExBounce/ExBounceTop, the bottom for ExBounce) and drop the bounce flags.
func (l *Line) bounce(m *Manager, env *Env) {
	c := &l.Common
	hx, hy := zun.Polar(c.Angle, c.Length)
	hx += c.Pos[0]
	hy += c.Pos[1]
	if !(hx <= -192 || 192 <= hx || hy <= 0 || 448 <= hy) {
		return
	}
	env.se(c.SE, 0, false)
	fired := false
	if hx < -192 || 192 <= hx {
		l.Tpl.Pos = [3]float32{hx, hy, 0}
		l.Tpl.Angle = zun.NormAngle(-c.Angle - zun.Pi)
		l.Tpl.Speed = l.bounceSpeed
		fired = true
		m.FireLine(&l.Tpl, env)
	}
	if l.exFlags&bullet.ExBounceSides != 0 || (0 <= hy && (hy < 448 || l.exFlags&bullet.ExBounce == 0)) {
		if !fired {
			return
		}
	} else {
		l.Tpl.Angle = -c.Angle
		l.Tpl.Pos = [3]float32{hx, hy, 0}
		l.Tpl.Speed = l.bounceSpeed
		m.FireLine(&l.Tpl, env)
	}
	l.exFlags &= 0xf7fff3ff
}

// cut is FUN_0041d880: cut the segments inside the size box centred on p.
func (l *Line) cut(m *Manager, p, size [3]float32, items bool, env *Env) int {
	return l.split(m, items, env, inBox(p, size))
}

// circle is FUN_0041dd80: cut the segments within r of p.
func (l *Line) circle(m *Manager, p [3]float32, r float32, items bool, env *Env) int {
	return l.split(m, items, env, inCircle(p, r))
}

// split is the body FUN_0041d880 and FUN_0041dd80 share. The laser keeps its
// first unmarked run (moving its tail up past a marked lead); each later run
// longer than 18px is fired as a new line laser from a copy of the template.
// A laser left with nothing, or with too little, is marked for deletion.
func (l *Line) split(m *Manager, items bool, env *Env, inside func([3]float32) bool) int {
	c := &l.Common
	if c.Length < 12 {
		return 0
	}
	orig := c.Pos
	marks, count := c.mark(int32(l.Tpl.Colour), true, items, env, inside)
	n := len(marks)
	if count == 0 {
		return 0
	}
	if n <= count {
		c.Mark = 1
		return count
	}
	ux, uy := zun.Polar(c.Angle, 6)
	step := [3]float32{ux + ux, uy + uy, 0}

	j := 0
	for j < n && marks[j] {
		j++
	}
	if j != 0 {
		fj := float32(j)
		for k := range 3 {
			c.Pos[k] = float32(step[k]*fj) + c.Pos[k]
		}
		d := float32(fj * 12)
		c.Length -= d
		if c.Length <= 18 {
			c.Mark = 1
			return count
		}
		l.Tpl.Full = c.Length
		c.Dist = d
	}
	run := 0
	for ; ; j++ {
		if n <= j {
			return count
		}
		if marks[j] {
			break
		}
		run++
	}
	d := float32(float32(run) * 12)
	l.Tpl.Full -= c.Length - d
	c.Length = d
	if d < 18 {
		c.Mark = 1
	}
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
		t := l.Tpl
		if size := float32(float32(j-start) * 12); 18 < size {
			fs := float32(start)
			for k := range 3 {
				t.Pos[k] = orig[k] + float32(step[k]*fs)
			}
			t.Full, t.Init = size, size
			m.FireLine(&t, env)
		}
	}
}

// clear is FUN_0041e260: a spark (and with items, on screen, an item) on
// every segment, then the laser is dead.
func (l *Line) clear(items bool, env *Env) {
	c := &l.Common
	if 12 < c.Length {
		id := int32(l.Tpl.Colour)*2 + 0x11
		c.walk(c.Pos[2], false, func(seg [3]float32) {
			env.effect(id, seg)
			if items && !outside7a0(seg, 32, 32) {
				env.cancelItem(seg)
			}
		})
	}
	c.State = StateDead
}

func abs32(v float32) float32 {
	return math.Float32frombits(math.Float32bits(v) &^ (1 << 31))
}
