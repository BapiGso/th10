package ecl

import (
	"math"
	"slices"
	"testing"

	"th10/anm"
	"th10/anm/anmdraw"
	"th10/game"
	"th10/gfx"
	"th10/input"
	"th10/scene/stage"
	"th10/std"
	"th10/supervisor"
	"th10/supervisor/supervisortest"
)

func TestShake337OperandOrderAndConstructor(t *testing.T) {
	vm := newTestVM()
	task := &eclTask{stack: []eVal{intVal(30), intVal(12), intVal(5)}}
	pop := Param{Type: 'S', IsVar: true, Raw: ^uint32(0)}
	vm.execGameOp(nil, task, &Instr{Opcode: opScreenShake, Params: []Param{pop, pop, pop}})
	if len(task.stack) != 0 || len(vm.shakes) != 1 {
		t.Fatalf("337 stack %v shakes %d", task.stack, len(vm.shakes))
	}
	s := vm.shakes[0]
	if s.duration != 30 || s.start != 12 || s.end != 5 || s.timer.Cur != 0 || s.timer.Prev != -1 || s.timer.F != 0 {
		t.Fatalf("constructor %+v", s)
	}
	if vm.rng.Count != 0 || vm.shakeShift != [2]float32{} {
		t.Fatal("constructor sampled before calc 0x0e")
	}
	for _, duration := range []int32{0, -1} {
		vm := newTestVM()
		vm.execGameOp(nil, &eclTask{}, &Instr{Opcode: opScreenShake, Params: []Param{
			intParam(duration), intParam(0), intParam(0),
		}})
		if len(vm.shakes) != 1 {
			t.Fatal("nonpositive duration was filtered at construction")
		}
		vm.CalcShake()
		if len(vm.shakes) != 0 || vm.rng.Count != 0 {
			t.Fatal("expired object sampled instead of taking return 7")
		}
	}
}

func TestShakeRuntimeSequenceAndLifetime(t *testing.T) {
	vm := newTestVM()
	vm.queueShake(30, 12, 0)
	for i, tc := range []struct {
		x, y float32
		seed uint16
	}{
		{-11.6, 11.6, 0x7c40},
		{-11.2, 0, 0xbd0c},
		{0, 0, 0xaecc},
		{0, -10.4, 0x6ee3},
		{-10, -10, 0x9223},
		{-9.6, 9.6, 0xd226},
		{9.2, 9.2, 0x84e6},
		{-8.8, -8.8, 0x4726},
	} {
		vm.CalcShake()
		if vm.shakeShift != [2]float32{tc.x, tc.y} || vm.rng.State != tc.seed || vm.rng.Count != uint32((i+1)*4) ||
			vm.shakes[0].timer.Cur != int32(i+1) {
			t.Fatalf("frame %d: shift %v rng %#x/%d timer %+v", i+1, vm.shakeShift, vm.rng.State, vm.rng.Count, vm.shakes[0].timer)
		}
	}
	for range 21 {
		vm.CalcShake()
	}
	if len(vm.shakes) != 1 || vm.shakes[0].timer.Cur != 29 || vm.rng.Count != 116 {
		t.Fatal("duration 30 did not sample exactly 29 times")
	}
	vm.CalcShake()
	if len(vm.shakes) != 0 || vm.shakeShift != [2]float32{} || vm.rng.Count != 116 {
		t.Fatal("terminal calc did not remove the object without sampling")
	}
	vm.CalcShake()
	if vm.rng.Count != 116 {
		t.Fatal("removed object still consumes RNG")
	}
}

