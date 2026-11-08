package dialog

import (
	"fmt"
	"slices"
	"strings"
	"testing"

	"th10/anm"
	"th10/anm/anmdraw"
	"th10/gamedata"
	"th10/gfx"
	"th10/zun"
)

func presentationFixture(t *testing.T, stage, char int, entry int32, code []Instr) (*MSG, *Presentation) {
	t.Helper()
	load := func(name string) *anm.File {
		f, err := gamedata.ANM(name)
		if err != nil {
			t.Fatal(err)
		}
		return f
	}
	w := &anm.World{}
	env := &anm.Env{GameSpeed: 1, RNG: &zun.RNG{State: 1234}, Alt: &zun.RNG{}, Spawn: w.Add}
	p := NewPresentation(PresentationConfig{
		Stage: stage, Character: char, World: w, Env: env,
		Front: load("front.anm"), Text: load("text.anm").OrdinalScripts(),
		Player: load(fmt.Sprintf("pl%02d.anm", char)), Enemy: load(fmt.Sprintf("stgenm%02d.anm", stage)),
		Logo: load(fmt.Sprintf("st%02dlogo.anm", stage)),
	})
	m := NewMSG(entry, code, Hooks{})
	m.SetPresentation(p)
	return m, p
}

func TestMSGPortraitScriptsAndFaces(t *testing.T) {
	for _, tc := range []struct{ stage, char, entry int }{
		{1, 0, 0}, {2, 1, 0}, {3, 0, 2}, {4, 1, 0}, {5, 0, 0},
		{6, 0, 0}, {6, 1, 0}, {7, 0, 0}, {7, 1, 1},
	} {
		t.Run(fmt.Sprintf("s%d/p%d/e%d", tc.stage, tc.char, tc.entry), func(t *testing.T) {
			m, p := presentationFixture(t, tc.stage, tc.char, int32(tc.entry), []Instr{
				{Op: msgShowPlayer}, {Op: msgShowBoss}, {Op: msgShowBox},
				{Op: msgPlayerFace, Arg: 6}, {Op: msgBossFace, Arg: 2},
				{Op: msgBossName}, {Op: msgBossBGM}, {Time: 100, Op: msgEnd},
			})
			seed := p.c.Env.RNG.State
			if !m.Update(false, false) || p.Player == nil || p.Boss == nil || p.Box == nil || p.Name == nil {
				t.Fatalf("missing portrait/box/name handles: %+v", p)
			}
			if p.Player.ScriptID != 29 || p.Box.ScriptID != 90 || p.Name.ScriptID != bossPlate(tc.stage) {
				t.Fatalf("wrong scripts: player %d box %d name %d", p.Player.ScriptID, p.Box.ScriptID, p.Name.ScriptID)
			}
			wantBoss := 30
			if tc.stage == 6 {
				wantBoss = 32
			} else if tc.stage == 7 && tc.entry == 0 {
				wantBoss = 37
			}
			if p.Boss.ScriptID != wantBoss {
				t.Fatalf("boss script %d, want %d", p.Boss.ScriptID, wantBoss)
			}
			ptop, pbottom := 0x34+6, 0x3c+6
			if tc.char == 1 {
				ptop, pbottom = 0x2d+6, 0x35+6
			}
			for _, pick := range []struct{ script, sprite int }{{23, ptop}, {26, pbottom}} {
				v := p.c.World.FindChild(p.Player, pick.script)
				if v == nil || v.Sprite != pick.sprite {
					t.Fatalf("player child %d = %v, want sprite %d", pick.script, v, pick.sprite)
				}
			}
			top, bottom, ct, cb := bossFace(tc.stage, int32(tc.entry))
			for _, pick := range []struct{ script, sprite int }{{ct, top + 2}, {cb, bottom + 2}} {
				v := p.c.World.FindChild(p.Boss, pick.script)
				if v == nil || v.Sprite != pick.sprite || v.File != p.c.Enemy {
					t.Fatalf("boss child %d = %v, want enemy sprite %d", pick.script, v, pick.sprite)
				}
			}
			if p.c.Env.RNG.State != seed {
				t.Fatal("MSG visual ops consumed the gameplay RNG")
			}
		})
	}
}

