package ui

import (
	"th10/anm"
	"th10/game"
	"th10/gamedata"
	"th10/input"
	"th10/supervisor"
	"th10/zun"

	"github.com/hajimehoshi/ebiten/v2"
)

type TitlePage int32

const (
	TitleInitialize       TitlePage = 0
	TitleIntro            TitlePage = 1
	TitleMain             TitlePage = 2
	TitleQuit             TitlePage = 3
	TitleOption           TitlePage = 4
	TitleKeys             TitlePage = 5
	TitleDifficulty       TitlePage = 6
	TitleCharacter        TitlePage = 7
	TitleShot             TitlePage = 8
	TitlePractice         TitlePage = 9
	TitlePlayerData       TitlePage = 11
	TitleReplay           TitlePage = 12
	TitleMusicRoom        TitlePage = 14
	TitlePlayerDataDetail TitlePage = 15
	TitlePlayerDataCards  TitlePage = 16
)

// TitlePageHooks are the original title object's separately owned calc
// and draw functions (t11's 0x431ee0/0x4315c0/0x433ef0/etc.).
type TitlePageHooks struct {
	Calc func(*TitleController, *game.Game, *input.State) error
	Draw func(*TitleController, *supervisor.Supervisor)
}

func titleASCII() *anm.File { return gamedata.MustANM(gamedata.SharedANM("ascii.anm")) }

// TitleController is DAT_0047784c, shared by the title and all menu pages.
// Roots are the native title +0x2c4 script-indexed handles, not inline VMs.
type TitleController struct {
	Game                         *game.Game
	State                        TitlePage // +0x1c
	Substate                     int32     // +0x20
	Timer                        zun.Timer // +0x2b0
	Selector                     Selector  // +0x24
	World                        *anm.World
	Env                          *anm.Env
	File, VersionFile            *anm.File // +0x14/+0x18
	Roots                        map[int]*anm.VM
	Hooks                        map[TitlePage]TitlePageHooks
	Fades                        []*Fade
	LoadingMarker                *anm.VM // ascii manager +0x89a4
	Idle                         int32   // DAT_00474ca4
	Practice                     bool    // the pending selection; actual RunFlags are written through t8 at launch
	Character, Shot, Difficulty  int
	PracticeStage, PracticeLives int
	entry                        int
	oldDifficulty                int      // +0x58f0, Extra/demo returns
	keyBindings                  [5]int16 // +0x59cc
	onStart                      func(*game.Game, *game.GameState) error
	onDemo                       func(*game.Game, int) error
	err                          error
	version, difficultyMarker    *anm.VM
}

// NewTitleController is FUN_0042c920 -> FUN_0042caa0. All free objects
// use the application's alternate RNG; no run/state is created for a menu.
func NewTitleController(g *game.Game, entry int) *TitleController {
	f := gamedata.MustANM(gamedata.SharedANM("title.anm"))
	v := gamedata.MustANM(gamedata.SharedANM("title_v.anm"))
	w := &anm.World{}
	c := &TitleController{Game: g, World: w, File: f, VersionFile: v, entry: entry,
		Roots: make(map[int]*anm.VM), Hooks: make(map[TitlePage]TitlePageHooks)}
	c.Env = &anm.Env{GameSpeed: 1, Alt: g.AlternateRNG(), Spawn: w.Add}
	c.Selector.Wrap = true // 0x42cb49
	if !g.Selection.Initialized {
		g.Selection.Initialized, g.Selection.Difficulty = true, game.DiffNormal
	}
	c.Character, c.Shot, c.Difficulty = g.Selection.Character, g.Selection.Shot, g.Selection.Difficulty
	c.PracticeStage = g.Selection.PracticeStage
	if g.RunState != nil {
		c.Character, c.Shot, c.Difficulty = g.RunState.Character, g.RunState.ShotType, g.RunState.Difficulty
		c.Practice = g.RunState.RunFlags&game.RunFlagPractice != 0
	}
	c.Timer.Reset()
	return c
}

// SetState / SetSubstate are FUN_0042c5c0 / FUN_0042c620. The next
// frame's state sees the post-tick clock, just as FUN_0042cdf0's tail.
func (c *TitleController) SetState(page TitlePage) {
	c.State, c.Substate = page, 0
	c.Timer.Reset()
}
func (c *TitleController) SetSubstate(state int32) { c.Substate = state; c.Timer.Reset() }

