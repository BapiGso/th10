package stage

import (
	"fmt"
	"math"
	"slices"
	"strings"
	"testing"

	"th10/anm"
	"th10/anm/anmdraw"
	"th10/game"
	"th10/gfx"
	"th10/supervisor"
	"th10/supervisor/supervisortest"
)

// guiRec uses the recording-device fixture pattern in anmdraw's tests:
// no GPU, one distinct texture per VM, and copies of submitted vertices.
type guiRec struct {
	supervisortest.Rec
	vertices [][]gfx.TLVertex
	prims    []gfx.Primitive
}

func (d *guiRec) DrawTL(p gfx.Primitive, v []gfx.TLVertex) {
	d.Rec.DrawTL(p, v)
	d.vertices = append(d.vertices, slices.Clone(v))
	d.prims = append(d.prims, p)
}

func guiFixture(t *testing.T, vms []*anm.VM) (*supervisor.Supervisor, *guiRec) {
	t.Helper()
	f := &anm.File{}
	sh := &anmdraw.Sheet{Sprites: make([]anmdraw.Sprite, len(vms))}
	d := &guiRec{Rec: supervisortest.Rec{Names: map[*gfx.Texture]string{}}}
	for i, v := range vms {
		*v = *anm.New()
		v.File, v.Pos = f, [3]float32{100, 100, 0}
		v.SetSprite(i)
		tex := &gfx.Texture{W: 1, H: 1}
		d.Names[tex] = fmt.Sprintf("%02d", i)
		sh.Sprites[i] = anmdraw.Sprite{Tex: tex, U1: 1, V1: 1, W: 1, H: 1}
	}
	sv := supervisor.New(d, nil)
	sv.NewSheet = func(g *anm.File) *anmdraw.Sheet {
		if g != f {
			t.Fatal("GUI drew a VM from another file")
		}
		return sh
	}
	sv.BeginFrame()
	sv.Enter2D()
	d.Take()
	return sv, d
}

// TestGUIArrayDrawOrder locks down all four unconditional walks of
// FUN_00415800, including interleaving the two ten-element score arrays.
func TestGUIArrayDrawOrder(t *testing.T) {
	g := &GUI{ctx: &Context{State: &game.GameState{}}}
	var vms []*anm.VM
	for i := range g.score {
		vms = append(vms, &g.hiScore[i], &g.score[i])
	}
	for i := range g.lives {
		vms = append(vms, &g.lives[i])
	}
	for i := range g.power {
		vms = append(vms, &g.power[i])
	}
	for i := range g.faith {
		vms = append(vms, &g.faith[i])
	}
	sv, d := guiFixture(t, vms)
	s := &Stage{ctx: g.ctx}
	s.ctx.GUI = g
	s.nodeGUI(sv)
	sv.ANM.Flush()
	var textures []string
	for _, c := range d.Take() {
		if strings.HasPrefix(c, "tex ") {
			textures = append(textures, c)
		}
	}
	want := make([]string, 40)
	for i := range want {
		want[i] = fmt.Sprintf("tex %02d", i)
	}
	if !slices.Equal(textures, want) {
		t.Fatalf("textures %v, want %v", textures, want)
	}
	if len(d.vertices) != 40 {
		t.Fatalf("%d submitted VMs, want 40", len(d.vertices))
	}
	for i, v := range d.vertices {
		if len(v) != 6 || d.prims[i] != gfx.TriangleList {
			t.Errorf("VM %d: primitive %d with %d vertices", i, d.prims[i], len(v))
		}
	}
}

func TestGUINumericCaps(t *testing.T) {
	for _, tc := range []struct {
		n    int64
		want string
	}{
		{-1, "0000000000"}, {0, "0000000000"}, {100, "0000000100"},
		{1234567890, "1234567890"}, {9999999990, "9999999990"},
		{9999999999, "9999999990"}, {math.MaxInt64, "9999999990"},
	} {
		d := scoreDigits(tc.n)
		var b strings.Builder
		for i := len(d) - 1; i >= 0; i-- {
			b.WriteByte(byte(d[i]) + '0')
		}
		if b.String() != tc.want {
			t.Errorf("score %d: %q, want %q", tc.n, b.String(), tc.want)
		}
	}
	for _, tc := range []struct {
		power, whole, percent int
	}{
		{-100, 0, 0}, {0, 0, 0}, {5, 0, 5}, {95, 0, 95},
		{100, 1, 0}, {495, 4, 95}, {500, 5, 0}, {9999, 5, 0},
	} {
		whole, percent := powerDigits(tc.power)
		if whole != tc.whole || percent != tc.percent {
			t.Errorf("power %d: %d.%02d, want %d.%02d", tc.power, whole, percent, tc.whole, tc.percent)
		}
	}
}

