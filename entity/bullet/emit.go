package bullet

import "th10/zun"

// Volley is FUN_004073e0: fire Count2 rings of Count1 bullets from t, aimed
// from t.Pos, stopping when the manager is full, then play t.Sound if flag
// 0x200 is set.
func (p *Pool) Volley(t *Template, env *Env) {
	aim := zun.Aim(t.Pos[0], t.Pos[1], env.Player[0], env.Player[1])
fire:
	for j := 0; j < int(t.Count2); j++ {
		for i := 0; i < int(t.Count1); i++ {
			if !p.spawn(t, i, j, aim, env) {
				break fire
			}
		}
	}
	if t.Flags&0x200 != 0 {
		env.se(t.Sound, t.Pos[0], true)
	}
}

// spawn is FUN_004067d0: bullet i of ring j. It reports false when no slot
// is free.
func (p *Pool) spawn(t *Template, i, j int, aim float32, env *Env) bool {
	idx := p.cursor
	for n := 0; p.Slots[idx].State != StateFree; n++ {
		if n == Slots-1 {
			return false
		}
		if idx++; idx == Slots {
			idx = 0
		}
	}
	b := &p.Slots[idx]

	fi, fj := float32(i), float32(j)
	n1 := float32(t.Count1)
	speed := t.Speed1
	if t.Count2 > 1 {
		speed = t.Speed1 - float32(float32(float32(t.Speed1-t.Speed2)*fj)/float32(t.Count2))
	}
	var a float32
	switch t.Aim {
	case 0, 1:
		var m float32
		if t.Count1&1 != 0 {
			m = float32((i + 1) / 2)
		} else {
			m = float32(i/2) + 0.5
		}
		a = float32(m * t.Angle2)
		if i&1 != 0 {
			a = float32(a * -1)
		}
		if t.Aim == 0 {
			a += aim
		}
		a += t.Angle1
	case 2, 3:
		if t.Aim == 2 {
			a = aim
		}
		a = float32(float32(float32(fi*zun.TwoPi)/n1)+a) + float32(fj*t.Angle2)
		a += t.Angle1
	case 4, 5:
		if t.Aim == 4 {
			a = aim
		}
		a = float32(zun.Pi/n1) + a
		a = float32(a+float32(float32(fi*zun.TwoPi)/n1)) + float32(fj*t.Angle2)
		a += t.Angle1
	case 6:
		a = float32(float32(t.Angle1-t.Angle2)*env.Anm.RNG.Float()) + t.Angle2
	case 7:
		speed = float32(float32(t.Speed1-t.Speed2)*env.Anm.RNG.Float()) + t.Speed2
		a = float32(float32(fi*zun.TwoPi)/n1) + float32(fj*t.Angle2)
		a += t.Angle1
	case 8:
		a = float32(float32(t.Angle1-t.Angle2)*env.Anm.RNG.Float()) + t.Angle2
		speed = float32(float32(t.Speed1-t.Speed2)*env.Anm.RNG.Float()) + t.Speed2
	}

	b.Flags |= FlagUsed
	b.State = StateActive
	b.T.Reset()
	b.T2.Reset()
	b.Speed = speed
	b.Angle = zun.AddAngle(a, 0)
	b.Pos = [3]float32{t.Pos[0], t.Pos[1], 0.1}
	b.Vel[0], b.Vel[1] = zun.Polar(a, speed)
	b.ExFlags = t.Flags
	b.Colour, b.Type = t.Colour, t.Type
	b.Flags = b.Flags&^(FlagGrazed|FlagDelete) | FlagHittable
	b.Unk454 = 0
	b.startScript(scriptBase[typeIndex(t.Type)]+int32(t.Colour), env)
	b.CancelScript = cancelScript(t.Type, t.Colour)
	b.Layer = layers[typeIndex(t.Type)]
	b.TransformSound = t.TransformSound
	b.Grace = 10
	b.W = sizes[typeIndex(t.Type)]
	b.H = b.W

	switch {
	case t.Flags&2 != 0:
		b.VM.Interrupt(7)
	case t.Flags&4 != 0:
		b.VM.Interrupt(8)
	case t.Flags&8 != 0:
		b.VM.Interrupt(9)
	default:
		b.VM.Interrupt(2)
	}
	if t.Flags&0xe != 0 {
		b.State = StateSpawning
		for k := range 3 {
			b.Pos[k] -= float32(b.Vel[k] * 4)
		}
	}

	b.Ex = t.Ex
	b.TplFlags = t.Flags
	b.ExFlags = 0
	b.ExIdx = t.StartEx
	b.runEx(p, env)
	// An ended script is not checked here; the next update frees the bullet.
	b.VM.Run(env.Anm)

	if idx++; idx == Slots {
		idx = 0
	}
	p.cursor = idx
	return true
}

// startScript is FUN_00404f30: restart the bullet's sprite script id with
// the positions zeroed. The script's first frame runs now.
func (b *Bullet) startScript(id int32, env *Env) {
	b.VM.Reset()
	b.VM.Offset, b.VM.Pos, b.VM.Pos2 = [3]float32{}, [3]float32{}, [3]float32{}
	b.VM.Start(env.File, int(id), env.Anm)
}
