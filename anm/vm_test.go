package anm_test

import (
	"math"
	"testing"

	"th10/anm"
	"th10/gamedata"
	"th10/zun"
)

func loadBulletAnm(t *testing.T) *anm.File {
	t.Helper()
	f, err := gamedata.ANM("bullet.anm")
	if err != nil {
		t.Fatalf("load bullet.anm: %v", err)
	}
	return f
}

// The graze spark (bullet.anm 434) draws one angle from the game RNG on its
// first frame (op 7 F0 = 10010) and eases out 64px along it over 12 frames.
func TestGrazeSparkDrawsOnceFromGameRNG(t *testing.T) {
	f := loadBulletAnm(t)
	env := &anm.Env{RNG: &zun.RNG{State: 0x1234}, Alt: &zun.RNG{State: 0x4321}}
	want := *env.RNG
	a := float32(want.Signed() * zun.Pi)
	x := float32(float32(math.Cos(float64(a))) * 64)
	y := float32(float32(math.Sin(float64(a))) * 64)

	vm := anm.New()
	if !vm.Start(f, 434, env) {
		t.Fatal("script 434 missing")
	}
	if env.RNG.Count != 2 || env.Alt.Count != 0 {
		t.Fatalf("draws after start: game %d alt %d, want 2 and 0", env.RNG.Count, env.Alt.Count)
	}
	if vm.F[0] != a {
		t.Errorf("F0 = %v, want %v", vm.F[0], a)
	}
	if !vm.Visible() || vm.Sprite != 308 || vm.Alpha != 128 {
		t.Errorf("after start: visible %v sprite %d alpha %d", vm.Visible(), vm.Sprite, vm.Alpha)
	}
	for i := range 11 {
		if vm.Run(env) {
			t.Fatalf("ended after %d frames, want 12", i+1)
		}
	}
	if vm.Pos != [3]float32{x, y, 0} {
		t.Errorf("final position %v, want (%v, %v, 0)", vm.Pos, x, y)
	}
	if !vm.Run(env) || !vm.Ended() {
		t.Error("op 1 at t=12 did not end the script")
	}
	if env.RNG.Count != 2 {
		t.Errorf("game RNG count %d after the run, want 2", env.RNG.Count)
	}
}

// The cancel effect (bullet.anm 17) draws nothing itself; its op-88 child
// (script 16) keeps flag 0x40000000 and draws three values from the alternate
// RNG, and inherits the parent's position and offset.
func TestCancelChildUsesAltRNG(t *testing.T) {
	f := loadBulletAnm(t)
	env := &anm.Env{RNG: &zun.RNG{State: 1}, Alt: &zun.RNG{State: 2}}
	var kids []*anm.VM
	env.Spawn = func(c *anm.VM) { kids = append(kids, c) }

	vm := anm.New()
	vm.Offset = [3]float32{224 + 10, 16 + 20, 0}
	vm.Layer = 3
	vm.Start(f, 17, env)
	if vm.Offset != [3]float32{234, 36, 0} {
		t.Errorf("reset cleared the offset: %v", vm.Offset)
	}
	if env.RNG.Count != 0 || env.Alt.Count != 6 {
		t.Fatalf("draws: game %d alt %d, want 0 and 6", env.RNG.Count, env.Alt.Count)
	}
	if vm.Flags&anm.FlagAltRNG != 0 {
		t.Error("parent has the alt-RNG flag; Start's reset should clear it")
	}
	if len(kids) != 1 {
		t.Fatalf("%d children, want 1", len(kids))
	}
	c := kids[0]
	if c.Flags&anm.FlagAltRNG == 0 || c.ScriptID != 16 || c.Offset != vm.Offset || c.Pos2 != vm.Pos {
		t.Errorf("child: flags %#x script %d offset %v pos2 %v", c.Flags, c.ScriptID, c.Offset, c.Pos2)
	}
	if c.Layer != 7 {
		t.Errorf("child layer %d, want 7 (op 68 after inheriting 3)", c.Layer)
	}
	for i := range 27 {
		if vm.Run(env) {
			t.Fatalf("parent ended after %d frames, want 28", i+1)
		}
	}
	if !vm.Run(env) {
		t.Error("parent did not end at t=28")
	}
}

// Op 0 is a no-op, not the end of a script: 370 and 371 continue past it.
func TestScriptsContinuePastOp0(t *testing.T) {
	f := loadBulletAnm(t)
	for _, id := range []int{370, 371} {
		s := f.Scripts[id]
		if s == nil {
			t.Fatalf("script %d missing", id)
		}
		zero := -1
		for i, in := range s.Instrs {
			if in.Op == 0 {
				zero = i
				break
			}
		}
		if zero < 0 || zero == len(s.Instrs)-1 {
			t.Errorf("script %d: op 0 at %d of %d instructions", id, zero, len(s.Instrs))
		}
	}
}

// World.SpawnFront (FUN_00448e30) flags its object for the alternate RNG
// before the first frame: bullet.anm 16 draws its three values from it. The
// layer goes in before that frame too, so the script's own op 68 (layer 7)
// overrides the 5 asked for, as in the binary (0x448e59, then FUN_00449870).
func TestSpawnFrontUsesAltRNG(t *testing.T) {
	f := loadBulletAnm(t)
	env := &anm.Env{RNG: &zun.RNG{State: 1}, Alt: &zun.RNG{State: 2}}
	w := &anm.World{}
	vm := w.SpawnFront(f, 16, 5, env)
	if env.RNG.Count != 0 || env.Alt.Count != 6 {
		t.Fatalf("draws: game %d alt %d, want 0 and 6", env.RNG.Count, env.Alt.Count)
	}
	if !w.Contains(vm) || vm.Layer != 7 {
		t.Errorf("contained %v layer %d, want true 7", w.Contains(vm), vm.Layer)
	}
}
