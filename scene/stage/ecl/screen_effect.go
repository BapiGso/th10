package ecl

import (
	"slices"

	"th10/gfx"
	"th10/supervisor"
	"th10/zun"
)

// calc dispatches FUN_0043c970's table 0x43cad8. Return 7 destroys the
// object; kind 4's return 0 detaches only its calc node (FUN_00449c40).
func (s *screenShake) calc(vm *VM, speed float32) int {
	exiting := vm.ctx != nil && vm.ctx.StageExiting // DAT_004918a4
	flags := uint32(0)
	if vm.ctx != nil {
		flags = vm.ctx.StageFlags // DAT_00477810+0x58, not DAT_00474ca0
		if vm.ctx.Menu != nil && vm.ctx.Menu.Active() {
			flags |= 0x10 // FUN_00418190's existing menu freeze bit
		}
	}
	switch s.kind {
	case 0, 3: // FUN_0043bd40: alpha uses the pre-tick clock
		if exiting {
			return 7
		}
		if s.duration != 0 {
			v := screenEffectRatio(s.timer.F, 255, s.duration) // DAT_00470bf8 = 0x437f0000
			s.alpha = max(int32(float32(255-v)), 0)            // 0x43bd59-0x43bd78: subtract BEFORE ftol
		}
		if s.timer.Cur >= s.duration { // 0x43bd7f-0x43bd85
			return 7
		}
		s.timer.Tick(speed)
	case 1: // FUN_0043c550 -> FUN_0043c5ca: tick, expire, sample
		if exiting {
			return 7
		}
		s.timer.Tick(speed) // 0x43c55f-0x43c5a5
		if s.timer.Cur >= s.duration {
			return 7
		}
		x, y := sampleShake(vm.rng, s.end-s.start, s.timer.F, s.duration, s.start)
		vm.shakeShift = [2]float32{x, y} // last callback wins
	case 2, 5: // FUN_0043c230: fade-out has two full-alpha tail calcs
		if exiting {
			return 7
		}
		if s.duration != 0 {
			if s.timer.Cur < s.duration {
				s.alpha = int32(screenEffectRatio(s.timer.F, 255, s.duration))
			} else {
				s.alpha = 255 // 0x43c265
			}
			s.alpha = max(s.alpha, 0)
		}
		if s.timer.Cur >= s.duration+2 { // 0x43c27a-0x43c285
			return 7
		}
		if flags&5 == 0 { // 0x43c28d-0x43c2a8: stage bits 0/2 hold the clock
			s.timer.Tick(speed)
		}
	case 4: // FUN_0043c460: repetitions in +0x20, ARGB in +0x24
		if exiting {
			return 0 // 0x43c4d5, NOT the destructor's return 7
		}
		if s.timer.Cur < s.duration {
			peak := int32(uint32(s.end) >> 24)
			s.alpha = max(peak-int32(screenEffectRatio(s.timer.F, peak, s.duration)), 0)
		} else {
			s.start-- // 0x43c4c1-0x43c4d3
			s.alpha = 0
			if s.start <= 0 {
				return 0
			}
			s.timer.Set(0) // 0x43c4dd-0x43c4e2; tick on this same calc
		}
		s.timer.Tick(speed)
	case 6, 7: // FUN_0043c310: translucent hold, then eight-tick release
		if s.phase == 0 {
			if s.duration != 0 && s.timer.Cur <= s.duration {
				s.alpha = int32(screenEffectRatio(s.timer.F, 128, s.duration)) // DAT_00470bf4 = 0x43000000
			}
		} else {
			if s.timer.Cur > 8 { // 0x43c342-0x43c346; Cur==8 still draws
				return 7
			}
			s.alpha = 128 - int32(float32(s.timer.F*16)) // 0x43c348-0x43c35d; no clamp
		}
		s.timer.Tick(speed) // 0x43c360-0x43c3af; no stage-exit/menu gate
	case 8: // FUN_0043c710: amplitude, rise, hold, fall in +0x1c..+0x28
		if exiting {
			return 7
		}
		if vm.ctx == nil || flags&0x77 != 0 { // 0x43c725-0x43c746
			return 1 // no tick or RNG while gated
		}
		s.timer.Tick(speed) // 0x43c74e-0x43c752
		var f float32
		switch {
		case s.timer.Cur < s.start: // 0x43c757-0x43c76c
			f = float32(float64(s.timer.F) / float64(s.start))
		case s.timer.Cur < s.start+s.end:
			f = 1
		case s.timer.Cur < s.start+s.end+s.third: // 0x43c780-0x43c7bd
			total := screenEffectUnsignedOperand(s.start + s.end + s.third)
			f = float32(total - float64(s.timer.F))
			f = float32(float64(f) / screenEffectUnsignedOperand(s.third))
		default:
			return 7 // 0x43c85c: terminal calc consumes no RNG
		}
		a := float32(float64(s.duration) * float64(f)) // 0x43c7bf-0x43c7ce, exact FILD operand
		x, y := sampleEnvelopeShake(vm.rng, a)
		vm.shakeShift = [2]float32{x, y}
	}
	return 1
}

