package ecl

import (
	"encoding/binary"
	"math"
	"reflect"
	"testing"

	"th10/anm"
	"th10/entity/enemy"
	"th10/scene/stage"
)

// seRec records the sound ids played through Context.Audio.
type seRec struct{ ids []int }

func (r *seRec) PlaySE(string)   {}
func (r *seRec) PlaySEID(id int) { r.ids = append(r.ids, id) }
func (r *seRec) PlayBGM(int)     {}
func (r *seRec) StopBGM()        {}
func (r *seRec) FadeOutBGM(int)  {}

func anmIns(op int, args ...uint32) anm.Instr {
	b := make([]byte, 4*len(args))
	for i, a := range args {
		binary.LittleEndian.PutUint32(b[4*i:], a)
	}
	return anm.Instr{Op: op, Args: b}
}

// visualANM is the ECL's ANM file 1 in these tests: scripts 0-9 show sprite
// 5 (32×48) at scale (2, 0.5) and stop; script 20 stops with no sprite.
const visualANM = "visual_test.anm"

func init() {
	f := &anm.File{Sprites: map[int]anm.Sprite{5: {ID: 5, W: 32, H: 48}}, Scripts: map[int]*anm.Script{}}
	for id := range 10 {
		f.Scripts[id] = &anm.Script{ID: id, Instrs: []anm.Instr{
			anmIns(3, 5),
			anmIns(50, math.Float32bits(2), math.Float32bits(0.5)),
			anmIns(63),
		}}
	}
	f.Scripts[20] = &anm.Script{ID: 20, Instrs: []anm.Instr{anmIns(63)}}
	anmFiles.Store(visualANM, f)
}

// visualVM is a VM with a full stage context (World, Anm, BulletEnv), the
// test file as its ANM file 1, a sound recorder, and one enemy whose final
// position is (10, 20, 3).
func visualVM(t *testing.T) (*VM, *stage.Context, *eclActor, *seRec) {
	t.Helper()
	ctx := newTestContext()
	rec := &seRec{}
	ctx.Audio = rec
	vm := newTestVM()
	vm.prog.AnimNames = []string{visualANM}
	vm.ctx = ctx
	e := enemy.New(0, 0, 100, enemy.TypeFairy)
	e.Damaged = vm.damaged
	a := vm.actorFor(e)
	a.mv.final.pos = [3]float32{10, 20, 3}
	return vm, ctx, a, rec
}

func gameOp(vm *VM, ctx *stage.Context, a *eclActor, op uint16, params ...Param) {
	vm.execGameOp(ctx, &eclTask{owner: a.e, asyncID: -1}, &Instr{Opcode: op, Params: params})
}

// slotVM is a stopped object of the test file in the world, as a slot op
// leaves one.
func slotVM(ctx *stage.Context) *anm.VM {
	return ctx.World.SpawnFront(loadANM(visualANM), 0, 5, ctx.Anm)
}

// TestFaceDir is the direction switch of FUN_0040dc80 (0x40e2e4-0x40e374):
// the bounds are the float32 constants at 0x470d30 (-0.1) and 0x470c18
// (0.1), both exclusive, and NaN fails both compares.
func TestFaceDir(t *testing.T) {
	nan := float32(math.NaN())
	for _, c := range []struct {
		old int32
		dx  float32
		dir int32
		k   int32
		ok  bool
	}{
		{0, 0, 0, 0, false},
		{0, 0.1, 0, 0, false},
		{0, -0.1, 0, 0, false},
		{0, nan, 0, 0, false},
		{0, -0.11, -1, 1, true},
		{0, 0.11, 1, 2, true},
		{-1, 0, 0, 3, true},
		{-1, nan, 0, 3, true},
		{-1, 1, 1, 2, true},
		{-1, -1, -1, 0, false},
		{1, 0.05, 0, 4, true},
		{1, -1, -1, 1, true},
		{1, 1, 1, 0, false},
		// An old value outside -1..1 restarts with the base script.
		{2, 0, 0, 0, true},
		{2, -1, -1, 0, true},
	} {
		dir, k, ok := faceDir(c.old, c.dx)
		if dir != c.dir || ok != c.ok || ok && k != c.k {
			t.Errorf("faceDir(%d, %v) = (%d, %d, %v), want (%d, %d, %v)", c.old, c.dx, dir, k, ok, c.dir, c.k, c.ok)
		}
	}
}

