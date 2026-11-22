package stage

import (
	"testing"
	"th10/game"
	"th10/input"
	"th10/replay"
)

type recordScript struct{}

func (*recordScript) Init(*Context)   {}
func (*recordScript) Update(*Context) {}
func (*recordScript) Finished() bool  { return false }

func TestRecordingModes(t *testing.T) {
	for _, mode := range []struct {
		name                            string
		practice, debug, playback, want bool
	}{
		{"normal", false, false, false, true},
		{"practice", true, false, false, true},
		{"debug", true, true, false, false},
		{"playback", false, false, true, false},
	} {
		t.Run(mode.name, func(t *testing.T) {
			state := game.NewState(0, 0, 1)
			state.Stage, state.SingleStage, state.DebugMode = 1, mode.practice, mode.debug
			if mode.playback {
				state.ReplayMode = 1
			}
			s := NewHeadless(state, &recordScript{})
			s.beginRecording()
			if (state.Recording != nil) != mode.want {
				t.Fatal("wrong recording mode")
			}
			if !mode.want {
				return
			}
			var in input.State
			in.Restore(1<<input.KeyRight | 1<<input.KeyShot)
			s.UpdateHeadless(&in)
			ch := state.Recording.Replay().Chapters[0]
			if len(ch.Records) != 1 || ch.Records[0].Keys != replay.KeyRight|replay.KeyShot {
				t.Fatal("input was not recorded before the world step")
			}
			if ch.PlayerY != 40000 || ch.Seed != 0x1234 || ch.Flags&1 == 0 {
				t.Fatalf("initial snapshot: %+v", ch)
			}
		})
	}
}
