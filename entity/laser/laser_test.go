package laser

import (
	"testing"

	"th10/anm"
	"th10/gamedata"
	"th10/zun"
)

type testEnv struct {
	Env
	kills, grazes, items int
	invuln               bool
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
		Half:    [2]float32{1, 1},
	}
	e.Kill = func() bool { e.kills++; return !e.invuln }
	e.Graze = func() { e.grazes++ }
	e.CancelItem = func(x, y float32) { e.items++ }
	return e
}

// down is a line laser fired straight down from (0, y).
func down(y, length, speed float32) LineTemplate {
	return LineTemplate{
		Pos: [3]float32{0, y, 0}, Angle: zun.HalfPi,
		Full: length, Init: length, Width: 16, Speed: speed, Colour: 1,
	}
}

func lasers(m *Manager) []Laser {
	var out []Laser
	for _, l := range m.list {
		if l != nil {
			out = append(out, l)
		}
	}
	return out
}

func near(a, b float32) bool { return a-b < 1e-3 && b-a < 1e-3 }

// A line laser grows from its tail at its speed, the tail fixed and the
// emitter shown, until it reaches the full length; then the whole beam
// travels.
func TestLineGrowth(t *testing.T) {
	e := newTestEnv(t)
	m := NewManager()
	tpl := down(100, 120, 4)
	tpl.Init = 0
	if id := m.FireLine(&tpl, &e.Env); id != 1 {
		t.Fatalf("id = %d, want 1", id)
	}
	l := m.Find(1).(*Line)
	for i := 1; i <= 30; i++ {
		m.Calc(&e.Env)
		if l.Length != float32(4*i) || l.Pos[1] != 100 || l.Emitter() == nil {
			t.Fatalf("calc %d: length %v tail %v emitter %v", i, l.Length, l.Pos[1], l.Emitter())
		}
	}
	m.Calc(&e.Env)
	if l.Length != 120 || !near(l.Pos[1], 104) || l.Dist != 4 || l.Emitter() != nil {
		t.Fatalf("calc 31: length %v tail %v dist %v", l.Length, l.Pos[1], l.Dist)
	}
}

// A line laser is safe from the offscreen test for its first 30 calcs.
func TestLineGrace(t *testing.T) {
	e := newTestEnv(t)
	m := NewManager()
	tpl := down(-300, 10, 4)
	tpl.Angle = -zun.HalfPi
	m.FireLine(&tpl, &e.Env)
	for range 30 {
		m.Calc(&e.Env)
	}
	if m.Count() != 1 {
		t.Fatalf("deleted during the grace period")
	}
	m.Calc(&e.Env)
	if m.Count() != 0 {
		t.Fatalf("not deleted after the grace period")
	}
}

// A hit cuts the beam round the player: the laser keeps the run before the
// cut, and the run beyond flies on as a new laser that waits a calc (it was
// fired by the tail of the walk).
func TestLineHitCuts(t *testing.T) {
	e := newTestEnv(t)
	m := NewManager()
	tpl := down(300, 200, 0)
	m.FireLine(&tpl, &e.Env)
	m.Calc(&e.Env)
	if e.kills != 1 {
		t.Fatalf("kills = %d, want 1", e.kills)
	}
	ls := lasers(m)
	if len(ls) != 2 {
		t.Fatalf("%d lasers after the cut, want 2", len(ls))
	}
	// Segments 7-9 (centres 390, 402, 414) lie in the 32px box round y 400.
	a, b := ls[0].(*Line), ls[1].(*Line)
	if a.Length != 84 || a.Tpl.Full != 84 || a.Mark != 0 {
		t.Errorf("kept run: length %v full %v mark %d", a.Length, a.Tpl.Full, a.Mark)
	}
	if !near(b.Pos[1], 420) || b.Length != 72 || b.Tpl.Full != 72 || b.ID != 2 {
		t.Errorf("new run: tail %v length %v full %v id %d", b.Pos[1], b.Length, b.Tpl.Full, b.ID)
	}
	if a.T.Cur != 1 || b.T.Cur != 0 {
		t.Errorf("timers %d, %d: the new laser should wait a calc", a.T.Cur, b.T.Cur)
	}
}

// An invulnerable player is neither hit nor cuts the beam.
func TestLineInvulnerable(t *testing.T) {
	e := newTestEnv(t)
	e.invuln = true
	m := NewManager()
	tpl := down(300, 200, 0)
	m.FireLine(&tpl, &e.Env)
	m.Calc(&e.Env)
	if ls := lasers(m); len(ls) != 1 || ls[0].Base().Length != 200 {
		t.Fatalf("beam cut while invulnerable")
	}
}