func TestShakeRuntimeZeroNegativeAndConstantAmplitudes(t *testing.T) {
	for _, tc := range []struct {
		start, end int32
		amplitude  float32
	}{{0, 0, 0}, {-6, 0, -4}, {0, 6, 2}, {6, 6, 6}} {
		vm := newTestVM()
		vm.execGameOp(nil, &eclTask{}, &Instr{Opcode: opScreenShake, Params: []Param{
			intParam(3), intParam(tc.start), intParam(tc.end),
		}})
		vm.CalcShake()
		if math.Float32bits(vm.shakeShift[0]) != math.Float32bits(-tc.amplitude) ||
			math.Float32bits(vm.shakeShift[1]) != math.Float32bits(tc.amplitude) || vm.rng.Count != 4 || vm.rng.State != 0x7c40 {
			t.Fatalf("start/end %d/%d: shift %v rng %#x/%d", tc.start, tc.end, vm.shakeShift, vm.rng.State, vm.rng.Count)
		}
	}
}

func TestShakeSlowFrameSampling(t *testing.T) {
	vm := newTestVM()
	vm.gameSpeed = 0.5
	vm.queueShake(3, 6, 0)
	for i, count := range []uint32{4, 8, 12, 16, 20, 20} {
		vm.CalcShake()
		if vm.rng.Count != count {
			t.Errorf("slow calc %d RNG count %d, want %d", i+1, vm.rng.Count, count)
		}
		if i < 5 && (vm.shakes[0].timer.Cur != int32((i+1)/2) || vm.shakes[0].timer.F != float32(i+1)*0.5) {
			t.Errorf("slow calc %d timer %+v", i+1, vm.shakes[0].timer)
		}
	}
	if len(vm.shakes) != 0 || vm.shakeShift != [2]float32{} {
		t.Fatal("slow terminal frame did not remove shake")
	}
	vm.gameSpeed = 0
	vm.queueShake(3, 0, 0)
	for range 3 {
		vm.CalcShake()
	}
	if vm.rng.Count != 32 || vm.shakes[0].timer.F != 0 || vm.shakes[0].timer.Cur != 0 {
		t.Fatal("zero speed suppressed per-calc RNG samples")
	}
}

func TestShakeEqualPriorityOrder(t *testing.T) {
	vm := newTestVM()
	vm.queueShake(3, 9, 9)
	vm.queueShake(3, 3, 3)
	vm.CalcShake()
	// Seed 1: first sample -/+3, second sample -9/0. The older node,
	// which is later at the same priority, overwrites the shared shift.
	if vm.shakes[0].start != 3 || vm.shakes[1].start != 9 || vm.shakeShift != [2]float32{-9, 0} || vm.rng.Count != 8 {
		t.Fatalf("equal priorities: shift %v rng %d", vm.shakeShift, vm.rng.Count)
	}
}

func TestShakeOffsetReadIsPure(t *testing.T) {
	vm := newTestVM()
	vm.queueShake(30, 12, 0)
	vm.CalcShake()
	timer, count := vm.shakes[0].timer, vm.rng.Count
	x, y := vm.ShakeOffset()
	for range 20 {
		if a, b := vm.ShakeOffset(); a != x || b != y {
			t.Fatal("draw accessor changed the sampled shift")
		}
	}
	if vm.shakes[0].timer != timer || vm.rng.Count != count {
		t.Fatal("draw accessor advanced the timer or RNG")
	}
}

type shakeStageScript struct {
	vm      *VM
	updates int
}

func (*shakeStageScript) Init(*stage.Context) {}
func (s *shakeStageScript) Update(c *stage.Context) {
	s.updates++
	s.vm.Update(c)
	if s.updates == 1 {
		s.vm.execGameOp(c, &eclTask{}, &Instr{Opcode: opScreenShake, Params: []Param{
			intParam(3), intParam(9), intParam(0),
		}})
	}
}
func (*shakeStageScript) Finished() bool                    { return false }
func (s *shakeStageScript) ShakeOffset() (float64, float64) { return s.vm.ShakeOffset() }
func (s *shakeStageScript) LateUpdate(c *stage.Context)     { s.vm.LateUpdate(c) }