// TestHitSE: 0x13, or 0x23 for a boss whose phase is nearly over.
func TestHitSE(t *testing.T) {
	for _, c := range []struct {
		boss, card, survival bool
		hp                   int32
		want                 int
	}{
		{false, false, false, 0, 0x13},
		{false, true, false, 0, 0x13},
		{true, true, true, 0, 0x13},
		{true, true, false, 299, 0x23},
		{true, true, false, 300, 0x13},
		{true, true, false, -5, 0x23},
		{true, false, false, 899, 0x23},
		{true, false, false, 900, 0x13},
		// Bit 3 counts only during a card.
		{true, false, true, 899, 0x23},
	} {
		if got := hitSE(c.boss, c.card, c.survival, c.hp); got != c.want {
			t.Errorf("hitSE(boss %v, card %v, survival %v, %d) = %#x, want %#x", c.boss, c.card, c.survival, c.hp, got, c.want)
		}
	}
}

// TestHitFlash: a hit lights slot 0 in colour 2 (blue) for one frame and
// holds the next flash off for four, so steady hits flash every fifth frame,
// each with the hit sound.
func TestHitFlash(t *testing.T) {
	vm, ctx, a, rec := visualVM(t)
	body := slotVM(ctx)
	a.vms[0] = body
	var on []int
	for f := range 12 {
		a.e.ECLFlags |= 0x2000
		vm.updateVisual(a)
		if body.Flags&0x8000 != 0 {
			on = append(on, f)
		}
	}
	if !reflect.DeepEqual(on, []int{0, 5, 10}) {
		t.Errorf("colour 2 on frames %v, want [0 5 10]", on)
	}
	if body.Color2 != [3]uint8{0, 0, 255} || body.Alpha2 != 255 {
		t.Errorf("colour 2 = %v/%d, want blue (0xff0000ff)", body.Color2, body.Alpha2)
	}
	if !reflect.DeepEqual(rec.ids, []int{0x13, 0x13, 0x13}) {
		t.Errorf("sounds %#x, want three 0x13", rec.ids)
	}
	if a.flash != 3 {
		t.Errorf("flash = %d, want 3 (4 at frame 10, one frame down)", a.flash)
	}

	t.Run("no hit", func(t *testing.T) {
		vm, ctx, a, rec := visualVM(t)
		body := slotVM(ctx)
		a.vms[0] = body
		for range 3 {
			vm.updateVisual(a)
		}
		if body.Flags&0x8000 != 0 || len(rec.ids) != 0 || a.flash != 0 {
			t.Errorf("flags %#x sounds %v flash %d without a hit", body.Flags, rec.ids, a.flash)
		}
	})
	t.Run("boss", func(t *testing.T) {
		vm, ctx, a, rec := visualVM(t)
		a.vms[0] = slotVM(ctx)
		a.e.ECLFlags |= 0x8000 | 0x2000
		vm.card.Active = true
		a.phaseHP = 200
		vm.updateVisual(a)
		if !reflect.DeepEqual(rec.ids, []int{0x23}) {
			t.Errorf("sounds %#x, want 0x23 (card, 200 HP left)", rec.ids)
		}
	})
	t.Run("no slot 0", func(t *testing.T) {
		vm, _, a, rec := visualVM(t)
		a.e.ECLFlags |= 0x2000
		vm.updateVisual(a)
		if a.flash != 4 || len(rec.ids) != 1 {
			t.Errorf("flash %d sounds %v, want the counter and sound without an object", a.flash, rec.ids)
		}
	})
	t.Run("departed slot 0", func(t *testing.T) {
		vm, ctx, a, _ := visualVM(t)
		body := slotVM(ctx)
		a.vms[0] = body
		ctx.World.Delete(body)
		ctx.World.Run(ctx.Anm)
		a.e.ECLFlags |= 0x2000
		vm.updateVisual(a)
		if a.vms[0] != nil {
			t.Error("slot 0 kept an object that left the world")
		}
		if body.Flags&0x8000 != 0 || body.Color2 != ([3]uint8{}) {
			t.Errorf("flashed a departed object: flags %#x colour 2 %v", body.Flags, body.Color2)
		}
	})
}