// Beside the beam, within 1.5 widths, the player grazes it every fifth frame.
func TestLineGraze(t *testing.T) {
	e := newTestEnv(t)
	e.Player = [2]float32{10, 400}
	m := NewManager()
	tpl := down(300, 200, 0)
	m.FireLine(&tpl, &e.Env)
	for range 10 {
		m.Calc(&e.Env)
	}
	if e.kills != 0 || e.grazes != 2 {
		t.Fatalf("kills %d grazes %d, want 0 and 2", e.kills, e.grazes)
	}
}

// A circle clear over the whole beam marks it for deletion by the next calc.
func TestCircleClear(t *testing.T) {
	e := newTestEnv(t)
	e.Player = [2]float32{150, 50}
	m := NewManager()
	tpl := down(300, 200, 0)
	m.FireLine(&tpl, &e.Env)
	n := m.CircleClear([3]float32{0, 300, 0}, 1000, true, &e.Env)
	if n != 16 || e.items != 15 {
		// Segment 15 (centre 486) is more than 32px past the bottom edge.
		t.Fatalf("cut %d segments, %d items; want 16, 15", n, e.items)
	}
	l := lasers(m)[0]
	if l.Base().Mark != 1 {
		t.Fatalf("mark = %d", l.Base().Mark)
	}
	m.Calc(&e.Env)
	if m.Count() != 0 {
		t.Fatalf("not deleted")
	}
}

// Ids count up from 1, and the manager refuses a laser while full.
func TestAlloc(t *testing.T) {
	e := newTestEnv(t)
	e.Player = [2]float32{150, 50}
	m := NewManager()
	tpl := down(100, 50, 0)
	for i := 1; i <= Max; i++ {
		if id := m.FireLine(&tpl, &e.Env); id != int32(i) {
			t.Fatalf("id %d, want %d", id, i)
		}
	}
	if id := m.FireLine(&tpl, &e.Env); id != 0 || m.Count() != Max {
		t.Fatalf("full manager gave id %d", id)
	}
	m.ClearAll(false, &e.Env)
	m.Calc(&e.Env)
	if m.Count() != 0 {
		t.Fatalf("%d lasers after a clear", m.Count())
	}
	if id := m.FireLine(&tpl, &e.Env); id != Max+1 {
		t.Fatalf("id %d after the clear, want %d", id, Max+1)
	}
}

// An infinite laser warns for Delay frames, widens over Expand, holds for
// Hold and narrows over Shrink.
func TestInfiniteStates(t *testing.T) {
	e := newTestEnv(t)
	e.Player = [2]float32{150, 50}
	m := NewManager()
	tpl := InfiniteTemplate{
		Pos: [3]float32{0, 100, 0}, Angle: zun.HalfPi, Max: 300, Width: 20, Speed: 8,
		Delay: 10, Expand: 10, Hold: 10, Shrink: 10, Colour: 1,
	}
	m.FireInfinite(&tpl, &e.Env)
	l := m.Find(1).(*Infinite)
	want := func(calc int) (int32, float32) {
		switch {
		case calc <= 11:
			return StateDelay + int32(calc/11), 2
		case calc <= 20:
			return StateExpand, float32(2 * (calc - 11))
		case calc <= 31:
			if calc == 31 {
				return StateShrink, 20
			}
			return StateLine, 20
		}
		return StateShrink, float32(20 - 2*(calc-31))
	}
	for i := 1; i <= 40; i++ {
		m.Calc(&e.Env)
		s, w := want(i)
		if l.State != s || !near(l.Width, w) {
			t.Fatalf("calc %d: state %d width %v, want %d %v", i, l.State, l.Width, s, w)
		}
	}
	if l.Length != 300 {
		t.Errorf("length %v, want 300", l.Length)
	}
	m.Calc(&e.Env)
	if m.Count() != 0 {
		t.Fatalf("not deleted after shrinking")
	}
}

// Cutting an infinite laser shortens it to the cut, and the rest flies off
// as a line laser at speed 8 that may travel no further than the beam
// reached.
func TestInfiniteCut(t *testing.T) {
	e := newTestEnv(t)
	m := NewManager()
	tpl := InfiniteTemplate{
		Pos: [3]float32{0, 300, 0}, Angle: zun.HalfPi, Max: 200, Init: 200, Width: 16,
		Speed: 8, Hold: 100, Shrink: 10, Colour: 1,
	}
	m.FireInfinite(&tpl, &e.Env)
	l := m.Find(1).(*Infinite)
	l.State = StateLine
	l.Width = 16
	m.Calc(&e.Env)
	ls := lasers(m)
	if e.kills != 1 || len(ls) != 2 {
		t.Fatalf("kills %d, %d lasers", e.kills, len(ls))
	}
	if l.Length != 84 {
		t.Errorf("length %v, want 84", l.Length)
	}
	r := ls[1].(*Line)
	if !near(r.Pos[1], 420) || r.Length != 72 || r.Speed != 8 || r.Tpl.MaxDist != 80 {
		t.Errorf("rest: tail %v length %v speed %v maxdist %v", r.Pos[1], r.Length, r.Speed, r.Tpl.MaxDist)
	}
}
