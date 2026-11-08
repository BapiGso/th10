package ecl

import (
	"testing"

	"th10/entity/enemy"
	"th10/game"
)

func TestSpellDeclarationOperandOrder(t *testing.T) {
	for _, c := range []struct {
		op uint16
		id int32
	}{{opSpellBg, 10}, {opSpellAlt, 10}, {opSpell, 13}, {opSpell2, 12}, {opSpell3, 11}} {
		ctx := newTestContext()
		ctx.State.Stage, ctx.State.Faith = 1, 50009
		vm := newTestVM()
		vm.difficulty = 3
		task := &eclTask{stack: []eVal{intVal(2700), intVal(777), intVal(10)}}
		vm.execGameOp(ctx, task, &Instr{Opcode: c.op, Params: []Param{
			{Type: 'S', Raw: ^uint32(0), IsVar: true},
			{Type: 'S', Raw: ^uint32(0), IsVar: true},
			{Type: 'S', Raw: ^uint32(0), IsVar: true},
			stringParam("card"),
		}})
		if len(task.stack) != 0 || vm.card.ID != c.id || vm.card.Duration != 2700 || vm.card.Bonus != 650000 || vm.card.InitialBonus != 650000 {
			t.Errorf("op %d stack=%v card=%+v", c.op, task.stack, vm.card)
		}
	}
}

func TestSpell343PaysOnceAndLeavesPoolsAlone(t *testing.T) {
	ctx := newTestContext()
	ctx.State.Stage = 1
	rec := &seRec{}
	ctx.Audio = rec
	vm := newTestVM()
	e := enemy.New(0, 0, 100, enemy.TypeBoss)
	vm.actorFor(e)
	vm.card.Declare(ctx.State, 2, "card", 2700)
	vm.card.Timer.Set(600)
	vm.card.Bonus = 649750
	bulletOp(vm, ctx, e, opETNew, intParam(0))
	bulletOp(vm, ctx, e, opETFire, intParam(0))
	bullets, items := ctx.Bullets.Live(), ctx.Items.ActiveCount()
	if bullets == 0 {
		t.Fatal("fixture has no bullet")
	}
	rec.ids = nil // discard the fixture's bullet-fire sound
	vm.execGameOp(ctx, &eclTask{owner: e}, &Instr{Opcode: opSpellEnd})
	vm.execGameOp(ctx, &eclTask{owner: e}, &Instr{Opcode: opSpellEnd})
	if ctx.State.Score != 649750 || vm.card.Active || !vm.card.Capture || vm.card.Timer.Cur != 600 {
		t.Fatalf("settlement score=%d card=%+v", ctx.State.Score, vm.card)
	}
	if caps, seen := ctx.State.SpellCounts(2); caps != 1 || seen != 1 {
		t.Fatalf("history=%d/%d, want 1/1", caps, seen)
	}
	if ctx.Bullets.Live() != bullets || ctx.Items.ActiveCount() != items {
		t.Fatal("343 cleared bullets or generated items")
	}
	if len(rec.ids) != 1 || rec.ids[0] != 0x2d {
		t.Fatalf("capture SE=%v", rec.ids)
	}
}

func TestSpellInterruptCapturePaths(t *testing.T) {
	for _, c := range []struct {
		name              string
		time              int32
		hp, survival, cap bool
	}{
		{"HP capture", 600, true, false, true},
		{"timeout failure", 600, false, false, false},
		{"opening timeout", 59, false, false, true},
		{"survival timeout", 600, false, true, true},
		{"survival HP end", 600, true, true, true},
	} {
		t.Run(c.name, func(t *testing.T) {
			ctx := newTestContext()
			ctx.State.Stage, ctx.State.Faith = 1, 90000
			vm := newTestVM()
			vm.ctx = ctx
			e := enemy.New(0, 0, 11, enemy.TypeBoss)
			a := vm.actorFor(e)
			a.slots[0] = interruptSlot{hp: 10, time: 1, sub: "HP", timeoutSub: "Time"}
			a.phase.set(1)
			vm.card.Declare(ctx.State, 2, "card", 2700)
			vm.card.Survival = c.survival
			vm.card.Timer.Set(c.time)
			if c.hp {
				e.HP = 10
			}
			name, ok := vm.checkInterrupt(e, a)
			if !ok || c.hp && name != "HP" || !c.hp && name != "Time" || vm.card.Capture != c.cap {
				t.Fatalf("interrupt=%q/%v card=%+v", name, ok, vm.card)
			}
			wantFaith := int64(60000)
			if c.hp {
				wantFaith = 90000
			}
			if ctx.State.Faith != wantFaith || (e.ECLFlags&0x10000 != 0) == c.hp {
				t.Fatalf("faith=%d flags=%#x", ctx.State.Faith, e.ECLFlags)
			}
			vm.endSpell(ctx)
			wantScore := int64(0)
			if c.cap {
				wantScore = 1170000
			}
			if ctx.State.Score != wantScore {
				t.Fatalf("score=%d, want %d", ctx.State.Score, wantScore)
			}
		})
	}
}

func TestSpellFailureEndDoesNotAward(t *testing.T) {
	ctx := newTestContext()
	ctx.State.Stage = 1
	rec := &seRec{}
	ctx.Audio = rec
	vm := newTestVM()
	vm.card.Declare(ctx.State, 2, "card", 2700)
	vm.card.Timer.Set(600)
	vm.card.Bombed()
	vm.card.Tick(1) // invisible negative bonus still must not be added
	vm.endSpell(ctx)
	if ctx.State.Score != 0 || len(rec.ids) != 0 || vm.card.Bonus != -240 {
		t.Fatalf("score=%d sound=%v bonus=%d", ctx.State.Score, rec.ids, vm.card.Bonus)
	}
	if caps, seen := ctx.State.SpellCounts(2); caps != 0 || seen != 1 {
		t.Fatalf("failed card's history=%d/%d", caps, seen)
	}
}

func TestSpellDeclarationIgnoresFaithChangesAfterStart(t *testing.T) {
	ctx := newTestContext()
	ctx.State.Stage, ctx.State.Faith = game.StageExtra, 50010
	vm := newTestVM()
	vm.card.Declare(ctx.State, 109, "card", 2700)
	ctx.State.Faith = 999990
	vm.card.Timer.Set(300)
	vm.LateUpdate(ctx)
	// (1550310 - 155031) / 2400 = 581, then 1549729 -> 1549720.
	if vm.card.Bonus != 1549720 || vm.card.InitialBonus != 1550310 || vm.card.Timer.Cur != 301 {
		t.Fatalf("fixed declaration base=%+v", vm.card)
	}
}
