package ecl

import (
	"fmt"
	"testing"

	"th10/entity/enemy"
	"th10/entity/player"
)

func rankTestVM(live bool) *VM {
	vm := newTestVM()
	vm.started = true
	if live {
		// The hidden main is in manager+0x60, even without a visible enemy.
		e := &enemy.Enemy{Active: true, ECLFlags: 0x51}
		a := vm.actorFor(e)
		a.main = &eclTask{owner: e, sub: &Sub{Instrs: []Instr{
			{Time: 100000, Opcode: opRetBig, RankMask: 0xff},
		}}}
	}
	return vm
}

func TestDynamicRankClockAndGates(t *testing.T) {
	for _, tc := range []struct {
		clock, start, want int
		live, dialog       bool
	}{
		{59, 0, 0, true, false}, {60, 0, 1, true, false}, {61, 0, 0, true, false},
		{0, 0, 1, true, false}, {120, 0, 0, false, false}, {120, 0, 0, true, true},
		{60, 1024, 1024, true, false}, {60, -2000, -1024, true, false},
	} {
		t.Run(fmt.Sprintf("%+v", tc), func(t *testing.T) {
			ctx := newTestContext()
			ctx.State.Rank = tc.start
			ctx.Player.StateTimer.Set(int32(tc.clock))
			if tc.dialog {
				ctx.LoadMSG("stage1", 0)
				ctx.OpenMSG(0)
				if !ctx.DialogActive() {
					t.Fatal("fixture did not open a dialog")
				}
			}
			vm := rankTestVM(tc.live)
			vm.Update(ctx)
			if ctx.State.Rank != tc.want || vm.rank != tc.want {
				t.Fatalf("rank state/VM = %d/%d, want %d", ctx.State.Rank, vm.rank, tc.want)
			}
		})
	}
}

func TestDynamicRankHeldClockAndAllStates(t *testing.T) {
	for _, state := range []player.State{
		player.StateRespawn, player.StateNormal, player.StateDying, player.State3, player.StateDeathbomb,
	} {
		t.Run(fmt.Sprint(state), func(t *testing.T) {
			ctx := newTestContext()
			ctx.Player.State = state
			ctx.Player.StateTimer.Set(60)
			vm := rankTestVM(true)
			vm.Update(ctx)
			vm.Update(ctx) // Cur remains 60 (e.g. a fractional-speed frame).
			if ctx.State.Rank != 2 {
				t.Fatalf("held timer added rank %d, want 2", ctx.State.Rank)
			}
		})
	}
}
