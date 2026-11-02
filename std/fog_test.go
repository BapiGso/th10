package std

import (
	"math"
	"testing"
)

func TestPack(t *testing.T) {
	nan := float32(math.NaN())
	for _, tc := range []struct {
		f    [6]float32
		want uint32
	}{
		{[6]float32{1, 2, 0x30, 0x20, 0x10, 0xff}, 0xff102030},
		// Truncation toward zero, then the low byte.
		{[6]float32{0, 0, 16.9, 255.99, -1.5, 300}, 0x2cffff10},
		{[6]float32{0, 0, nan, 1, 2, 3}, 0x03020100},
	} {
		if got := pack(tc.f); got != tc.want {
			t.Errorf("pack(%v) = %08x, want %08x", tc.f, got, tc.want)
		}
	}
}

func TestRecordOps(t *testing.T) {
	a := fogRec{F: [6]float32{100, 900, 16, 32, 64, 255}, C: 0xdeadbeef}
	b := fogRec{F: [6]float32{1.5, -100, 0.99999994, 1, -64, 0}, C: 0x01020304}

	// 16 + 0.99999994 rounds to 17 before the truncation (x87 at 24 bits).
	if got, want := add7(a, b), (fogRec{F: [6]float32{101.5, 800, 17, 33, 0, 255}, C: 0xff002111}); got != want {
		t.Errorf("add7 = %+v, want %+v", got, want)
	}
	if got, want := sub7(a, b), (fogRec{F: [6]float32{98.5, 1000, 15, 31, 128, 255}, C: 0xff801f0f}); got != want {
		t.Errorf("sub7 = %+v, want %+v", got, want)
	}
	if got, want := scale7(0.5, a), (fogRec{F: [6]float32{50, 450, 8, 16, 32, 127.5}, C: 0x7f201008}); got != want {
		t.Errorf("scale7 = %+v, want %+v", got, want)
	}
}

func TestFogRec(t *testing.T) {
	c := Camera{FogColor: 0x01020304, FogNear: 1, FogFar: 2, FogBytes: [4]float32{3, 4, 5, 6}}
	r := c.fogRec()
	if want := (fogRec{F: [6]float32{1, 2, 3, 4, 5, 6}, C: 0x01020304}); r != want {
		t.Errorf("fogRec = %+v, want %+v", r, want)
	}
	var d Camera
	d.setFogRec(r)
	if d != c {
		t.Errorf("setFogRec gave %+v, want %+v", d, c)
	}
}

// rec is a fog record whose colour is packed from its floats.
func rec(f ...float32) fogRec {
	r := fogRec{F: [6]float32(f)}
	r.C = pack(r.F)
	return r
}

