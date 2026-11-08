package player

import (
	"slices"
	"testing"

	"th10/game"
	"th10/input"
)

func TestBombPowerGaugeHookAfterRefreshBeforeFaithLoss(t *testing.T) {
	p := testPlayer(game.CharReimu)
	p.GS.Power, p.GS.Faith = 245, 100000
	p.RefreshOptions()
	calls := 0
	p.PowerChanged = func() {
		calls++
		if !p.Bomb.Active() || p.GS.Power != 145 || len(p.Options()) != 1 || p.GS.Faith != 100000 {
			t.Fatal("power gauge was not refreshed after subtraction/options and before faith loss")
		}
	}
	var in input.State
	in.Restore(1 << input.KeyBomb)
	p.UpdateInput(&in)
	if calls != 1 || p.GS.Faith != 97000 {
		t.Fatal("bomb power hook count or subsequent faith loss differs")
	}
}

func TestDeathPowerGaugeHookBeforeDrops(t *testing.T) {
	p := testPlayer(game.CharReimu)
	p.State, p.GS.Power = StateDying, 400
	p.StateTimer.Set(3)
	var order []string
	p.PowerChanged = func() {
		if p.GS.Power != 80 {
			t.Fatal("death gauge read the old power")
		}
		order = append(order, "gauge")
	}
	p.SpawnItem = func(_, _ float32, _ int32, _, _ float32) {
		order = append(order, "drop")
	}
	var in input.State
	p.UpdateInput(&in)
	if !slices.Equal(order, []string{"gauge", "drop", "drop", "drop", "drop", "drop", "drop", "drop"}) {
		t.Fatalf("death visual order %v", order)
	}
}
