package anm

// The ANM script VM, ported from th10.exe's FUN_0043ee30. One VM is one
// 0x3ac-byte animation object; Run executes a frame of its script and then
// advances its velocities, tweens and timer. Arithmetic follows the binary at
// x87 24-bit precision (see scene/stage/ecl/motion.go): products are wrapped
// in float32() so they cannot fuse.
//
// Rendering state (sprite matrices, the 0x2400000 ring buffer) is kept only as
// far as later frames depend on it; drawing is M3.

import (
	"math"
	"slices"

	"th10/zun"
)

// VM flag bits (+0x35c).
const (
	FlagVisible    uint32 = 1
	flagRotDirty   uint32 = 4
	flagScaleDirty uint32 = 8
	flagBlendMask  uint32 = 0x30
	flagPos2       uint32 = 0x100 // position ops target Pos2
	FlagFlipX      uint32 = 0x200
	FlagFlipY      uint32 = 0x400
	flagWaiting    uint32 = 0x1000 // parked at a stop until an interrupt
	flagScroll     uint32 = 0x2000 // Offset follows the stage scroll
	flagFrozen     uint32 = 0x20000
	flagRealtime   uint32 = 0x20000000 // runs at game speed 1
	FlagAltRNG     uint32 = 0x40000000 // draws from the alternate RNG
)

// Env is what a VM reads from the game while it runs.
type Env struct {
	// GameSpeed is DAT_00476f78; 0 means 1.
	GameSpeed float32
	// RNG is the game RNG DAT_004918b0, Alt the alternate DAT_004918a8.
	RNG, Alt *zun.RNG
	// Spawn receives each op-88 child after its first frame, once it has
	// copied the parent's Pos (into Pos2) and Offset. Nil drops children.
	Spawn func(*VM)
	// Globals are variables 10016-10021 (DAT_00491d7c/80/84/a0/a4/a8).
	Globals [6]float32
}

// tween is an integer colour/alpha tween (+0xbc, +0x108 and their colour2
// twins): FUN_00441600 for 3 components, FUN_00441950 for 1.
type tween struct {
	start, end, tan0, tan1 [3]int32
	t                      zun.Timer
	dur, mode              int32
}

// begin restarts the timer and zeroes the tangents (FUN_00442220).
func (w *tween) begin(dur, mode int32) {
	w.dur, w.mode = dur, mode
	w.tan0, w.tan1 = [3]int32{}, [3]int32{}
	w.t.Reset()
}

// finish is the common head of both steps: tick, and on the last frame stop
// and return the final value.
func (w *tween) finish(gs float32) (out [3]int32, done bool) {
	if w.dur > 0 {
		w.t.Tick(gs)
		if w.dur <= w.t.Cur {
			w.t.Set(w.dur)
			w.dur = 0
			if w.mode == 7 {
				return w.start, true
			}
			return w.end, true
		}
	}
	return out, false
}

// hermite returns the basis weights h00, h01, h10, h11 at t = F/dur.
func (w *tween) hermite() (h00, h01, h10, h11 float32) {
	t := w.t.F / float32(w.dur)
	a := t - 1
	h00 = float32(float32(((t+t)+1)*a) * a)
	h01 = float32(float32((3-(t+t))*t) * t)
	h10 = float32(float32((1-t)*(1-t)) * t)
	h11 = float32(float32(a*t) * t)
	return
}

// step1 is FUN_00441950 (alpha): the eased value truncates after adding start.
func (w *tween) step1(gs float32) int32 {
	if out, done := w.finish(gs); done {
		return out[0]
	}
	switch w.mode {
	case 7:
		w.start[0] += w.end[0]
		return w.start[0]
	case 17:
		w.start[0] += w.tan1[0]
		w.tan1[0] += w.end[0]
		return w.start[0]
	case 8:
		h00, h01, h10, h11 := w.hermite()
		v := float32(h01 * float32(w.end[0]))
		v = v + float32(h11*float32(w.tan1[0]))
		v = v + float32(h10*float32(w.tan0[0]))
		v = v + float32(h00*float32(w.start[0]))
		return int32(v)
	}
	e := zun.Ease(w.mode, w.t.F, float32(w.dur))
	return int32(float32(float32(e*float32(w.end[0]-w.start[0])) + float32(w.start[0])))
}

