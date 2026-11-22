package ending

import (
	"fmt"

	"th10/anm"
	"th10/game"
	"th10/gamedata"
	"th10/gfx"
	"th10/input"
	"th10/scene/result"
	"th10/supervisor"

	"github.com/hajimehoshi/ebiten/v2"
)

// EndingIndex is FUN_0040b560, 0x40b684..0x40b6b4, and PE 0x474440.
// Extra/practice clear never enters EndingInf in the original controller.
func EndingIndex(state *game.GameState) (int, error) {
	if state == nil || state.Character < 0 || state.Character > 1 || state.ShotType < 0 || state.ShotType > 2 || state.Difficulty < 0 || state.Difficulty > 3 {
		return 0, fmt.Errorf("ending: invalid normal-clear selection")
	}
	index := state.Character*3 + state.ShotType
	if state.ReplayState90 == 0 && state.Difficulty >= game.DiffNormal {
		index += 6
	}
	return index, nil
}

// Ending is DAT_00477700 (FUN_0040b940/0040b560). Staff is the same
// object's second MSG, not a separate approximate story or credit timer.
type Ending struct {
	state       game.GameState
	extra       bool
	index       int
	back        game.Scene
	vm          *msgVM
	world       anm.World
	env         anm.Env
	resources   [4]*anm.File // ANM slots 0x1d..0x20 outlive MSG opcode 12
	loading     *anm.VM
	pending     *resourceLoad
	fades       []*screenFade
	frame       int32 // EndingInf +0x24, kept across the Staff MSG switch
	firstEnding bool  // EndingInf +0x20 bit 0
	firstStaff  bool  // EndingInf +0x20 bit 1
	entered     bool
	closed      bool
	err         error

	// After retains t5's scene-15 ranking/replay entry on the same Stage.
	After func(*game.Game) game.Scene
	// ResolveScene lets the supervisor map scene 15/2 without moving its
	// lifecycle into this package. Nil uses After/back, as existing callers do.
	ResolveScene   func(*game.Game, int32) game.Scene
	RequestedScene int32 // FUN_0040b9f0, 0x40b9fd..0x40ba11
}

func New(state *game.GameState, extra bool, back game.Scene) *Ending {
	e := &Ending{extra: extra, back: back, firstEnding: true, firstStaff: true}
	if state != nil {
		e.state = *state
	}
	if extra {
		// Compatibility only: FUN_00415e90 sends Extra to the clear menu,
		// without any of the twelve normal ending scripts.
		return e
	}
	e.index, e.err = EndingIndex(&e.state)
	if e.err == nil {
		e.init(fmt.Sprintf("e%02d.msg", e.index), false)
	}
	return e
}

// Staff exposes the native Staff segment as a game.Scene for a supervisor
// or a headless viewer. Good endings normally reach it through MSG op 12.
type Staff struct{ *Ending }

func NewStaff(state *game.GameState, back game.Scene) *Staff {
	e := &Ending{back: back, index: -1, firstEnding: true, firstStaff: true}
	if state != nil {
		e.state = *state
	}
	e.init("staff.msg", true)
	return &Staff{Ending: e}
}

func animation(name string) (*anm.File, error) {
	if name == "text.anm" {
		return gamedata.SharedOrdinalANM(name, true) // FUN_00447470's runtime text-script table
	}
	return gamedata.SharedPresentationANM(name)
}

func (e *Ending) init(name string, staff bool) {
	e.env.Alt = e.state.AlternateRNG()
	e.env.Spawn = e.world.Add
	var instructions []instruction
	instructions, e.err = loadMSG(name)
	if e.err != nil {
		return
	}
	e.vm = &msgVM{owner: e}
	e.vm.reset(name, instructions, staff)
	if !staff {
		// FUN_0040ba90: five text.anm VMs, ordinal 0x42..0x46, font 16.
		var text *anm.File
		text, e.err = animation("text.anm")
		if e.err != nil {
			return
		}
		for i := range e.vm.lines {
			e.vm.lines[i] = e.world.SpawnTail(text, 0x42+i, 15, &e.env)
		}
	}
	// FUN_0040b560 (0x40b650) starts the existing loading ANM at (480,392).
	e.showLoading([3]float32{480, 392, 0})
}

func (e *Ending) showLoading(pos [3]float32) {
	if e.world.Contains(e.loading) {
		return
	}
	f, err := animation("ascii.anm")
	if err != nil {
		e.err = err
		return
	}
	// FUN_00448d50 copies a screen-space Offset before Start's reset;
	// unlike World.Spawn, it does not add the playfield's (224,16).
	v := anm.New()
	v.Offset = pos
	v.Flags |= anm.FlagAltRNG
	v.Start(f, 6, &e.env) // 0x448d95 clears the alternate flag, as the binary does
	e.world.Add(v)
	e.loading = v
}

