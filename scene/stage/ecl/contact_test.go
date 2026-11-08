package ecl

import (
	"testing"

	"th10/entity/enemy"
	"th10/entity/player"
	"th10/input"
)

func contactActor(t *testing.T) (*VM, *eclActor) {
	t.Helper()
	vm := newTestVM()
	vm.ctx = newTestContext()
	vm.ctx.Spell = &vm.card
	p := vm.ctx.Player
	p.SetFixed(0, 40000)
	p.Invinc.Set(0)
	vm.ctx.SyncPlayer()
	e := enemy.New(224, 416, 100, enemy.TypeFairy)
	a := vm.actorFor(e)
	a.mv.layer[0].pos = [3]float32{0, 400, 0}
	a.main = &eclTask{owner: e, sub: &Sub{Instrs: []Instr{{Time: 1000, Opcode: opRetBig, RankMask: 0xff}}}}
	return vm, a
}

func TestEnemyContactFlagsAndTimers(t *testing.T) {
	for _, tc := range []struct {
		name  string
		flags int
		off   int32
		state player.State
	}{
		{"normal", 0, 0, player.StateDeathbomb},
		{"damage-disabled-still-touches", 1, 0, player.StateDeathbomb},
		{"contact-disabled", 2, 0, player.StateNormal},
		{"hidden", 0x10, 0, player.StateNormal},
		{"contact-timer", 0, 1, player.StateNormal},
	} {
		t.Run(tc.name, func(t *testing.T) {
			vm, a := contactActor(t)
			a.e.ECLFlags = tc.flags
			a.contactOff.set(tc.off)
			if !vm.updateActor(a) {
				t.Fatal("body contact removed the enemy")
			}
			if p := vm.ctx.Player; p.State != tc.state {
				t.Fatalf("player state %d, want %d", p.State, tc.state)
			}
			if a.contactOff.cur != 0 {
				t.Fatalf("post-contact timer %d, want 0", a.contactOff.cur)
			}
			if tc.off == 1 {
				a.e.ECLFlags &^= 0x400
				vm.updateActor(a)
				if vm.ctx.Player.State != player.StateDeathbomb {
					t.Fatal("expired contact timer still blocked the next frame")
				}
			}
		})
	}
}

func TestEnemyContactUsesBodyAndPostMovementBox(t *testing.T) {
	vm, a := contactActor(t)
	a.e.HitWidth, a.e.HitHeight = 200, 200
	a.e.BodyWidth, a.e.BodyHeight = 16, 16
	a.mv.layer[0].pos[0] = 13.5
	vm.updateActor(a)
	if vm.ctx.Player.State != player.StateNormal {
		t.Fatal("shootable hurtbox used as the contact body")
	}
	var in input.State
	in.Restore(1 << input.KeyRight)
	vm.ctx.Player.UpdateInput(&in) // x = 4.5, right edge = 5.5
	vm.ctx.SyncPlayer()
	a.e.ECLFlags &^= 0x400
	vm.updateActor(a) // body left edge = 5.5: inclusive contact
	if p := vm.ctx.Player; p.State != player.StateDeathbomb || p.Invinc.Cur != 6 || p.IX != 450 {
		t.Fatalf("post-move contact: state=%d invinc=%d ix=%d", p.State, p.Invinc.Cur, p.IX)
	}
	vm.ctx.Player.UpdateInput(&in)
	if vm.ctx.Player.IX != 450 {
		t.Fatal("deathbomb state moved on the next player calc")
	}
}

func TestEnemyContact361SeparateFromDamageInvulnerability(t *testing.T) {
	vm, a := contactActor(t)
	vm.execGameOp(vm.ctx, a.main, &Instr{Opcode: opContactOff, Params: []Param{intParam(2)}})
	if a.contactOff.cur != 2 || a.invuln.cur != 0 {
		t.Fatalf("361 set contact=%d damage=%d", a.contactOff.cur, a.invuln.cur)
	}
	vm.execGameOp(vm.ctx, a.main, &Instr{Opcode: opSetInvuln, Params: []Param{intParam(60)}})
	if a.contactOff.cur != 2 || a.invuln.cur != 60 {
		t.Fatalf("335 set contact=%d damage=%d", a.contactOff.cur, a.invuln.cur)
	}
}
