package ecl

import (
	"math"
	"testing"
)

// The float32 constants must carry the exact .rdata bit patterns.
func TestMotionConstantBits(t *testing.T) {
	for _, c := range []struct {
		name string
		v    float32
		bits uint32
	}{
		{"pi (DAT_00470b18)", pi32, 0x40490fdb},
		{"2pi (DAT_00470b14)", twoPi32, 0x40c90fdb},
		{"pi/2 (DAT_00470b94)", halfPi32, 0x3fc90fdb},
		{"pi/3 (DAT_00470d50)", thirdPi32, 0x3f860a92},
		{"1/3 (DAT_00470cc4)", third32, 0x3eaaaaab},
	} {
		if got := math.Float32bits(c.v); got != c.bits {
			t.Errorf("%s = %#08x, want %#08x", c.name, got, c.bits)
		}
	}
}

func TestNormAngle(t *testing.T) {
	if got := normAngle(pi32); got != pi32 {
		t.Errorf("normAngle(pi) = %v, want pi unchanged", got)
	}
	if got, want := normAngle(4), float32(4)-twoPi32; got != want {
		t.Errorf("normAngle(4) = %v, want %v", got, want)
	}
	if got, want := normAngle(-4), float32(-4)+twoPi32; got != want {
		t.Errorf("normAngle(-4) = %v, want %v", got, want)
	}
	if got := normAngle(100); got < -pi32 || got > pi32 {
		t.Errorf("normAngle(100) = %v, outside [-pi, pi]", got)
	}
	// FUN_0044bc70 gives up after 34 turns.
	if got := normAngle(1000); got <= pi32 {
		t.Errorf("normAngle(1000) = %v, want the 34-turn cap to leave it above pi", got)
	}
}

func TestRound01(t *testing.T) {
	for _, c := range []struct{ in, want float32 }{
		{1.234, float32(float32(123) * 0.01)},
		{1.239, float32(float32(123) * 0.01)},
		{-0.001, float32(float32(-1) * 0.01)}, // floor, not truncation
		{0, 0},
	} {
		if got := round01(c.in); got != c.want {
			t.Errorf("round01(%v) = %v, want %v", c.in, got, c.want)
		}
	}
}

func TestEase(t *testing.T) {
	for _, m := range []int32{0, 1, 2, 3, 4, 5, 6, 9, 10, 11, 12, 13, 14} {
		if got := ease(m, 0, 10); got != 0 {
			t.Errorf("ease(%d) at 0 = %v, want 0", m, got)
		}
		if got := ease(m, 10, 10); got != 1 {
			t.Errorf("ease(%d) at 1 = %v, want 1", m, got)
		}
	}
	for _, c := range []struct {
		mode     int32
		num, den float32
		want     float32
	}{
		{0, 1, 4, 0.25},
		{1, 1, 2, 0.25},
		{2, 1, 2, 0.125},
		{4, 1, 2, 0.75},
		{9, 1, 4, 0.125},
		{12, 1, 2, 0.5},
		{15, 1, 2, 0},
		{16, 1, 2, 1},
	} {
		if got := ease(c.mode, c.num, c.den); got != c.want {
			t.Errorf("ease(%d, %v/%v) = %v, want %v", c.mode, c.num, c.den, got, c.want)
		}
	}
}

func TestPosBlockAdvanceRounds(t *testing.T) {
	b := posBlock{pos: [3]float32{10, 20, 0}}
	b.delta[0], b.delta[1] = polar(0, 1.239)
	b.advance()
	if want := round01(11.239); b.pos[0] != want || b.pos[1] != 20 {
		t.Errorf("pos = %v, want (%v, 20)", b.pos, want)
	}
}

func TestMotionOffscreen(t *testing.T) {
	leaving := func(flags int) (bool, int) {
		var m motion
		m.layer[0].pos = [3]float32{0, -100, 0}
		m.final.pos = m.layer[0].pos
		ok := m.step(&flags, 1, [3]float32{})
		return ok, flags
	}
	if ok, _ := leaving(0); !ok {
		t.Error("an enemy that never entered the field was deleted")
	}
	if ok, _ := leaving(0x100); ok {
		t.Error("an enemy that left the field (0x100) was kept")
	}
	if ok, _ := leaving(0x100 | 0x4); !ok {
		t.Error("flag 0x4 did not keep an enemy outside the field")
	}

	var m motion
	flags := 0
	m.step(&flags, 1, [3]float32{})
	if flags&0x100 == 0 {
		t.Error("an enemy at (0, 0) did not get 0x100 (inside the field)")
	}
}

func TestClampAxis(t *testing.T) {
	for _, c := range []struct{ v, want float32 }{{-50, -10}, {0, 0}, {50, 10}} {
		v := c.v
		clampAxis(&v, 0, 20)
		if v != c.want {
			t.Errorf("clampAxis(%v, 0, 20) = %v, want %v", c.v, v, c.want)
		}
	}
}
