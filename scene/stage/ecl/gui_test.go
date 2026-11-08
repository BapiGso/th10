package ecl

import (
	"fmt"
	"math"
	"testing"

	"th10/entity/enemy"
	"th10/scene/stage"
)

func TestECL347GUIRecords(t *testing.T) {
	vm := newTestVM()
	ctx := newTestContext()
	ctx.State.Stage = 1
	ctx.GUI = stage.NewGUI(ctx)
	vm.ctx = ctx
	e := enemy.New(100, 100, 1000, enemy.TypeFairy) // not boss slot 0
	vm.actors[e] = newActor(e)
	task := &eclTask{owner: e, stack: []eVal{intVal(2), floatVal(333), intVal(0x7f123456)}}
	in := Instr{Opcode: opLifeMarker, Params: []Param{
		{Type: 'S', Raw: math.MaxUint32, IsVar: true},
		{Type: 'f', Raw: math.Float32bits(-1), IsVar: true},
		{Type: 'S', Raw: math.MaxUint32, IsVar: true},
	}}
	vm.execGameOp(ctx, task, &in)
	want := stage.HPMarker{Fraction: float32(333) / 1000, Color: 0x7f123456}
	if vm.hudMarkers[2] != want || ctx.GUI.HPMarkers[2] != want || len(task.stack) != 0 {
		t.Fatalf("347 records %+v / %+v, stack %v; want %+v", vm.hudMarkers, ctx.GUI.HPMarkers, task.stack, want)
	}
	// FUN_0040e770 case 331 clears all GUI records, not just this actor's.
	in = Instr{Opcode: opSetHP, Params: []Param{{Type: 'S', Raw: 2000}}}
	vm.execGameOp(ctx, task, &in)
	if vm.hudMarkers != ([4]stage.HPMarker{}) || ctx.GUI.HPMarkers != ([4]stage.HPMarker{}) {
		t.Fatal("331 did not clear the global GUI markers")
	}
}

func TestECL360RawStars(t *testing.T) {
	vm := newTestVM()
	ctx := newTestContext()
	ctx.State.Stage = 1
	ctx.GUI = stage.NewGUI(ctx)
	for _, count := range []int32{-1, 0, 3, 10, 15} {
		task := &eclTask{stack: []eVal{intVal(count)}}
		in := Instr{Opcode: 360, Params: []Param{{Type: 'S', Raw: math.MaxUint32, IsVar: true}}}
		vm.execGameOp(ctx, task, &in)
		if vm.hudStars != count || ctx.GUI.BossStars != count || len(task.stack) != 0 {
			t.Errorf("360(%d): %d/%d, stack %v", count, vm.hudStars, ctx.GUI.BossStars, task.stack)
		}
	}
}

func TestSpellNamesPresentationOnly(t *testing.T) {
	for stageNum := 1; stageNum <= 7; stageNum++ {
		vm := NewStageVM(fmt.Sprintf("ecl/stage%02d.ecl", stageNum))
		if len(vm.displayNames) == 0 {
			t.Errorf("stage %d has no th10c.dat spell-name pairs", stageNum)
		}
		if vm.prog.SubByName["main"] == nil {
			t.Errorf("stage %d lost its original main", stageNum)
		}
	}
}