func TestMSGSpeakerAndTextVMOperations(t *testing.T) {
	m, p := presentationFixture(t, 2, 0, 0, []Instr{
		{Op: msgShowPlayer}, {Op: msgShowBoss}, {Op: msgShowBox},
		{Op: msgSpeakerPlayer}, {Op: msgText, Text: "raw", DisplayText: "甲"},
		{Op: msgText, Text: "line2"},
		{Time: 30, Op: msgSpeakerBoss}, {Time: 30, Op: msgText, Text: "boss"},
		{Time: 50, Op: msgDimAll}, {Time: 100, Op: msgEnd},
	})
	for f := 0; f < 16; f++ {
		m.Update(false, false)
		p.c.World.Run(p.c.Env)
	}
	player := p.c.World.FindChild(p.Player, 23)
	boss := p.c.World.FindChild(p.Boss, 24)
	if player.Color != [3]uint8{255, 255, 255} || boss.Color != [3]uint8{96, 96, 96} {
		t.Fatalf("speaker colours %v/%v, want ANM bright/dim", player.Color, boss.Color)
	}
	if p.Lines[0].Offset != [3]float32{8, 0, 0} || p.Lines[1].Offset != [3]float32{8, 0, 0} ||
		p.TextContent != [2]string{"甲", "line2"} || m.View.Lines[0].Text != "raw" {
		t.Fatalf("player text offsets/content: %v / %v", p.Lines[0].Offset, p.TextContent)
	}
	for f := 16; f < 48; f++ {
		m.Update(false, false)
		p.c.World.Run(p.c.Env)
	}
	if player.Color != [3]uint8{96, 96, 96} || boss.Color != [3]uint8{255, 255, 255} ||
		p.Lines[0].Offset != [3]float32{24, 0, 0} {
		t.Fatalf("boss speaker colours/offset %v/%v/%v", player.Color, boss.Color, p.Lines[0].Offset)
	}
	for f := 48; f < 56; f++ {
		m.Update(false, false)
		p.c.World.Run(p.c.Env)
	}
	if p.Lines[0].Alpha != 0 || p.Lines[1].Alpha != 0 || boss.Color != [3]uint8{255, 255, 255} {
		t.Fatalf("op 17 changed portraits or failed to dim text: alpha %d/%d, boss %v", p.Lines[0].Alpha, p.Lines[1].Alpha, boss.Color)
	}
}

func TestMSG22And23InterruptSeven(t *testing.T) {
	m, p := presentationFixture(t, 1, 0, 0, []Instr{
		{Op: msgShowPlayer}, {Op: msgShowBoss}, {Op: msgSpeakerPlayer},
		{Time: 20, Op: msgSpeakerBoss}, {Time: 40, Op: msgAnimatePlayer, Arg: 99},
		{Time: 40, Op: msgAnimateBoss, Arg: -1}, {Time: 100, Op: msgEnd},
	})
	for range 40 {
		m.Update(false, false)
		p.c.World.Run(p.c.Env)
	}
	m.Update(false, false)
	p.c.World.Run(p.c.Env)
	for _, pair := range []struct {
		root *anm.VM
		id   int
	}{{p.Player, 23}, {p.Boss, 24}} {
		v := p.c.World.FindChild(pair.root, pair.id)
		if v == nil || v.T.Cur != 31 { // label 7 begins at ANM time 30
			t.Fatalf("op 22/23 child %d timer %v, want label-7 clock 31", pair.id, v)
		}
	}
}

func TestMSGHideUsesExitAndEndDeletes(t *testing.T) {
	m, p := presentationFixture(t, 1, 0, 0, []Instr{
		{Op: msgShowPlayer}, {Op: msgShowBoss}, {Op: msgShowBox},
		{Time: 1, Op: msgHidePlayer}, {Time: 2, Op: msgHideAll}, {Time: 3, Op: msgEnd},
	})
	m.Update(false, false)
	p.c.World.Run(p.c.Env)
	oldPlayer := p.Player
	m.Update(false, false)
	if p.Player != nil || oldPlayer.Flags&anm.FlagDelete != 0 {
		t.Fatal("op 4 immediately deleted rather than interrupting the exit script")
	}
	p.c.World.Run(p.c.Env)
	m.Update(false, false)
	p.c.World.Run(p.c.Env)
	if m.Update(false, false) || p.Lines[0].Flags&anm.FlagDelete == 0 || p.Boss.Flags&anm.FlagDelete == 0 {
		t.Fatal("MSG end did not delete its remaining handles")
	}
}

