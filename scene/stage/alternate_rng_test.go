package stage

import (
	"testing"

	"th10/game"
)

func TestAlternateRNGPersistsAcrossContexts(t *testing.T) {
	app := &game.Game{}
	state := app.NewState(game.CharReimu, game.ShotA, game.DiffNormal)
	a := NewContext(state)
	a.Anm.Alt.Raw()
	want := *a.Anm.Alt
	state.Stage = 2
	state.RNGSeed, state.HasSeed = 0xabcd, true
	b := NewContext(state)
	if a.Anm.Alt != b.Anm.Alt || *b.Anm.Alt != want {
		t.Fatalf("stage start reset alternate RNG: got %+v, want %+v", b.Anm.Alt, want)
	}
	if b.RNG.State != 0xabcd || b.RNG.Count != 0 {
		t.Fatalf("main RNG = %+v, want replay seed abcd and count 0", b.RNG)
	}
	b.Anm.Alt.Raw()
	if a.Anm.Alt.Count != want.Count+2 || a.RNG.Count != 0 || b.RNG.Count != 0 {
		t.Fatalf("alternate draw touched the wrong state: first=%+v second=%+v", a.RNG, b.RNG)
	}
	next := app.NewState(game.CharMarisa, game.ShotC, game.DiffExtra)
	c := NewContext(next)
	if c.Anm.Alt != a.Anm.Alt || c.Anm.Alt.Count != want.Count+2 {
		t.Fatal("new run did not reuse the application RNG")
	}
}
