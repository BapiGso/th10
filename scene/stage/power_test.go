package stage

import (
	"testing"
	"th10/audio"
	"th10/entity/bullet"
	"th10/entity/effect"
	"th10/entity/item"
	"th10/entity/player"
	"th10/game"
	"th10/input"
)

func powerTestStage() *Stage {
	state := game.NewState(game.CharReimu, game.ShotA, game.DiffNormal)
	return &Stage{ctx: &Context{
		State:  state,
		Player: player.New(state, game.FieldLeft, game.FieldTop, game.FieldRight, game.FieldBottom),
		Items:  item.NewPool(32), Bullets: bullet.NewPool(32), Effects: effect.NewPool(32),
		Audio: &audio.Manager{}, // zero-volume manager, no device required
	}}
}

func TestCollectedPowerValues(t *testing.T) {
	for _, tc := range []struct {
		kind         item.ItemType
		before, want int
	}{
		{item.PowerItem, 0, 5}, {item.BigPower, 0, 100},
		{item.PowerItem, 499, 500}, {item.BigPower, 450, 500},
	} {
		s := powerTestStage()
		s.ctx.State.Power = tc.before
		p := s.ctx.Player
		s.ctx.Items.Spawn(p.X, p.Y, tc.kind)
		s.checkCollisions()
		if s.ctx.State.Power != tc.want {
			t.Fatalf("item %d: %d -> %d, want %d", tc.kind, tc.before, s.ctx.State.Power, tc.want)
		}
	}
}

func TestBigPointIsNotBombStock(t *testing.T) {
	s := powerTestStage()
	p := s.ctx.Player
	s.ctx.State.Faith = 80000
	s.ctx.Items.Spawn(p.X, p.Y, item.BigPoint)
	s.checkCollisions()
	if s.ctx.State.Score != 80000 || s.ctx.State.HiScore != 80000 || s.ctx.State.Power != 0 {
		t.Fatalf("big point reward: %+v", s.ctx.State)
	}
}

func TestDeathDropsPowerButDeathbombDoesNot(t *testing.T) {
	for _, bomb := range []bool{false, true} {
		s := powerTestStage()
		p := s.ctx.Player
		p.GS.Power = 500
		p.X, p.Y = 100, 200
		p.Hit()
		var in input.State
		if bomb {
			in.Restore(1 << input.KeyBomb)
		}
		for i := 0; i < 9; i++ {
			s.updatePlayer(&in)
		}
		small, big := 0, 0
		s.ctx.Items.Each(func(it *item.Item) {
			if it.X != 100 || it.Y != 200 {
				t.Fatal("drop emitted at respawn position")
			}
			if it.Type == item.PowerItem {
				small++
			}
			if it.Type == item.BigPower {
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