func (c *TitleController) animations() TitleAnimations {
	return TitleAnimations{World: c.World, Env: c.Env, File: c.File, Roots: c.Roots}
}

func (c *TitleController) Root(script int) *anm.VM {
	return c.animations().Root(script)
}

// Spawn is FUN_0042c670: tail insertion, initial layer 15, StartKeep.
func (c *TitleController) Spawn(script int) *anm.VM {
	return c.animations().Spawn(script)
}

// LeaveScript is FUN_0042c6d0: interrupt 1 and clear the root handle.
func (c *TitleController) LeaveScript(script int) {
	c.animations().Leave(script)
}
func (c *TitleController) Interrupt(script int, label int16) {
	c.animations().Interrupt(script, label)
}
func (c *TitleController) Child(root, script int) *anm.VM {
	return c.animations().Child(root, script)
}
func (c *TitleController) InterruptChild(root, script int, label int16) {
	c.animations().InterruptChild(root, script, label)
}

// SetChildSprite is FUN_00449670, distinct from script restart 0x4496d0.
func (c *TitleController) SetChildSprite(root, script, sprite int) {
	if vm := c.Child(root, script); vm != nil {
		vm.SetSprite(sprite)
	}
}

// ShowLoading is FUN_0040c540: it creates ascii script 6, not a capture.
func (c *TitleController) ShowLoading() {
	if !c.World.Contains(c.LoadingMarker) {
		c.LoadingMarker = c.World.SpawnTail(titleASCII(), 6, 15, c.Env)
		c.World.SetOffset(c.LoadingMarker, [3]float32{480, 392, 0})
	}
}

func (c *TitleController) AddFade(kind, frames int32, rgb uint32) {
	c.Fades = append([]*Fade{NewFade(kind, frames, rgb)}, c.Fades...) // newest equal-priority callback first
}

// Calc supplies the shared tail for every page; page logic is installed
// below or by t11's Hooks. The world remains one object across menu pages.
func (c *TitleController) Calc(g *game.Game, in *input.State) error {
	if c.err != nil {
		return c.err
	}
	if h := c.Hooks[c.State]; h.Calc != nil {
		if err := h.Calc(c, g, in); err != nil {
			return err
		}
	} else if err := c.calcPage(g, in); err != nil {
		return err
	}
	c.Remember(g)
	c.Timer.Tick(c.Env.GameSpeed) // FUN_0042cdf0, 0x42d1eb..0x42d23f
	for _, fade := range c.Fades {
		fade.Calc(c.Env.GameSpeed, false, 0)
	}
	c.animations().Run()
	return nil
}

// DrawPorted preserves the supervisor and ANM renderer. Title draw's
// priority-3 producer queues ASCII; its text pass follows the menu worlds.
func (c *TitleController) DrawPorted(sv *supervisor.Supervisor) {
	sv.BeginFrame()
	sv.Start()
	sv.Use(&sv.Screen)
	sv.Fog(false)
	for layer := 0; layer < anm.Layers; layer++ {
		sv.ANM.DrawLayer(c.World, layer)
		if layer == 16 { // selection fade draw priority 0x2b, after menu layers
			for _, fade := range c.Fades {
				fade.Draw(sv)
			}
		}
	}
	// FUN_0042d260 (priority 3) queues ASCII text; its renderer submits
	// after the world/menu layers. GlyphText directly emits that queued pass.
	if h := c.Hooks[c.State]; h.Draw != nil {
		h.Draw(c, sv)
	} else if c.State == TitlePractice {
		c.drawPractice(sv)
	}
	sv.Finish()
	sv.EndFrame()
}

func (c *TitleController) Draw(screen *ebiten.Image) {
	sv := supervisor.Get()
	c.DrawPorted(sv)
	if sv.Frame != nil {
		screen.DrawImage(sv.Frame, nil)
	}
}

// Dispose is FUN_0042cb60's callback/handle release. Embedded files are
// immutable cache entries, not runtime paths reopened on every scene.
func (c *TitleController) Dispose() {
	c.World.Clear()
	clear(c.Roots)
	c.Fades, c.LoadingMarker = nil, nil
}
