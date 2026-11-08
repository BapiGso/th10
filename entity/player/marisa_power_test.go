package player

import (
	"testing"

	"th10/game"
	"th10/sht"
)

func TestMarisaPowerCeilingAndShotThresholds(t *testing.T) {
	for shot := range 3 {
		p := New(game.NewState(game.CharMarisa, shot, game.DiffNormal), game.FieldLeft, game.FieldTop, game.FieldRight, game.FieldBottom)
		if p.GS.MaxPower != 500 {
			t.Fatalf("Marisa shot %d ceiling=%d, want 500", shot, p.GS.MaxPower)
		}
		// FUN_00418930's signed raw Power cap is 0x64; FUN_00428160
		// chooses tiers at raw 20/40/60/80, clamping raw 100 back to tier 4.
		for raw := 0; raw <= 100; raw++ {
			p.GS.Power = raw * 5
			p.RefreshOptions()
			want := min(raw/20, 4)
			if sht.PowerLevel(p.GS.Power) != want || p.optionCount != want {
				t.Fatalf("Marisa shot %d raw %d tier/options=%d/%d, want %d", shot, raw, sht.PowerLevel(p.GS.Power), p.optionCount, want)
			}
			for _, focused := range []bool{false, true} {
				group := want
				if focused {
					group += 5
				}
				got := p.shotFile.Shots(p.GS.Power, focused)
				if len(got) != len(p.shotFile.Groups[group]) || &got[0] != &p.shotFile.Groups[group][0] {
					t.Fatalf("Marisa shot %d raw %d focus %v selected the wrong SHT group", shot, raw, focused)
				}
			}
		}
		p.GS.Power = 495
		p.GS.GainPower(1)
		p.GS.GainPower(20)
		if p.GS.Power != 500 {
			t.Fatalf("Marisa shot %d saturated pickup=%d, want 500", shot, p.GS.Power)
		}
		p.AddPower(100)
		if p.GS.Power != 500 {
			t.Fatalf("Marisa shot %d direct pickup ceiling=%d, want 500", shot, p.GS.Power)
		}
	}
}
