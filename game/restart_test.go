package game

import "testing"

func TestRestartStageOriginalFreshState(t *testing.T) {
	s := NewState(0, 0, DiffNormal)
	s.StartStage, s.Stage, s.Score, s.Life, s.Power, s.Frame = 1, 3, 99999, -1, 40, 300
	s.RestartStage()
	if s.Stage != 3 || s.StartStage != 1 || s.Score != 0 || s.Frame != 0 || s.Faith != 50000 || s.Power != 400 || s.Life != 2 || s.Rank != -512 || s.ReplayState90 != 1 || !s.Continued {
		t.Fatalf("restart differs: %+v", s)
	}
	for range 20 {
		s.RestartStage()
	}
	if s.ReplayState90 != 9 {
		t.Fatal("continue count not capped at nine")
	}
	s = NewState(0, 0, DiffNormal)
	s.StartStage, s.Stage = 1, 1
	s.RestartStage()
	if s.ReplayState90 != 0 || s.Power != 0 || s.Rank != 0 || s.Continued {
		t.Fatal("initial-stage retry counted as continue")
	}
}
