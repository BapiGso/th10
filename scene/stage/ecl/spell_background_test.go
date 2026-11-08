package ecl

import (
	"fmt"
	"testing"

	"th10/anm"
	"th10/entity/enemy"
	"th10/scene/stage"
	"th10/std"
)

func TestSpellBackgroundScriptsAndOwnership(t *testing.T) {
	for _, tc := range []struct {
		stage, card, section int
		background           [2]int
		cutin                int
	}{
		{1, 0, 0, [2]int{0xc, 0xb}, 0xf},
		{1, 2, 24, [2]int{0xc, 0xb}, 0xe},
		{2, 10, 24, [2]int{0xe, 0xf}, 0x11},
		{3, 20, 24, [2]int{0x12, 0x13}, 0x15},
		{4, 30, 24, [2]int{0x13, 0x14}, 0x16},
		{5, 50, 24, [2]int{0xc, 0xd}, 0xf},
		{6, 60, 24, [2]int{0x21, 0x22}, 0x24},
		{7, 109, 23, [2]int{0x26, -1}, 0x28}, // branch depends on section, not card id
		{7, 0, 24, [2]int{0x1b, 0x1c}, 0x1e},
	} {
		t.Run(fmt.Sprintf("stage%d_section%d", tc.stage, tc.section), func(t *testing.T) {
			vm, ctx, boss, _ := visualVM(t)
			ctx.State.Stage = tc.stage
			ctx.SetBossSection(int32(tc.section))
			vm.prog.AnimNames = []string{"enemy.anm", fmt.Sprintf("stgenm%02d.anm", tc.stage)}
			vm.bossSlots[0] = boss.e
			for i, script := range tc.background {
				if script >= 0 {
					vm.card.Background[i].Offset = [3]float32{99, 88, 77}
					vm.card.Background[i].Flags = anm.FlagAltRNG
				}
			}
			before := ctx.RNG.Count
			gameOp(vm, ctx, boss, opSpellBg, intParam(int32(tc.card)), intParam(2700), intParam(0), stringParam("card"))
			f := vm.fileAt(ctx, 2)
			for i, script := range tc.background {
				v := &vm.card.Background[i]
				if script < 0 {
					if v.File != nil {
						t.Fatal("Extra midboss initialized the unused second cdbg")
					}
					continue
				}
				timer := int32(0) // static cdbg scripts stop at time zero
				if i == 1 && tc.stage >= 3 {
					timer = 1 // colour-cycling overlays have their next instruction at 60
				}
				if v.File != f || v.ScriptID != script || v.Pos != [3]float32{224, 240, 0} || v.Offset != [3]float32{} ||
					v.Pos2 != [3]float32{} || v.Flags&anm.FlagAltRNG != 0 || v.T.Cur != timer || ctx.World.Contains(v) {
					t.Fatalf("cdbg %d = %+v", i, v)
				}
			}
			ctx.UpdateWorld()
			found := false
			ctx.World.Each(3, func(v *anm.VM) {
				if v.File == f && v.ScriptID == tc.cutin {
					found = true
				}
			})
			if !found || ctx.RNG.Count != before {
				t.Fatalf("cut-in found %v primary RNG %d -> %d", found, before, ctx.RNG.Count)
			}
			want := [2]anm.VM{}
			for i, script := range tc.background {
				if script >= 0 {
					want[i] = vm.card.Background[i]
					want[i].Run(ctx.Anm)
				}
			}
			vm.LateUpdate(ctx)
			ctx.UpdateWorld()
			for i, script := range tc.background {
				v := &vm.card.Background[i]
				// Parked scripts still advance UV/colour/alpha; cycling ones
				// advance T too. Neither gets a second step from the World.
				if script >= 0 && (v.Alpha != want[i].Alpha || v.Color != want[i].Color || v.UV != want[i].UV || v.Rot != want[i].Rot) {
					t.Errorf("cdbg %d did not take exactly one calc-0x16 step", i)
				}
			}
		})
	}
}

func TestSpellExtraLeavesSecondBackgroundUntouched(t *testing.T) {
	vm, ctx, boss, _ := visualVM(t)
	ctx.State.Stage = 7
	ctx.SetBossSection(23)
	vm.prog.AnimNames = []string{"enemy.anm", "stgenm07.anm"}
	vm.bossSlots[0] = boss.e
	v := &vm.card.Background[1]
	v.Start(spellPlacardFile(), 1, ctx.Anm)
	v.Offset = [3]float32{1, 2, 3}
	before := *v
	gameOp(vm, ctx, boss, opSpellBg, intParam(109), intParam(2700), intParam(0), stringParam("card"))
	if v.File != before.File || v.T != before.T || v.Offset != before.Offset || v.Flags != before.Flags {
		t.Fatal("Extra <0x18 branch modified cdbg VM 1")
	}
}

func TestSpellBackgroundSTDTransition(t *testing.T) {
	vm, ctx, boss, _ := visualVM(t)
	ctx.State.Stage = 1
	ctx.STD = std.New(&std.File{})
	vm.prog.AnimNames = []string{"enemy.anm", "stgenm01.anm"}
	vm.bossSlots[0] = boss.e
	gameOp(vm, ctx, boss, opSpellBg, intParam(2), intParam(2700), intParam(0), stringParam("card"))
	for range 60 {
		vm.LateUpdate(ctx)
		ctx.UpdateWorld()
	}
	if !ctx.STD.BackgroundVisible || vm.card.Timer.Cur != 60 {
		t.Fatal("STD disappeared before the pre-tick timer reached 60")
	}
	for i, v := range vm.card.Background {
		if v.Alpha != 255 || v.T.Cur != 0 {
			t.Errorf("cdbg %d not ready at the STD handover: alpha %d timer %d", i, v.Alpha, v.T.Cur)
		}
	}
	vm.LateUpdate(ctx)
	if ctx.STD.BackgroundVisible || vm.card.Timer.Cur != 61 {
		t.Fatal("STD did not hand over to cdbg at timer 60")
	}
	vm.endSpell(ctx)
	if !ctx.STD.BackgroundVisible || vm.card.Active {
		t.Fatal("spell end did not restore STD and stop cdbg drawing")
	}
}

func TestECL344WritesBossSection(t *testing.T) {
	vm, ctx, boss, _ := visualVM(t)
	vm.bossSlots[0] = boss.e
	boss.e.Type = enemy.TypeMidBoss
	task := &eclTask{stack: []eVal{intVal(0x18)}}
	vm.execGameOp(ctx, task, &Instr{Opcode: 344, Params: []Param{{Type: 'S', Raw: ^uint32(0), IsVar: true}}})
	if ctx.BossSection != 0x18 || len(task.stack) != 0 || vm.BossHUD().Midboss {
		t.Fatalf("344 section %d stack %v midboss %v", ctx.BossSection, task.stack, vm.BossHUD().Midboss)
	}
	boss.e.Type = enemy.TypeBoss
	gameOp(vm, ctx, boss, 344, intParam(0x17))
	if ctx.BossSection != 0x17 || !vm.BossHUD().Midboss {
		t.Fatal("Boss HUD still selects the name from enemy.Type")
	}
	ctx.GUI = stage.NewGUI(ctx)
	ctx.GUI.SetBossSection(-1)
	if ctx.BossSection != -1 {
		t.Fatal("GUI section interface did not write the shared original field")
	}
}