// step3 is FUN_00441600 (colour): each term truncates on its own before the
// integer sum, so negative deltas round toward start instead of toward 0.
func (w *tween) step3(gs float32) (out [3]int32) {
	if out, done := w.finish(gs); done {
		return out
	}
	switch w.mode {
	case 7:
		for i := range 3 {
			w.start[i] += w.end[i]
		}
		return w.start
	case 17:
		for i := range 3 {
			w.start[i] += w.tan1[i]
			w.tan1[i] += w.end[i]
		}
		return w.start
	case 8:
		h00, h01, h10, h11 := w.hermite()
		for i := range 3 {
			out[i] = int32(float32(float32(w.end[i])*h01)) + int32(float32(float32(w.start[i])*h00)) +
				int32(float32(float32(w.tan0[i])*h10)) + int32(float32(float32(w.tan1[i])*h11))
		}
		return out
	}
	e := zun.Ease(w.mode, w.t.F, float32(w.dur))
	for i := range 3 {
		out[i] = w.start[i] + int32(float32(float32(w.end[i]-w.start[i])*e))
	}
	return out
}

// VM is one ANM animation object.
type VM struct {
	File     *File
	ScriptID int
	script   *Script
	pc       int // instruction index; -1 once the script has ended

	T       zun.Timer // +0x5c
	pending int16     // +0x304: interrupt label waiting to be taken
	Flags   uint32    // +0x35c

	I      [4]int32   // variables 10000-10003
	F      [4]float32 // 10004-10007
	I8, I9 int32      // 10008, 10009

	Sprite     int   // -1 = none
	SpriteTime int32 // timer value when the sprite was set (+0x380)

	Pos, Offset, Pos2 [3]float32

	Rot, AngVel     [3]float32
	Scale, ScaleVel [2]float32
	UV, UVVel       [2]float32

	Color, Color2 [3]uint8 // R, G, B
	Alpha, Alpha2 uint8
	Layer         int32

	posI, rotI, scaleI               zun.Interp
	colorI, alphaI, color2I, alpha2I tween

	savedT  zun.Timer // op 81 restores these (saved at each interrupt)
	savedPC int

	// RingN is op 84's segment count (the ring buffer itself is not drawn yet).
	RingN int32

	// scratch absorbs writes to a non-variable destination, which the binary
	// stores into the script bytes themselves (no shipped script does this).
	scratchI int32
	scratchF float32

	// chain is the children op 88 spawned under this object and under them
	// (the list at +0x10 FUN_00413200 links each child into, after its
	// parent); root is the object whose chain holds a child, nil on a root.
	chain []*VM
	root  *VM

	env *Env
	gs  float32
}

// New returns a VM in the reset state with no script.
func New() *VM {
	vm := &VM{}
	vm.Reset()
	return vm
}

// Reset is FUN_00401de0. It keeps Offset and Layer, which the binary's spawn
// helpers set before starting the script.
func (vm *VM) Reset() {
	off, layer := vm.Offset, vm.Layer
	*vm = VM{Offset: off, Layer: layer}
	vm.pc = -1
	vm.Sprite = -1
	vm.Scale = [2]float32{1, 1}
	vm.Color = [3]uint8{255, 255, 255}
	vm.Alpha = 255
	vm.Flags = 7
	vm.T = zun.Timer{Prev: -999999}
}

// Start is FUN_0043e710: reset, load script id of f and run its first frame.
// A missing script leaves the VM zeroed and ended. It reports whether the
// script was found.
func (vm *VM) Start(f *File, id int, env *Env) bool {
	if f == nil || f.Scripts[id] == nil {
		*vm = VM{pc: -1, Sprite: -1}
		return false
	}
	vm.Reset()
	vm.startKeep(f, id, env)
	return true
}

// StartKeep is FUN_0043e7e0: Start without the reset, so flags (notably
// FlagAltRNG), variables and positions carry over.
func (vm *VM) StartKeep(f *File, id int, env *Env) bool {
	if f == nil || f.Scripts[id] == nil {
		return false
	}
	vm.startKeep(f, id, env)
	return true
}

func (vm *VM) startKeep(f *File, id int, env *Env) {
	vm.File, vm.ScriptID, vm.script = f, id, f.Scripts[id]
	vm.Flags &^= FlagFlipX | FlagFlipY
	vm.pc = 0
	vm.T.Reset()
	vm.Flags &^= FlagVisible
	vm.Run(env)
}