func TestFogTween(t *testing.T) {
	start := rec(100, 900, 48, 32, 16, 255)
	end := rec(200, 1000, 128, 96, 64, 255)

	t.Run("linear", func(t *testing.T) {
		p := fogTween{Start: start, End: end, Dur: 4, Mode: 0}
		p.T.Reset()
		for i, want := range []fogRec{
			rec(125, 925, 68, 48, 28, 255),
			rec(150, 950, 88, 64, 40, 255),
			rec(175, 975, 108, 80, 52, 255),
			end,
		} {
			if got := p.step(1); got != want {
				t.Fatalf("frame %d: %+v, want %+v", i+1, got, want)
			}
		}
		if p.Dur != 0 || p.T.Prev != 3 || p.T.Cur != 4 || p.T.F != 4 {
			t.Errorf("finished: Dur %d, T %+v; want 0, {3 4 4}", p.Dur, p.T)
		}
	})

	t.Run("game speed", func(t *testing.T) {
		p := fogTween{Start: start, End: end, Dur: 4, Mode: 0}
		p.T.Reset()
		if got, want := p.step(0.5), rec(112.5, 912.5, 58, 40, 22, 255); got != want {
			t.Errorf("half speed: %+v, want %+v", got, want)
		}
	})

	// Mode 7 adds End each frame and finishes on Start without the last
	// addition; finishing on the first frame returns Start as copied.
	t.Run("mode 7", func(t *testing.T) {
		p := fogTween{Start: rec(0, 0, 1, 2, 3, 4), End: rec(1, 1, 1, 1, 1, 1), Dur: 3, Mode: 7}
		p.T.Reset()
		for i, want := range []fogRec{
			rec(1, 1, 2, 3, 4, 5),
			rec(2, 2, 3, 4, 5, 6),
			rec(2, 2, 3, 4, 5, 6),
		} {
			if got := p.step(1); got != want {
				t.Fatalf("frame %d: %+v, want %+v", i+1, got, want)
			}
		}
		odd := fogRec{F: [6]float32{5, 6, 7, 8, 9, 10}, C: 0xdeadbeef}
		q := fogTween{Start: odd, End: end, Dur: 1, Mode: 7}
		q.T.Reset()
		if got := q.step(1); got != odd {
			t.Errorf("Dur 1: %+v, want Start %+v", got, odd)
		}
	})

	// Mode 17 adds Tan1 to Start, then End to Tan1.
	t.Run("mode 17", func(t *testing.T) {
		p := fogTween{End: rec(0, 0, 2, 0, 0, 0), Tan1: rec(0, 0, 1, 0, 0, 0), Dur: 3, Mode: 17}
		p.T.Reset()
		for i, want := range []fogRec{rec(0, 0, 1, 0, 0, 0), rec(0, 0, 4, 0, 0, 0), p.End} {
			if got := p.step(1); got != want {
				t.Fatalf("frame %d: %+v, want %+v", i+1, got, want)
			}
		}
	})

	// Mode 8 with zero tangents: the first of 5 frames from 1 to 0 is h00(1/5).
	t.Run("mode 8", func(t *testing.T) {
		p := fogTween{Start: rec(1, 1, 1, 1, 1, 1), End: rec(0, 0, 0, 0, 0, 0), Dur: 5, Mode: 8}
		p.T.Reset()
		if got, want := p.step(1), rec(0.896, 0.896, 0.896, 0.896, 0.896, 0.896); got != want {
			t.Errorf("frame 1: %+v, want %+v", got, want)
		}
	})
}

func f32bits(f float32) int32 { return int32(math.Float32bits(f)) }

// TestOp9 runs op 8, then op 9 two frames later followed by op 0, which
// holds the clock: the fog tween still steps every frame.
func TestOp9(t *testing.T) {
	var s []byte
	s = stdIns(s, 0, 8, int32(-0xefdfd0), f32bits(100), f32bits(900))        // 0xff102030
	s = stdIns(s, 2, 9, 4, 0, int32(-0xbf9f80), f32bits(200), f32bits(1000)) // 0xff406080
	s = stdIns(s, 2, 0)
	v := New(&File{Script: s})

	fog := func() fogRec { return v.Cam.fogRec() }
	v.Update(1)
	if want := (fogRec{F: [6]float32{100, 900, 0x30, 0x20, 0x10, 0xff}, C: 0xff102030}); fog() != want {
		t.Fatalf("after op 8: %+v, want %+v", fog(), want)
	}
	v.Update(1)
	for i, want := range []fogRec{
		rec(125, 925, 68, 48, 28, 255),
		rec(150, 950, 88, 64, 40, 255),
		rec(175, 975, 108, 80, 52, 255),
		rec(200, 1000, 128, 96, 64, 255),
		rec(200, 1000, 128, 96, 64, 255),
	} {
		v.Update(1)
		if fog() != want {
			t.Fatalf("frame %d of the tween: %+v, want %+v", i+1, fog(), want)
		}
	}
	if v.Cam.FogColor != 0xff406080 {
		t.Errorf("FogColor = %08x, want ff406080", v.Cam.FogColor)
	}
}
