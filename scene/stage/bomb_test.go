package stage

import (
	"testing"
	"th10/game"
)

func bombTestContext(character int) *Context {
	return NewContext(game.NewState(character, game.ShotA, game.DiffNormal))
}

// TestBombLifetime runs a bomb as the calc chain does (FUN_00405750, then
// the world): it lasts while its bullet.anm object does, 130 frames for
// Reimu's script 0x190 and 170 for Marisa's 0x197, rising from the player
// with the object's y scale as its radius.
func TestBombLifetime(t *testing.T) {
	for _, tc := range []struct {
		character int
		frames    int
		rise      float32
		radius    float32
	}{
		{game.CharReimu, 130, 1, 160},
		{game.CharMarisa, 170, 1.3, 112},
	} {
		c := bombTestContext(tc.character)
		y0 := float32(c.Player.Y - gameY)
		rank := c.State.Rank
		c.Bomb.Start()
		if c.State.Rank != rank-0x80 {
			t.Fatalf("character %d: rank %d after the bomb, want %d", tc.character, c.State.Rank, rank-0x80)
		}
		frames := 0
		var radius float32
		for c.Bomb.Update(); c.Bomb.Active(); c.Bomb.Update() {
			if frames == 0 && c.Bomb.Pos[1] != y0-tc.rise {
				t.Fatalf("character %d: bomb at y %v, want %v", tc.character, c.Bomb.Pos[1], y0-tc.rise)
			}
			frames++
			radius = c.Bomb.Radius
			c.UpdateWorld()
		}
		if frames != tc.frames || radius != tc.radius {
			t.Fatalf("character %d: %d frames, last radius %v; want %d, %v",
				tc.character, frames, radius, tc.frames, tc.radius)
		}
	}
}

// TestBombDamage is FUN_004059f0's table: inside the circle 5, 38 with a
// boss, 3 during a card unless the bomb went off a second into it (38);
// outside only a boss is hit, by 20/16 without a card, 26/23 by a strong bomb
// during one.
func TestBombDamage(t *testing.T) {
	for _, tc := range []struct {
		marisa, boss, spell, strong bool
		in, out                     int
	}{
		{false, false, false, false, 5, 0},
		{false, true, false, false, 38, 20},
		{true, true, false, false, 38, 16},
		{false, true, false, true, 38, 0},
		{false, true, true, false, 3, 0},
		{false, false, true, false, 3, 0},
		{false, true, true, true, 38, 26},
		{true, false, true, true, 38, 23},
	} {
		char := game.CharReimu
		if tc.marisa {
			char = game.CharMarisa
		}
		c := bombTestContext(char)
		c.Bomb.Start()
		c.Bomb.strong = tc.strong
		c.Boss0 = func() ([3]float32, bool) { return [3]float32{}, tc.boss }
		c.Spell = &SpellCard{Active: tc.spell}
		x, y := c.Bomb.Pos[0], c.Bomb.Pos[1]
		r := c.Bomb.Radius
		if got := c.Bomb.Damage(x, y-r+0.5); got != tc.in {
			t.Errorf("%+v: %d inside, want %d", tc, got, tc.in)
		}
		if got := c.Bomb.Damage(x, y-r); got != tc.out {
			t.Errorf("%+v: %d on the edge, want %d", tc, got, tc.out)
		}
	}
	c := bombTestContext(game.CharReimu)
	if c.Bomb.Damage(0, 0) != 0 {
		t.Fatal("an idle bomb dealt damage")
	}
}

// TestBombDuringCard: FUN_00405860 loses the capture a second into a card
// (FUN_00405500) and marks the bomb strong; the heavy cards run script 0x1b9,
// which rises at half speed.
func TestBombDuringCard(t *testing.T) {
	for _, tc := range []struct {
		id      int32
		timer   int32
		strong  bool
		capture bool
		rise    float32
		script  int
	}{
		{10, 59, false, true, 1, 0x190},
		{10, 60, true, false, 1, 0x190},
		{0x5d, 0, false, true, 0.5, 0x1b9},
		{0x6d, 120, true, false, 0.5, 0x1b9},
	} {
		c := bombTestContext(game.CharReimu)
		card := &SpellCard{Active: true, Capture: true, ID: tc.id, Bonus: 1000}
		card.Timer.Set(tc.timer)
		c.Spell = card
		y0 := float32(c.Player.Y - gameY)
		c.Bomb.Start()
		c.Bomb.Update()
		if c.Bomb.strong != tc.strong || card.Capture != tc.capture || (card.Bonus == 0) == tc.capture {
			t.Errorf("card %#x at %d: strong %v capture %v bonus %d", tc.id, tc.timer, c.Bomb.strong, card.Capture, card.Bonus)
		}
		if c.Bomb.vm.ScriptID != tc.script || c.Bomb.Pos[1] != y0-tc.rise {
			t.Errorf("card %#x: script %#x at y %v, want %#x at %v", tc.id, c.Bomb.vm.ScriptID, c.Bomb.Pos[1], tc.script, y0-tc.rise)
		}
	}
}
