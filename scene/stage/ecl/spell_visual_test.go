package ecl

import (
	"fmt"
	"math"
	"testing"

	"th10/anm"
	"th10/game"
	"th10/std"
)

func TestSpellDeclarationWorldRings(t *testing.T) {
	vm, ctx, boss, _ := visualVM(t)
	ctx.State.Stage = 1
	vm.bossSlots[0] = boss.e
	boss.mv.final.pos = [3]float32{10, 20, 3}
	vm.execGameOp(ctx, &eclTask{owner: boss.e}, &Instr{Opcode: opSpellBg, Params: []Param{
		intParam(2), intParam(3456), intParam(999), stringParam("card"),
	}})
	s := &vm.card
	if s.Ring == nil || s.Ring.File != ctx.BulletEnv.File || s.Ring.ScriptID != 0x1a1 || s.Ring.Layer != 15 || s.Ring.Offset != [3]float32{234, 36, 3} || s.Position != boss.mv.final.pos {
		t.Fatalf("declaration ring=%+v position=%v", s.Ring, s.Position)
	}
	for _, id := range []int{0x19f, 0x1a0} {
		v := s.Ring.Child(id)
		if v == nil || v.I[2] != 3456 || v.Layer != 3 || v.Offset != [3]float32{234, 36, 3} || v.Flags&anm.FlagAltRNG == 0 {
			t.Fatalf("ring child %#x=%+v", id, v)
		}
	}
	ctx.UpdateWorld()
	flash := false
	ctx.World.Each(15, func(v *anm.VM) {
		if v.ScriptID == 0x1ab {
			flash = true
			if v.File != ctx.BulletEnv.File || v.Offset != [3]float32{} || v.Flags&anm.FlagAltRNG == 0 {
				t.Fatalf("declaration flash=%+v", v)
			}
		}
	})
	if !flash || ctx.RNG.Count != 0 {
		t.Fatalf("flash=%v game RNG count=%d", flash, ctx.RNG.Count)
	}
}

func TestSpellCutinScriptsAndLayers(t *testing.T) {
	for _, c := range []struct {
		stage, id, script int
	}{{1, 0, 0xf}, {1, 2, 0xe}, {2, 10, 0x11}, {3, 20, 0x15}, {4, 30, 0x16}, {5, 50, 0xf}, {6, 60, 0x24}} {
		vm, ctx, boss, _ := visualVM(t)
		ctx.State.Stage = c.stage
		vm.prog.AnimNames = []string{"enemy.anm", fmt.Sprintf("stgenm%02d.anm", c.stage)}
		vm.bossSlots[0] = boss.e
		vm.execGameOp(ctx, &eclTask{owner: boss.e}, &Instr{Opcode: opSpellBg, Params: []Param{
			intParam(int32(c.id)), intParam(2700), intParam(777), stringParam("card"),
		}})
		ctx.UpdateWorld()
		found := false
		ctx.World.Each(3, func(v *anm.VM) {
			if v.File == vm.fileAt(ctx, 2) && v.ScriptID == c.script {
				found = true
				if v.Offset != [3]float32{} || v.Flags&anm.FlagAltRNG == 0 {
					t.Fatalf("stage %d cut-in offset=%v flags=%#x", c.stage, v.Offset, v.Flags)
				}
			}
		})
		if !found || ctx.RNG.Count != 0 {
			t.Errorf("stage %d id %d cut-in %#x found=%v game RNG count=%d", c.stage, c.id, c.script, found, ctx.RNG.Count)
		}
	}
}

func TestSpellVisualConstantBits(t *testing.T) {
	for _, c := range []struct {
		v    float32
		bits uint32
	}{{spellFadeY, 0x42c00000}, {spellShowY, 0x43000000}, {spellFollow, 0x3d4ccccd}} {
		if math.Float32bits(c.v) != c.bits {
			t.Errorf("constant=%#08x, want %#08x", math.Float32bits(c.v), c.bits)
		}
	}
}

// A stopped placard with interrupt 3 -> alpha 128, 2 -> alpha 255,
// 1 -> delete. This tests the labels, not guesses about front.anm art.
func spellPlacardFile() *anm.File {
	return &anm.File{Scripts: map[int]*anm.Script{1: {ID: 1, Instrs: []anm.Instr{
		anmIns(63),
		anmIns(64, 3), anmIns(51, 128), anmIns(63),
		anmIns(64, 2), anmIns(51, 255), anmIns(63),
		anmIns(64, 1), anmIns(1),
	}}}}
}

func TestSpellFadeHysteresis(t *testing.T) {
	vm, ctx, _, _ := visualVM(t)
	vm.card.Declare(ctx.State, 2, "card", 2700)
	for i := range vm.card.Placards {
		vm.card.Placards[i] = ctx.World.SpawnTail(spellPlacardFile(), 1, 15, ctx.Anm)
	}
	for _, c := range []struct {
		timer int32
		y     float64
		faded bool
		alpha uint8
	}{
		{118, 95, false, 255}, // tick reaches 119: no fading yet
		{119, 96, false, 255}, // exactly 96 fails the strict comparison
		{119, 95, true, 128},  // tick reaches 120: first fade
		{130, 96, true, 128},
		{131, 128, true, 128}, // exactly 128 does not restore it
		{132, 129, false, 255},
		{133, math.NaN(), false, 255},
	} {
		vm.card.Timer.Set(c.timer)
		ctx.Player.Y = c.y + game.FieldTop
		vm.LateUpdate(ctx)
		ctx.UpdateWorld()
		if vm.card.Faded != c.faded {
			t.Fatalf("at %d y=%v faded=%v, want %v", c.timer, c.y, vm.card.Faded, c.faded)
		}
		for _, v := range vm.card.Placards {
			if v.Alpha != c.alpha {
				t.Fatalf("at %d y=%v alpha=%d, want %d", c.timer, c.y, v.Alpha, c.alpha)
			}
		}
	}
}

