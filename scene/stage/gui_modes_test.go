package stage

import (
	"testing"

	"th10/anm"
	"th10/game"
)

func TestGUINativeModeGates(t *testing.T) {
	for _, tc := range []struct {
		name      string
		flags     uint32
		mode      int32
		fresh     bool
		continues int32
		seed      bool
		hint      bool
		title     int
	}{
		{"seeded-live", 0, 0, true, 0, true, true, 0},
		{"practice-not-demo", game.RunFlagPractice, 0, true, 0, true, true, 0},
		{"replay-without-seed", 0, 1, true, 0, false, false, 0},
		{"continued", 0, 0, false, 1, false, false, 0},
		{"demo", game.RunFlagDemo, 1, false, 0, true, false, 0x71},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s := game.NewState(0, 0, game.DiffNormal)
			s.Stage, s.SingleStage, s.HasSeed = 1, true, tc.seed
			s.RunFlags, s.StageMode, s.FreshRun, s.ReplayState90 = tc.flags, tc.mode, tc.fresh, tc.continues
			c := NewContext(s)
			g := NewGUI(c)
			if g.titles[0] == nil || g.titles[0].ScriptID != tc.title {
				t.Fatal("title used SingleStage instead of DAT_00474ca0 bit 0x20")
			}
			if (g.practice != nil) != tc.fresh {
				t.Fatal("fresh-run indicator used a practice policy")
			}
			c.World.Run(c.Anm)
			hint := false
			c.World.Each(15, func(v *anm.VM) {
				if v.File == g.front && v.ScriptID == 0x4f {
					hint = true
				}
			})
			if hint != tc.hint {
				t.Fatalf("hint = %v, want %v", hint, tc.hint)
			}
		})
	}
}
