package game

import "testing"

func TestOriginalClearBonusBranches(t *testing.T) {
	for _, tc := range []struct {
		stage, difficulty int
		flags             uint32
		want              int64
	}{
		{6, DiffEasy, 0, 53555000},
		{6, DiffNormal, 0, 63555000},
		{6, DiffHard, 0, 91555000},
		{6, DiffLuna, 0, 109555000},
		{6, DiffExtra, 0, 117555000},
		{7, DiffExtra, 0, 117555000},
		{6, DiffNormal, RunFlagPractice, 0},
		{7, DiffExtra, RunFlagPractice, 0},
		{3, DiffNormal, 0, 0},
	} {
		s := NewState(0, 0, tc.difficulty)
		s.Stage, s.RunFlags, s.Faith, s.Life, s.Power = tc.stage, tc.flags, 55559, 2, 401
		s.AwardClearBonus()
		if s.Score != tc.want || s.Faith != 55559 || s.Power != 401 || s.Life != 2 {
			t.Fatalf("stage %d difficulty %d flags %#x: score %d, want %d", tc.stage, tc.difficulty, tc.flags, s.Score, tc.want)
		}
	}
	s := NewState(0, 0, DiffLuna)
	s.Stage, s.Score, s.Power = 6, 9999999980, 500
	s.AwardClearBonus()
	if s.Score != 9999999990 {
		t.Fatal("clear call sequence did not use FUN_00409d90's per-call cap")
	}
}
