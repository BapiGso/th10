package ecl

import (
	"math"
	"testing"

	"th10/game"
	"th10/gfx"
	"th10/input"
	"th10/scene/stage"
	"th10/supervisor"
	"th10/zun"

	"github.com/hajimehoshi/ebiten/v2"
)

func TestScreenEffectAllKindConstructors(t *testing.T) {
	for kind := int32(0); kind <= 8; kind++ {
		vm := newTestVM()
		vm.RequestShake(kind, 120, 1, 2, 3)
		s := vm.shakes[0]
		wantAlpha := int32(0)
		if kind == 3 {
			wantAlpha = 255
		}
		if s.kind != kind || s.duration != 120 || s.start != 1 || s.end != 2 || s.third != 3 ||
			s.drawPriority != 0x31 || s.alpha != wantAlpha || s.phase != 0 || s.calcStopped ||
			s.timer.Cur != 0 || s.timer.Prev != -1 || s.timer.F != 0 || vm.rng.Count != 0 {
			t.Fatalf("kind %d constructor %+v rng=%d", kind, s, vm.rng.Count)
		}
	}
}

func TestScreenEffectFadeClocksAndLifetime(t *testing.T) {
	for _, kind := range []int32{0, 2, 3, 5} {
		vm := newTestVM()
		vm.RequestShake(kind, 120, 0, 0, 0)
		s := vm.shakes[0]
		vm.CalcShake()
		vm.CalcShake()
		want := int32(2)
		if kind == 0 || kind == 3 {
			want = 252 // ftol(255 - 255/120), not 255 - ftol(255/120)
		}
		if s.alpha != want || s.timer.Cur != 2 || vm.rng.Count != 0 {
			t.Fatalf("kind %d second calc %+v rng=%d", kind, s, vm.rng.Count)
		}
		for s.timer.Cur < 120 {
			vm.CalcShake()
		}
		if kind == 2 || kind == 5 {
			vm.CalcShake()
			vm.CalcShake()
			if s.alpha != 255 || s.timer.Cur != 122 || len(vm.shakes) != 1 {
				t.Fatalf("kind %d full-alpha tail %+v", kind, s)
			}
		}
		vm.CalcShake()
		if len(vm.shakes) != 0 || vm.rng.Count != 0 {
			t.Fatalf("kind %d did not expire without RNG", kind)
		}
	}
	for _, kind := range []int32{2, 5} {
		vm := newTestVM()
		vm.RequestShake(kind, 0, 0, 0, 0)
		vm.CalcShake()
		vm.CalcShake()
		if len(vm.shakes) != 1 || vm.shakes[0].alpha != 0 {
			t.Fatal("zero duration changed the constructor's alpha")
		}
		vm.CalcShake()
		if len(vm.shakes) != 0 {
			t.Fatal("zero-duration fade-out did not expire after its two calcs")
		}
	}
}

func TestScreenEffectKind4RetainsDrawNode(t *testing.T) {
	vm := newTestVM()
	argb := uint32(0x80aabbcc)
	vm.RequestShake(4, 3, 2, int32(argb), 0)
	s := vm.shakes[0]
	for i, alpha := range []int32{128, 86, 43, 0, 86, 43, 0} {
		vm.CalcShake()
		if s.alpha != alpha || s.calcStopped != (i == 6) || len(vm.shakes) != 1 {
			t.Fatalf("flash calc %d %+v, want alpha=%d", i, s, alpha)
		}
	}
	clock := s.timer
	vm.CalcShake()
	if s.timer != clock || vm.rng.Count != 0 {
		t.Fatal("detached flash calc still ran")
	}
	vm.ClearScreenEffects()
	if len(vm.shakes) != 0 {
		t.Fatal("stage teardown left the detached flash draw node")
	}
}

