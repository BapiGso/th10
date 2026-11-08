package game

import "testing"

func TestOriginalScoreExtendTablesAndGates(t *testing.T) {
	for _, tc := range []struct {
		difficulty int
		threshold  int32
	}{
		{DiffEasy, 2000000}, {DiffNormal, 2000000}, {DiffHard, 2000000},
		{DiffLuna, 2000000}, {DiffExtra, 3000000},
	} {
		s := NewState(0, 0, tc.difficulty)
		s.Rank, s.ReplayState90 = 123, 2
		if s.UpdateScoreController(tc.threshold-1, false) || s.Life != 2 || s.ReplayState9C != 0 {
			t.Fatal("extend happened before its original threshold")
		}
		if !s.UpdateScoreController(tc.threshold, false) || s.Life != 3 || s.ReplayState9C != 1 || s.Rank != 123 {
			t.Fatal("extend did not update only life and cursor")
		}
		if s.HiScoreContinues != 2 || s.RunFlags&RunFlagHighScore == 0 || s.HiScore != int64(tc.threshold)*10 {
			t.Fatal("high-score resource fields differ")
		}
		s.Life, s.ReplayState9C = 9, 0
		if s.UpdateScoreController(tc.threshold, false) || s.Life != 9 || s.ReplayState9C != 1 {
			t.Fatal("capped extend did not advance silently")
		}
		s.ReplayState9C = 0
		s.UpdateScoreController(tc.threshold, true)
		if s.ReplayState9C != 0 {
			t.Fatal("GUI final-clear bit did not inhibit extends")
		}
	}
	s := NewState(0, 0, DiffNormal)
	for cursor := int32(1); cursor <= 4; cursor++ {
		s.UpdateScoreController(999999999, false)
		if s.ReplayState9C != cursor {
			t.Fatal("controller processed more than one threshold in one frame")
		}
	}
	s.UpdateScoreController(999999999, false)
	if s.ReplayState9C != 4 {
		t.Fatal("unreachable sentinel became a repeating extend")
	}
	for _, cursor := range []int32{-1, 100, 0x7fffffff} {
		s.ReplayState9C = cursor
		s.UpdateScoreController(999999999, false)
		if s.ReplayState9C != cursor {
			t.Fatal("invalid replay cursor was not guarded")
		}
	}
}

func TestRunHighScoreUsesSelectedRanking(t *testing.T) {
	g := &Game{Save: DefaultSaveData(), Scores: NewMemoryScoreStore()}
	g.Save.HighScore = 90000000
	g.Save.Rankings[0][DiffNormal][0] = ScoreEntry{Score: 100, Continues: 3}
	s := g.NewState(0, 0, DiffNormal)
	s.Stage = 1
	g.BeginRun(s)
	if s.HiScore != 1000 || s.HiScoreContinues != 3 {
		t.Fatal("new run used aggregate high score instead of its ranking row")
	}
	s.BindScoreController()
	s.AddPoints(3200)
	if s.HiScore != 1000 {
		t.Fatal("staged AddPoints bypassed the rolling-score controller")
	}
	s.UpdateScoreController(101, false)
	if s.HiScore != 1010 || s.HiScoreContinues != 0 {
		t.Fatal("controller high-score update did not use shown score/current continues")
	}
}
