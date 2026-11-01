package ecl

import (
	"testing"
	"th10/entity/enemy"
	"th10/entity/item"
)

// dropTestActor is a fairy with its ECL actor at game position (0, 100).
func dropTestActor(vm *VM) (*enemy.Enemy, *eclActor, *eclTask) {
	e := enemy.New(0, 0, 100, enemy.TypeFairy)
	a := vm.actorFor(e)
	a.mv.final.pos = [3]float32{0, 100, 0}
	a.syncPos()
	return e, a, &eclTask{owner: e}
}

func countItems(m *item.Manager) map[int32]int {
	counts := map[int32]int{}
	m.Each(func(it *item.Item) { counts[it.Kind]++ })
	return counts
}

// TestOriginalPowerDropIDs: ins_327 assigns per-kind counts (FUN_0040e770
// case 0x147) and ins_329 drops them once (FUN_0040c9a0, FUN_0040c9d0).
func TestOriginalPowerDropIDs(t *testing.T) {
	ctx := newTestContext()
	vm := newTestVM()
	_, a, task := dropTestActor(vm)
	for _, d := range [][2]int32{{1, 5}, {1, 2}, {4, 3}, {5, 4}, {7, 1}} {
		vm.execGameOp(ctx, task, &Instr{Opcode: opDropExtra, Params: []Param{intParam(d[0]), intParam(d[1])}})
	}
	vm.execGameOp(ctx, task, &Instr{Opcode: opDropItems})
	counts := countItems(ctx.Items)
	// 1 is small power (2: the second ins_327 replaced the first), 4 big
	// power (not an extend), 5 a full point item (not a bomb), 7 an extend.
	for kind, want := range map[int32]int{item.KindPower: 2, item.KindBigPower: 3, item.KindFullPoint: 4, item.KindLife: 1} {
		if counts[kind] != want {
			t.Fatalf("drop %d: %d, want %d", kind, counts[kind], want)
		}
	}
	if a.drops.items != [13]int32{} {
		t.Fatalf("drop counts not cleared: %v", a.drops.items)
	}
	vm.execGameOp(ctx, task, &Instr{Opcode: opDropItems})
	if n := ctx.Items.ActiveCount(); n != 10 {
		t.Fatalf("%d items after a second ins_329, want 10", n)
	}
}

// TestDropMainItem: ins_330 sets the main item, dropped at the enemy and used
// up; ins_326 clears only the scatter counts. A boss (0x8000) drops kinds
// 10/11 for its power items.
func TestDropMainItem(t *testing.T) {
	ctx := newTestContext()
	vm := newTestVM()
	e, a, task := dropTestActor(vm)
	vm.execGameOp(ctx, task, &Instr{Opcode: opDropMain, Params: []Param{intParam(4)}})
	vm.execGameOp(ctx, task, &Instr{Opcode: opDropExtra, Params: []Param{intParam(2), intParam(3)}})
	vm.execGameOp(ctx, task, &Instr{Opcode: opDropClear})
	vm.execGameOp(ctx, task, &Instr{Opcode: opDropItems})
	counts := countItems(ctx.Items)
	if counts[item.KindBigPower] != 1 || ctx.Items.ActiveCount() != 1 {
		t.Fatalf("items %v, want the main big power alone", counts)
	}
	pos := a.mv.final.pos
	ctx.Items.Each(func(it *item.Item) {
		if it.Pos != pos {
			t.Fatalf("main item at %v, want the enemy at %v", it.Pos, pos)
		}
	})
	if a.drops.items[0] != 0 {
		t.Fatal("main item not used up")
	}

	e.ECLFlags |= 0x8000
	vm.execGameOp(ctx, task, &Instr{Opcode: opDropMain, Params: []Param{intParam(1)}})
	vm.execGameOp(ctx, task, &Instr{Opcode: opDropExtra, Params: []Param{intParam(4), intParam(2)}})
	if a.drops.items[0] != 10 || a.drops.items[11] != 2 || a.drops.items[4] != 0 {
		t.Fatalf("boss drops %v, want main 10 and two of 11", a.drops.items)
	}
}