type msgDrawRec struct {
	textures []*gfx.Texture
	bound    *gfx.Texture
}

func (d *msgDrawRec) SetTexture(t *gfx.Texture)               { d.bound = t }
func (*msgDrawRec) SetFilter(gfx.Filter)                      {}
func (*msgDrawRec) SetBlend(gfx.BlendMode)                    {}
func (*msgDrawRec) SetTransform(gfx.TransformState, gfx.Mat4) {}
func (d *msgDrawRec) DrawTL(_ gfx.Primitive, _ []gfx.TLVertex) {
	d.textures = append(d.textures, d.bound)
}
func (*msgDrawRec) Draw3D(gfx.Primitive, []gfx.Vertex3D) {}

// The same fake-device pattern as anmdraw's fixture: each sprite gets a
// unique texture, which reveals the layer-node drawing order without a GPU.
func TestMSGWorldLayerDraws(t *testing.T) {
	m, p := presentationFixture(t, 1, 0, 0, []Instr{
		{Op: msgShowPlayer}, {Op: msgShowBoss}, {Op: msgShowBox}, {Op: msgSpeakerPlayer},
		{Op: msgText, Text: "a"}, {Op: msgText, Text: "b"}, {Time: 100, Op: msgEnd},
	})
	for range 20 {
		m.Update(false, false)
		p.c.World.Run(p.c.Env)
	}
	d := &msgDrawRec{}
	sheets := map[*anm.File]*anmdraw.Sheet{}
	sheet := func(f *anm.File) *anmdraw.Sheet {
		if sh := sheets[f]; sh != nil {
			return sh
		}
		sh := &anmdraw.Sheet{Sprites: make([]anmdraw.Sprite, len(f.Sprites))}
		for id, s := range f.Sprites {
			sh.Sprites[id] = anmdraw.Sprite{Tex: &gfx.Texture{W: 1, H: 1}, U1: 1, V1: 1, W: s.W, H: s.H}
		}
		sheets[f] = sh
		return sh
	}
	r := anmdraw.NewRenderer(d, sheet)
	r.View = [4]uint32{0, 0, 640, 480}
	var want []*gfx.Texture
	for _, layer := range []int{13, 14, 15, 16} {
		p.c.World.Each(layer, func(v *anm.VM) {
			if v.Flags&3 == 3 && v.Alpha > 0 {
				want = append(want, sheet(v.File).Sprites[v.Sprite].Tex)
			}
		})
		r.DrawLayer(p.c.World, layer)
		r.Flush()
	}
	if len(want) < 6 || !slices.Equal(d.textures, want) {
		t.Fatalf("MSG layers drew %d sprites, want %d in World order", len(d.textures), len(want))
	}
}

func TestMSGTraditionalOverlayKeepsInstructions(t *testing.T) {
	for stage := 1; stage <= 7; stage++ {
		name := fmt.Sprintf("stage%d", stage)
		if stage == 7 {
			name = "extra"
		}
		for char := 0; char < 2; char++ {
			data, err := gamedata.ReadFile(fmt.Sprintf("msg/st%02d_%02d.msg", stage, char))
			if err != nil {
				t.Fatal(err)
			}
			raw, localized := ParseMSG(data), LoadStage(name, char)
			displays := 0
			for i, code := range raw {
				for j, in := range code {
					got := localized.Entries[i][j]
					if got.Time != in.Time || got.Op != in.Op || got.Arg != in.Arg || got.Text != in.Text {
						t.Fatalf("stage %d/%d entry %d instruction %d changed logic", stage, char, i, j)
					}
					if got.DisplayText != "" {
						if strings.ContainsRune(got.DisplayText, 0) {
							t.Errorf("stage %d/%d retained data after the text's NUL", stage, char)
						}
						displays++
					}
				}
			}
			if displays == 0 {
				t.Errorf("stage %d/%d has no th10c.dat display text", stage, char)
			}
		}
	}
}