// Ended reports whether the script has finished (pc == 0 in the binary).
func (vm *VM) Ended() bool { return vm.pc < 0 }

// Interrupt queues label n (ANM op 64 arg) for the next Run.
func (vm *VM) Interrupt(n int16) { vm.pending = n }

// Visible reports flag bit 0.
func (vm *VM) Visible() bool { return vm.Flags&FlagVisible != 0 }

// Blend is the blend mode op 66 sets (flag bits 4-5): 0 alpha, 1 additive.
func (vm *VM) Blend() uint32 { return (vm.Flags & flagBlendMask) >> 4 }

// instr returns the current instruction; past the end it is the 0xffff
// terminator (time 0), which the loader drops.
func (vm *VM) instr() Instr {
	if vm.pc < len(vm.script.Instrs) {
		return vm.script.Instrs[vm.pc]
	}
	return Instr{Op: 0xffff}
}

// jump moves to byte offset off within the script and sets the timer to t.
// An offset that is not an instruction boundary ends the script.
func (vm *VM) jump(off int32, t int32) {
	vm.T.Set(t)
	i, ok := vm.script.byOffset[int(off)]
	if !ok {
		if int(off) == vm.scriptEnd() {
			i = len(vm.script.Instrs)
		} else {
			vm.pc = -1
			return
		}
	}
	vm.pc = i
}

// scriptEnd is the byte offset of the terminator.
func (vm *VM) scriptEnd() int {
	n := len(vm.script.Instrs)
	if n == 0 {
		return 0
	}
	last := vm.script.Instrs[n-1]
	return last.Offset + 8 + len(last.Args)
}

// rng is the generator a draw uses right now.
func (vm *VM) rng() *zun.RNG {
	if vm.Flags&FlagAltRNG != 0 {
		return vm.env.Alt
	}
	return vm.env.RNG
}

// readI is FUN_0043ec70 behind the mask test: parameter i as an int.
func (vm *VM) readI(in Instr, i int) int32 {
	v := in.Arg(i)
	if in.Mask&(1<<i) == 0 {
		return v
	}
	switch {
	case v >= 10000 && v <= 10003:
		return vm.I[v-10000]
	case v >= 10004 && v <= 10007:
		return int32(vm.F[v-10004])
	case v == 10008:
		return vm.I8
	case v == 10009:
		return vm.I9
	}
	return v
}

// readF is FUN_0043eac0 behind the mask test: parameter i as a float.
func (vm *VM) readF(in Instr, i int) float32 {
	v := in.ArgF(i)
	if in.Mask&(1<<i) == 0 {
		return v
	}
	switch id := int32(v); {
	case id >= 10000 && id <= 10003:
		return float32(vm.I[id-10000])
	case id >= 10004 && id <= 10007:
		return vm.F[id-10004]
	case id == 10008:
		return float32(vm.I8)
	case id == 10009:
		return float32(vm.I9)
	case id == 10010:
		return float32(vm.rng().Signed() * zun.Pi)
	case id == 10011:
		return vm.rng().Float()
	case id == 10012:
		return vm.rng().Signed()
	case id >= 10013 && id <= 10015:
		return vm.Pos[id-10013]
	case id >= 10016 && id <= 10021:
		return vm.env.Globals[id-10016]
	}
	return v
}

// intPtr is FUN_0043eda0: the int variable parameter i names.
func (vm *VM) intPtr(in Instr, i int) *int32 {
	if in.Mask&(1<<i) != 0 {
		switch v := in.Arg(i); {
		case v >= 10000 && v <= 10003:
			return &vm.I[v-10000]
		case v == 10008:
			return &vm.I8
		case v == 10009:
			return &vm.I9
		}
	}
	vm.scratchI = in.Arg(i)
	return &vm.scratchI
}

// floatPtr is FUN_0043ed00: the float variable parameter i names.
func (vm *VM) floatPtr(in Instr, i int) *float32 {
	if in.Mask&(1<<i) != 0 {
		switch id := int32(in.ArgF(i)); {
		case id >= 10004 && id <= 10007:
			return &vm.F[id-10004]
		case id >= 10013 && id <= 10015:
			return &vm.Pos[id-10013]
		}
	}
	vm.scratchF = in.ArgF(i)
	return &vm.scratchF
}

