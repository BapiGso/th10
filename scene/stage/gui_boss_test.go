package stage

import (
	"math"
	"slices"
	"testing"

	"th10/anm"
	"th10/game"
	"th10/gfx"
)

type guiBossScript struct {
	stubScript
	info BossHUDInfo
}

func (s *guiBossScript) BossHUD() BossHUDInfo { return s.info }

func guiBossFixture() (*GUI, *guiBossScript) {
	g := newTestGUI()
	s := &guiBossScript{info: BossHUDInfo{Active: true, HP: 1000, MaxHP: 1000, PhaseHP: 1000}}
	g.ctx.stage = &Stage{ctx: g.ctx, script: s}
	return g, s
}

func TestGUIBossNames(t *testing.T) {
	for _, tc := range []struct{ stage, mid, boss int }{
		{1, 0x85, 0x86}, {2, -1, 0x87}, {3, -1, 0x88}, {4, 0x89, 0x8a},
		{5, 0x8b, 0x8b}, {6, 0x8c, 0x8c}, {7, 0x8c, 0x8d},
	} {
		if a, b := bossNameScript(tc.stage, true), bossNameScript(tc.stage, false); a != tc.mid || b != tc.boss {
			t.Errorf("stage %d names %#x/%#x, want %#x/%#x", tc.stage, a, b, tc.mid, tc.boss)
		}
	}
}

func TestGUIBossFillAndClear(t *testing.T) {
	g, s := guiBossFixture()
	s.info.HP = 500
	g.Calc()
	if math.Float32bits(g.hp) != 0x3ccccccd {
		t.Fatalf("first fill %g (%#x), want DAT_00470cf8", g.hp, math.Float32bits(g.hp))
	}
	for range 20 {
		g.Calc()
	}
	if g.hp != 0.5 {
		t.Fatalf("filled HP %g, want 0.5", g.hp)
	}
	s.info.HP = 200
	g.Calc()
	if g.hp != float32(200)/1000 {
		t.Fatalf("damage fill %g, want immediate 0.2", g.hp)
	}
	g.SetHPMarker(3, 0.5, 0x7f010203)
	s.info.Active = false
	g.Calc()
	if g.hp != 0 || g.HPMarkers[3] != (HPMarker{0, 0x7f010203}) || g.bossName != nil {
		t.Fatalf("no boss: hp %g marker %+v name %v", g.hp, g.HPMarkers[3], g.bossName)
	}
}

func TestGUIBossSegmentsInSlotOrder(t *testing.T) {
	g := &GUI{ctx: &Context{State: &game.GameState{}}, hp: 0.75, timeLeft: -1}
	g.HPMarkers[0] = HPMarker{0.6, 0x80203040}
	g.HPMarkers[1] = HPMarker{0.9, 0xff405060}
	g.HPMarkers[3] = HPMarker{0.2, 0x00708090}
	sv, _ := guiFixture(t, nil)
	type rectCall struct {
		v [4]float32
		c gfx.Color
	}
	var got []rectCall
	g.draw(sv.ANM, func(v [4]float32, c gfx.Color) { got = append(got, rectCall{v, c}) })
	black, white := hpRects(0.75)
	want := []rectCall{
		{black, 0xff000000}, {white, 0xffffffff},
		{[4]float32{40, 22, float32(float32(0.6)*342) + 40, 24}, 0x80203040},
		{[4]float32{40, 22, float32(float32(0.75)*342) + 40, 24}, 0xff405060},
		{[4]float32{40, 22, float32(float32(0.2)*342) + 40, 24}, 0x00708090},
	}
	if !slices.Equal(got, want) {
		t.Fatalf("segments %+v, want %+v", got, want)
	}
}

