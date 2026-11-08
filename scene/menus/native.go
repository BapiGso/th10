package menus

import (
	"math"
	"sync"

	"th10/anm"
	"th10/game"
	"th10/gamedata"
	"th10/gfx"
	"th10/input"
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
	loadingBurst       bool
	fade               *menuFade
}

var menuFiles = struct {
	sync.Mutex
	files map[string]*anm.File
}{files: map[string]*anm.File{}}

func menuAnimation(name string) (*anm.File, error) {
	menuFiles.Lock()
	defer menuFiles.Unlock()
	if f := menuFiles.files[name]; f != nil {
		return f, nil
	}
	f, err := gamedata.PresentationANM(name)
	if err != nil {
		return nil, err
	}
	if name == "text.anm" {
		f = f.OrdinalScripts() // FUN_00447470, labels -73..-1 are not caller IDs
	}
	menuFiles.files[name] = f
	return f, nil
}

func (m *nativeMenu) init(g *game.Game, background, heading int) error {
	m.world.Clear()
	m.loadingBurst = false
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
	return nil
}

func (m *nativeMenu) spawnTitle(id int) *anm.VM {
	return m.world.SpawnTail(m.title, id, 15, &m.env) // FUN_0042c670
}

func (m *nativeMenu) spawnText(id int) *anm.VM {
	return m.world.SpawnTail(m.text, id, 15, &m.env)
}

func (m *nativeMenu) interrupt(v *anm.VM, label int16) {
	m.world.Interrupt(v, label) // FUN_0042c6d0/0042c770/00449210
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
	if m.fade != nil && m.fade.calc() {
		m.fade = nil
	}
	loadingTime := m.loading.T.Cur
	m.world.Run(&m.env)
	m.calcLoading(loadingTime)
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

// calcLoading is the local ANM-91 subset used only by ascii.anm script 8.
// FUN_0043ee30 0x43ef55/0x43f00d creates front children: sixteen at time 1,
// then loops just the last emission from time 4 back to time 1. The common
// VM currently treats 91 as a no-op. Spawn after Run so front-born children
// are not visited again on their birth pass, just as the native list walk.
func (m *nativeMenu) calcLoading(before int32) {
	if !m.world.Contains(m.loading) {
		return
	}
	count := 0
	if !m.loadingBurst && before == 1 && m.loading.T.Cur == 2 {
		count, m.loadingBurst = 16, true
	} else if before == 4 && m.loading.T.Cur == 2 {
		count = 1
	}
	for range count {
		v := m.world.SpawnFront(m.ascii, 7, m.loading.Layer, &m.env)
		v.Pos2, v.Offset = m.loading.Pos, m.loading.Offset
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
		sv.ANM.Rect([4]float32{0, 0, 640, 480}, gfx.Color(uint32(m.fade.alpha)<<24))
	}
	sv.Finish()
	sv.EndFrame()
}

// menuFade is kind 5 of FUN_0043c8b0/0043c230/0043c1a0, registered by
// replay confirmation at 0x4319cd with duration 32 and draw priority 0x2b.
type menuFade struct {
	timer    zun.Timer
	duration int32
	alpha    int32
}

func newMenuFade(duration int32) *menuFade {
	f := &menuFade{duration: duration}
	f.timer.Reset()
	return f
}

func (f *menuFade) calc() bool {
	if f.timer.Cur >= f.duration+2 {
		return true
	}
	if f.timer.Cur >= f.duration {
		f.alpha = 255
	} else if f.duration > 0 {
		x := float32(f.timer.F * math.Float32frombits(0x437f0000)) // PE 0x470bf8
		f.alpha = int32(float32(float64(x) / float64(f.duration)))
	}
	f.timer.Tick(1)
	return false
}

// menuRepeat is the local minimum of FUN_0040ac20/0040ace0: first press,
// then held frame 26 and every eight frames. Merge with t10's collector.
func menuRepeat(in *input.State, key input.Key) bool {
	n := in.HoldFrames(key)
	return in.JustPressed(key) || (n >= 26 && (n-26)%8 == 0)
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
	v := anm.New()
	v.File = m.ascii
	v.Flags = v.Flags&0xffd7ffff | 0x140001
	v.Color = [3]uint8{byte(colour >> 16), byte(colour >> 8), byte(colour)}
	v.Alpha = byte(colour >> 24)
	start := x
	for i := range len(text) {
		c := text[i]
		if c == '\n' {
			x, y = start, y+16
			continue
		}
		if c < 0x20 || c > 0x81 {
			c = '?'
		}
		v.SetSprite(int(c) + 66) // ASCII 0x20 -> sprite 98, 0x40176f
		v.Pos = [3]float32{x, y, 0}
		sv.ANM.Draw(v)
		x += math.Float32frombits(0x41100000) // PE 0x470c10, 9
	}
}
