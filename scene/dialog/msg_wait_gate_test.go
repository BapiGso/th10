package dialog

import (
	"fmt"
	"testing"
)

func TestMSGWaitKeepsSeparateSkipGatesAndAdvanceSound(t *testing.T) {
	for _, allowed := range []bool{false, true} {
		for _, held := range []bool{false, true} {
			for _, advance := range []bool{false, true} {
				t.Run(fmt.Sprintf("allowed_%v/held_%v/advance_%v", allowed, held, advance), func(t *testing.T) {
					sounds := 0
					m := NewMSG(0, []Instr{{Op: msgWait, Arg: 5}, {Op: msgText, Text: "ready"}, {Time: 1, Op: msgEnd}}, Hooks{SE: func(int) { sounds++ }})
					m.skip = allowed
					if !m.Update(advance, held) {
						t.Fatal("the future end opcode ran too early")
					}
					passed := advance || allowed && held
					clock, wait, _, _ := m.Pos()
					if passed {
						if clock != 1 || wait != 0 || m.View.Lines[0].Text != "ready" {
							t.Fatalf("passed wait: clock=%d wait=%d view=%+v", clock, wait, m.View)
						}
					} else if clock != 0 || wait != 4 || m.View.Lines[0].Text != "" {
						t.Fatalf("held wait: clock=%d wait=%d view=%+v", clock, wait, m.View)
					}
					wantSounds := 0
					if advance {
						wantSounds = 1
					}
					if sounds != wantSounds {
						t.Fatalf("advance sounds=%d, want %d", sounds, wantSounds)
					}
				})
			}
		}
	}
}
