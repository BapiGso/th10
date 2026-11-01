package std

import (
	"encoding/binary"
	"math"

	"th10/zun"
)

// Camera is the stage object's camera block (stage+0x2a4c, 0x46 dwords),
// copied each frame to DAT_00491d7c where the rest of the game reads it.
type Camera struct {
	Pos    [3]float32 // +0x00
	Facing [3]float32 // +0x0c
	Up     [3]float32 // +0x18
	FOV    float32    // +0x48
	// Delta (+0xf0, DAT_00491e6c..74) is how far op 2 moved Pos this frame;
	// the calc zeroes it before the script runs. Enemies flagged 0x40000
	// (ECL 270/271) move along with it.
	Delta    [3]float32
	FogColor uint32  // +0x114, ARGB
	FogNear  float32 // +0xfc
	FogFar   float32 // +0x100
}

// VM is the stage object's script state: the timer at +0x38, the
// instruction pointer at +0x4c and the camera tweens.
type VM struct {
	f      *File
	pc     int
	T      zun.Timer
	Cam    Camera
	facing zun.Interp // +0x50, live while Dur (+0x94) != 0
	pos    zun.Interp // +0x9c, live while Dur (+0xe0) != 0
	// Op 12's mode byte (+0x20) and timer (+0x24); op 13's value
	// (DAT_004923a8). No TH10 .std uses either; what mode 1 does to the
	// camera (FUN_00404ed0) is not ported.
	shake  byte
	shakeT zun.Timer
	Op13   int32
}

// New starts f's script with the camera FUN_00402230 sets up: at
// (0, 0, -600) facing (0, 300, 600), up (0, 1, 0), and the field of view of
// the camera defaults (FUN_004216f0).
func New(f *File) *VM {
	v := &VM{f: f}
	v.Cam.Pos = [3]float32{0, 0, -600}
	v.Cam.Facing = [3]float32{0, 300, 600}
	v.Cam.Up = [3]float32{0, 1, 0}
	v.Cam.FOV = math.Float32frombits(0x3f060a92)
	v.T.Reset()
	return v
}

// Update is one frame of the STD calc FUN_00402720: clear the frame's
// position delta, then run the script (FUN_00403c80). gs is the game speed
// DAT_00476f78.
func (v *VM) Update(gs float32) {
	v.Cam.Delta = [3]float32{}
	v.run(gs)
}

// u32, f32 read the script at byte offset o.
func (v *VM) u32(o int) uint32  { return binary.LittleEndian.Uint32(v.f.Script[o:]) }
func (v *VM) i32(o int) int32   { return int32(v.u32(o)) }
func (v *VM) f32(o int) float32 { return math.Float32frombits(v.u32(o)) }
func (v *VM) vec(o int) [3]float32 {
	return [3]float32{v.f32(o), v.f32(o + 4), v.f32(o + 8)}
}

// run is FUN_00403c80. Instructions are {u32 time, u16 op, u16 size, args}
// and run while their time has come; op 0 stops the clock, op 1 jumps.
func (v *VM) run(gs float32) {
	s := v.f.Script
	for v.pc+8 <= len(s) && v.i32(v.pc) <= v.T.Cur {
		pc := v.pc
		a := pc + 8 // first argument
		switch binary.LittleEndian.Uint16(s[pc+4:]) {
		case 0:
			v.tweens(gs)
			return
		case 1:
			v.T.Set(v.i32(a + 4))
			v.pc = int(v.i32(a))
			continue
		case 2:
			old := v.Cam.Pos
			v.Cam.Pos = v.vec(a)
			for i := range old {
				v.Cam.Delta[i] = v.Cam.Pos[i] - old[i]
			}
		case 3:
			// The tangents keep whatever an earlier op 10 left.
			v.pos.Dur, v.pos.Mode = v.i32(a), v.i32(a+4)
			v.pos.Start, v.pos.End = v.Cam.Pos, v.vec(a+8)
			v.pos.T.Reset()
		case 4:
			v.Cam.Facing = v.vec(a)
		case 5:
			v.facing.Dur, v.facing.Mode = v.i32(a), v.i32(a+4)
			v.facing.Start, v.facing.End = v.Cam.Facing, v.vec(a+8)
			v.facing.T.Reset()
		case 6:
			v.Cam.Up = v.vec(a)
		case 7:
			v.Cam.FOV = v.f32(a)
		case 8:
			v.Cam.FogColor = v.u32(a)
			v.Cam.FogNear, v.Cam.FogFar = v.f32(a+4), v.f32(a+8)
		case 9:
			// The fog tween (FUN_00405040 / FUN_004049a0) only feeds
			// rendering: M4.
		case 10:
			// Hermite path (mode 8); the word after the duration is unused.
			v.pos.Dur, v.pos.Mode = v.i32(a), 8
			v.pos.Start = v.Cam.Pos
			v.pos.Tan0, v.pos.End, v.pos.Tan1 = v.vec(a+8), v.vec(a+20), v.vec(a+32)
			v.pos.T.Reset()
		case 11:
			v.facing.Dur, v.facing.Mode = v.i32(a), 8
			v.facing.Start = v.Cam.Facing
			v.facing.Tan0, v.facing.End, v.facing.Tan1 = v.vec(a+8), v.vec(a+20), v.vec(a+32)
			v.facing.T.Reset()
		case 12:
			v.shake = s[a]
			v.shakeT.Reset()
		case 13:
			v.Op13 = v.i32(a)
		case 14:
			// Interrupts one of the stage's ANM VMs: M4.
		}
		v.pc += int(int16(binary.LittleEndian.Uint16(s[pc+6:])))
	}
	v.T.Tick(gs)
	v.tweens(gs)
}

// tweens is the tail of FUN_00403c80, run even when op 0 holds the clock.
func (v *VM) tweens(gs float32) {
	if v.facing.Dur != 0 {
		v.Cam.Facing = v.facing.Step(gs, 3)
	}
	if v.pos.Dur != 0 {
		v.Cam.Pos = v.pos.Step(gs, 3)
	}
}
