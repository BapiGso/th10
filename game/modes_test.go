package game

import "testing"

func TestNativePracticeAndRestartModes(t *testing.T) {
	s := NewState(0, 0, DiffHard)
	s.Practice(3)
	if s.RunFlags != 0x10 || s.StageMode != 0 || s.ReplayMode != 0 || s.Stage != 3 || s.StartStage != 3 || s.Life != 9 || s.Power != 400 || s.Rank != 0 {
		t.Fatalf("practice fields differ: %+v", s)
	}
	s.PracticeLives = 4
	s.RestartStage()
	if s.Life != 3 || s.Rank != 0 || s.RunFlags&RunFlagPractice == 0 || !s.FreshRun {
		t.Fatalf("practice retry fields differ: %+v", s)
	}
	s.RunFlags |= RunFlagContinued | RunFlagHighScore
	s.RestartStage()
	if s.Rank != -512 || s.RunFlags&RunFlagHighScore != 0 || s.StageMode != 0 || s.ReplayMode != 0 {
		t.Fatalf("bit 8 restart fields differ: %+v", s)
	}
	s.BeginStageTransition()
	if s.FreshRun || s.RunFlags != RunFlagPractice {
		t.Fatalf("transition changed practice or retained transient bits: %+v", s)
	}
}

func TestSpellHistoryNonPlaybackModeValues(t *testing.T) {
	for _, mode := range []int32{-1, 0, 1, 2, 3} {
		s := NewState(0, 1, DiffEasy)
		s.ReplayMode, s.RunFlags = mode, RunFlagPractice
		s.RecordSpellDeclaration(0, "card")
		s.RecordSpellCapture(0)
		captures, attempts := s.SpellCounts(0)
		want := int32(1)
		if mode == 1 {
			want = 0
		}
		if captures != want || attempts != want {
			t.Fatalf("mode %d has %d/%d, want %d/%d", mode, captures, attempts, want, want)
		}
	}
}
