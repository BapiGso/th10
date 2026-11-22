//go:build !th10nostory

// Dialogue/ending behaviour; th10nostory removes it (gamedata.Story).

package stage

import (
	"testing"

	"th10/scene/dialog"
)

func TestMSGBossSectionLifecycle(t *testing.T) {
	g := newTestGUI()
	c := g.ctx
	c.GUI = nil // exercise MSG logic without requiring a stage-logo presentation
	c.msgs = &dialog.StageMSG{Entries: make([][]dialog.Instr, 24)}
	c.msgs.Entries[23] = []dialog.Instr{{Op: 10, Arg: 2}, {Op: 0}}
	c.SetBossSection(7)
	c.OpenMSG(24) // missing entry must not change the original global
	if c.BossSection != 7 || c.MSG() != nil {
		t.Fatal("missing MSG changed section")
	}
	c.OpenMSG(23)
	if c.BossSection != 24 || c.MSG() == nil || c.MSG().ID != 23 {
		t.Fatal("MSG open did not write entry id+1")
	}
	c.UpdateMSG(false, false)
	if c.BossSection != 24 || c.MSG() == nil {
		t.Fatal("waiting MSG lost its section")
	}
	c.UpdateMSG(false, false)
	if c.BossSection != 0 || c.MSG() != nil {
		t.Fatal("MSG op 0 did not reset DAT_00474c84")
	}
	c.SetBossSection(0x18)
	c.UpdateMSG(false, false)
	if c.BossSection != 0x18 {
		t.Fatal("an absent MSG reset an ECL 344 section")
	}
}

func TestGUIBossNamesUseOriginalSection(t *testing.T) {
	for _, tc := range []struct{ stage, mid, boss int }{
		{1, 0x85, 0x86}, {2, -1, 0x87}, {3, -1, 0x88}, {4, 0x89, 0x8a},
		{5, 0x8b, 0x8b}, {6, 0x8c, 0x8c}, {7, 0x8c, 0x8d},
	} {
		for _, section := range []int32{23, 24} {
			g, script := guiBossFixture()
			g.ctx.State.Stage = tc.stage
			g.SetBossSection(section)
			script.info.Midboss = section >= 24 // deliberately contradict the old adapter
			g.Calc()
			want := tc.mid
			if section >= 24 {
				want = tc.boss
			}
			if want < 0 {
				if g.bossName != nil {
					t.Fatalf("stage %d section %d unexpectedly drew a name", tc.stage, section)
				}
			} else if g.bossName == nil || g.bossName.ScriptID != want {
				t.Fatalf("stage %d section %d name %+v, want %#x", tc.stage, section, g.bossName, want)
			}
		}
	}
}