// posTarget is the vector position ops write: Pos, or Pos2 under flag 0x100.
func (vm *VM) posTarget() *[3]float32 {
	if vm.Flags&flagPos2 != 0 {
		return &vm.Pos2
	}
	return &vm.Pos
}

// takeInterrupt jumps to the pending label, or to the last label -1 seen on
// the way when there is no match. It reports whether a label was taken.
func (vm *VM) takeInterrupt() bool {
	want := int32(vm.pending)
	found := -1
	for i, in := range vm.script.Instrs {
		if in.Op == 0x40 && in.Arg(0) == want {
			found = i
			break
		}
		if in.Op == 0x40 && in.Arg(0) == -1 {
			found = i
		}
	}
	vm.pending = 0
	vm.Flags &^= flagWaiting
	if found < 0 {
		return false
	}
	vm.savedT, vm.savedPC = vm.T, vm.pc
	vm.T.Set(int32(vm.script.Instrs[found].Time))
	vm.pc = found + 1
	vm.Flags |= FlagVisible
	return true
}

// Run executes one frame (FUN_0043ee30). It reports true once the script has
// ended; an ended VM does nothing.
func (vm *VM) Run(env *Env) bool {
	if vm.pc < 0 {
		return true
	}
	if vm.Flags&flagFrozen != 0 {
		return false
	}
	vm.env = env
	vm.gs = env.GameSpeed
	if vm.gs == 0 || vm.Flags&flagRealtime != 0 {
		vm.gs = 1
	}
	if vm.pending != 0 && !vm.takeInterrupt() {
		vm.T.Add(-1, vm.gs)
		vm.endRun()
		return false
	}
	for {
		in := vm.instr()
		if vm.T.Cur < int32(in.Time) {
			break
		}
		switch vm.exec(in) {
		case execEnd:
			return true
		case execStall:
			vm.endRun()
			return false
		}
	}
	vm.endRun()
	return false
}

type execResult int

const (
	execNext  execResult = iota // keep executing this frame
	execStall                   // stop executing, finish the frame
	execEnd                     // the script ended (no end-of-frame work)
)

