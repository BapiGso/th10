package ecl

import (
	"testing"

	"th10/entity/enemy"
)

type eclPanRecord struct {
	seRec
	xs []float32
}

func (s *eclPanRecord) PlaySEIDAt(id int, x float32) {
	s.PlaySEID(id)
	s.xs = append(s.xs, x)
}

func TestEnemySoundPanning(t *testing.T) {
	ctx := newTestContext()
	s := &eclPanRecord{}
	ctx.Audio = s
	vm := newTestVM()
	e := &enemy.Enemy{Active: true}
	a := vm.actorFor(e)
	a.mv.final.pos = [3]float32{-96, 100, 0}
	vm.execGameOp(ctx, &eclTask{owner: e}, &Instr{Opcode: opPlaySound, Params: []Param{intParam(7)}})
	vm.ctx = ctx
	a.deathSE, a.deathFx = 2, -1
	vm.kill(a)
	if len(s.ids) != 2 || s.ids[0] != 7 || s.ids[1] != 2 || len(s.xs) != 2 || s.xs[0] != -96 || s.xs[1] != -96 {
		t.Fatalf("enemy sounds = %v at %v, want ids 7/2 at game x -96", s.ids, s.xs)
	}
}
