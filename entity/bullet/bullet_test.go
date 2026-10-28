package bullet

import (
	"testing"

	"th10/anm"
	"th10/gamedata"
	"th10/zun"
)

type testEnv struct {
	Env
	se    []int32
	hits  int
	items int
}

func newTestEnv(t *testing.T) *testEnv {
	t.Helper()
	f, err := gamedata.ANM("bullet.anm")
	if err != nil {
		t.Fatal(err)
	}
	e := &testEnv{}
	e.Env = Env{
		Anm:     &anm.Env{GameSpeed: 1, RNG: &zun.RNG{}, Alt: &zun.RNG{}},
		File:    f,
		Effects: &anm.World{},
		Player:  [2]float32{0, 400},
	}
	e.Box = [4]float32{-1, 399, 1, 401}
	e.PlaySE = func(id int32, x float32, pan bool) { e.se = append(e.se, id) }
	e.Hit = func() bool { e.hits++; return true }
	e.CancelItem = func(x, y float32) { e.items++ }
	return e
}

// ecl400 is the template ECL op 400 leaves.
func ecl400() Template {
	return Template{Count1: 1, Count2: 1, Speed1: 2, Sound: 7, TransformSound: 24, Flags: 0x203}
}

func live(p *Pool) []*Bullet {
	var out []*Bullet
	p.Each(func(b *Bullet) { out = append(out, b) })
	return out
}

func TestRingSpawn(t *testing.T) {
	e := newTestEnv(t)
	p := NewPool()
	tpl := ecl400()
	tpl.Type, tpl.Colour = 3, 2
	tpl.Pos = [3]float32{0, 100, 0}
	tpl.Aim = 3 // ring, not aimed
	tpl.Count1 = 8
	p.Volley(&tpl, &e.Env)

	bs := live(p)
	if len(bs) != 8 {
		t.Fatalf("live = %d, want 8", len(bs))
	}
	for i, b := range bs {
		raw := float32(float32(float32(i)*zun.TwoPi) / 8)
		if want := zun.AddAngle(raw, 0); b.Angle != want {
			t.Errorf("bullet %d angle = %v, want %v", i, b.Angle, want)
		}
		// Flag 2 picks the spawn interrupt and the step back by 4 frames.
		if b.State != StateSpawning {
			t.Errorf("bullet %d state = %d, want spawning", i, b.State)
		}
		// The velocity is taken from the angle before it is normalised.
		vx, vy := zun.Polar(raw, 2)
		if wx := 0 - float32(vx*4); b.Pos[0] != wx {
			t.Errorf("bullet %d x = %v, want %v", i, b.Pos[0], wx)
		}
		if wy := 100 - float32(vy*4); b.Pos[1] != wy {
			t.Errorf("bullet %d y = %v, want %v", i, b.Pos[1], wy)
		}
		if b.W != 4 || b.Layer != 4 || b.CancelScript != 2*2+0x11 || b.Grace != 10 {
			t.Errorf("bullet %d W %v layer %d cancel %d grace %d", i, b.W, b.Layer, b.CancelScript, b.Grace)
		}
		if b.VM.ScriptID != 96+2 {
			t.Errorf("bullet %d script = %d, want 98", i, b.VM.ScriptID)
		}
	}
	if len(e.se) != 1 || e.se[0] != 7 {
		t.Errorf("sounds = %v, want [7]", e.se)
	}
}

// The spawn animation ends by setting I0, which turns the bullet active.
func TestSpawningBecomesActive(t *testing.T) {
	e := newTestEnv(t)
	p := NewPool()
	tpl := ecl400()
	tpl.Pos = [3]float32{0, 100, 0}
	tpl.Angle1 = zun.HalfPi
	p.Volley(&tpl, &e.Env)
	b := &p.Slots[0]
	for f := 1; f <= 60; f++ {
		p.Calc(&e.Env)
		if b.State == StateActive {
			t.Logf("active after %d frames", f)
			return
		}
	}
	t.Fatalf("still state %d after 60 frames", b.State)
}

func TestStepChain(t *testing.T) {
	e := newTestEnv(t)
	p := NewPool()
	tpl := ecl400()
	tpl.Flags = 1 // no spawn animation
	tpl.Pos = [3]float32{0, 100, 0}
	tpl.Ex[0] = Ex{Type: ExStepAbs, A: 1, B: 3, C: 10, D: 1}
	tpl.Ex[1] = Ex{Type: ExImmune, C: 5}
	p.Volley(&tpl, &e.Env)
	b := &p.Slots[0]
	if b.State != StateActive || b.ExFlags != ExStepAbs || b.ExIdx != 1 {
		t.Fatalf("state %d flags %#x idx %d", b.State, b.ExFlags, b.ExIdx)
	}
	for range 10 {
		p.Calc(&e.Env)
	}
	if b.Angle == 1 {
		t.Fatalf("turned early")
	}
	p.Calc(&e.Env)
	if b.Angle != 1 || b.Speed != 3 || b.ExFlags != 0 || b.ExIdx != 1 {
		t.Fatalf("angle %v speed %v flags %#x idx %d", b.Angle, b.Speed, b.ExFlags, b.ExIdx)
	}
	// The chain ran before the step finished, so the synchronous immune entry
	// waits for the next frame, which also ages it once.
	p.Calc(&e.Env)
	if b.ExIdx != 2 || b.Immune != 4 {
		t.Fatalf("idx %d immune %d", b.ExIdx, b.Immune)
	}
	if n := len(e.se); n != 1 || e.se[0] != 24 {
		t.Fatalf("sounds = %v, want [24]", e.se)
	}
}