// exec runs one instruction and advances pc unless it jumped.
func (vm *VM) exec(in Instr) execResult {
	next := true
	switch in.Op {
	case 1, 0xffff:
		vm.Flags &^= FlagVisible
		vm.pc = -1
		return execEnd
	case 2:
		vm.pc = -1
		return execEnd
	case 3:
		vm.Flags |= FlagVisible
		vm.Sprite = int(vm.readI(in, 0))
		vm.SpriteTime = vm.T.Cur
	case 4:
		vm.jump(in.Arg(0), in.Arg(1))
		next = false
	case 5:
		*vm.intPtr(in, 0) -= 1
		if vm.readI(in, 0) >= 1 {
			vm.jump(in.Arg(1), in.Arg(2))
			next = false
		}
	case 6:
		*vm.intPtr(in, 0) = vm.readI(in, 1)
	case 7:
		*vm.floatPtr(in, 0) = vm.readF(in, 1)

	case 8, 10, 12, 14, 16:
		b := vm.readI(in, 1)
		p := vm.intPtr(in, 0)
		*p = intOp(in.Op-8, *p, b)
	case 9, 11, 13, 15:
		b := vm.readF(in, 1)
		p := vm.floatPtr(in, 0)
		*p = floatOp(in.Op-9, *p, b)
	case 17:
		b := vm.readF(in, 1)
		a := vm.readF(in, 0)
		*vm.floatPtr(in, 0) = float32(math.Mod(float64(a), float64(b)))
	case 18, 20, 22, 24, 26:
		a := vm.readI(in, 1)
		b := vm.readI(in, 2)
		p := vm.intPtr(in, 0)
		*p = intOp(in.Op-18, a, b)
	case 19, 21, 23, 25:
		a := vm.readF(in, 1)
		b := vm.readF(in, 2)
		*vm.floatPtr(in, 0) = floatOp(in.Op-19, a, b)
	case 27:
		b := vm.readF(in, 2)
		a := vm.readF(in, 1)
		*vm.floatPtr(in, 0) = float32(math.Mod(float64(a), float64(b)))

	case 28, 30, 32, 34, 36, 38:
		a, b := vm.readI(in, 0), vm.readI(in, 1)
		if compare((in.Op-28)/2, a, b) {
			vm.jump(in.Arg(2), in.Arg(3))
			next = false
		}
	case 29, 31, 33, 35, 37, 39:
		a, b := vm.readF(in, 0), vm.readF(in, 1)
		if compare((in.Op-29)/2, a, b) {
			vm.jump(in.Arg(2), in.Arg(3))
			next = false
		}

	case 40:
		r := vm.rng()
		n := vm.readI(in, 1)
		*vm.intPtr(in, 0) = int32(r.Intn(uint32(n)))
	case 41:
		r := vm.rng()
		b := vm.readF(in, 1)
		*vm.floatPtr(in, 0) = float32(r.Float() * b)
	case 42:
		*vm.floatPtr(in, 0) = float32(math.Sin(float64(vm.readF(in, 1))))
	case 43:
		*vm.floatPtr(in, 0) = float32(math.Cos(float64(vm.readF(in, 1))))
	case 44:
		*vm.floatPtr(in, 0) = float32(math.Tan(float64(vm.readF(in, 1))))
	case 45:
		// FUN_00463bf0, a CRT routine (likely acos); no shipped script uses it.
		*vm.floatPtr(in, 0) = float32(math.Acos(float64(vm.readF(in, 1))))
	case 46:
		*vm.floatPtr(in, 0) = float32(math.Atan(float64(vm.readF(in, 1))))
	case 47:
		*vm.floatPtr(in, 0) = zun.AddAngle(vm.readF(in, 0), 0)

	case 48:
		z := vm.readF(in, 2)
		y := vm.readF(in, 1)
		x := vm.readF(in, 0)
		*vm.posTarget() = [3]float32{x, y, z}
	case 49:
		for i := range 3 {
			vm.Rot[i] = vm.readF(in, i)
		}
		vm.Flags |= flagRotDirty
	case 50:
		vm.Scale[0] = vm.readF(in, 0)
		vm.Scale[1] = vm.readF(in, 1)
		vm.Flags |= flagScaleDirty
	case 51:
		vm.Alpha = uint8(vm.readI(in, 0))
	case 52:
		for i := range 3 {
			vm.Color[i] = uint8(vm.readI(in, i))
		}
	case 53:
		for i := range 3 {
			vm.AngVel[i] = vm.readF(in, i)
		}
		vm.Flags |= flagRotDirty
	case 54:
		vm.ScaleVel[0] = vm.readF(in, 0)
		vm.ScaleVel[1] = vm.readF(in, 1)
	case 55:
		vm.alphaI.begin(vm.readI(in, 1), 0)
		vm.alphaI.start[0] = int32(vm.Alpha)
		vm.alphaI.end[0] = int32(uint8(in.Arg(0)))
	case 56:
		dur := vm.readI(in, 0)
		mode := in.Arg(1)
		z := vm.readF(in, 4)
		y := vm.readF(in, 3)
		x := vm.readF(in, 2)
		vm.posI.Begin(dur, mode)
		vm.posI.Start = *vm.posTarget()
		vm.posI.End = [3]float32{x, y, z}
	case 57, 78:
		w, cur := &vm.colorI, &vm.Color
		if in.Op == 78 {
			w, cur = &vm.color2I, &vm.Color2
		}
		var start, end [3]int32
		for i := range 3 {
			start[i] = int32(cur[i])
			end[i] = vm.readI(in, 2+i)
		}
		w.begin(vm.readI(in, 0), int32(uint8(in.Arg(1))))
		w.start, w.end = start, end
	case 58, 79:
		w, cur := &vm.alphaI, vm.Alpha
		if in.Op == 79 {
			w, cur = &vm.alpha2I, vm.Alpha2
		}
		end := int32(uint8(vm.readI(in, 2)))
		w.begin(vm.readI(in, 0), int32(uint8(in.Arg(1))))
		w.start[0], w.end[0] = int32(cur), end
	case 59:
		z := vm.readF(in, 4)
		y := vm.readF(in, 3)
		x := vm.readF(in, 2)
		vm.rotI.Begin(vm.readI(in, 0), in.Arg(1))
		vm.rotI.Start = vm.Rot
		vm.rotI.End = [3]float32{x, y, z}
		vm.Flags |= flagRotDirty
	case 60:
		y := vm.readF(in, 3)
		x := vm.readF(in, 2)
		// FUN_0041ab70 restarts the timer but keeps the old tangents.
		vm.scaleI.Dur = vm.readI(in, 0)
		vm.scaleI.Mode = int32(uint8(in.Arg(1)))
		vm.scaleI.Start = [3]float32{vm.Scale[0], vm.Scale[1]}
		vm.scaleI.End = [3]float32{x, y}
		vm.scaleI.T.Reset()
		vm.Flags |= flagScaleDirty
	case 61:
		vm.Scale[0] = -vm.Scale[0]
		vm.Flags ^= FlagFlipX
		vm.Flags |= flagScaleDirty
	case 62:
		vm.Scale[1] = -vm.Scale[1]
		vm.Flags ^= FlagFlipY
		vm.Flags |= flagScaleDirty
	case 63, 69:
		if in.Op == 69 {
			vm.Flags &^= FlagVisible
		}
		if vm.pending == 0 {
			vm.Flags |= flagWaiting
			vm.T.Add(-1, vm.gs)
			return execStall
		}
		if !vm.takeInterrupt() {
			vm.T.Add(-1, vm.gs)
			return execStall
		}
		return execNext
	case 64:
		// label
	case 65:
		lo := uint32(uint16(in.Arg(0)))
		hi := uint32(uint16(in.Arg(0) >> 16))
		vm.Flags = vm.Flags&^0x3c0000 | lo<<18&0xc0000 | hi<<20&0x300000
	case 66:
		vm.Flags = vm.Flags&^flagBlendMask | uint32(in.Arg(0))<<4&flagBlendMask
	case 67:
		vm.Flags = vm.Flags&^0x3c00000 | uint32(in.Arg(0))<<22&0x3c00000
	case 68:
		vm.Layer = int32(uint8(in.Arg(0)))
	case 70:
		vm.UVVel[0] = vm.readF(in, 0)
	case 71:
		vm.UVVel[1] = vm.readF(in, 0)
	case 72:
		vm.Flags = vm.Flags&^FlagVisible | uint32(in.Arg(0))&1
	case 73:
		vm.Flags = vm.Flags&^0x800 | uint32(in.Arg(0))<<11&0x800
	case 74:
		vm.Flags = vm.Flags&^flagScroll | uint32(in.Arg(0))<<13&flagScroll
	case 75:
		vm.T.Add(-float32(vm.readI(in, 0)), vm.gs)
	case 76:
		for i := range 3 {
			vm.Color2[i] = uint8(vm.readI(in, i))
		}
	case 77:
		vm.Alpha2 = uint8(vm.readI(in, 0))
	case 80:
		vm.Flags = vm.Flags&^0x8000 | uint32(uint8(in.Arg(0)))<<15&0x8000
	case 81:
		vm.T, vm.pc = vm.savedT, vm.savedPC
		next = false
	case 82:
		vm.Flags = vm.Flags&^0x8000000 | uint32(in.Arg(0))<<27&0x8000000
	case 83:
		vm.Pos, vm.Offset = vm.Offset, [3]float32{}
	case 84:
		vm.Flags = vm.Flags&0xfe7fffff | 0x2400000
		vm.RingN = vm.readI(in, 0)
	case 85:
		vm.Flags = vm.Flags&^0x10000000 | uint32(in.Arg(0))<<28&0x10000000
	case 86:
		vm.Flags = vm.Flags&^flagRealtime | uint32(vm.readI(in, 0))<<29&flagRealtime
	case 87:
		vm.Flags = vm.Flags&^FlagAltRNG | uint32(uint8(in.Arg(0)))<<30&FlagAltRNG
	case 88:
		vm.spawnChild(int(vm.readI(in, 0)))
	case 89:
		vm.Flags = vm.Flags&^0x80000000 | uint32(in.Arg(0))<<31
	}
	if next {
		vm.pc++
	}
	return execNext
}

