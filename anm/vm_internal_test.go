package anm

import (
	"encoding/binary"
	"testing"

	"th10/zun"
)

// ins builds one instruction; synth lays instructions out with their byte
// offsets as the loader would.
func ins(t, op, mask int, args ...int32) Instr {
	b := make([]byte, 4*len(args))
	for i, a := range args {
		binary.LittleEndian.PutUint32(b[4*i:], uint32(a))
	}
	return Instr{Time: t, Op: op, Mask: mask, Args: b}
}

func synth(instrs ...Instr) *File {
	off := 0
	for i := range instrs {
		instrs[i].Offset = off
		off += 8 + len(instrs[i].Args)
	}
	return &File{Scripts: map[int]*Script{0: newScript(0, instrs)}}
}

func testEnv() *Env { return &Env{RNG: &zun.RNG{State: 1}, Alt: &zun.RNG{State: 2}} }

// A stop parks the script with its timer held until an interrupt jumps to the
// matching label; op 81 then returns to the stop.
func TestStopAndInterrupt(t *testing.T) {
	f := synth(
		ins(0, 63, 0),
		ins(10, 64, 0, 7),
		ins(10, 6, 1, 10000, 42),
		ins(11, 81, 0),
	)
	env := testEnv()
	vm := New()
	vm.Start(f, 0, env)
	for range 3 {
		vm.Run(env)
	}
	if vm.T.Cur != 0 || vm.Flags&flagWaiting == 0 || vm.Visible() {
		t.Fatalf("parked: cur %d flags %#x", vm.T.Cur, vm.Flags)
	}
	vm.Interrupt(99) // no such label and no -1 fallback
	vm.Run(env)
	if vm.T.Cur != 0 || vm.pending != 0 || vm.pc != 0 {
		t.Fatalf("unmatched interrupt: cur %d pending %d pc %d", vm.T.Cur, vm.pending, vm.pc)
	}
	vm.Interrupt(7)
	vm.Run(env)
	if vm.I[0] != 42 || !vm.Visible() || vm.T.Cur != 11 {
		t.Fatalf("after interrupt: I0 %d visible %v cur %d", vm.I[0], vm.Visible(), vm.T.Cur)
	}
	vm.Run(env) // op 81 restores the stop's timer and pc
	if vm.pc != 0 || vm.T.Cur != 0 || vm.Flags&flagWaiting == 0 {
		t.Errorf("after op 81: pc %d cur %d flags %#x", vm.pc, vm.T.Cur, vm.Flags)
	}
}

// With no matching label the last label -1 is taken.
func TestInterruptFallback(t *testing.T) {
	f := synth(
		ins(0, 63, 0),
		ins(5, 64, 0, -1),
		ins(5, 6, 1, 10000, 1),
		ins(6, 63, 0),
		ins(8, 64, 0, -1),
		ins(8, 6, 1, 10000, 2),
		ins(9, 63, 0),
	)
	env := testEnv()
	vm := New()
	vm.Start(f, 0, env)
	vm.Interrupt(3)
	vm.Run(env)
	if vm.I[0] != 2 || vm.T.Cur != 9 {
		t.Errorf("fallback: I0 %d cur %d, want the last -1 label (I0 2, cur 8+1)", vm.I[0], vm.T.Cur)
	}
}

// Op 75 rewinds the timer, delaying what follows by n frames.
func TestWaitOp75(t *testing.T) {
	f := synth(
		ins(0, 75, 0, 5),
		ins(0, 6, 1, 10000, 1),
		ins(9, 1, 0),
	)
	env := testEnv()
	vm := New()
	vm.Start(f, 0, env)
	for i := range 5 {
		if vm.I[0] != 0 {
			t.Fatalf("I0 set after %d frames, want 5", i)
		}
		vm.Run(env)
	}
	if vm.I[0] != 1 {
		t.Errorf("I0 = %d after 5 frames, want 1", vm.I[0])
	}
}

