package game

import "testing"

// The values below follow the asm of FUN_00412ff0, FUN_0044bf40 and
// FUN_00418a00, not the Go code. The game stores faith / 10; GameState keeps
// it times ten, so 60000 here is stored 6000.

func TestFaithHoldCountsDownBeforeDecay(t *testing.T) {
	s := NewState(CharReimu, ShotA, DiffNormal)
	s.BeginStageFaith() // FUN_00417870: DAT_00474c50 = 9
	s.Faith = 60000
	s.AddFaithHold(3)
	for i, want := range []int{2, 1, 0} {
		s.TickFaith() // FUN_0044bf40(-1.0): the timer counts down, no decay
		if s.FaithHold != want || s.Faith != 60000 {
			t.Fatalf("hold tick %d: timer %d faith %d, want %d and 60000", i+1, s.FaithHold, s.Faith, want)
		}
	}
	s.TickFaith() // timer is 0: stored 6000 - 9 = 5991, rate rises to 18
	if s.Faith != 59910 || s.FaithRate != 18 {
		t.Fatalf("first decay: faith %d rate %d, want 59910 and 18", s.Faith, s.FaithRate)
	}
	s.TickFaith() // stored 5991 - 18 = 5973
	if s.Faith != 59730 {
		t.Fatalf("second decay: faith %d, want 59730", s.Faith)
	}
}

func TestFaithHoldCap(t *testing.T) {
	s := NewState(CharReimu, ShotA, DiffNormal)
	s.AddFaithHold(100)
	s.AddFaithHold(60) // 160, held at the cap 130
	if s.FaithHold != 130 {
		t.Fatalf("timer %d after 160 frames, want 130", s.FaithHold)
	}
	s.AddFaithHold(3) // FUN_00412ff0 adds only while timer < 0x82
	if s.FaithHold != 130 {
		t.Fatalf("timer %d after an add at the cap, want 130", s.FaithHold)
	}

	s = NewState(CharReimu, ShotA, DiffNormal)
	s.AddFaithHold(129)
	s.AddFaithHold(1) // exactly 130 is not over the cap
	if s.FaithHold != 130 {
		t.Fatalf("timer %d, want 130", s.FaithHold)
	}
}

func TestRestoreFaithHoldThenAdd(t *testing.T) {
	s := NewState(CharReimu, ShotA, DiffNormal)
	s.RestoreFaithHold(47) // FUN_0042a6a0: timer and accumulator from the chapter
	s.AddFaithHold(100)
	if s.FaithHold != 130 {
		t.Fatalf("timer %d after restore 47 and +100, want 130", s.FaithHold)
	}
}

func TestFaithDecayFloorsAtStored5000(t *testing.T) {
	s := NewState(CharReimu, ShotA, DiffNormal)
	s.FaithRate = 18
	s.Faith = 50100 // stored 5010: 5010 - 18 = 4992, held at 5000
	s.TickFaith()
	if s.Faith != 50000 {
		t.Fatalf("faith %d, want the floor 50000", s.Faith)
	}
	s.TickFaith() // stored 5000 is not above the floor: nothing happens
	if s.Faith != 50000 || s.FaithRate != 18 {
		t.Fatalf("faith %d rate %d at the floor, want 50000 and 18", s.Faith, s.FaithRate)
	}
}

// An enemy frame sets the rate to 1 (FUN_0040dc80 0x40e26b), so the decay
// step that follows loses 1, and the rate goes back to 18.
func TestDecayRateOneStep(t *testing.T) {
	s := NewState(CharReimu, ShotA, DiffNormal)
	s.BeginStageFaith()
	s.Faith = 60000
	s.FaithRate = 1
	s.TickFaith()
	if s.Faith != 59990 || s.FaithRate != 18 {
		t.Fatalf("faith %d rate %d after a rate-1 step, want 59990 and 18", s.Faith, s.FaithRate)
	}
}
