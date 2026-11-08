package ecl

import (
	"slices"

	"th10/zun"
)

// ScreenEffect is FUN_0043c8b0's 0x44-byte screen-effect object. Kind 1
// uses duration (+0x1c) and start/end amplitudes (+0x20/+0x24); the other
// kinds interpret these words as colours, repetitions or envelope times.
// It is not an ANM VM. Only kinds 1/8 have no draw-chain node.
type ScreenEffect struct {
	duration, start, end int32
	timer                zun.Timer
	kind, third          int32 // +0x10, +0x28
	alpha, phase         int32 // +0x18, +0x2c
	drawPriority         int32 // hidden EBX argument to FUN_0043c970
	calcStopped          bool  // return 0 removes only the calc node (kind 4)
}

type screenShake = ScreenEffect

// SetPhase writes the caller-owned +0x2c word read at 0x43c314. It does
// not guess the caller's timer reset or other scene-transition writes.
func (s *ScreenEffect) SetPhase(phase int32) { s.phase = phase }

// SetTimer is FUN_00405410 on the object's timer at +0x30.
func (s *ScreenEffect) SetTimer(frames int32) { s.timer.Set(frames) }

func newScreenShake(duration, start, end int32) *screenShake {
	return newScreenEffect(1, duration, start, end, 0, 0)
}

func newScreenEffect(kind, duration, start, end, third, priority int32) *screenShake {
	// FUN_0043c8b0 zeroes the 0x44-byte object; FUN_0043c970 selects
	// the callbacks through 0x43cad8 and resets the timer to 0. Invalid
	// kinds leave a null calc node in the binary; report that misuse here.
	if kind < 0 || kind > 8 {
		panic("ecl: invalid screen-effect kind")
	}
	s := &screenShake{kind: kind, duration: duration, start: start, end: end,
		third: third, drawPriority: priority}
	if kind == 3 {
		s.alpha = 255 // FUN_0043c970, 0x43c9e0: field fade-in starts opaque
	}
	s.timer.Reset() // 0x43caa5-0x43cace; construction consumes no RNG
	return s
}

func (vm *VM) queueShake(duration, start, end int32) {
	vm.queueScreenEffect(newScreenShake(duration, start, end))
}

func (vm *VM) queueScreenEffect(s *screenShake) {
	// FUN_00449ae0 inserts before equal priorities: newest samples first.
	// An ECL-337 insertion at calc 0x12 cannot run until the next frame.
	vm.shakes = slices.Insert(vm.shakes, 0, s)
}

// RequestShake preserves Context's five-word hook for FUN_0043c8b0.
// MSG's final-clear caller (0x416d85) passes kind 5 and EBX=0x31: a
// full-screen fade-out, not a random shake. No constructor samples RNG.
func (vm *VM) RequestShake(kind, frames, x, y, z int32) {
	vm.NewScreenEffect(kind, frames, x, y, z, 0x31)
}

// NewScreenEffect is FUN_0043c8b0, including its returned object and
// FUN_0043c970's hidden draw-priority EBX argument. Kinds 6/7 retain a
// caller-controlled phase; out-of-stage callers own those raw writes.
func (vm *VM) NewScreenEffect(kind, frames, x, y, z, priority int32) *ScreenEffect {
	s := newScreenEffect(kind, frames, x, y, z, priority)
	vm.queueScreenEffect(s)
	return s
}

// CalcShake runs FUN_0043c970's callbacks at calc 0x0e, after STD and
// before player/enemies. Kind 1 (FUN_0043c550 -> FUN_0043c5ca) ticks
// before sampling and takes return 7 without RNG on its terminal calc;
// the other kinds retain their own timer and node-removal order.
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
		if !s.calcStopped {
			switch s.calc(vm, speed) {
			case 7: // FUN_0043c870 -> FUN_0043cb00: remove both nodes
				continue
			case 0: // FUN_00449c40 -> FUN_00449f60: calc only, no destructor
				s.calcStopped = true
			}
		}
		vm.shakes[n] = s
		n++
	}
	clear(vm.shakes[n:])
	vm.shakes = vm.shakes[:n]
}

// ClearScreenEffects is FUN_0043cb00's stage-lifetime counterpart. A
// scene replacement releases both nodes, including a kind-4 draw node
// whose calc previously returned 0; none may consume the next stage's RNG.
func (vm *VM) ClearScreenEffects() {
	clear(vm.shakes)
	vm.shakes = nil
	vm.shakeShift = [2]float32{}
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
