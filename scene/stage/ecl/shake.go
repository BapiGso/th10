package ecl

import (
	"slices"

	"th10/zun"
)

// screenShake is kind 1 of FUN_0043c8b0/0043c970: a calc-only object
// with duration (+0x1c), start/end amplitudes (+0x20/+0x24), and the
// timer (+0x30). It is not an ANM VM and has no draw-chain node.
type screenShake struct {
	duration, start, end int32
	timer                zun.Timer
}

func newScreenShake(duration, start, end int32) *screenShake {
	// FUN_0043c8b0 zeroes the 0x44-byte object; FUN_0043c970 selects
	// callback 0x43c550 at calc priority 0x0e and resets the timer to 0.
	s := &screenShake{duration: duration, start: start, end: end}
	s.timer.Reset() // 0x43caa5-0x43cace; construction consumes no RNG
	return s
}

func (vm *VM) queueShake(duration, start, end int32) {
	// FUN_00449ae0 inserts before equal priorities: newest samples first.
	// An ECL-337 insertion at calc 0x12 cannot run until the next frame.
	vm.shakes = slices.Insert(vm.shakes, 0, newScreenShake(duration, start, end))
}

// CalcShake is FUN_0043c550 -> FUN_0043c5ca. Stage calls it at calc
// 0x0e, after STD (0x0c) and before score popups/player/enemies. The
// timer ticks before sampling; an expired object takes return 7 and is
// removed (FUN_0043c870/0043cb00) without consuming any RNG that frame.
func (vm *VM) CalcShake() {
	// FUN_00402720 clears its shift slots at 0x402787/0x402791, then
	// copies the camera to Field at 0x402835, before this priority.
	// Keep that reset in the logic-side sample, not in ShakeOffset/Draw.
	vm.shakeShift = [2]float32{}
	speed := vm.gameSpeed
	if vm.ctx != nil && vm.ctx.Anm != nil {
		speed = vm.ctx.Anm.GameSpeed
	}
	n := 0
	for _, s := range vm.shakes {
		s.timer.Tick(speed)            // 0x43c55f-0x43c5a5
		if s.timer.Cur >= s.duration { // 0x43c5a8-0x43c5bd
			continue
		}
		x, y := sampleShake(vm.rng, s.end-s.start, s.timer.F, s.duration, s.start)
		vm.shakeShift = [2]float32{x, y} // DAT_00491e64/68; the last callback wins
		vm.shakes[n] = s
		n++
	}
	clear(vm.shakes[n:])
	vm.shakes = vm.shakes[:n]
}

// ShakeOffset supplies the already sampled Field.Shift to the stage
// wrappers/Draw. Repeated draws never advance a timer or take RNG steps.
func (vm *VM) ShakeOffset() (float64, float64) {
	return float64(vm.shakeShift[0]), float64(vm.shakeShift[1])
}

// sampleShake is FUN_0043c5ca (0x43c5e9-0x43c6e1). The caller supplies
// end-start, the post-tick timer accumulator, duration, then start. Each
// axis takes a repeated-low two-step RNG value, then chooses 0/+/-.
// Even an amplitude of zero consumes four game-RNG steps in total.
func sampleShake(rng *zun.RNG, numerator int32, timer float32, denominator, floor int32) (x, y float32) {
	// FILD/FIDIV/FIADD use the exact int32 operand; each arithmetic result
	// rounds at x87 24-bit precision before the next operation.
	a := float32(float64(numerator) * float64(timer))
	a = float32(float64(a) / float64(denominator))
	a = float32(float64(a) + float64(floor))
	axis := func() float32 {
		switch rng.Dup() % 3 {
		case 1:
			return a
		case 2:
			return -a
		}
		return 0
	}
	x = axis()
	y = axis()
	return x, y
}