func TestShakeStageCalcPriorityAndBirthFrame(t *testing.T) {
	state := game.NewState(game.CharReimu, game.ShotA, game.DiffEasy)
	state.HasSeed, state.RNGSeed = true, 1
	script := &shakeStageScript{vm: newTestVM()}
	s := stage.NewHeadless(state, script)
	c := stage.ExportContext(s)
	before := c.RNG.Count
	s.UpdateHeadless(&input.State{})
	if len(script.vm.shakes) != 1 || script.vm.shakeShift != [2]float32{} || c.RNG.Count != before || c.CalcShake == nil {
		t.Fatal("ECL-created shake sampled on its birth frame or did not install calc")
	}
	c.STD = std.New(&std.File{})
	c.STD.Cam.Delta = [3]float32{1, 2, 3}
	playerTimer := c.Player.StateTimer.Cur
	x, y := c.Player.X, c.Player.Y
	calc, called := c.CalcShake, false
	c.CalcShake = func() {
		called = true
		if c.STD.Cam.Delta != [3]float32{} || c.Player.StateTimer.Cur != playerTimer {
			t.Fatal("shake calc is not after STD and before player")
		}
		calc()
	}
	s.UpdateHeadless(&input.State{})
	if !called || script.vm.shakeShift != [2]float32{-6, 6} || c.RNG.Count != before+4 || c.Player.X != x || c.Player.Y != y {
		t.Fatalf("first sample: called %v shift %v RNG %d -> %d player %g/%g", called, script.vm.shakeShift,
			before, c.RNG.Count, c.Player.X, c.Player.Y)
	}
	s.UpdateHeadless(&input.State{})
	s.UpdateHeadless(&input.State{})
	if len(script.vm.shakes) != 0 || script.vm.shakeShift != [2]float32{} || c.RNG.Count != before+8 {
		t.Fatal("stage did not remove the expired shake at calc 0x0e")
	}
}

type shakeDrawRec struct {
	supervisortest.Rec
	vertices [][]gfx.TLVertex
}

func (r *shakeDrawRec) DrawTL(p gfx.Primitive, v []gfx.TLVertex) {
	r.Rec.DrawTL(p, v)
	r.vertices = append(r.vertices, slices.Clone(v))
}

func TestShakeOffsetFeedsFieldOnly(t *testing.T) {
	vm := newTestVM()
	vm.queueShake(3, 9, 0)
	vm.CalcShake()
	d := &shakeDrawRec{}
	sv := supervisor.New(d, nil)
	f := &anm.File{}
	tex := &gfx.Texture{W: 1, H: 1}
	sv.NewSheet = func(*anm.File) *anmdraw.Sheet {
		return &anmdraw.Sheet{Sprites: []anmdraw.Sprite{{Tex: tex, W: 8, H: 8, U1: 1, V1: 1}}}
	}
	v := anm.New()
	v.File, v.Pos = f, [3]float32{224, 120, 0}
	v.SetSprite(0)
	pos, offset, viewport := v.Pos, v.Offset, sv.Field.Viewport
	sv.BeginFrame()
	sv.EnterField()
	sv.ANM.Draw(v)
	sv.ANM.Flush()
	x, y := vm.ShakeOffset()
	sv.Field.Shift = [2]float32{float32(x), float32(y)}
	sv.EnterField()
	sv.ANM.Draw(v)
	sv.ANM.Flush()
	sv.Unshake() // draw 0x24: every later GUI VM must be unshifted
	sv.ANM.Draw(v)
	sv.ANM.Flush()
	if len(d.vertices) != 3 {
		t.Fatalf("draw batches %d, want 3", len(d.vertices))
	}
	for i, b := range d.vertices[0] {
		shifted := d.vertices[1][i]
		if shifted.X != b.X-6 || shifted.Y != b.Y+6 || d.vertices[2][i] != b {
			t.Errorf("vertex %d did not shift then unshake: base %+v shifted %+v restored %+v", i, b, shifted, d.vertices[2][i])
		}
	}
	if v.Pos != pos || v.Offset != offset || sv.Field.Viewport != viewport || sv.Field.Shift != [2]float32{} || sv.ANM.Shift != [2]float32{} {
		t.Fatal("shake moved logical coordinates/viewport or escaped the GUI unshake node")
	}
}
