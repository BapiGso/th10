package stage

import (
	"testing"

	"th10/anm"
	"th10/entity/player"
)

func TestSpellGUIBombAndHitFailureWrites(t *testing.T) {
	for _, hit := range []bool{false, true} {
		for _, timer := range []int32{59, 60} {
			g := newTestGUI()
			c := g.ctx
			c.Spell = &SpellCard{}
			c.Spell.Declare(c.State, 2, "card", 2700)
			g.BeginSpellPresentation(c.Spell.Name)
			c.Spell.Timer.Set(timer)
			c.Player.State = player.StateNormal
			c.Player.Invinc.Set(0)
			if hit {
				c.hitPlayer()
			} else {
				c.Bomb.Start()
			}
			for i, v := range c.Spell.HUD.Bonus {
				if want := timer < 60 || i == 2; (v.Flags&anm.FlagDraw != 0) != want {
					t.Errorf("hit %v timer %d VM %d flags %#x, want draw %v", hit, timer, i, v.Flags, want)
				}
			}
			if c.Spell.Capture != (timer < 60) {
				t.Errorf("hit %v timer %d capture %v", hit, timer, c.Spell.Capture)
			}
		}
	}
}

func TestSpellGUIFailureDoesNotChangeOtherFlagBits(t *testing.T) {
	h := &SpellHUD{}
	for i := range h.Bonus {
		h.Bonus[i].Flags = 0xffffffff
	}
	h.loseCapture()
	for i, v := range h.Bonus {
		want := uint32(0xfffffffd)
		if i == 2 {
			want = 0xffffffff
		}
		if v.Flags != want {
			t.Errorf("VM %d flags %#x, want %#x", i, v.Flags, want)
		}
	}
}