func TestScreenEffectTranslucentHoldAndRelease(t *testing.T) {
	for _, kind := range []int32{6, 7} {
		vm := newTestVM()
		vm.RequestShake(kind, 3, 0, 0, 0)
		s := vm.shakes[0]
		for _, alpha := range []int32{0, 42, 85, 128, 128} {
			vm.CalcShake()
			if s.alpha != alpha || len(vm.shakes) != 1 {
				t.Fatalf("kind %d hold %+v, want alpha %d", kind, s, alpha)
			}
		}
		// Emulate the caller's raw +0x2c / timer writes, not a guessed API.
		s.SetPhase(1)
		s.SetTimer(0)
		for i := int32(0); i <= 8; i++ {
			vm.CalcShake()
			if s.alpha != 128-i*16 || len(vm.shakes) != 1 {
				t.Fatalf("kind %d release tick %d %+v", kind, i, s)
			}
		}
		vm.CalcShake()
		if len(vm.shakes) != 0 || vm.rng.Count != 0 {
			t.Fatal("release lifetime/RNG differs")
		}
	}
	vm := newTestVM()
	vm.gameSpeed = 0.5
	vm.RequestShake(6, 3, 0, 0, 0)
	s := vm.shakes[0]
	for range 9 {
		vm.CalcShake()
	}
	if s.alpha != 149 { // Cur==3, F==3.5 computes before the hold; no clamp to 128
		t.Fatalf("fractional hold alpha %d, want 149", s.alpha)
	}
	s.phase = 1
	s.timer.Set(8)
	s.timer.F = 8.5
	vm.CalcShake()
	if s.alpha != -8 || len(vm.shakes) != 1 {
		t.Fatal("fractional release was clamped instead of retaining its low byte")
	}
}

func TestScreenEffectStageGatesAndExit(t *testing.T) {
	for _, bit := range []uint32{1, 4} {
		vm := newTestVM()
		vm.ctx = newTestContext()
		vm.ctx.StageFlags = bit
		vm.RequestShake(5, 120, 0, 0, 0)
		vm.RequestShake(0, 120, 0, 0, 0)
		vm.CalcShake()
		if vm.shakes[0].timer.Cur != 1 || vm.shakes[1].timer.Cur != 0 || vm.rng.Count != 0 {
			t.Fatal("fade-out gate was incorrectly applied to fade-in")
		}
	}
	for kind := int32(0); kind <= 8; kind++ {
		vm := newTestVM()
		vm.ctx = newTestContext()
		vm.ctx.StageExiting = true
		vm.RequestShake(kind, 30, 2, 2, 2)
		vm.CalcShake()
		want := 0
		if kind == 4 || kind == 6 || kind == 7 {
			want = 1
		}
		if len(vm.shakes) != want || vm.rng.Count != 0 {
			t.Fatalf("exit kind %d nodes=%d rng=%d", kind, len(vm.shakes), vm.rng.Count)
		}
		if kind == 4 && !vm.shakes[0].calcStopped {
			t.Fatal("kind 4 exit did not take return 0")
		}
	}
}

func TestScreenEffectKind8EnvelopeAndRNG(t *testing.T) {
	vm := newTestVM()
	vm.ctx = newTestContext()
	vm.RequestShake(8, 6, 2, 1, 3)
	control := &zun.RNG{State: 1}
	axis := func(a float32) float32 {
		switch control.Raw() % 3 {
		case 1:
			return a
		case 2:
			return -a
		}
		return 0
	}
	for i, amplitude := range []float32{3, 6, 6, 4, 2} {
		vm.CalcShake()
		want := [2]float32{axis(amplitude), axis(amplitude)}
		if vm.shakeShift != want || vm.rng.Count != uint32((i+1)*4) || vm.rng.State != control.State {
			t.Fatalf("envelope tick %d shift=%v want=%v rng=%d", i+1, vm.shakeShift, want, vm.rng.Count)
		}
		if i == 0 && vm.shakeShift != [2]float32{3, 3} {
			t.Fatal("kind 8 used Dup()%3 instead of Raw()%3")
		}
	}
	vm.CalcShake()
	if len(vm.shakes) != 0 || vm.rng.Count != 20 || vm.shakeShift != [2]float32{} {
		t.Fatal("envelope terminal calc sampled RNG")
	}
	if screenEffectUnsignedOperand(-1) != 4294967296 || screenEffectUnsignedOperand(16777217) != 16777217 {
		t.Fatal("unsigned FILD/FADD conversion differs")
	}
	if math.Float32bits(screenEffectRatio(33554432, 1, 16777217)) != 0x3fffffff {
		t.Fatal("alpha callback rounded the integer divisor before FIDIV")
	}
}

