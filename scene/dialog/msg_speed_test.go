package dialog

import "testing"

func TestMSGQuarterSpeedTimeline(t *testing.T) {
	m := NewMSG(0, []Instr{{Time: 2, Op: msgText, Text: "a"}, {Time: 3, Op: msgEnd}}, Hooks{})
	m.GameSpeed = func() float32 { return 0.25 }
	for i := 0; i < 8; i++ {
		if !m.Update(false, false) || m.View.Lines[0].Text != "" {
			t.Fatal("quarter-speed MSG executed a future instruction")
		}
	}
	if m.clock != 2 || m.clockTimer.F != 2 || !m.Update(false, false) || m.View.Lines[0].Text != "a" {
		t.Fatal("MSG clock did not retain quarter-frame precision")
	}
	if f := run(m, false, false, 10); f != 4 {
		t.Fatalf("quarter-speed end %d updates later, want 4", f)
	}
}

func TestMSGScaledWaitUsesTruncatedTimer(t *testing.T) {
	m := NewMSG(0, []Instr{{Op: msgWait, Arg: 3}, {Op: msgText, Text: "a"}, {Time: 1, Op: msgEnd}}, Hooks{})
	m.GameSpeed = func() float32 { return 0.25 }
	for i := 0; i < 8; i++ {
		if !m.Update(false, false) || m.View.Lines[0].Text != "" || m.clock != 0 {
			t.Fatal("wait passed before truncated Cur reached zero")
		}
	}
	if !m.Update(false, false) || m.View.Lines[0].Text != "a" || m.wait != 0 || m.waitTimer.F != 0 || m.clockTimer.F != 0.25 {
		t.Fatal("scaled wait did not clear its whole timer and tick the timeline once")
	}
}

func TestMSGSkipResetsFractionalClock(t *testing.T) {
	m := NewMSG(0, []Instr{{Op: msgSkippable, Arg: 1}, {Time: 50, Op: msgText, Text: "a"}, {Time: 90, Op: msgEnd}}, Hooks{})
	m.GameSpeed = func() float32 { return 0.25 }
	m.Update(false, false)
	if !m.Update(false, true) || m.clock != 50 || m.clockTimer.F != 50.25 || m.View.Lines[0].Text != "a" {
		t.Fatal("Ctrl did not reset both integer and fractional clock components")
	}
	if m.Update(false, true) {
		t.Fatal("Ctrl did not jump to the following end instruction")
	}
}
