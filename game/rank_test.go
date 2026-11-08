package game

import "testing"

func TestResetRank(t *testing.T) {
	s := NewState(CharReimu, ShotA, DiffNormal)
	if s.Rank != 0 {
		t.Fatalf("new-run rank = %d, want 0", s.Rank)
	}
	s.Rank = 123
	s.ResetRank(true)
	if s.Rank != -512 {
		t.Fatalf("boss-practice rank = %d, want -512", s.Rank)
	}
	s.ResetRank(false)
	if s.Rank != 0 {
		t.Fatalf("reset normal-run rank = %d, want 0", s.Rank)
	}
}

func TestAddRankClamp(t *testing.T) {
	for _, tc := range []struct{ start, delta, want int }{
		{0, 12, 12}, {1000, 24, 1024}, {1024, 256, 1024},
		{100, -1024, -924}, {-1000, -128, -1024}, {-2000, 1, -1024},
	} {
		s := &GameState{Rank: tc.start}
		s.AddRank(tc.delta)
		if s.Rank != tc.want {
			t.Errorf("rank %d %+d = %d, want %d", tc.start, tc.delta, s.Rank, tc.want)
		}
	}
}
