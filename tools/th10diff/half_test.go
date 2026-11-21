package main

import (
	"fmt"
	"strings"
	"testing"

	"th10/trace"
)

func TestDiffHalfFrameSkipAndCounts(t *testing.T) {
	dir := t.TempDir()
	lines := strings.SplitAfter(baseFrame, "\n")
	marked := strings.Replace(lines[0], "p 224.000", "p 999.000", 1) + "H unstable attempts 4 before 1 after 1\n" + lines[1]
	for _, sides := range []struct {
		a, b  string
		count string
	}{
		{marked, baseFrame, "A 1, B 0"}, {baseFrame, marked, "A 0, B 1"}, {marked, marked, "A 1, B 1"},
	} {
		a := writeTrace(t, dir, "a.txt", sides.a)
		b := writeTrace(t, dir, "b.txt", sides.b)
		code, out := runDiffStdout(t, a, b)
		if code != 0 || !strings.Contains(out, "MATCH: 1 frames compared") ||
			!strings.Contains(out, "SKIP: 1 half-frame indices ("+sides.count+")") {
			t.Fatalf("half skip/count: code=%d output=%s", code, out)
		}
	}
}

func TestDiffHalfFramesWithGapsShiftAndCutoffs(t *testing.T) {
	header := func(index int, half bool) string {
		line := fmt.Sprintf("F %d p 0.000 400.000 life 2 power 0 faith 50000 score 0 graze 0 state 1 invinc 0 bomb 0 | e 0 b 0 i 0 pb 0\n", index)
		if half {
			line += fmt.Sprintf("H cross-frame attempts 1 before %d after %d\n", index, index+1)
		}
		return line
	}
	a := header(1, true) + header(3, false) + header(4, true) + header(5, true)
	b := header(11, true) + header(12, false) + header(13, false) + header(14, false)
	d := &differ{shift: 10, eps: .002, angEps: .00011}
	if code := d.compare(trace.NewReader(strings.NewReader(a)), trace.NewReader(strings.NewReader(b))); code != 0 ||
		d.frames != 1 || d.half != 3 || d.halfA != 3 || d.halfB != 1 || d.gaps != 1 {
		t.Fatalf("aligned half/gaps: code=%d differ=%+v", code, d)
	}
	d = &differ{shift: 10, from: 3, after: 2, eps: .002, angEps: .00011}
	if code := d.compare(trace.NewReader(strings.NewReader(a)), trace.NewReader(strings.NewReader(b))); code != 0 || d.frames != 1 || d.half != 2 {
		t.Fatalf("cutoff and half: code=%d differ=%+v", code, d)
	}
}

func TestDiffDoesNotClaimAllHalfFramesMatched(t *testing.T) {
	dir := t.TempDir()
	line := strings.SplitAfter(baseFrame, "\n")[0]
	a := writeTrace(t, dir, "a.txt", line+"H calc-unconfirmed attempts 0 before 1 after 1\n")
	b := writeTrace(t, dir, "b.txt", line)
	code, out := runDiffStdout(t, a, b)
	if code != 2 || !strings.Contains(out, "INCOMPLETE: no stable overlapping samples") || strings.Contains(out, "MATCH:") {
		t.Fatalf("all-half output: code=%d out=%s", code, out)
	}
}