// TestFaceMotion: with 0x1000 (ins_262 on slot 0) slot 0 restarts with the
// base script plus the turn's offset whenever the direction changes.
func TestFaceMotion(t *testing.T) {
	vm, ctx, a, _ := visualVM(t)
	gameOp(vm, ctx, a, opAnmSelect, intParam(1))
	gameOp(vm, ctx, a, opAnmSetMain, intParam(0), intParam(0))
	body := a.vms[0]
	if body == nil || a.e.ECLFlags&0x1000 == 0 || a.dirScript != 0 || a.dir != 0 {
		t.Fatalf("after 262(0, 0): slot 0 %v flags %#x dirScript %d dir %d", body, a.e.ECLFlags, a.dirScript, a.dir)
	}

	a.mv.final.delta[0] = 1
	a.e.ECLFlags |= 0x2000
	vm.updateVisual(a)
	if a.vms[0] != body || body.ScriptID != 2 || a.dir != 1 {
		t.Fatalf("still → right: script %d dir %d, want 2 and 1 on the same object", body.ScriptID, a.dir)
	}
	if body.Flags&0x8000 == 0 {
		t.Error("the restart undid this frame's flash: the flash goes after the direction")
	}

	a.mv.final.delta[0] = 0.05
	vm.updateVisual(a)
	if body.ScriptID != 4 || a.dir != 0 {
		t.Fatalf("right → still: script %d dir %d, want 4 and 0", body.ScriptID, a.dir)
	}

	body.Color = [3]uint8{1, 2, 3}
	vm.updateVisual(a)
	if body.Color != [3]uint8{1, 2, 3} {
		t.Error("restarted without a change of direction")
	}

	a.e.ECLFlags &^= 0x1000
	a.mv.final.delta[0] = -1
	vm.updateVisual(a)
	if a.dir != 0 || body.ScriptID != 4 {
		t.Errorf("without 0x1000: dir %d script %d, want both kept", a.dir, body.ScriptID)
	}
}

// TestSlotsFollow: slots 0-7 move to the enemy every frame, in screen space
// (FUN_00449350) or as is with 0x40000 (FUN_004492f0); 8 and 9 stay put.
func TestSlotsFollow(t *testing.T) {
	vm, ctx, a, _ := visualVM(t)
	for _, i := range []int{0, 7, 8} {
		a.vms[i] = slotVM(ctx)
	}
	vm.updateVisual(a)
	want := [3]float32{234, 36, 3}
	if a.vms[0].Offset != want || a.vms[7].Offset != want {
		t.Errorf("offsets %v %v, want %v", a.vms[0].Offset, a.vms[7].Offset, want)
	}
	if a.vms[8].Offset != ([3]float32{}) {
		t.Errorf("slot 8 moved to %v", a.vms[8].Offset)
	}

	a.e.ECLFlags |= 0x40000
	a.mv.final.pos = [3]float32{-5, 6, 7}
	vm.updateVisual(a)
	if want := [3]float32{-5, 6, 7}; a.vms[0].Offset != want {
		t.Errorf("0x40000 offset %v, want %v", a.vms[0].Offset, want)
	}
}