func TestGUIPowerSprites(t *testing.T) {
	g := &GUI{}
	g.power[1].Sprite = 19
	g.SetPower(4, 95)
	if got := [4]int{g.power[0].Sprite, g.power[1].Sprite, g.power[2].Sprite, g.power[3].Sprite}; got != [4]int{12, 19, 17, 13} {
		t.Fatalf("power sprites %v, want [12 19 17 13]", got)
	}
}

func TestGUILivesKeepVMs(t *testing.T) {
	g := &GUI{}
	for i := range g.lives {
		g.lives[i].Flags = 0x12340003
	}
	g.SetLives(3)
	for i, v := range g.lives {
		if (v.Flags&anm.FlagDraw != 0) != (i < 3) || v.Flags&^anm.FlagDraw != 0x12340001 {
			t.Errorf("life %d flags %#x", i, v.Flags)
		}
	}
	g.SetLives(100)
	for i, v := range g.lives {
		if v.Flags&anm.FlagDraw == 0 {
			t.Errorf("capped lives: VM %d is hidden", i)
		}
	}
}

func TestGUIFaithRectCallSites(t *testing.T) {
	black, white, col := faithRects(130, 32)
	if black != [4]float32{51, 461, 89, 463} || white != [4]float32{50, 460, 88, 462} || col != 0x20000000 {
		t.Fatalf("faith rectangles %v %v %#x", black, white, col)
	}
	if math.Float32bits(guiHoldUnit) != 0x3e95a95b {
		t.Fatalf("hold unit %#x, want DAT_00470cb8", math.Float32bits(guiHoldUnit))
	}
}

func newTestGUI() *GUI {
	st := game.NewState(game.CharReimu, game.ShotA, game.DiffNormal)
	st.Stage = 1
	ctx := NewContext(st)
	ctx.GUI = NewGUI(ctx)
	return ctx.GUI
}

func TestGUIScoreRollIsVisualOnly(t *testing.T) {
	g := newTestGUI()
	g.ctx.State.Score = 3200
	before := *g.ctx.State
	seed := g.ctx.RNG.State
	g.Numbers()
	if g.shownScore != 10 || g.scoreStep != 10 {
		t.Fatalf("first roll %d step %d, want 10 / 10", g.shownScore, g.scoreStep)
	}
	for range 100 {
		g.Calc()
		g.Numbers()
	}
	if g.shownScore != 320 || g.scoreStep != 0 {
		t.Fatalf("settled score %d step %d, want 320 / 0", g.shownScore, g.scoreStep)
	}
	if *g.ctx.State != before || g.ctx.RNG.State != seed {
		t.Fatalf("GUI changed game state or game RNG: %+v / %#x", *g.ctx.State, g.ctx.RNG.State)
	}
}

func TestGUIFaithDigitsAndHysteresis(t *testing.T) {
	g := newTestGUI()
	g.ctx.State.Faith = 123450
	for range 40 {
		g.Calc()
	}
	for i, want := range []int{31, 32, 33, 34, 35} {
		if got := g.faith[i+1].Sprite; got != want {
			t.Errorf("faith digit %d sprite %d, want %d", i, got, want)
		}
	}
	g.ctx.Player.X, g.ctx.Player.Y = gameX-129, gameY+417
	for range 9 {
		g.Calc()
	}
	if !g.faithDim || g.faith[0].Alpha != 32 {
		t.Fatalf("covered faith: dim %v alpha %d", g.faithDim, g.faith[0].Alpha)
	}
	g.ctx.Player.X = gameX - 120 // inside the hysteresis band
	g.Calc()
	if !g.faithDim {
		t.Fatal("faith brightened inside the hysteresis band")
	}
	g.ctx.Player.X = gameX - 111
	for range 9 {
		g.Calc()
	}
	if g.faithDim || g.faith[0].Alpha != 255 {
		t.Fatalf("uncovered faith: dim %v alpha %d", g.faithDim, g.faith[0].Alpha)
	}
}