// Op 5 decrements its counter and jumps while it stays >= 1.
func TestLoopOp5(t *testing.T) {
	f := synth(
		ins(0, 6, 1, 10000, 3),
		ins(0, 8, 1, 10001, 1),
		ins(0, 5, 1, 10000, 16, 0),
		ins(0, 1, 0),
	)
	env := testEnv()
	vm := New()
	if vm.Start(f, 0, env); !vm.Ended() {
		t.Fatal("script did not end in its first frame")
	}
	if vm.I[0] != 0 || vm.I[1] != 3 {
		t.Errorf("I0 %d I1 %d, want 0 and 3", vm.I[0], vm.I[1])
	}
}

// The colour tween truncates the eased delta before adding start; the alpha
// tween truncates the sum. 255 -> 0 at 1/4: colour 255 + trunc(-63.75) = 192,
// alpha trunc(191.25) = 191.
func TestTweenTruncation(t *testing.T) {
	f := synth(
		ins(0, 57, 0, 4, 0, 0, 0, 0),
		ins(0, 58, 0, 4, 0, 0),
		ins(10, 1, 0),
	)
	env := testEnv()
	vm := New()
	vm.Start(f, 0, env)
	if vm.Color != [3]uint8{192, 192, 192} || vm.Alpha != 191 {
		t.Errorf("after one frame: colour %v alpha %d, want 192 and 191", vm.Color, vm.Alpha)
	}
	for range 3 {
		vm.Run(env)
	}
	if vm.Color != [3]uint8{} || vm.Alpha != 0 || vm.colorI.dur != 0 {
		t.Errorf("finished: colour %v alpha %d dur %d", vm.Color, vm.Alpha, vm.colorI.dur)
	}
}

// Reset keeps Offset and Layer; a missing script zeroes the object.
func TestResetKeepsOffset(t *testing.T) {
	vm := New()
	vm.Offset, vm.Layer, vm.I[0] = [3]float32{1, 2, 3}, 5, 9
	vm.Reset()
	if vm.Offset != [3]float32{1, 2, 3} || vm.Layer != 5 || vm.I[0] != 0 || vm.Scale != [2]float32{1, 1} {
		t.Errorf("after reset: %+v", vm)
	}
	if vm.Start(synth(), 1, testEnv()) || !vm.Ended() || vm.Offset != [3]float32{} {
		t.Errorf("missing script: ended %v offset %v", vm.Ended(), vm.Offset)
	}
}

// The sprite op marks Mat stale for the renderer and leaves Mat itself
// alone; Reset clears both.
func TestSpriteMarksMatStale(t *testing.T) {
	f := synth(
		ins(0, 3, 0, 5),
		ins(2, 3, 0, 6),
		ins(9, 0, 0),
	)
	env := testEnv()
	vm := New()
	if vm.MatStale {
		t.Fatal("New: MatStale set")
	}
	vm.Start(f, 0, env)
	if vm.Sprite != 5 || !vm.MatStale || vm.SpriteTime != 0 {
		t.Fatalf("op 3 at 0: sprite %d stale %v time %d", vm.Sprite, vm.MatStale, vm.SpriteTime)
	}
	m := [4][4]float32{{2, 0, 0, 0}, {0, 3, 0, 0}, {0, 0, 1, 0}, {0, 0, 0, 1}}
	vm.Mat, vm.MatStale = m, false
	for range 5 {
		if vm.Sprite == 6 {
			break
		}
		vm.Run(env)
	}
	if vm.Sprite != 6 || !vm.MatStale || vm.Mat != m || vm.SpriteTime != 2 {
		t.Fatalf("op 3 at 2: sprite %d stale %v mat %v time %d", vm.Sprite, vm.MatStale, vm.Mat, vm.SpriteTime)
	}
	vm.Reset()
	if vm.MatStale || vm.Mat != ([4][4]float32{}) {
		t.Errorf("Reset: stale %v mat %v", vm.MatStale, vm.Mat)
	}

	vm.SetSprite(3)
	if vm.Sprite != 3 || !vm.MatStale {
		t.Errorf("SetSprite(3): sprite %d stale %v", vm.Sprite, vm.MatStale)
	}
}
