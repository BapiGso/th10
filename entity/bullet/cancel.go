package bullet

// Cancel is FUN_00408030: turn an active or spawning bullet into its cancel
// animation (state 3, script interrupt 1, the cancel effect), or delete it
// outright when it is more than 8px off the playfield. It reports whether the
// bullet was cancelled.
func (b *Bullet) Cancel(env *Env) bool {
	if b.State != StateSpawning && b.State != StateActive {
		return false
	}
	outside := outside4086b0(b.Pos, 8, 8)
	b.VM.Interrupt(1)
	b.State = StateCancelling
	if outside {
		b.Flags |= FlagDelete
		return true
	}
	if b.CancelScript >= 0 {
		env.effect(b.CancelScript, b.Pos)
	}
	b.T.Reset()
	return true
}

// CancelAll cancels every live bullet, as the MSG constructor (FUN_00415b00)
// does.
func (p *Pool) CancelAll(env *Env) {
	for i := range p.Slots {
		p.Slots[i].Cancel(env)
	}
}

// CancelVulnerable cancels every live bullet that is not cancel-immune,
// without items: the player's respawn and the 30 frames after it
// (FUN_00425730 states 0 and 1).
func (p *Pool) CancelVulnerable(env *Env) {
	for i := range p.Slots {
		if p.Slots[i].Immune == 0 {
			p.Slots[i].Cancel(env)
		}
	}
}

// CircleClear is FUN_00408100: cancel every bullet whose hit circle touches
// the circle of radius r round (cx, cy), skipping cancel-immune bullets when
// checkImmune is set. A cancelled bullet on the playfield leaves an item when
// items is set.
func (p *Pool) CircleClear(cx, cy, r float32, items, checkImmune bool, env *Env) {
	for i := range p.Slots {
		b := &p.Slots[i]
		if b.State == StateFree || b.State == StateCancelling || (checkImmune && b.Immune != 0) {
			continue
		}
		dx, dy := b.Pos[0]-cx, b.Pos[1]-cy
		rr := float32(b.W*0.5) + r
		if float32(float32(dx*dx)+float32(dy*dy)) > float32(rr*rr) {
			continue
		}
		b.Cancel(env)
		if !outside4086b0(b.Pos, 2, 2) && items {
			env.cancelItem(b.Pos)
		}
	}
}

// RectClear is FUN_00408210: delete every bullet touching Rect, leaving a
// spark tinted by the bullet's colour and, when items is set, an item.
func (p *Pool) RectClear(items bool, env *Env) {
	hx, hy := float32(p.Rect.W*0.5), float32(p.Rect.H*0.5)
	cx, cy := p.Rect.X, p.Rect.Y
	for i := range p.Slots {
		b := &p.Slots[i]
		if b.State == StateFree || b.State == StateCancelling {
			continue
		}
		hw, hh := float32(b.W*0.5), float32(b.H*0.5)
		x, y := b.Pos[0], b.Pos[1]
		if !(cx-hx <= hw+x && x-hw <= hx+cx && cy-hy <= hh+y && y-hh <= hy+cy) {
			continue
		}
		b.Flags |= FlagDelete
		if items {
			env.cancelItem(b.Pos)
		}
		vm := env.effect(effectClear, b.Pos)
		if w, _, ok := b.sprite(); ok && vm != nil {
			if c, ok := clearColour(w, b.Colour); ok {
				vm.Color2 = [3]uint8{uint8(c >> 16), uint8(c >> 8), uint8(c)}
				vm.Alpha2 = uint8(c >> 24)
			}
		}
		b.Clear = 0
	}
}

// cancelItem is FUN_0041bb00(8, -1, -pi/2, 0.6) at pos.
func (e *Env) cancelItem(pos [3]float32) {
	if e.CancelItem != nil {
		e.CancelItem(pos[0], pos[1])
	}
}
