package main

import (
	"strings"
	"testing"
)

func TestDiffFromFrameIsInclusive(t *testing.T) {
	dir := t.TempDir()
	a := writeTrace(t, dir, "a.txt", baseFrame)
	b := writeTrace(t, dir, "b.txt", strings.Replace(baseFrame, "F 1 p 224.000", "F 1 p 230.000", 1))
	if code := runDiff(t, a, b, "-scope", "motion", "-from-frame", "2"); code != 0 {
		t.Fatalf("excluded frame returned %d, want 0", code)
	}
	b = writeTrace(t, dir, "b.txt", strings.Replace(baseFrame, "F 2 p 224.000", "F 2 p 230.000", 1))
	if code := runDiff(t, a, b, "-scope", "motion", "-from-frame", "2"); code != 1 {
		t.Fatalf("first included frame returned %d, want 1", code)
	}
}

func TestDiffFromFrameUsesReferenceIndexBeforeShift(t *testing.T) {
	dir := t.TempDir()
	a := writeTrace(t, dir, "a.txt", baseFrame)
	b := writeTrace(t, dir, "b.txt", strings.NewReplacer("F 1 ", "F 11 ", "F 2 ", "F 12 ").Replace(
		strings.Replace(baseFrame, "F 1 p 224.000", "F 1 p 230.000", 1)))
	if code := runDiff(t, a, b, "-scope", "motion", "-from-frame", "2", "-shift", "10"); code != 0 {
		t.Fatalf("shifted range returned %d, want 0", code)
	}
}

func TestDiffFromFramePreservesBudgetAndMissingTail(t *testing.T) {
	dir := t.TempDir()
	a := writeTrace(t, dir, "a.txt", baseFrame)
	b := writeTrace(t, dir, "b.txt", strings.Replace(baseFrame, "F 2 p 224.000", "F 2 p 224.003", 1))
	if code := runDiff(t, a, b, "-scope", "motion", "-from-frame", "2"); code != 1 {
		t.Fatalf("range changed position budget: returned %d, want 1", code)
	}
	b = writeTrace(t, dir, "b.txt", strings.SplitAfter(baseFrame, "\n")[0])
	if code := runDiff(t, a, b, "-scope", "motion", "-from-frame", "2"); code != 1 {
		t.Fatalf("missing included tail returned %d, want 1", code)
	}
	if code := runDiff(t, a, b, "-from-frame", "-1"); code != 2 {
		t.Fatalf("negative reference index returned %d, want 2", code)
	}
}
