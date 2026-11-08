package stage

import (
	"slices"
	"testing"

	"th10/anm"
	"th10/game"
	"th10/input"
)

func TestBombPowerGaugeBeforeAndAfterCallerCost(t *testing.T) {
	c := bombTestContext(game.CharReimu)
	c.State.Power, c.State.Faith = 245, 100000
	c.Player.RefreshOptions()
	c.GUI = &GUI{}
	c.GUI.power[1].Sprite = 19
	c.Spell = &SpellCard{Active: true, Capture: true, Bonus: 1000}
	c.Spell.Timer.Set(60)
	refresh := c.Player.PowerChanged
	calls := 0
	c.Player.PowerChanged = func() {
		calls++
		got := [4]int{c.GUI.power[0].Sprite, c.GUI.power[1].Sprite, c.GUI.power[2].Sprite, c.GUI.power[3].Sprite}
		if got != [4]int{10, 19, 12, 13} || c.State.Power != 145 || c.State.Faith != 100000 ||
			c.Spell.Capture || c.Spell.Bonus != 0 {
			t.Fatal("start gauge/cost/card/faith ordering differs from FUN_00405860/FUN_00425730")
		}
		refresh()
	}
	var in input.State
	in.Restore(1 << input.KeyBomb)
	c.Player.UpdateInput(&in)
	got := [4]int{c.GUI.power[0].Sprite, c.GUI.power[1].Sprite, c.GUI.power[2].Sprite, c.GUI.power[3].Sprite}
	if calls != 1 || got != [4]int{9, 19, 12, 13} || c.State.Power != 145 || c.State.Faith != 97000 {
		t.Fatalf("after bomb: calls=%d sprites=%v power=%d faith=%d", calls, got, c.State.Power, c.State.Faith)
	}
}

func TestBombVisualsStayInOriginalANMScripts(t *testing.T) {
	for _, tc := range []struct {
		character int
		cardID    int32
		active    bool
		main      int
		children  []int
		timed     int
		rise      float32
	}{
		{game.CharReimu, 0, false, 0x190, []int{0x18a, 0x18b, 0x18c}, 0x18d, 1},
		{game.CharMarisa, 0, false, 0x197, []int{0x191, 0x192, 0x193}, 0x194, 1.3},
		{game.CharReimu, 0x5d, true, 0x1b9, []int{0x1b3, 0x1b4, 0x1b5}, 0x1b6, 0.5},
		{game.CharMarisa, 0x6d, false, 0x1b9, []int{0x1b3, 0x1b4, 0x1b5}, 0x1b6, 1.3},
	} {
		c := bombTestContext(tc.character)
		c.Player.SetFixed(2525, 34575)
		c.Spell = &SpellCard{ID: tc.cardID, Active: tc.active}
		var born []int
		c.Anm.Spawn = func(v *anm.VM) {
			born = append(born, v.ScriptID)
			c.World.Add(v)
		}
		c.Bomb.Start()
		main := c.Bomb.vm
		if main.ScriptID != tc.main || main.File != c.BulletEnv.File || main.Flags&anm.FlagAltRNG != 0 || !slices.Equal(born, tc.children) {
			t.Fatalf("bomb %#x: initial children=%v", tc.main, born)
		}
		y := c.Bomb.Pos[1]
		c.Bomb.Update()
		if c.Bomb.Pos[1] != y-tc.rise {
			t.Fatal("visual wiring changed bomb rise")
		}
		for _, id := range tc.children {
			v := c.World.FindChild(main, id)
			if v == nil || v.Offset != main.Offset {
				t.Fatal("bomb child did not follow FUN_004492f0")
			}
		}
		for range 12 {
			c.UpdateWorld()
			c.Bomb.Update()
		}
		if !slices.Contains(born, tc.timed) {
			t.Fatal("ANM did not emit its timed child; no manual VM approximation is allowed")
		}
	}
}