// TestSlotOps: 259/262/269 replace a slot's object with a new one on layer
// 5; slot 0's sizes the enemy and 262 makes it face the motion.
func TestSlotOps(t *testing.T) {
	vm, ctx, a, _ := visualVM(t)
	f := loadANM(visualANM)
	gameOp(vm, ctx, a, opAnmSelect, intParam(1))
	if a.anmFile != 1 {
		t.Fatalf("anmFile = %d after 258(1)", a.anmFile)
	}

	gameOp(vm, ctx, a, opAnmSetSlot, intParam(0), intParam(3))
	v := a.vms[0]
	if v == nil || !ctx.World.Contains(v) || v.File != f || v.ScriptID != 3 || v.Layer != 5 || v.Flags&anm.FlagAltRNG == 0 {
		t.Fatalf("259(0, 3): slot 0 = %+v", v)
	}
	if a.curScript != 3 || a.bodyFile != 1 || a.e.ECLFlags&0x1000 != 0 {
		t.Errorf("259(0, 3): curScript %d bodyFile %d flags %#x", a.curScript, a.bodyFile, a.e.ECLFlags)
	}
	if want := [2]float32{24, 64}; a.mv.size != want { // (48·0.5, 32·2)
		t.Errorf("size %v, want %v", a.mv.size, want)
	}

	gameOp(vm, ctx, a, opAnmSetSlot, intParam(0), intParam(20))
	if a.vms[0] == v || v.Flags&anm.FlagDelete == 0 {
		t.Error("259 did not replace slot 0's object")
	}
	if a.mv.size != ([2]float32{}) {
		t.Errorf("size %v with no sprite, want 0", a.mv.size)
	}

	old, n := a.vms[0], ctx.World.Len()
	gameOp(vm, ctx, a, opAnmSetSlot, intParam(0), intParam(-1))
	if a.vms[0] != nil || old.Flags&anm.FlagDelete == 0 || ctx.World.Len() != n || a.curScript != 20 {
		t.Errorf("259(0, -1): slot %v deleted %v len %d→%d curScript %d; want only the slot emptied",
			a.vms[0], old.Flags&anm.FlagDelete != 0, n, ctx.World.Len(), a.curScript)
	}

	gameOp(vm, ctx, a, opAnmSetMain, intParam(2), intParam(6))
	if v := a.vms[2]; v == nil || v.ScriptID != 6 || a.e.ECLFlags&0x1000 != 0 || a.dirScript != 0 {
		t.Errorf("262(2, 6): slot 2 %v flags %#x dirScript %d; want no facing off slot 0", v, a.e.ECLFlags, a.dirScript)
	}

	a.dir = 1
	gameOp(vm, ctx, a, opAnmSetMain, intParam(0), intParam(1))
	if a.e.ECLFlags&0x1000 == 0 || a.dirScript != 1 || a.curScript != 1 || a.dir != 0 || a.bodyFile != 1 {
		t.Errorf("262(0, 1): flags %#x dirScript %d curScript %d dir %d bodyFile %d",
			a.e.ECLFlags, a.dirScript, a.curScript, a.dir, a.bodyFile)
	}
	if want := [2]float32{24, 64}; a.mv.size != want {
		t.Errorf("size %v after 262(0, 1), want %v", a.mv.size, want)
	}

	gameOp(vm, ctx, a, opAnmSetDir5, intParam(4))
	if v := a.vms[4]; v == nil || v.ScriptID != 6 || v.Flags&anm.FlagDraw == 0 {
		t.Errorf("269(4): slot 4 = %+v, want script dirScript+5 = 6, drawn", v)
	}

	n = ctx.World.Len()
	gameOp(vm, ctx, a, opAnmSetSlot, intParam(10), intParam(0))
	gameOp(vm, ctx, a, opAnmSetSlot, intParam(-1), intParam(0))
	if ctx.World.Len() != n {
		t.Error("a slot outside 0-9 spawned")
	}

	a.e.ECLFlags |= 0x10
	gameOp(vm, ctx, a, opAnmSetSlot, intParam(5), intParam(0))
	if v := a.vms[5]; v == nil || v.Flags&anm.FlagDraw != 0 {
		t.Error("a hidden enemy's (0x10) new object is drawn")
	}
	a.e.ECLFlags &^= 0x10

	gameOp(vm, ctx, a, opAnmSelect, intParam(0))
	gameOp(vm, ctx, a, opAnmSetSlot, intParam(1), intParam(16))
	if v := a.vms[1]; v == nil || v.File != ctx.BulletEnv.File || v.ScriptID != 16 {
		t.Errorf("file 0 is bullet.anm: slot 1 = %+v", v)
	}

	gameOp(vm, ctx, a, opAnmSelect, intParam(9))
	gameOp(vm, ctx, a, opAnmSetSlot, intParam(1), intParam(0))
	v = a.vms[1]
	if v == nil || !ctx.World.Contains(v) {
		t.Fatal("a file the ECL lacks left the slot empty, want an ended object")
	}
	ctx.World.Run(ctx.Anm)
	if ctx.World.Contains(v) {
		t.Error("the ended object outlived a Run")
	}
}

