package anm

import (
	"reflect"
	"slices"
	"testing"
)

// synthN is synth for several scripts, script i from the i-th list.
func synthN(scripts ...[]Instr) *File {
	f := &File{Scripts: map[int]*Script{}}
	for id, instrs := range scripts {
		f.Scripts[id] = synth(instrs...).Scripts[0]
		f.Scripts[id].ID = id
	}
	return f
}

// parked is a script that shows nothing and stops at once.
func parked() *File { return synth(ins(0, 63, 0)) }

// SpawnFront puts its object at the head of the list, SpawnTail at the end
// (FUN_00448a50, FUN_004489d0); both set the layer and the alternate-RNG
// flag, which the start without a reset keeps (FUN_00448e30, FUN_00448d00).
func TestSpawnFrontTail(t *testing.T) {
	env := testEnv()
	w := &World{}
	old := New()
	old.Start(parked(), 0, env)
	w.Add(old)
	front := w.SpawnFront(parked(), 0, 5, env)
	tail := w.SpawnTail(parked(), 0, 6, env)
	if want := []*VM{front, old, tail}; !slices.Equal(w.list, want) {
		t.Fatalf("list order wrong: front at %d, old at %d, tail at %d",
			slices.Index(w.list, front), slices.Index(w.list, old), slices.Index(w.list, tail))
	}
	for _, c := range []struct {
		vm    *VM
		layer int32
	}{{front, 5}, {tail, 6}} {
		vm := c.vm
		if vm.Layer != c.layer || vm.Flags&FlagAltRNG == 0 || vm.Offset != [3]float32{} {
			t.Errorf("layer %d flags %#x offset %v, want layer %d with alt RNG and zero offset",
				vm.Layer, vm.Flags, vm.Offset, c.layer)
		}
		if vm.Ended() || !w.Contains(vm) {
			t.Errorf("layer %d: ended %v contained %v", c.layer, vm.Ended(), w.Contains(vm))
		}
	}
}

// A missing script still inserts the object, ended, until the next Run.
func TestSpawnMissingScript(t *testing.T) {
	env := testEnv()
	w := &World{}
	for _, spawn := range []func(*File, int, int32, *Env) *VM{w.SpawnFront, w.SpawnTail} {
		vm := spawn(parked(), 99, 5, env)
		if vm == nil || !vm.Ended() || !w.Contains(vm) {
			t.Fatalf("missing script: vm %v ended %v contained %v", vm != nil, vm != nil && vm.Ended(), w.Contains(vm))
		}
	}
	w.Run(env)
	if w.Len() != 0 {
		t.Errorf("%d objects after the run, want 0", w.Len())
	}
}

// Contains follows Add, AddFront, Run's removal and Clear.
func TestContains(t *testing.T) {
	env := testEnv()
	w, other := &World{}, &World{}
	a, b := New(), New()
	a.Start(parked(), 0, env)
	b.Start(parked(), 0, env)
	if w.Contains(nil) || w.Contains(a) {
		t.Fatal("contains nil or a VM never added")
	}
	w.Add(a)
	w.AddFront(b)
	if !w.Contains(a) || !w.Contains(b) || other.Contains(a) {
		t.Fatal("after Add/AddFront: wrong membership")
	}
	if w.list[0] != b {
		t.Fatal("AddFront did not insert at the head")
	}
	w.Delete(a)
	w.Run(env)
	if w.Contains(a) || !w.Contains(b) {
		t.Fatalf("after deleting a: contains a %v b %v", w.Contains(a), w.Contains(b))
	}
	w.Clear()
	if w.Contains(b) {
		t.Error("contains b after Clear")
	}
}

// chained returns a root with two op-88 children in its chain, of which only
// c1 is in w (c2 has left it).
func chained(w *World) (root, c1, c2 *VM) {
	root, c1, c2 = New(), New(), New()
	c1.root, c2.root = root, root
	root.chain = []*VM{c1, c2}
	w.Add(root)
	w.Add(c1)
	return
}