func TestGoto(t *testing.T) {
	e := newTestEnv(t)
	p := NewPool()
	tpl := ecl400()
	tpl.Flags = 1
	tpl.Ex[0] = Ex{Type: ExGoto, C: 2}
	tpl.Ex[1] = Ex{Type: ExCancel}
	tpl.Ex[2] = Ex{Type: ExValue, C: 9}
	p.Volley(&tpl, &e.Env)
	b := &p.Slots[0]
	if b.State != StateActive || b.Value != 9 || b.ExIdx != 3 {
		t.Fatalf("state %d value %d idx %d", b.State, b.Value, b.ExIdx)
	}
}

func TestChildVolley(t *testing.T) {
	e := newTestEnv(t)
	p := NewPool()
	tpl := ecl400()
	tpl.Flags = 1
	tpl.Pos = [3]float32{10, 50, 0}
	// Colour 5 from entry 2 onward, 4 per ring at speed 1, then cancel self.
	tpl.Ex[0] = Ex{Type: ExVolley, A: 1, B: 1, C: int32(-0x80000000 | 5<<8 | 2), D: 4}
	tpl.Ex[1] = Ex{A: 0, B: 0, C: 1, D: 1}
	tpl.Ex[2] = Ex{Type: ExValue, C: 7}
	p.Volley(&tpl, &e.Env)
	bs := live(p)
	if len(bs) != 5 {
		t.Fatalf("live = %d, want 5", len(bs))
	}
	if bs[0].State != StateCancelling || bs[0].ExIdx != 3 {
		t.Errorf("parent state %d idx %d", bs[0].State, bs[0].ExIdx)
	}
	for _, c := range bs[1:] {
		if c.Colour != 5 || c.Value != 7 || c.Pos[0] != 10 {
			t.Errorf("child colour %d value %d x %v", c.Colour, c.Value, c.Pos[0])
		}
	}
}

func TestOffscreenDelete(t *testing.T) {
	e := newTestEnv(t)
	p := NewPool()
	tpl := ecl400()
	tpl.Flags = 1
	tpl.Pos = [3]float32{0, -70, 0}
	tpl.Angle1 = -zun.HalfPi
	tpl.Speed1 = 0.5
	p.Volley(&tpl, &e.Env)
	b := &p.Slots[0]
	for f := 1; f <= 10; f++ {
		p.Calc(&e.Env)
		if !b.Live() {
			t.Fatalf("deleted during the grace period, frame %d", f)
		}
	}
	p.Calc(&e.Env)
	if b.Live() {
		t.Fatalf("still live at y %v", b.Pos[1])
	}
}

func TestBoundsStrict(t *testing.T) {
	cases := []struct {
		pos     [3]float32
		outside bool
	}{
		{[3]float32{-194, 100, 0}, true}, // right edge exactly on -192
		{[3]float32{-193.9, 100, 0}, false},
		{[3]float32{0, -66, 0}, true}, // bottom edge exactly on -64
		{[3]float32{0, 450, 0}, true},
		{[3]float32{0, 449.9, 0}, false},
	}
	for _, c := range cases {
		if got := outside406160(c.pos, 4, 4); got != c.outside {
			t.Errorf("outside406160(%v) = %v", c.pos, got)
		}
	}
}

func TestHitAndGraze(t *testing.T) {
	e := newTestEnv(t)
	p := NewPool()
	tpl := ecl400()
	tpl.Flags = 1
	tpl.Pos = [3]float32{20, 400, 0}
	tpl.Angle1 = zun.Pi
	tpl.Speed1 = 0
	p.Volley(&tpl, &e.Env)
	b := &p.Slots[0]
	e.se = nil
	p.Calc(&e.Env)
	p.Calc(&e.Env)
	if b.Flags&FlagGrazed == 0 || len(e.se) != 1 || e.se[0] != seGraze {
		t.Fatalf("graze flags %#x sounds %v", b.Flags, e.se)
	}
	b.Pos[0] = 2
	p.Calc(&e.Env)
	if e.hits != 1 || b.State != StateCancelling {
		t.Fatalf("hits %d state %d", e.hits, b.State)
	}
}

func TestCancel(t *testing.T) {
	e := newTestEnv(t)
	p := NewPool()
	tpl := ecl400()
	tpl.Flags = 1
	tpl.Count1 = 2
	tpl.Pos = [3]float32{0, 100, 0}
	p.Volley(&tpl, &e.Env)
	p.Slots[1].Pos = [3]float32{-200, 100, 0}
	n := e.Effects.Len()
	p.CircleClear(0, 100, 1000, true, false, &e.Env)
	if p.Slots[0].State != StateCancelling || p.Slots[0].Flags&FlagDelete != 0 {
		t.Errorf("on screen: state %d flags %#x", p.Slots[0].State, p.Slots[0].Flags)
	}
	if p.Slots[1].Flags&FlagDelete == 0 {
		t.Errorf("off screen: flags %#x", p.Slots[1].Flags)
	}
	if e.Effects.Len() != n+1 || e.items != 1 {
		t.Errorf("effects %d (was %d), items %d", e.Effects.Len(), n, e.items)
	}
	p.Calc(&e.Env)
	if p.Slots[1].Live() {
		t.Errorf("deleted bullet still live")
	}
}

func TestPoolFull(t *testing.T) {
	e := newTestEnv(t)
	p := NewPool()
	tpl := ecl400()
	tpl.Flags = 1
	tpl.Count1, tpl.Count2 = 100, 25
	p.Volley(&tpl, &e.Env)
	if n := p.Live(); n != Slots {
		t.Fatalf("live = %d, want %d", n, Slots)
	}
	if p.cursor != 0 {
		t.Fatalf("cursor = %d", p.cursor)
	}
}
