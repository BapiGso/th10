package stage

import (
	"testing"

	"th10/game"
	"th10/input"
	"th10/supervisor"
	"th10/supervisor/supervisortest"
	"th10/zun"
)

func TestScreenEffectDrawRangesInterleaveStaticChain(t *testing.T) {
	state := game.NewState(0, 0, game.DiffNormal)
	state.Stage = 1
	s := NewHeadless(state, &recordScript{})
	s.UpdateHeadless(&input.State{})
	var ranges [][2]int32
	s.ctx.DrawScreenEffects = func(_ *supervisor.Supervisor, first, last int32) {
		ranges = append(ranges, [2]int32{first, last})
	}
	s.runChain(supervisor.New(&supervisortest.Rec{}, nil), false)
	if len(ranges) != len(drawChain)+1 {
		t.Fatal("dynamic draw ranges missing")
	}
	first := int32(-1 << 31)
	foundClear := false
	for i, n := range drawChain {
		if ranges[i] != [2]int32{first, int32(n.prio)} {
			t.Fatalf("range %d = %v", i, ranges[i])
		}
		if n.prio == 0x32 {
			foundClear = first == 0x31
		}
		first = int32(n.prio) + 1
	}
	if !foundClear || ranges[len(drawChain)] != [2]int32{first, 1<<31 - 1} {
		t.Fatal("MSG draw 0x31 did not run immediately before Finish")
	}
}

func TestLiveSceneTransferCarriesRNGAndReleasesEffects(t *testing.T) {
	s, g := clearControllerFixture(t, 3, game.DiffNormal, 0, false)
	s.ctx.RNG.State, s.ctx.RNG.Count = 1, 0
	s.ctx.RNG.Dup()
	s.ctx.RNG.Dup()
	control := &zun.RNG{State: s.ctx.RNG.State, Count: 4}
	cleared, called := false, false
	s.ctx.ClearScreenEffects = func() { cleared = true }
	s.NextScene = func(_ *game.Game, state *game.GameState) game.Scene {
		called = true
		if !cleared || !s.ctx.StageExiting || state.StageRNG() != s.ctx.RNG {
			t.Fatal("scene callback preceded effect teardown/RNG transfer")
		}
		state.Stage, state.Frame = 4, 0
		state.BeginStageTransition()
		next := NewHeadless(state, &recordScript{})
		if next.ctx.RNG != s.ctx.RNG || next.ctx.RNG.Count != control.Count || next.ctx.RNG.Raw() != control.Raw() {
			t.Fatal("actual scene-0x0b adapter re-seeded the next stage")
		}
		return next
	}
	if err := s.Update(g); err != nil {
		t.Fatal(err)
	}
	if !called {
		t.Fatal("scene-0x0b transfer did not execute")
	}
}