func TestSpellSTDGateAndWorldEnd(t *testing.T) {
	vm, ctx, _, _ := visualVM(t)
	ctx.STD = std.New(&std.File{})
	vm.card.Declare(ctx.State, 2, "card", 2700)
	for i := range vm.card.Placards {
		vm.card.Placards[i] = ctx.World.SpawnTail(spellPlacardFile(), 1, 15, ctx.Anm)
	}
	vm.card.Ring = ctx.World.SpawnTail(loadANM("bullet.anm"), 0x1a1, 15, ctx.Anm)
	ring := vm.card.Ring
	vm.card.Timer.Set(59)
	vm.LateUpdate(ctx)
	if !ctx.STD.BackgroundVisible {
		t.Fatal("STD hid on pre-tick frame 59")
	}
	vm.LateUpdate(ctx)
	if ctx.STD.BackgroundVisible {
		t.Fatal("STD stayed visible on pre-tick frame 60")
	}
	vm.endSpell(ctx)
	if !ctx.STD.BackgroundVisible || vm.card.Active || vm.card.Ring != nil || ring.Flags&anm.FlagDelete == 0 {
		t.Fatalf("end: visible=%v active=%v ring=%p flags=%#x", ctx.STD.BackgroundVisible, vm.card.Active, vm.card.Ring, ring.Flags)
	}
	ctx.UpdateWorld()
	if ctx.World.Contains(ring) {
		t.Fatal("ring survived World.Run after the end")
	}
	for _, v := range vm.card.Placards {
		if ctx.World.Contains(v) {
			t.Fatal("placard did not take end label 1")
		}
	}
	ctx.STD.BackgroundVisible = false
	vm.endSpell(ctx)
	if ctx.STD.BackgroundVisible {
		t.Fatal("inactive card end changed STD")
	}
}

func TestSpellRingFollowSequence(t *testing.T) {
	vm, ctx, boss, _ := visualVM(t)
	vm.bossSlots[0] = boss.e
	vm.card.Declare(ctx.State, 2, "card", 2700)
	vm.card.Position = [3]float32{100, 200, 3}
	boss.mv.final.pos = [3]float32{120, 180, 7}
	vm.card.Ring = ctx.World.SpawnTail(loadANM("bullet.anm"), 0x1a1, 15, ctx.Anm)
	// The third step includes the rounded second step, not the decimal
	// values 101.95 and 3.39 evaluated at higher precision.
	for _, want := range [][3]float32{{101, 199, 3.2}, {101.95, 198.05, 3.39},
		{math.Float32frombits(0x42cdb47a), 197.1475, math.Float32frombits(0x40648313)}} {
		vm.LateUpdate(ctx)
		if vm.card.Position != want {
			t.Fatalf("position=%v (%08x/%08x/%08x), want %v", vm.card.Position,
				math.Float32bits(vm.card.Position[0]), math.Float32bits(vm.card.Position[1]), math.Float32bits(vm.card.Position[2]), want)
		}
		ctx.UpdateWorld()
		off := [3]float32{want[0] + 224, want[1] + 16, want[2]}
		ctx.World.Each(3, func(v *anm.VM) {
			if v.ScriptID == 0x19f || v.ScriptID == 0x1a0 {
				if v.Offset != off {
					t.Fatalf("ring child %#x offset=%v, want %v", v.ScriptID, v.Offset, off)
				}
			}
		})
	}
}

func TestSpellBackgroundStepBetweenDecayAndTimer(t *testing.T) {
	vm, ctx, _, _ := visualVM(t)
	vm.card.Declare(ctx.State, 2, "card", 2700)
	vm.card.InitialBonus, vm.card.Bonus = 650000, 650000
	vm.card.Timer.Set(300)
	f := &anm.File{Scripts: map[int]*anm.Script{
		1: {ID: 1, Instrs: []anm.Instr{anmIns(48, 0, 0, 0), {Time: 1, Op: 88, Args: anmIns(88, 2).Args}, {Time: 1, Op: 63}}},
		2: {ID: 2, Instrs: []anm.Instr{anmIns(63)}},
	}}
	vm.card.Background[0] = *anm.New()
	vm.card.Background[0].Start(f, 1, ctx.Anm)
	called := false
	ctx.Anm.Spawn = func(v *anm.VM) {
		called = true
		if vm.card.Timer.Cur != 300 || vm.card.Bonus != 649750 {
			t.Fatalf("background ran at timer=%d bonus=%d", vm.card.Timer.Cur, vm.card.Bonus)
		}
		ctx.World.Add(v)
	}
	vm.LateUpdate(ctx)
	if !called || vm.card.Timer.Cur != 301 {
		t.Fatalf("background step=%v timer=%d", called, vm.card.Timer.Cur)
	}
}