// TestEffectOps: 263/264/272/273 spawn objects on layer 6 the enemy does not
// keep; 263/272/273 put them at the enemy, 273 also turns them.
func TestEffectOps(t *testing.T) {
	vm, ctx, a, _ := visualVM(t)
	gameOp(vm, ctx, a, opAnmPlayTail, intParam(1), intParam(0))
	gameOp(vm, ctx, a, opAnmPlay, intParam(1), intParam(1))
	a.e.ECLFlags |= 0x40000
	gameOp(vm, ctx, a, opAnmPlayRot, intParam(1), intParam(2), floatParam(1.5))
	gameOp(vm, ctx, a, opAnmPlayAbs, intParam(1), intParam(3))
	if a.vms != [slotCount]*anm.VM{} {
		t.Fatal("an effect op filled a slot")
	}
	if ctx.World.Len() != 4 {
		t.Fatalf("world has %d objects, want 4", ctx.World.Len())
	}
	ctx.World.Run(ctx.Anm)
	var got []*anm.VM
	ctx.World.Each(6, func(v *anm.VM) { got = append(got, v) })
	if len(got) != 4 {
		t.Fatalf("layer 6 has %d objects, want 4", len(got))
	}
	for i, c := range []struct {
		script int
		off    [3]float32
	}{
		{3, [3]float32{}},           // 264: placed nowhere
		{2, [3]float32{10, 20, 3}},  // 273 with 0x40000: as is
		{1, [3]float32{234, 36, 3}}, // 263: screen space
		{0, [3]float32{234, 36, 3}}, // 272: at the tail
	} {
		if v := got[i]; v.ScriptID != c.script || v.Offset != c.off {
			t.Errorf("layer 6 [%d]: script %d offset %v, want %d %v", i, v.ScriptID, v.Offset, c.script, c.off)
		}
	}
	if r := got[1]; r.Rot[2] != 1.5 || r.Flags&anm.FlagRotDirty == 0 {
		t.Errorf("273: Rot %v flags %#x, want Rot[2] 1.5 and FlagRotDirty", r.Rot, r.Flags)
	}
	for _, v := range got {
		if v.Flags&anm.FlagAltRNG == 0 {
			t.Errorf("script %d: not on the alternate RNG", v.ScriptID)
		}
	}

	n := ctx.World.Len()
	gameOp(vm, ctx, a, opAnmPlay, intParam(1), intParam(1))
	vm.execGameOp(ctx, &eclTask{asyncID: -1}, &Instr{Opcode: opAnmPlay, Params: []Param{intParam(1), intParam(1)}})
	if ctx.World.Len() != n+1 {
		t.Errorf("world %d→%d, want one spawn (263 without an owner does nothing)", n, ctx.World.Len())
	}
}

// TestFlagOpsHideSlots: 322 hides every slot when the flags it leaves have
// 0x10, 323 shows them when the flags it leaves lack it.
func TestFlagOpsHideSlots(t *testing.T) {
	vm, ctx, a, _ := visualVM(t)
	a.vms[0], a.vms[9] = slotVM(ctx), slotVM(ctx)
	check := func(step string, want bool) {
		t.Helper()
		for _, i := range []int{0, 9} {
			if got := a.vms[i].Flags&anm.FlagDraw != 0; got != want {
				t.Errorf("%s: slot %d drawn %v, want %v", step, i, got, want)
			}
		}
	}
	gameOp(vm, ctx, a, opFlagSet, intParam(0x20))
	check("322(0x20)", true)
	gameOp(vm, ctx, a, opFlagSet, intParam(0x10))
	check("322(0x10)", false)
	gameOp(vm, ctx, a, opFlagClear, intParam(0x4))
	check("323(0x4) with 0x10 left", false)
	gameOp(vm, ctx, a, opFlagClear, intParam(0x10))
	check("323(0x10)", true)
	if a.e.ECLFlags != 0x20 {
		t.Errorf("flags %#x, want 0x20", a.e.ECLFlags)
	}
}

