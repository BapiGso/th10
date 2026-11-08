package game

import "testing"

func TestRetryInitialStageIsNotLaterStageContinue(t *testing.T) {
	s := NewState(1, 2, 3)
	s.Stage, s.StartStage, s.ReplayState90, s.RunFlags = 5, 1, 3, RunFlagContinued
	s.Score, s.Power, s.Paused = 123450, 500, true
	s.RetryInitialStage()
	if s.Stage != 1 || s.StartStage != 1 || s.ReplayState90 != 0 || s.Continued || s.RunFlags&RunFlagContinued != 0 || s.Score != 0 || s.Power != 0 || s.Paused || s.Rank != 0 || s.Life != 2 || !s.FreshRun {
		t.Fatalf("retry fresh state %+v", s)
	}
	s.Practice(4)
	s.PracticeLives = 4
	s.RetryInitialStage()
	if s.Stage != 4 || s.Life != 3 || s.Power != 400 || s.RunFlags&RunFlagPractice == 0 {
		t.Fatal("practice retry lost native resources")
	}
}
