package main

import (
	"strings"
	"testing"

	"th10/trace"
)

func TestDiffAfterKeepsLaterDivergences(t *testing.T) {
	dir := t.TempDir()
	a := writeTrace(t, dir, "a.txt", baseFrame)
	b := writeTrace(t, dir, "b.txt", strings.Replace(baseFrame, "F 1 p 224.000", "F 1 p 230.000", 1))
	if code := runDiff(t, a, b, "-scope", "motion"); code != 1 {
		t.Fatalf("default comparison returned %d, want first-frame divergence", code)
	}
	if code := runDiff(t, a, b, "-scope", "motion", "-after", "1"); code != 0 {
		t.Fatalf("cutoff comparison returned %d, want match on frame 2", code)
	}
	b = writeTrace(t, dir, "c.txt", strings.ReplaceAll(baseFrame, "p 224.000", "p 230.000"))
	if code := runDiff(t, a, b, "-scope", "motion", "-after", "1"); code != 1 {
		t.Fatalf("later position divergence returned %d, want 1", code)
	}
	if code := runDiff(t, a, b, "-after", "-1"); code != 2 {
		t.Fatalf("negative cutoff returned %d, want usage error", code)
	}
}

func TestDiffAfterUsesReferenceFrameBeforeShift(t *testing.T) {
	dir := t.TempDir()
	a := writeTrace(t, dir, "a.txt", baseFrame)
	bBody := strings.NewReplacer("F 1 ", "F 2 ", "F 2 ", "F 3 ").Replace(baseFrame)
	bBody = strings.Replace(bBody, "F 2 p 224.000", "F 2 p 230.000", 1)
	b := writeTrace(t, dir, "b.txt", bBody)
	if code := runDiff(t, a, b, "-shift", "1", "-after", "1"); code != 0 {
		t.Fatalf("shifted cutoff returned %d, want match on A2/B3", code)
	}
	b = writeTrace(t, dir, "c.txt", strings.Replace(bBody, "F 3 p 224.000", "F 3 p 230.000", 1))
	if code := runDiff(t, a, b, "-shift", "1", "-after", "1"); code != 1 {
		t.Fatalf("shifted later divergence returned %d, want 1", code)
	}
}

func TestDiffAfterPreservesSampleGapsAndLengthCheck(t *testing.T) {
	frame := strings.SplitAfter(baseFrame, "\n")[0]
	makeFrame := func(n string) string { return strings.Replace(frame, "F 1 ", "F "+n+" ", 1) }
	a := frame + makeFrame("3") + makeFrame("5") + makeFrame("7")
	b := frame + makeFrame("2") + makeFrame("3") + makeFrame("4") + makeFrame("5") + makeFrame("6") + makeFrame("7")
	d := &differ{eps: 1e-3, angEps: 1e-4, after: 3}
	if code := d.compare(trace.NewReader(strings.NewReader(a)), trace.NewReader(strings.NewReader(b))); code != 0 || d.frames != 2 || d.gaps != 2 {
		t.Fatalf("cutoff/gaps: code=%d frames=%d gaps=%d, want 0/2/2", code, d.frames, d.gaps)
	}
	d = &differ{eps: 1e-3, angEps: 1e-4, after: 3}
	b = frame + makeFrame("2") + makeFrame("3") + makeFrame("4") + makeFrame("5")
	if code := d.compare(trace.NewReader(strings.NewReader(a)), trace.NewReader(strings.NewReader(b))); code != 1 {
		t.Fatalf("replica ended after cutoff: code=%d, want 1", code)
	}
}