func TestScreenEffectKind8ZeroSlowAndGatedSamples(t *testing.T) {
	for _, speed := range []float32{0, 0.25} {
		vm := newTestVM()
		vm.ctx = newTestContext()
		vm.ctx.Anm.GameSpeed = speed
		vm.RequestShake(8, 0, 2, 2, 2)
		for range 3 {
			vm.CalcShake()
		}
		if vm.rng.Count != 12 || vm.shakes[0].timer.Cur != 0 {
			t.Fatal("held integer clock/zero amplitude suppressed a sample")
		}
	}
	for _, bit := range []uint32{1, 2, 4, 0x10, 0x20, 0x40} {
		vm := newTestVM()
		vm.ctx = newTestContext()
		vm.ctx.StageFlags = bit
		vm.RequestShake(8, 6, 2, 2, 2)
		vm.CalcShake()
		if vm.rng.Count != 0 || vm.shakes[0].timer.Cur != 0 || len(vm.shakes) != 1 {
			t.Fatalf("gated kind 8 bit=%#x sampled/expired", bit)
		}
	}
}

func TestScreenEffectDrawKindsGeometryAndPurity(t *testing.T) {
	for kind := int32(0); kind <= 8; kind++ {
		vm := newTestVM()
		vm.RequestShake(kind, 120, 0x00223344, 0x00223344, 0)
		s := vm.shakes[0]
		s.alpha = 17
		d := &shakeDrawRec{}
		sv := supervisor.New(d, nil)
		sv.BeginFrame()
		sv.Field.Viewport.MinZ, sv.Field.Viewport.MaxZ = 0.2, 0.8
		sv.EnterField()
		field, screen := sv.Field.Viewport, sv.Screen.Viewport
		sv.ANM.Shift = [2]float32{13, -7}
		d.Take()
		vm.DrawScreenEffects(sv, 0x31, 0x31)
		vm.DrawScreenEffects(sv, 0x31, 0x31)
		if kind == 1 || kind == 8 {
			if len(d.vertices) != 0 {
				t.Fatal("shake kind acquired a draw node")
			}
			continue
		}
		if len(d.vertices) != 2 || d.vertices[0][0].Diffuse != gfx.Color(0x11223344) {
			t.Fatalf("kind %d rectangle/colour differs: %v", kind, d.vertices)
		}
		v := d.vertices[0]
		if kind == 0 || kind == 5 || kind == 6 {
			if v[0].X != 0 || v[0].Y != 0 || v[3].X != 640 || v[3].Y != 480 {
				t.Fatal("full rectangle was shifted or changed")
			}
		} else if v[0].X != 32 || v[0].Y != 16 || v[3].X != 416 || v[3].Y != 464 {
			t.Fatal("field rectangle was shifted or changed")
		}
		if vm.rng.Count != 0 || s.timer.Cur != 0 || sv.Field.Viewport != field || sv.Screen.Viewport != screen || sv.ANM.Shift != [2]float32{13, -7} {
			t.Fatal("drawing changed logic, camera viewports, timer or RNG")
		}
	}
}

type effectClearScript struct{ vm *VM }