// The handle helpers (FUN_004492a0, FUN_004495e0, FUN_00449590,
// FUN_00449350, FUN_004492f0) act on the object and, on a root, on the
// children still in the world; on a child only on the child; on an object
// no longer in the world (or nil) not at all.
func TestHandleHelpers(t *testing.T) {
	w := &World{}
	root, c1, c2 := chained(w)
	draws := func() [3]bool {
		return [3]bool{root.Flags&FlagDraw != 0, c1.Flags&FlagDraw != 0, c2.Flags&FlagDraw != 0}
	}

	w.Hide(root)
	if got := draws(); got != [3]bool{false, false, true} {
		t.Errorf("Hide(root): draw bits %v, want [false false true]", got)
	}
	w.Show(root)
	if got := draws(); got != [3]bool{true, true, true} {
		t.Errorf("Show(root): draw bits %v, want all set", got)
	}
	w.Hide(c1)
	if got := draws(); got != [3]bool{true, false, true} {
		t.Errorf("Hide(c1): draw bits %v, want [true false true]", got)
	}
	w.Show(c1)
	w.Hide(c2)
	w.Hide(nil)
	if got := draws(); got != [3]bool{true, true, true} {
		t.Errorf("Hide(c2), Hide(nil): draw bits %v, want all set", got)
	}

	w.Place(root, [3]float32{1, 2, 3})
	want := [3]float32{225, 18, 3}
	if root.Offset != want || c1.Offset != want || c2.Offset != [3]float32{} {
		t.Errorf("Place: offsets %v %v %v, want %v %v and zero", root.Offset, c1.Offset, c2.Offset, want, want)
	}
	w.SetOffset(root, [3]float32{4, 5, 6})
	want = [3]float32{4, 5, 6}
	if root.Offset != want || c1.Offset != want || c2.Offset != [3]float32{} {
		t.Errorf("SetOffset: offsets %v %v %v, want %v %v and zero", root.Offset, c1.Offset, c2.Offset, want, want)
	}
	w.Place(c1, [3]float32{0, 0, 0})
	if root.Offset != want || c1.Offset != [3]float32{224, 16, 0} {
		t.Errorf("Place(c1): offsets %v %v", root.Offset, c1.Offset)
	}
	w.Place(c2, [3]float32{9, 9, 9})
	w.SetOffset(nil, [3]float32{9, 9, 9})
	if c2.Offset != [3]float32{} {
		t.Errorf("Place on an object out of the world moved it to %v", c2.Offset)
	}

	w.Delete(c2)
	if c2.Flags&FlagDelete != 0 {
		t.Error("Delete marked an object out of the world")
	}
	w.Delete(root)
	if root.Flags&FlagDelete == 0 || c1.Flags&FlagDelete == 0 || c2.Flags&FlagDelete != 0 {
		t.Errorf("Delete(root): flags %#x %#x %#x", root.Flags, c1.Flags, c2.Flags)
	}
	w.Run(testEnv())
	if w.Contains(root) || w.Contains(c1) || w.Len() != 0 {
		t.Errorf("after the run: contains root %v c1 %v, len %d", w.Contains(root), w.Contains(c1), w.Len())
	}
	root.Flags &^= FlagDraw
	w.Show(root) // no longer in w
	if root.Flags&FlagDraw != 0 {
		t.Error("Show acted on an object the run removed")
	}
}

// restartFile's script 0 flips X, starts all seven interpolations, sets
// colour 2, the colour-2 select bit, additive blend and I0, then stops at 5;
// script 1 sets I1 and stops at 3.
func restartFile() *File {
	return synthN(
		[]Instr{
			ins(0, 76, 0, 10, 20, 30),
			ins(0, 77, 0, 40),
			ins(0, 61, 0),
			ins(0, 56, 0, 100, 0, f32arg(0), f32arg(0), f32arg(0)),
			ins(0, 57, 0, 100, 0, 0, 0, 0),
			ins(0, 58, 0, 100, 0, 0),
			ins(0, 59, 0, 100, 0, f32arg(0), f32arg(0), f32arg(0)),
			ins(0, 60, 0, 100, 0, f32arg(-1), f32arg(1)),
			ins(0, 78, 0, 100, 0, 200, 200, 200),
			ins(0, 79, 0, 100, 0, 200),
			ins(0, 80, 0, 1),
			ins(0, 66, 0, 1),
			ins(0, 6, 1, 10000, 42),
			ins(5, 63, 0),
		},
		[]Instr{
			ins(0, 6, 1, 10001, 7),
			ins(3, 63, 0),
		},
	)
}