// TestVMDropAreaSpreadsOpcodeDrops: the scatter (FUN_0040c9d0) puts each
// item on the drop ellipse (radii w, h) scaled by 0.5-1, and draws the RNG
// once for the start angle and twice (two steps each) per item.
func TestVMDropAreaSpreadsOpcodeDrops(t *testing.T) {
	ctx := newTestContext()
	vm := newTestVM()
	_, a, task := dropTestActor(vm)
	pos := a.mv.final.pos

	vm.execGameOp(ctx, task, &Instr{Opcode: opDropClear})
	vm.execGameOp(ctx, task, &Instr{Opcode: opDropExtra, Params: []Param{intParam(1), intParam(4)}})
	vm.execGameOp(ctx, task, &Instr{Opcode: opDropArea, Params: []Param{floatParam(64), floatParam(32)}})
	before := vm.rng.Count
	vm.execGameOp(ctx, task, &Instr{Opcode: opDropItems})
	if n := vm.rng.Count - before; n != 2+4*4 {
		t.Fatalf("scatter drew %d steps, want 18", n)
	}

	count := 0
	ctx.Items.Each(func(it *item.Item) {
		count++
		dx, dy := (it.Pos[0]-pos[0])/64, (it.Pos[1]-pos[1])/32
		if r2 := dx*dx + dy*dy; r2 < 0.25-1e-4 || r2 > 1+1e-4 {
			t.Fatalf("item at %.2f,%.2f: %.3f of the ellipse's radius squared", it.Pos[0], it.Pos[1], r2)
		}
	})
	if count != 4 {
		t.Fatalf("spawned %d items, want 4", count)
	}

	before = vm.rng.Count
	vm.execGameOp(ctx, task, &Instr{Opcode: opDropItems})
	if n := vm.rng.Count - before; n != 2 {
		t.Fatalf("an empty scatter drew %d steps, want 2", n)
	}
}

// TestKillScoresAndDrops: a defeat (FUN_0040dc80, FUN_0040e5f0) scores a
// tenth of the score times 10 and drops the main item.
func TestKillScoresAndDrops(t *testing.T) {
	ctx := newTestContext()
	vm := newTestVM()
	vm.ctx = ctx
	e, a, _ := dropTestActor(vm)
	e.Damaged = vm.damaged
	a.score = 1234
	a.drops.items[0] = 1
	before := vm.rng.Count
	if !e.TakeDamage(1000) {
		t.Fatal("not defeated")
	}
	if e.Active {
		t.Fatal("defeated enemy still active")
	}
	if got := ctx.State.Score; got < 1230 {
		t.Fatalf("score %d, want at least the defeat's 1230", got)
	}
	if counts := countItems(ctx.Items); counts[item.KindPower] != 1 {
		t.Fatalf("drops %v, want one small power", counts)
	}
	if n := vm.rng.Count - before; n != 2 {
		t.Fatalf("defeat drew %d steps, want 2 (the scatter's start)", n)
	}
}

// TestFullPowerDropsFaith: at full power FUN_0041bb00 turns power items
// into faith items.
func TestFullPowerDropsFaith(t *testing.T) {
	ctx := newTestContext()
	ctx.State.Power = ctx.State.MaxPower
	vm := newTestVM()
	_, _, task := dropTestActor(vm)
	vm.execGameOp(ctx, task, &Instr{Opcode: opDropMain, Params: []Param{intParam(1)}})
	vm.execGameOp(ctx, task, &Instr{Opcode: opDropExtra, Params: []Param{intParam(4), intParam(2)}})
	vm.execGameOp(ctx, task, &Instr{Opcode: opDropItems})
	if counts := countItems(ctx.Items); counts[item.KindFaith] != 3 || ctx.Items.ActiveCount() != 3 {
		t.Fatalf("drops %v, want three faith items", counts)
	}
}
