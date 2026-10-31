package dialog

import "testing"

// run updates m with the same keys until it ends, at most limit frames, and
// returns the frame it ended on (0 = still running).
func run(m *MSG, advance, skip bool, limit int) int {
	for f := 1; f <= limit; f++ {
		if !m.Update(advance, skip) {
			return f
		}
	}
	return 0
}

// TestMSGWaitTimesOut: op 10 holds the clock until its timer runs out on the
// arg-th frame, which advances with the sound like a key press would.
func TestMSGWaitTimesOut(t *testing.T) {
	var se []int
	code := []Instr{{Op: msgWait, Arg: 600}, {Time: 0, Op: msgText, Text: "a"}, {Time: 1, Op: msgEnd}}
	m := NewMSG(0, code, Hooks{SE: func(id int) { se = append(se, id) }})
	if f := run(m, false, false, 599); f != 0 || m.View.Lines[0].Text != "" {
		t.Fatalf("ended on frame %d with line %q before the timeout", f, m.View.Lines[0].Text)
	}
	if !m.Update(false, false) || m.View.Lines[0].Text != "a" || len(se) != 1 || se[0] != 0 {
		t.Fatalf("frame 600: line %q, sounds %v; want the wait passed with SE 0", m.View.Lines[0].Text, se)
	}
	if m.Update(false, false) {
		t.Fatal("op 0 at time 1 did not end the entry on frame 601")
	}
}

// TestMSGWaitAdvances: the shot key's press edge passes op 10 at once, and
// the rest of that time's instructions run the same frame.
func TestMSGWaitAdvances(t *testing.T) {
	code := []Instr{{Op: msgWait, Arg: 600}, {Op: msgText, Text: "a"}, {Time: 1, Op: msgEnd}}
	m := NewMSG(0, code, Hooks{})
	if !m.Update(true, false) || m.View.Lines[0].Text != "a" {
		t.Fatalf("frame 1: line %q, want the wait passed", m.View.Lines[0].Text)
	}
	if f := run(m, false, false, 10); f != 1 {
		t.Fatalf("ended %d frames later, want 1", f)
	}
}

// TestMSGSkip: with op 9 on, holding Ctrl passes waits silently and jumps the
// clock to each next instruction's time; without op 9 Ctrl does nothing.
func TestMSGSkip(t *testing.T) {
	code := func(skippable int32) []Instr {
		return []Instr{
			{Op: msgSkippable, Arg: skippable}, {Op: msgWait, Arg: 600},
			{Time: 50, Op: msgText, Text: "a"}, {Time: 90, Op: msgEnd},
		}
	}
	se := 0
	m := NewMSG(0, code(1), Hooks{SE: func(int) { se++ }})
	if f := run(m, false, true, 10); f != 3 || se != 0 {
		t.Fatalf("skipping: ended on frame %d with %d sounds, want frame 3 and none", f, se)
	}
	m = NewMSG(0, code(0), Hooks{})
	if f := run(m, false, true, 700); f != 600+40+50 {
		t.Fatalf("not skippable: ended on frame %d, want %d", f, 600+40+50)
	}
}

// TestMSGSignal: op 11 raises the signal for one frame, so ECL 339 (which runs
// before the MSG) sees it exactly once.
func TestMSGSignal(t *testing.T) {
	code := []Instr{{Time: 2, Op: msgSignal}, {Time: 10, Op: msgEnd}}
	m := NewMSG(0, code, Hooks{})
	var seen []int
	for f := 1; f <= 6; f++ {
		if m.Signal != 0 {
			seen = append(seen, f) // what this frame's ECL sees
		}
		m.Update(false, false)
	}
	if len(seen) != 1 || seen[0] != 4 {
		t.Fatalf("ECL saw the signal on frames %v, want [4]", seen)
	}
}

// TestMSGHooksAndView: the show/speaker/text ops build the view, and ops 18,
// 20 and 21 call their hooks.
func TestMSGHooksAndView(t *testing.T) {
	var calls []string
	hooks := Hooks{
		BossBGM: func() { calls = append(calls, "bgm") },
		Clear:   func() { calls = append(calls, "clear") },
		FadeBGM: func() { calls = append(calls, "fade") },
	}
	code := []Instr{
		{Op: msgShowPlayer}, {Op: msgShowBox}, {Op: msgSpeakerPlayer},
		{Op: msgText, Text: "p1"}, {Op: msgText, Text: "p2"},
		{Time: 1, Op: msgShowBoss}, {Time: 1, Op: msgSpeakerBoss}, {Time: 1, Op: msgText, Text: "b1"},
		{Time: 2, Op: msgBossBGM}, {Time: 2, Op: msgFadeBGM}, {Time: 2, Op: msgStageClear},
		{Time: 3, Op: msgEnd},
	}
	m := NewMSG(0, code, hooks)
	m.Update(false, false)
	v := &m.View
	if !v.Player || !v.Box || v.Boss || v.Lines != [2]TextLine{{"p1", 0}, {"p2", 0}} {
		t.Fatalf("frame 1 view %+v", *v)
	}
	m.Update(false, false)
	if !v.Boss || v.Speaker != 1 || v.Lines != [2]TextLine{{"b1", 1}, {}} {
		t.Fatalf("frame 2 view %+v", *v)
	}
	m.Update(false, false)
	if len(calls) != 3 || calls[0] != "bgm" || calls[1] != "fade" || calls[2] != "clear" {
		t.Fatalf("hooks %v, want [bgm fade clear]", calls)
	}
}