// TestDeletionReleasesSlots: the manager's deletion (FUN_0040dae0) and a
// hit's removal take the enemy's objects with it.
func TestDeletionReleasesSlots(t *testing.T) {
	vm, ctx, a, _ := visualVM(t)
	vm.started = true
	a.main = &eclTask{owner: a.e, asyncID: -1} // no sub: ends on its first run
	b := vm.actorFor(enemy.New(0, 0, 100, enemy.TypeFairy))
	b.e.ECLFlags |= 0x20000
	var objs []*anm.VM
	for _, x := range []*eclActor{a, b} {
		x.vms[0], x.vms[9] = slotVM(ctx), slotVM(ctx)
		objs = append(objs, x.vms[0], x.vms[9])
	}
	vm.Update(ctx)
	for i, x := range []*eclActor{a, b} {
		if x.e.Active || x.vms != [slotCount]*anm.VM{} {
			t.Errorf("enemy %d: active %v slots %v, want deleted and emptied", i, x.e.Active, x.vms)
		}
	}
	for i, o := range objs {
		if o.Flags&anm.FlagDelete == 0 {
			t.Errorf("object %d not marked for deletion", i)
		}
	}
	ctx.World.Run(ctx.Anm)
	if n := ctx.World.Len(); n != 0 {
		t.Errorf("%d objects outlived their enemies", n)
	}

	t.Run("hit", func(t *testing.T) {
		_, ctx, a, _ := visualVM(t)
		a.e.HP = 1
		body := slotVM(ctx)
		a.vms[0] = body
		if !a.e.TakeDamage(5) {
			t.Fatal("the hit did not destroy the enemy")
		}
		if a.e.Active || a.vms[0] != nil || body.Flags&anm.FlagDelete == 0 {
			t.Errorf("active %v slot 0 %v deleted %v", a.e.Active, a.vms[0], body.Flags&anm.FlagDelete != 0)
		}
	})
}

// TestSweepDeathFx: the sweep (FUN_0040e6a0) spawns the death effect of
// every enemy it flags, and only of those.
func TestSweepDeathFx(t *testing.T) {
	for _, c := range []struct {
		name   string
		fx     int32
		flags  int
		active bool
		spawn  bool
	}{
		{"swept", 0x167, 0, true, true},
		{"no effect", -1, 0, true, false},
		{"exempt", 0x167, 0x40, true, false},
		{"deleted", 0x167, 0, false, false},
	} {
		t.Run(c.name, func(t *testing.T) {
			vm, ctx, a, _ := visualVM(t)
			a.deathFx = c.fx
			a.e.ECLFlags = c.flags
			a.e.Active = c.active
			n := ctx.World.Len()
			vm.sweepEnemies()
			if got := ctx.World.Len() > n; got != c.spawn {
				t.Errorf("world %d→%d, want a spawn %v", n, ctx.World.Len(), c.spawn)
			}
		})
	}
}

// TestStageSlotsFollow: a stage enemy's sprite is its slot 0, sized from
// its first frame and placed at it every frame (stage 2's boss, whose
// time-0 ins_259 fills it).
func TestStageSlotsFollow(t *testing.T) {
	vm, ctx, boss := spawnStageSub(t, "ecl/stage02.ecl", "Boss")
	for range 30 {
		stepFrame(vm, ctx)
	}
	a := vm.actors[boss]
	v := a.vms[0]
	if v == nil || !ctx.World.Contains(v) {
		t.Fatal("the boss has no slot-0 object in the world")
	}
	p := a.mv.final.pos
	if want := [3]float32{p[0] + 224, p[1] + 16, p[2]}; v.Offset != want {
		t.Errorf("slot 0 at %v, want %v", v.Offset, want)
	}
	if a.mv.size == ([2]float32{}) {
		t.Error("slot 0 did not size the boss")
	}
}