// FMUL followed by FIDIV in the alpha callbacks: integer operands remain
// exact until each x87 24-bit arithmetic result.
func screenEffectRatio(timer float32, numerator, denominator int32) float32 {
	v := float32(float64(timer) * float64(numerator))
	return float32(float64(v) / float64(denominator))
}

// FUN_0043c710, 0x43c79c/0x43c7b1: unsigned operands use FILD and, when
// negative, FADD 2^32 (DAT_00470b98 = 0x4f800000) before the next operation.
func screenEffectUnsignedOperand(v int32) float64 {
	if v < 0 {
		return float64(float32(float64(v) + 4294967296.0))
	}
	return float64(v)
}

// FUN_0043c710's 0x43c7d4/0x43c812 calls use FUN_0043cb80, Raw()%3,
// not FUN_0043c5ca's repeated-low Dup()%3. Both take four main steps,
// including zero amplitude, but can choose different signs.
func sampleEnvelopeShake(rng *zun.RNG, amplitude float32) (x, y float32) {
	axis := func() float32 {
		switch rng.Intn(3) {
		case 1:
			return amplitude
		case 2:
			return -amplitude
		}
		return 0
	}
	x, y = axis(), axis()
	return x, y
}

// DrawScreenEffects supplies dynamic FUN_0043c970 draw nodes in the
// inclusive priority range. Stage interleaves ranges before its static
// nodes; newest registration wins equal priority, as FUN_00449b70 does.
func (vm *VM) DrawScreenEffects(sv *supervisor.Supervisor, first, last int32) {
	var draw []*screenShake
	for _, s := range vm.shakes {
		if s.kind != 1 && s.kind != 8 && s.drawPriority >= first && s.drawPriority <= last {
			draw = append(draw, s)
		}
	}
	slices.SortStableFunc(draw, func(a, b *screenShake) int {
		switch {
		case a.drawPriority < b.drawPriority:
			return -1
		case a.drawPriority > b.drawPriority:
			return 1
		}
		return 0
	})
	for _, s := range draw {
		// FUN_0043c1a0 (0/5), 43c2c0 (2/3), 43c500 (4), 43c3c0
		// (6), 43c410 (7): all use FUN_0043bda0 without Shift/snap.
		rect := [4]float32{32, 16, 416, 464} // immediate words 42000000/41800000/43d00000/43e80000
		colour := uint32(s.start)
		if s.kind == 4 {
			colour = uint32(s.end) & 0xffffff // 0x43c507-0x43c514
		}
		if s.kind == 0 || s.kind == 5 || s.kind == 6 {
			rect = [4]float32{0, 0, 640, 480} // 0x43c1ae-0x43c1c6 / 0x43c3d4-0x43c3f4
		}
		if s.kind == 0 || s.kind == 5 {
			// FUN_0043c1a0 flushes and sets the device viewport to the
			// whole back buffer; it does not move either camera or Shift.
			sv.ANM.Flush()
			vp := sv.Screen.Viewport
			if sv.Cur != nil {
				vp = sv.Cur.Viewport
			}
			vp.X, vp.Y, vp.W, vp.H = 0, 0, 640, 480
			sv.Dev.SetViewport(vp)
		}
		sv.ANM.Rect(rect, gfx.Color(uint32(s.alpha)<<24|colour))
	}
}