func (e *Ending) enter(g *game.Game) error {
	e.entered = true
	e.env.Alt = g.AlternateRNG()
	if g.Save == nil || e.state.ReplayMode == 1 {
		return nil
	}
	if g.Save.Native == nil {
		g.Save.Native = game.DefaultScoreFile(0)
	}
	native := g.Save.Native
	e.firstEnding, e.firstStaff = !native.EndingSeen(e.index), !native.StaffSeen()
	if e.index < 0 {
		e.firstEnding = e.firstStaff
		native.MarkStaffSeen()
	} else {
		native.MarkEndingSeen(e.index)
	}
	return g.SaveProgress()
}

func (e *Ending) IsStaff() bool { return e.vm != nil && e.vm.flags&msgStaff != 0 }

func (e *Ending) ScriptName() string {
	if e.vm == nil {
		return ""
	}
	return e.vm.name
}

func (e *Ending) Update(g *game.Game) error {
	if e.closed {
		return nil
	}
	if e.err != nil {
		return e.err
	}
	if e.extra {
		e.finish(g)
		return nil
	}
	if !e.entered {
		if err := e.enter(g); err != nil {
			return err
		}
	}
	advance, skip := input.Global.JustPressed(input.KeyOk), input.Global.IsPressed(input.KeySkip)
	for {
		// FUN_00449c00's return-6 target 0x449c30 restarts the entire
		// calc chain. Repeat just this scene's prior calc nodes, not Draw.
		if err := e.completeLoad(); err != nil {
			return err
		}
		e.calcFades() // calc 0x0e, before World 0x1a and Ending 0x1c
		e.world.Run(&e.env)
		done, err := e.vm.step(g, advance, skip)
		if err != nil {
			return err
		}
		if done {
			e.finish(g)
			return nil
		}
		e.vm.elapsed.Tick(1) // FUN_0040bd20, also ticks during a blocked wait
		e.frame++
		if e.vm.flags&msgLoading != 0 || e.firstStaff || !e.IsStaff() || !skip || e.frame%12 == 0 {
			break // FUN_0040b9f0, 0x40ba27..0x40ba56, return 1
		}
		advance = false // repeated input calc no longer has the press edge
	}
	return nil
}

// completeLoad is the deferred FUN_0040c3a0 -> FUN_0040c3c0 callback.
// Embedded bytes need no worker thread; it still executes on the next calc.
func (e *Ending) completeLoad() error {
	if e.pending == nil {
		return nil
	}
	q := e.pending
	e.pending = nil
	f, err := animation(q.name)
	if err != nil {
		return fmt.Errorf("ending: load %s: %w", q.name, err)
	}
	old := e.resources[q.slot]
	for l := range anm.Layers {
		e.world.Each(l, func(vm *anm.VM) {
			if vm.File == old && old != nil {
				e.world.Delete(vm) // FUN_004477d0 unloads the old file's VMs
			}
		})
	}
	e.resources[q.slot], e.vm.files[q.slot] = f, f
	e.vm.flags &^= msgLoading
	e.world.Interrupt(e.loading, 1) // FUN_00409e50 at 0x40c42e
	e.loading = nil
	return nil
}

func (e *Ending) finish(g *game.Game) {
	e.RequestedScene = 15
	if g.SystemFlags&0x1000 != 0 {
		e.RequestedScene = 2
	}
	e.Close()
	if e.ResolveScene != nil {
		g.GoTo(e.ResolveScene(g, e.RequestedScene))
	} else if e.RequestedScene == 2 {
		g.GoTo(e.back)
	} else if e.After != nil {
		g.GoTo(e.After(g))
	} else {
		mode := result.ModeStageClear
		if e.extra {
			mode = result.ModeExtraClear
		}
		g.GoTo(result.New(&e.state, mode, e.back))
	}
}

// Close is FUN_0040b7b0/0040b3a0's scene-owned teardown. It does not clear
// the retained Stage, the application's RNG, or shared score/history data.
func (e *Ending) Close() {
	if e.closed {
		return
	}
	e.closed = true
	e.world.Clear()
	e.resources = [4]*anm.File{}
	e.pending, e.loading, e.fades = nil, nil, nil
}

func (e *Ending) Draw(screen *ebiten.Image) {
	sv := supervisor.Get()
	e.DrawPorted(sv)
	if screen != nil && sv.Frame != nil {
		screen.DrawImage(sv.Frame, nil)
	}
}

// DrawPorted follows the native World draw order; EndingInf's own draw
// 0x06 is a stub. It also works with a nil back buffer/recording device.
func (e *Ending) DrawPorted(sv *supervisor.Supervisor) {
	sv.BeginFrame()
	sv.ClearColor = gfx.Color(0xff000000)
	sv.Start()
	sv.Use(&sv.Screen)
	for l := range anm.Layers {
		sv.ANM.DrawLayer(&e.world, l)
	}
	for _, fade := range e.fades {
		fade.draw(sv) // MSG 13/14 registers draw 0x31 (0x40c255)
	}
	sv.Finish()
	sv.EndFrame()
}
