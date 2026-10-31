package stage

import (
	"testing"

	"th10/entity/player"
	"th10/input"

	"github.com/hajimehoshi/ebiten/v2"
)

// stubScript does nothing and reports done when told.
type stubScript struct{ done bool }

func (s *stubScript) Init(*Context)        {}
func (s *stubScript) Update(*Context)      {}
func (s *stubScript) BgDraw(*ebiten.Image) {}
func (s *stubScript) Finished() bool       { return s.done }

// TestMSGClearEndsStage: the post entry plays while the world runs and its op
// 20 clears the stage; until then the dialog holds the script's own end back.
// Stage 6's post entry has no text or waits: op 20 runs when its clock reaches
// 540, on the 541st frame, and the MSG lives on until op 0 at 1140.
func TestMSGClearEndsStage(t *testing.T) {
	for _, tc := range []struct {
		stage    string
		clearsOn int // the frame op 20 runs on, 0 = depends on the waits
	}{
		{"stage4", 0},
		{"stage6", 541},
	} {
		script := &stubScript{}
		s := powerTestStage()
		s.script, s.ctx.stage = script, s
		s.ctx.LoadMSG(tc.stage, 0)
		s.ctx.OpenMSG(2)
		if s.ctx.DialogActive() {
			t.Fatalf("%s: opened a missing entry", tc.stage)
		}
		s.ctx.OpenMSG(1)
		if !s.ctx.DialogActive() {
			t.Fatalf("%s: entry 1 did not open", tc.stage)
		}
		script.done = true // main ended, as stages 1-3 do after the post dialog

		var in input.State
		f := 0
		for !s.finished() {
			if f++; f > 5000 {
				t.Fatalf("%s: not finished after %d frames", tc.stage, f)
			}
			if s.ctx.MSG() == nil {
				t.Fatalf("%s: the dialog ended on frame %d before clearing the stage", tc.stage, f)
			}
			in.Restore(uint16(f%2) << input.KeyOk) // tap Z every other frame
			s.update(&in)
		}
		if !s.ctx.Cleared() {
			t.Fatalf("%s: finished on frame %d without op 20", tc.stage, f)
		}
		if tc.clearsOn != 0 && f != tc.clearsOn {
			t.Errorf("%s: op 20 ran on frame %d, want %d", tc.stage, f, tc.clearsOn)
		}
	}
}

// TestMSGDisarmsPlayer: while a dialog is up the player neither shoots nor
// bombs, and enemy bullets do not hit (FUN_004266b0).
func TestMSGDisarmsPlayer(t *testing.T) {
	s := powerTestStage()
	s.script, s.ctx.stage = &stubScript{}, s
	s.ctx.State.Power = 400
	s.ctx.LoadMSG("stage2", 0)
	s.ctx.OpenMSG(0)
	if !s.ctx.DialogActive() {
		t.Fatal("stage2 entry 0 did not open")
	}
	var in input.State
	for f := 0; f < 30; f++ {
		in.Restore(1<<input.KeyShot | 1<<input.KeyBomb)
		s.update(&in)
	}
	p := s.ctx.Player
	if n := p.Bullets.ActiveCount(); n != 0 {
		t.Errorf("%d player bullets during the dialog, want none", n)
	}
	if p.State != player.StateNormal || s.ctx.State.Power != 400 {
		t.Errorf("bombed during the dialog: state %v, power %d", p.State, s.ctx.State.Power)
	}
	if s.ctx.bulletHit() || p.State != player.StateNormal {
		t.Errorf("a bullet hit during the dialog: state %v", p.State)
	}
}
