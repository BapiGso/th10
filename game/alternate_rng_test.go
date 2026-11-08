package game

import "testing"

func TestAlternateRNGDeterministicSeed(t *testing.T) {
	appA, appB := &Game{}, &Game{}
	a := appA.NewState(CharReimu, ShotA, DiffNormal)
	b := appB.NewState(CharMarisa, ShotC, DiffExtra)
	a.RNGSeed, a.HasSeed = 0xffff, true
	if a.AlternateRNG().State != 0 || a.AlternateRNG().Count != 0 {
		t.Fatalf("alternate starts at %+v, want state/count 0/0", a.AlternateRNG())
	}
	if a.AlternateRNG() == b.AlternateRNG() {
		t.Fatal("independent runs share an alternate RNG")
	}
	for i := range 20 {
		if x, y := a.AlternateRNG().Raw(), b.AlternateRNG().Raw(); x != y {
			t.Fatalf("draw %d differs: %08x vs %08x", i, x, y)
		}
	}
	want := *a.AlternateRNG()
	next := appA.NewState(CharMarisa, ShotB, DiffHard)
	if next.AlternateRNG() != a.AlternateRNG() || *next.AlternateRNG() != want {
		t.Fatal("a new run reset the application's alternate RNG")
	}
}