// spawnChild is op 88 (FUN_00448d00 + FUN_00449870): a child of the same
// file on the parent's layer, flagged for the alternate RNG and started
// without a reset, so the flag survives its first frame.
func (vm *VM) spawnChild(id int) {
	if vm.File.Scripts[id] == nil {
		return
	}
	c := New()
	c.Layer = vm.Layer
	c.Offset = [3]float32{}
	c.Flags |= FlagAltRNG
	c.StartKeep(vm.File, id, vm.env)
	c.Pos2 = vm.Pos
	c.Offset = vm.Offset
	if vm.env.Spawn != nil {
		vm.env.Spawn(c)
	}
	root := vm
	if vm.root != nil {
		root = vm.root
	}
	c.root = root
	if len(root.chain) == cap(root.chain) {
		root.chain = slices.DeleteFunc(root.chain, (*VM).Ended)
	}
	root.chain = append(root.chain, c)
}

// SetOffset is FUN_004492f0: the object's Offset, and on a root that of every
// child in its chain.
func (vm *VM) SetOffset(pos [3]float32) {
	vm.Offset = pos
	if vm.root != nil {
		return
	}
	for _, c := range vm.chain {
		if !c.Ended() {
			c.Offset = pos
		}
	}
}

func intOp(k int, a, b int32) int32 {
	switch k {
	case 0:
		return a + b
	case 2:
		return a - b
	case 4:
		return a * b
	case 6:
		if b == 0 {
			return a
		}
		return a / b
	case 8:
		if b == 0 {
			return a
		}
		return a % b
	}
	return a
}