// Restart (FUN_0043e8b0) undoes the flip, resets the low flag bits, the
// colour and every interpolation, and runs the new script's first frame,
// keeping the variables, the high flag bits, positions and colour 2.
func TestRestart(t *testing.T) {
	env := testEnv()
	vm := New()
	vm.Start(restartFile(), 0, env)
	for range 9 {
		vm.Run(env)
	}
	if vm.Scale != [2]float32{-1, 1} || vm.Flags&FlagFlipX == 0 || vm.Flags&0x8000 == 0 || vm.Color == [3]uint8{255, 255, 255} {
		t.Fatalf("test setup: scale %v flags %#x colour %v", vm.Scale, vm.Flags, vm.Color)
	}
	vm.Flags |= FlagAltRNG
	vm.Pos, vm.Offset, vm.Layer = [3]float32{7, 8, 9}, [3]float32{1, 2, 3}, 4
	vm.savedPC = 3
	color2, alpha2 := vm.Color2, vm.Alpha2

	vm.Restart(1, env)
	if vm.ScriptID != 1 || vm.Ended() || vm.pc != 1 || vm.savedPC != 0 || vm.T.Cur != 1 {
		t.Errorf("script %d ended %v pc %d savedPC %d T %d, want 1 false 1 0 1",
			vm.ScriptID, vm.Ended(), vm.pc, vm.savedPC, vm.T.Cur)
	}
	if vm.Scale != [2]float32{1, 1} {
		t.Errorf("scale %v, want the flip undone to [1 1]", vm.Scale)
	}
	if lo := vm.Flags & 0xffff; lo != FlagDraw|FlagRotDirty {
		t.Errorf("low flags %#x, want %#x", lo, FlagDraw|FlagRotDirty)
	}
	if vm.Flags&FlagAltRNG == 0 {
		t.Error("lost the alt-RNG flag")
	}
	if vm.Color != [3]uint8{255, 255, 255} || vm.Alpha != 255 {
		t.Errorf("colour %v alpha %d, want white 255", vm.Color, vm.Alpha)
	}
	if vm.Color2 != color2 || vm.Alpha2 != alpha2 {
		t.Errorf("colour 2 %v/%d, want it kept at %v/%d", vm.Color2, vm.Alpha2, color2, alpha2)
	}
	durs := []int32{vm.posI.Dur, vm.rotI.Dur, vm.scaleI.Dur, vm.colorI.dur, vm.alphaI.dur, vm.color2I.dur, vm.alpha2I.dur}
	if slices.ContainsFunc(durs, func(d int32) bool { return d != 0 }) {
		t.Errorf("interpolation durations %v, want all 0", durs)
	}
	if vm.I[0] != 42 || vm.I[1] != 7 {
		t.Errorf("I0 %d I1 %d, want 42 7", vm.I[0], vm.I[1])
	}
	if vm.Pos != [3]float32{7, 8, 9} || vm.Offset != [3]float32{1, 2, 3} || vm.Layer != 4 {
		t.Errorf("pos %v offset %v layer %d changed", vm.Pos, vm.Offset, vm.Layer)
	}
}

// Restart with no file or a missing script does nothing.
func TestRestartMissing(t *testing.T) {
	env := testEnv()
	vm := New()
	vm.Start(restartFile(), 0, env)
	before := *vm
	vm.Restart(99, env)
	if !reflect.DeepEqual(*vm, before) {
		t.Error("Restart with a missing script changed the VM")
	}
	bare := New()
	before = *bare
	bare.Restart(0, env)
	if !reflect.DeepEqual(*bare, before) {
		t.Error("Restart with no file changed the VM")
	}
}

// World.Restart (FUN_004496d0) restarts the object with its own file, and
// only while it is in the world; its chain is left alone.
func TestWorldRestart(t *testing.T) {
	env := testEnv()
	w := &World{}
	f := restartFile()
	vm, kid := New(), New()
	vm.Start(f, 0, env)
	kid.Start(f, 0, env)
	kid.root, vm.chain = vm, []*VM{kid}
	w.Restart(vm, 1, env)
	if vm.ScriptID != 0 {
		t.Fatal("restarted an object not in the world")
	}
	w.Add(vm)
	w.Add(kid)
	w.Restart(vm, 1, env)
	if vm.ScriptID != 1 || vm.I[1] != 7 {
		t.Errorf("in the world: script %d I1 %d, want 1 7", vm.ScriptID, vm.I[1])
	}
	if kid.ScriptID != 0 {
		t.Error("restarted the chain too")
	}
	w.Restart(nil, 1, env)
}
