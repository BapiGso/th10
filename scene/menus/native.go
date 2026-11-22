package menus

import (
	"th10/anm"
	"th10/game"
	"th10/gamedata"
	"th10/gfx"
	sceneui "th10/scene/ui"
	"th10/supervisor"
	"th10/zun"
)

// nativeMenu contains only the TitleInf subscene's ANM objects. Title's
// manager, main menu and input collector remain owned by t10.
type nativeMenu struct {
	world              anm.World
	env                anm.Env
	title, text, ascii *anm.File
	positions          map[*anm.VM]*zun.Interp
	loading            *anm.VM
	loadingEmitter     sceneui.LoadingEmitter
	fade               *sceneui.Fade
}

func menuAnimation(name string) (*anm.File, error) {
	if name == "text.anm" {
		return gamedata.SharedOrdinalANM(name, true) // FUN_00447470, labels -73..-1 are not caller IDs
	}
	return gamedata.SharedPresentationANM(name)
}

func (m *nativeMenu) init(g *game.Game, background, heading int) error {
	m.world.Clear()
	m.fade = nil
	var err error
	if m.title, err = menuAnimation("title.anm"); err != nil {
		return err
	}
	if m.text, err = menuAnimation("text.anm"); err != nil {
		return err
	}
	if m.ascii, err = menuAnimation("ascii.anm"); err != nil {
		return err
	}
	m.env.Alt, m.env.Spawn = g.AlternateRNG(), m.world.Add
	m.positions = make(map[*anm.VM]*zun.Interp)
	m.spawnTitle(background)
	m.spawnTitle(heading)
	m.loading = m.world.SpawnTail(m.ascii, 8, 15, &m.env) // FUN_004315c0/00431ee0/00433ef0
	m.loadingEmitter = sceneui.LoadingEmitter{Parent: m.loading, File: m.ascii}
	return nil
}

func (m *nativeMenu) animations() sceneui.TitleAnimations {
	return sceneui.TitleAnimations{World: &m.world, Env: &m.env, File: m.title}
}

func (m *nativeMenu) spawnTitle(id int) *anm.VM {
	return m.animations().Spawn(id)
}

func (m *nativeMenu) spawnText(id int) *anm.VM {
	return m.world.SpawnTail(m.text, id, 15, &m.env)
}

func (m *nativeMenu) interrupt(v *anm.VM, label int16) {
	m.animations().InterruptVM(v, label)
}

// move is the private minimum of FUN_00434a80: a four-frame, mode-0
// position tween. It does not expose or change ANM's private interpolator.
func (m *nativeMenu) move(v *anm.VM, target [3]float32) {
	if !m.world.Contains(v) {
		return
	}
	p := &zun.Interp{Start: v.Pos, End: target}
	p.Begin(4, 0) // 0x43426d/0x4345f2 pass duration 4, mode 0
	m.positions[v] = p
}

func (m *nativeMenu) calc() {
	// Title calc 0x06 precedes World calc 0x1a (FUN_0042caa0).
	if m.fade != nil {
		m.fade.Calc(1, false, 0)
		if !m.fade.Active {
			m.fade = nil
		}
	}
	m.animations().Run(&m.loadingEmitter)
	for v, p := range m.positions {
		if !m.world.Contains(v) {
			delete(m.positions, v)
			continue
		}
		v.Pos = p.Step(1, 3)
		if p.Dur == 0 {
			delete(m.positions, v)
		}
	}
}

func (m *nativeMenu) close() {
	m.world.Clear() // FUN_0042cb60's subscene-owned teardown only
	clear(m.positions)
	m.fade = nil
}

func (m *nativeMenu) draw(sv *supervisor.Supervisor, background func(*supervisor.Supervisor), text func()) {
	sv.BeginFrame()
	sv.ClearColor = gfx.Color(0xff000000)
	sv.Start()
	sv.Use(&sv.Screen)
	if background != nil {
		background(sv) // optional t10-owned persistent title backdrop
	}
	for l := range anm.Layers {
		sv.ANM.DrawLayer(&m.world, l)
	}
	if text != nil {
		text() // FUN_00401760's small-font queue, without a legacy image
	}
	if m.fade != nil {
		m.fade.DrawRect(sv) // replay's draw 0x2b remains after the native text pass
	}
	sv.Finish()
	sv.EndFrame()
}

func wrapSelection(at, count int) int {
	if count <= 0 {
		return 0
	}
	at %= count
	if at < 0 {
		at += count
	}
	return at // FUN_0044bea0's full-wrap mode
}

func menuSE(g *game.Game, id int) {
	if a := g.Audio(); a != nil {
		a.PlaySEID(id) // FUN_0043dc90, pan zero
	}
}

// smallText is the 8x8/9-pixel-advance subset of FUN_00401760 used by
// TitleInf's Player Data/replay draw routines (+0x8988 = 1).
func (m *nativeMenu) smallText(sv *supervisor.Supervisor, text string, x, y float32, colour uint32) {
	sceneui.GlyphTextFile(sv, m.ascii, text, x, y, colour)
}