func (*effectClearScript) Init(c *stage.Context) { c.LoadMSG("stage6", 0) }
func (s *effectClearScript) Update(c *stage.Context) {
	s.vm.Update(c)
	if c.State.Frame == 2 {
		c.OpenMSG(1)
	}
}
func (*effectClearScript) BgDraw(*ebiten.Image)              {}
func (*effectClearScript) Finished() bool                    { return false }
func (s *effectClearScript) LateUpdate(c *stage.Context)     { s.vm.LateUpdate(c) }
func (s *effectClearScript) ShakeOffset() (float64, float64) { return s.vm.ShakeOffset() }

func TestFinalClearConnectsKind5AtOriginalBirthFrame(t *testing.T) {
	state := game.NewState(0, 0, game.DiffNormal)
	state.Stage = 6
	vm := newTestVM()
	s := stage.NewHeadless(state, &effectClearScript{vm: vm})
	c := stage.ExportContext(s)
	s.UpdateHeadless(&input.State{})
	if c.RequestShake == nil || c.DrawScreenEffects == nil || c.ClearScreenEffects == nil {
		t.Fatal("ECL wrapper did not connect the screen-effect hooks")
	}
	s.UpdateHeadless(&input.State{})
	// The original post-entry places op 20 at clock 540, not time zero.
	for len(vm.shakes) == 0 && state.Frame < 1000 {
		s.UpdateHeadless(&input.State{})
	}
	if len(vm.shakes) != 1 || vm.shakes[0].kind != 5 || vm.shakes[0].duration != 120 || vm.shakes[0].timer.Cur != 0 || vm.shakes[0].alpha != 0 {
		t.Fatal("MSG final clear did not construct an unsampled kind 5")
	}
	s.UpdateHeadless(&input.State{})
	s.UpdateHeadless(&input.State{})
	if vm.shakes[0].timer.Cur != 2 || vm.shakes[0].alpha != 2 || vm.shakeShift != [2]float32{} {
		t.Fatal("final clear used post-tick alpha or a random shake")
	}
	d := &shakeDrawRec{}
	sv := supervisor.New(d, nil)
	sv.BeginFrame()
	c.DrawScreenEffects(sv, 0x31, 0x31)
	if len(d.vertices) != 1 || d.vertices[0][0].Diffuse != gfx.Color(0x02000000) {
		t.Fatal("final clear did not draw its full-screen black fade")
	}
}

func TestShakeContinuousCarryAndReplayChapterReset(t *testing.T) {
	state := game.NewState(0, 0, game.DiffEasy)
	state.RNGSeed, state.HasSeed = 1, true
	c := stage.NewContext(state)
	vm := newTestVM()
	vm.ctx, vm.rng = c, c.RNG
	vm.queueShake(30, 12, 0)
	for range 30 {
		vm.CalcShake()
	}
	if c.RNG.Count != 116 || len(vm.shakes) != 0 {
		t.Fatal("continuous shake did not take exactly 29 four-step samples")
	}
	control := &zun.RNG{State: 1}
	for range 58 {
		control.Dup()
	}
	state.CarryStageRNG(c.RNG)
	state.Stage = 2
	state.BeginStageTransition()
	next := stage.NewContext(state)
	if next.RNG != c.RNG || next.Anm.RNG != c.RNG || next.RNG.Count != 116 || next.RNG.Raw() != control.Raw() {
		t.Fatal("next live stage discarded the shake's RNG steps")
	}
	if next.RNG.State == (&zun.RNG{State: 1}).State {
		t.Fatal("live stage reverted to its original seed")
	}
	state.StageMode, state.ReplayMode = 1, 1
	state.RNGSeed = 0xabcd
	playback := stage.NewContext(state)
	if playback.RNG == c.RNG || playback.RNG.State != 0xabcd || playback.RNG.Count != 0 {
		t.Fatal("replay chapter did not override the live continuation seed")
	}
}
