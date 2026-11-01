package stage

import (
	"testing"
	"th10/audio"
	"th10/entity/item"
	"th10/game"
	"th10/input"
	"th10/zun"
)

func powerTestStage() *Stage {
	state := game.NewState(game.CharReimu, game.ShotA, game.DiffNormal)
	ctx := NewContext(state)
	ctx.Audio = &audio.Manager{} // zero-volume manager, no device required
	return &Stage{ctx: ctx}
}

// dropOnPlayer drops a still item of kind on the player, in its collect box.
func dropOnPlayer(s *Stage, kind int32) {
	p := s.ctx.Player
	s.ctx.SpawnItem(float32(p.X-gameX), float32(p.Y-gameY), kind, -zun.HalfPi, 0)
}

func TestCollectedPowerValues(t *testing.T) {
	for _, tc := range []struct {
		kind         int32
		before, want int
	}{
		{item.KindPower, 0, 5}, {item.KindBigPower, 0, 100},
		{item.KindPower, 495, 500}, {item.KindBigPower, 450, 500},
		{item.KindBossPower, 0, 5}, {item.KindBossBigPower, 0, 100},
	} {
		s := powerTestStage()
		s.ctx.State.Power = tc.before
		dropOnPlayer(s, tc.kind)
		s.ctx.UpdateItems(false)
		if s.ctx.State.Power != tc.want || s.ctx.Items.ActiveCount() != 0 {
			t.Fatalf("item %d: %d -> %d, want %d", tc.kind, tc.before, s.ctx.State.Power, tc.want)
		}
	}
}

// TestPointItemValue: a point item (FUN_0041afd0 kind 2) is worth the faith
// times ten above y 144, and less the lower the player is below it.
func TestPointItemValue(t *testing.T) {
	for _, tc := range []struct {
		y     float64 // the player's game y
		score int64
		rank  int
	}{
		{130, 80000, 8},
		// Half of 80000, less 1.00000012 of 8000/2-5000, truncated.
		{448, 41000, 1},
	} {
		s := powerTestStage()
		s.ctx.State.Faith = 80000
		s.ctx.Player.SetPos(gameX, gameY+tc.y)
		dropOnPlayer(s, item.KindPoint)
		s.ctx.UpdateItems(false)
		st := s.ctx.State
		if st.Score != tc.score || st.HiScore != tc.score || st.Rank != tc.rank || st.Power != 0 {
			t.Fatalf("point item at y %v: score %d rank %d, want %d and %d", tc.y, st.Score, st.Rank, tc.score, tc.rank)
		}
	}
}

// TestCancelItemRisesThenCollects: a cancel item waits out its delay (0 for
// the first slot), appears, then is collected for 10 faith and 10 points.
func TestCancelItemRisesThenCollects(t *testing.T) {
	s := powerTestStage()
	faith := s.ctx.State.Faith
	s.ctx.Player.SetPos(gameX, gameY+400)
	s.ctx.cancelItem(0, 400)
	s.ctx.UpdateItems(false)
	if n := s.ctx.Items.ActiveCount(); n != 1 || s.ctx.State.Score != 0 {
		t.Fatalf("collected while appearing: %d items, score %d", n, s.ctx.State.Score)
	}
	s.ctx.UpdateItems(false)
	if n := s.ctx.Items.ActiveCount(); n != 0 || s.ctx.State.Score != 10 || s.ctx.State.Faith != faith+10 {
		t.Fatalf("%d items, score %d, faith %d", n, s.ctx.State.Score, s.ctx.State.Faith-faith)
	}
}

func TestDeathDropsPowerButDeathbombDoesNot(t *testing.T) {
	for _, bomb := range []bool{false, true} {
		s := powerTestStage()
		p := s.ctx.Player
		p.GS.Power = 500
		p.SetPos(100, 200)
		p.Hit()
		var in input.State
		if bomb {
			in.Restore(1 << input.KeyBomb)
		}
		for i := 0; i < 12; i++ { // 9 frames to the death, power at its 3rd
			s.updatePlayer(&in)
		}
		small, big := 0, 0
		s.ctx.Items.Each(func(it *item.Item) {
			if it.Pos[0] != 100-gameX || it.Pos[1] != 200-gameY {
				t.Fatal("drop emitted at respawn position")
			}
			if it.Kind == item.KindPower {
				small++
			}
			if it.Kind == item.KindBigPower {
				big++
			}
		})
		if bomb {
			if small+big != 0 {
				t.Fatal("deathbomb produced death drops")
			}
		} else if small != 4 || big != 3 {
			t.Fatalf("drops small=%d big=%d", small, big)
		}
	}
}
