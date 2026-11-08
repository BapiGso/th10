package stage

import (
	"testing"

	"th10/game"
	"th10/input"
)

func TestScoreControllerStageTimingAndOneUpNotice(t *testing.T) {
	state := game.NewState(0, 0, game.DiffNormal)
	state.Stage, state.Rank, state.ReplayState90 = 1, 77, 2
	s := NewHeadless(state, &recordScript{})
	sound := &panSoundRecord{}
	s.ctx.Audio = sound
	state.AddPoints(20000000)
	if state.HiScore != 0 {
		t.Fatal("AddPoints updated staged high score before the controller")
	}
	var in input.State
	for range 31 {
		s.UpdateHeadless(&in)
		if state.Life != 2 || state.ReplayState9C != 0 {
			t.Fatal("extend used real score instead of rolled GUI score")
		}
	}
	s.UpdateHeadless(&in)
	if state.Life != 3 || state.ReplayState9C != 1 || state.Rank != 77 {
		t.Fatal("32nd rolling frame did not grant the extend")
	}
	if len(sound.ids) != 1 || sound.ids[0] != 0x2c || s.ctx.GUI.notice[1] == nil || s.ctx.GUI.notice[1].ScriptID != 0x4b || s.ctx.GUI.lastLife != 3 {
		t.Fatal("one-up sound/VM/resource write did not happen in the same frame")
	}
	if state.HiScore != 20000000 || state.HiScoreContinues != 2 || s.ctx.GUI.hiScore[0].Sprite != 10 || s.ctx.GUI.score[0].Sprite != 10 {
		t.Fatal("controller high score/continuation digits differ")
	}
	s.UpdateHeadless(&in)
	if len(sound.ids) != 1 {
		t.Fatal("GUI's next Calc duplicated the controller one-up")
	}
}

func TestNumbersAloneNeverGrantsExtend(t *testing.T) {
	state := game.NewState(0, 0, game.DiffNormal)
	state.Stage = 1
	s := NewHeadless(state, &recordScript{})
	state.AddPoints(20000000)
	for range 64 {
		s.ctx.GUI.Numbers()
	}
	if state.Life != 2 || state.HiScore != 0 || state.ReplayState9C != 0 {
		t.Fatal("GUI number rendering mutated gameplay")
	}
	s.updateScoreController()
	if state.Life != 3 || state.HiScore != 20000000 {
		t.Fatal("stage controller did not apply the completed roll")
	}
}
