package ecl

import (
	"testing"

	"th10/entity/enemy"
	"th10/game"
	"th10/scene/stage"
)

func bulletOp(vm *VM, ctx *stage.Context, e *enemy.Enemy, op uint16, params ...Param) {
	vm.execBulletOp(ctx, &eclTask{owner: e}, &Instr{Opcode: op, Params: params})
}

// 400 resets a template to the binary's defaults and zeroes the offset's x
// and y, keeping z.
func TestETNewDefaults(t *testing.T) {
	vm := newTestVM()
	e := &enemy.Enemy{Active: true}
	a := vm.actorFor(e)
	a.tpl[3].Angle1, a.tpl[3].Ex[2].C = 1, 5
	a.tplOff[3] = [3]float32{4, 5, 6}
	bulletOp(vm, nil, e, opETNew, intParam(3))
	if a.tpl[3] != newTemplate() {
		t.Fatalf("template = %+v, want the defaults", a.tpl[3])
	}
	if a.tplOff[3] != [3]float32{0, 0, 6} {
		t.Fatalf("offset = %v, want [0 0 6]", a.tplOff[3])
	}
	bulletOp(vm, nil, e, opETNew, intParam(16)) // past the slots: ignored
}

// 401 fires from the enemy plus the slot's offset, unless the player is
// inside the radius 346 set.
func TestETFireRadius(t *testing.T) {
	ctx := stage.NewContext(game.NewState(game.CharReimu, game.ShotA, game.DiffNormal))
	vm := newTestVM()
	e := &enemy.Enemy{Active: true}
	a := vm.actorFor(e)
	a.mv.final.pos = [3]float32{0, 100, 0}
	bulletOp(vm, ctx, e, opETNew, intParam(0))
	bulletOp(vm, ctx, e, opETOffset, intParam(0), floatParam(8), floatParam(0))
	vm.execGameOp(ctx, &eclTask{owner: e}, &Instr{Opcode: 346, Params: []Param{floatParam(20)}})
	if a.fireR != 400 {
		t.Fatalf("fire radius² = %v, want 400", a.fireR)
	}

	vm.playerECL = [2]float32{8, 110}
	bulletOp(vm, ctx, e, opETFire, intParam(0))
	if n := ctx.Bullets.Live(); n != 0 {
		t.Fatalf("fired %d bullets with the player inside the radius", n)
	}
	if a.tpl[0].Pos != [3]float32{8, 100, 0} {
		t.Fatalf("template pos = %v, want [8 100 0]", a.tpl[0].Pos)
	}
	vm.playerECL = [2]float32{8, 300}
	bulletOp(vm, ctx, e, opETFire, intParam(0))
	if n := ctx.Bullets.Live(); n != 1 {
		t.Fatalf("fired %d bullets, want 1", n)
	}
}

// 422 picks a speed pair by rank tier; 424 and 427 blend by rank.
func TestETRankOps(t *testing.T) {
	vm := newTestVM()
	e := &enemy.Enemy{Active: true}
	a := vm.actorFor(e)
	tier := []Param{intParam(0),
		floatParam(1), floatParam(2), floatParam(3), floatParam(4), floatParam(5), floatParam(6)}
	for _, tc := range []struct {
		rank   int
		s1, s2 float32
	}{{-513, 1, 2}, {-512, 3, 4}, {511, 3, 4}, {512, 5, 6}} {
		vm.rank = tc.rank
		bulletOp(vm, nil, e, opETSpeedR3, tier...)
		if a.tpl[0].Speed1 != tc.s1 || a.tpl[0].Speed2 != tc.s2 {
			t.Errorf("rank %d: speeds %v, %v, want %v, %v", tc.rank, a.tpl[0].Speed1, a.tpl[0].Speed2, tc.s1, tc.s2)
		}
	}

	vm.rank = 0
	bulletOp(vm, nil, e, opETSpeedR2, intParam(0), floatParam(1), floatParam(4), floatParam(3), floatParam(0))
	if a.tpl[0].Speed1 != 2 || a.tpl[0].Speed2 != 2 {
		t.Errorf("424 at rank 0: speeds %v, %v, want 2, 2", a.tpl[0].Speed1, a.tpl[0].Speed2)
	}
	// (hi - lo) * 1024 >> 11 rounds toward zero: 3/2 -> 1, -3/2 -> -1.
	bulletOp(vm, nil, e, opETCountR2, intParam(0), intParam(2), intParam(5), intParam(5), intParam(2))
	if a.tpl[0].Count1 != 3 || a.tpl[0].Count2 != 4 {
		t.Errorf("427 at rank 0: counts %d, %d, want 3, 4", a.tpl[0].Count1, a.tpl[0].Count2)
	}
}