func floatOp(k int, a, b float32) float32 {
	switch k {
	case 0:
		return a + b
	case 2:
		return a - b
	case 4:
		return float32(a * b)
	case 6:
		return float32(a / b)
	}
	return a
}

// compare evaluates jump condition k: ==, !=, <, <=, >, >=.
func compare[T int32 | float32](k int, a, b T) bool {
	switch k {
	case 0:
		return a == b
	case 1:
		return a != b
	case 2:
		return a < b
	case 3:
		return a <= b
	case 4:
		return a > b
	case 5:
		return a >= b
	}
	return false
}

// endRun is the frame's tail after the instructions: velocities, tweens and
// the timer tick, in the binary's order.
func (vm *VM) endRun() {
	gs := vm.gs
	for i := range 3 {
		if vm.AngVel[i] != 0 {
			vm.Rot[i] = zun.AddAngle(vm.Rot[i], float32(gs*vm.AngVel[i]))
			vm.Flags |= flagRotDirty
		}
	}
	if vm.ScaleVel[1] != 0 {
		vm.Flags |= flagScaleDirty
		vm.Scale[1] = float32(gs*vm.ScaleVel[1]) + vm.Scale[1]
	}
	if vm.ScaleVel[0] != 0 {
		vm.Flags |= flagRotDirty | flagScaleDirty
		vm.Scale[0] = float32(gs*vm.ScaleVel[0]) + vm.Scale[0]
	}
	for i := range 2 {
		u := float32(gs*vm.UVVel[i]) + vm.UV[i]
		if u >= 1 {
			u -= 1
		} else if u < 0 {
			u += 1
		}
		vm.UV[i] = u
	}
	// Flag 0x2000 adds the stage scroll DAT_00491e6c.. to Offset; not traced,
	// taken as 0.
	if vm.posI.Dur != 0 {
		*vm.posTarget() = vm.posI.Step(gs, 3)
	}
	if vm.colorI.dur != 0 {
		c := vm.colorI.step3(gs)
		for i := range 3 {
			vm.Color[i] = uint8(c[i])
		}
	}
	if vm.alphaI.dur != 0 {
		vm.Alpha = uint8(vm.alphaI.step1(gs))
	}
	if vm.scaleI.Dur != 0 {
		o := vm.scaleI.Step(gs, 2)
		vm.Scale = [2]float32{o[0], o[1]}
		vm.Flags |= flagScaleDirty
	}
	if vm.rotI.Dur != 0 {
		vm.Rot = vm.rotI.Step(gs, 3)
		vm.Flags |= flagRotDirty
	}
	if vm.color2I.dur != 0 {
		c := vm.color2I.step3(gs)
		for i := range 3 {
			vm.Color2[i] = uint8(c[i])
		}
	}
	if vm.alpha2I.dur != 0 {
		vm.Alpha2 = uint8(vm.alpha2I.step1(gs))
	}
	vm.T.Tick(gs)
}
