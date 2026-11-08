package player

import (
	"testing"

	"th10/game"
	"th10/input"
)

func TestRespawnAdvancesOptionHistoryWithoutMovement(t *testing.T) {
	for _, power := range []int{0, 100, 400} {
		p := New(game.NewState(game.CharMarisa, game.ShotA, game.DiffNormal), game.FieldLeft, game.FieldTop, game.FieldRight, game.FieldBottom)
		p.GS.Power = power
		p.RefreshOptions()
		p.State = StateRespawn
		p.StateTimer.Set(10)
		p.focus = true // state 0 ignores the focus/movement gate of state 1
		for i := range p.positionHistory {
			p.positionHistory[i] = [2]int32{int32(i * 37), int32(40000 + i*53)}
		}
		before := p.positionHistory
		var in input.State
		p.UpdateInput(&in)
		// FUN_00425730, 0x4257b3-0x425833: repeat the old terminal sample,
		// shift through 8*n, then write the newly computed respawn position.
		end := p.optionCount * 8
		if p.positionHistory[0] != [2]int32{0, 48000 - 10*8000/60} {
			t.Fatalf("power %d head=%v, want this tick's respawn position", power, p.positionHistory[0])
		}
		for i := 1; i < len(p.positionHistory); i++ {
			want := before[min(i-1, end)]
			if p.positionHistory[i] != want {
				t.Fatalf("power %d history[%d]=%v, want %v", power, i, p.positionHistory[i], want)
			}
		}
	}
}