func TestGUIBossStarsAndTimer(t *testing.T) {
	g, _ := guiBossFixture()
	g.SetBossStars(3)
	g.SetBossTimer(11)
	g.Calc()
	g.AfterMSG()
	for i, v := range g.stars {
		if (v != nil) != (i < 3) {
			t.Errorf("star %d = %v", i, v)
		}
	}
	old := g.stars[2]
	g.SetBossStars(1)
	g.Calc()
	if g.stars[2] != nil || old == nil {
		t.Fatal("star removal did not detach its handle")
	}
	if g.timer[0].Sprite != 9 || g.timer[1].Sprite != 9 {
		t.Fatalf("11 seconds sprites %d/%d", g.timer[0].Sprite, g.timer[1].Sprite)
	}
	g.SetBossTimer(10)
	g.AfterMSG()
	g.Calc()
	if g.timer[0].Color != [3]uint8{255, 144, 144} {
		t.Fatalf("10-second warning colour %v", g.timer[0].Color)
	}
	g.SetBossTimer(5)
	g.AfterMSG()
	g.Calc()
	if g.timer[0].Color != [3]uint8{255, 64, 64} {
		t.Fatalf("5-second warning colour %v", g.timer[0].Color)
	}
}

func TestGUIArrowThresholdAndAlpha(t *testing.T) {
	g := newTestGUI()
	g.info = BossHUDInfo{Active: true, PhaseHP: 699}
	g.ctx.Player.X = gameX
	g.calcArrow()
	if g.arrowStyle != 1 || g.arrow.Alpha != 64 {
		t.Fatalf("arrow state %d alpha %d, want 1/64", g.arrowStyle, g.arrow.Alpha)
	}
	g.info.PhaseHP = 0
	g.ctx.Player.X = gameX + 32
	g.calcArrow()
	if g.arrowStyle != 2 || g.arrow.Alpha != 159 {
		t.Fatalf("arrow state %d alpha %d, want one step and alpha 159", g.arrowStyle, g.arrow.Alpha)
	}
	g.ctx.Player.X = gameX + 64
	g.calcArrow()
	if g.arrowStyle != 3 || g.arrow.Alpha != 255 {
		t.Fatalf("arrow state %d alpha %d, want 3/255", g.arrowStyle, g.arrow.Alpha)
	}
	g.info.Position[0] = 193
	g.calcArrow()
	if g.arrow.Alpha != 0 {
		t.Fatal("arrow outside the field is visible")
	}
}

func TestGUISpellDrawOrder(t *testing.T) {
	g := &GUI{ctx: &Context{State: &game.GameState{}}}
	g.ctx.GUI = g
	g.spellHUD.Active = true
	var vms []*anm.VM
	for i := range g.spellHUD.Bonus {
		vms = append(vms, &g.spellHUD.Bonus[i])
	}
	for i := range g.spellHUD.History {
		vms = append(vms, &g.spellHUD.History[i])
	}
	sv, d := guiFixture(t, vms)
	(&Stage{ctx: g.ctx}).nodeSpellHUD(sv)
	sv.ANM.Flush()
	if len(d.vertices) != 13 {
		t.Fatalf("spell node submitted %d VMs, want 13", len(d.vertices))
	}
	d.Take()
	g.spellHUD.Active = false
	(&Stage{ctx: g.ctx}).nodeSpellHUD(sv)
	sv.ANM.Flush()
	if len(d.Take()) != 0 {
		t.Fatal("inactive spell HUD issued device calls")
	}
}

func TestGUISpellNumbersAndCapturePopup(t *testing.T) {
	g := newTestGUI()
	g.BeginSpellPresentation("Spell")
	g.UpdateSpellPresentation(123456789, true, 123, 7, 90)
	for i, v := range g.spellHUD.Bonus {
		if v.Sprite != 39 {
			t.Errorf("capped bonus digit %d sprite %d, want 39", i, v.Sprite)
		}
	}
	if got := [5]int{g.spellHUD.History[0].Sprite, g.spellHUD.History[1].Sprite,
		g.spellHUD.History[2].Sprite, g.spellHUD.History[3].Sprite, g.spellHUD.History[4].Sprite}; got != [5]int{39, 39, 40, 30, 37} {
		t.Fatalf("history sprites %v", got)
	}
	before := *g.ctx.State
	g.ShowCaptureBonus(1230)
	for i, v := range g.captureDigits {
		if (v.Flags&anm.FlagDraw != 0) != (i >= 4) {
			t.Errorf("capture digit %d draw %v, want %v", i, v.Flags&anm.FlagDraw != 0, i >= 4)
		}
	}
	if *g.ctx.State != before {
		t.Fatal("capture popup changed game state")
	}
}
