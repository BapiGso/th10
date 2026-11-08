package ecl

import (
	"math"
	"testing"

	"th10/zun"
)

func TestShakeSampleSequence(t *testing.T) {
	// Explicit timer inputs: this verifies FUN_0043c5ca's offset body,
	// not the still-undumped constructor/calc entry's timer lifecycle.
	rng := &zun.RNG{State: 1}
	for i, c := range []struct {
		x, y float32
		seed uint16
	}{
		{-12, 12, 0x7c40},
		{-11.6, 0, 0xbd0c},
		{0, 0, 0xaecc},
		{0, -10.8, 0x6ee3},
		{-10.4, -10.4, 0x9223},
		{-10, 10, 0xd226},
		{9.6, 9.6, 0x84e6},
		{-9.2, -9.2, 0x4726},
	} {
		x, y := sampleShake(rng, 12, float32(30-i), 30, 0)
		if x != c.x || y != c.y || rng.State != c.seed || rng.Count != uint32((i+1)*4) {
			t.Fatalf("sample %d = %v/%v rng=%#x/%d, want %v/%v rng=%#x/%d", i, x, y, rng.State, rng.Count, c.x, c.y, c.seed, (i+1)*4)
		}
	}
}

func TestShakeSampleIntegerOperands(t *testing.T) {
	for _, c := range []struct {
		n, d, floor int32
		timer       float32
		want        float32
	}{
		{12, 30, 3, 0, 3},
		{16777217, 1, 0, 3, 50331652}, // FILD is exact before FMUL
		{1, 16777217, 0, 33554432, math.Float32frombits(0x3fffffff)}, // FIDIV
		{2, 1, 16777217, 1, 16777220},                                // FIADD
	} {
		rng := &zun.RNG{State: 1}
		x, y := sampleShake(rng, c.n, c.timer, c.d, c.floor)
		if math.Float32bits(x) != math.Float32bits(-c.want) || math.Float32bits(y) != math.Float32bits(c.want) {
			t.Errorf("operands %+v = %v/%v (%08x/%08x), want -/+%v", c, x, y, math.Float32bits(x), math.Float32bits(y), c.want)
		}
	}
}

func TestShakeSampleZeroStillConsumesRNG(t *testing.T) {
	rng := &zun.RNG{State: 1}
	x, y := sampleShake(rng, 0, 30, 30, 0)
	if math.Float32bits(x) != 0x80000000 || math.Float32bits(y) != 0 || rng.Count != 4 || rng.State != 0x7c40 {
		t.Fatalf("zero amplitude = %08x/%08x rng=%#x/%d", math.Float32bits(x), math.Float32bits(y), rng.State, rng.Count)
	}
}
