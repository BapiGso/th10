package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// writeTrace writes a small synthetic trace and returns its path.
func writeTrace(t *testing.T, dir, name, body string) string {
	t.Helper()
	p := filepath.Join(dir, name)
	if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

const baseFrame = `F 1 p 224.000 416.000 life 2 power 0 faith 49999 score 0 graze 0 state 0 invinc 0 bomb 0 | e 0 b 0 i 0 pb 0
F 2 p 224.000 416.000 life 2 power 0 faith 49998 score 0 graze 0 state 0 invinc 0 bomb 0 | e 0 b 0 i 0 pb 0
`

func TestDiffIdentical(t *testing.T) {
	dir := t.TempDir()
	a := writeTrace(t, dir, "a.txt", baseFrame)
	b := writeTrace(t, dir, "b.txt", baseFrame)
	if code := runDiff(t, a, b); code != 0 {
		t.Fatalf("identical traces returned %d, want 0", code)
	}
}

func TestDiffPositionDivergence(t *testing.T) {
	dir := t.TempDir()
	a := writeTrace(t, dir, "a.txt", baseFrame)
	b := writeTrace(t, dir, "b.txt", strings.Replace(baseFrame,
		"F 2 p 224.000", "F 2 p 230.000", 1))
	if code := runDiff(t, a, b); code != 1 {
		t.Fatalf("divergent traces returned %d, want 1", code)
	}
}

func TestDiffIntegerDivergence(t *testing.T) {
	dir := t.TempDir()
	a := writeTrace(t, dir, "a.txt", baseFrame)
	b := writeTrace(t, dir, "b.txt", strings.Replace(baseFrame,
		"F 2 p 224.000 416.000 life 2", "F 2 p 224.000 416.000 life 1", 1))
	if code := runDiff(t, a, b); code != 1 {
		t.Fatalf("life divergence returned %d, want 1", code)
	}
}

func TestDiffLengthMismatch(t *testing.T) {
	dir := t.TempDir()
	a := writeTrace(t, dir, "a.txt", baseFrame)
	b := writeTrace(t, dir, "b.txt",
		"F 1 p 224.000 416.000 life 2 power 0 faith 49999 score 0 graze 0 state 0 invinc 0 bomb 0 | e 0 b 0 i 0 pb 0\n")
	if code := runDiff(t, a, b); code != 1 {
		t.Fatalf("length mismatch returned %d, want 1", code)
	}
}

func TestDiffEpsilonBudget(t *testing.T) {
	dir := t.TempDir()
	a := writeTrace(t, dir, "a.txt", baseFrame)
	// 0.0005 px is inside the default 1e-3 budget.
	b := writeTrace(t, dir, "b.txt", strings.Replace(baseFrame,
		"F 2 p 224.000", "F 2 p 224.0005", 1))
	if code := runDiff(t, a, b); code != 0 {
		t.Fatalf("sub-epsilon drift returned %d, want 0", code)
	}
}

// runDiff invokes the command's run() with flags parsed from args.
func runDiff(t *testing.T, a, b string) int {
	t.Helper()
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()
	os.Args = []string{"th10diff", "-a", a, "-b", b, "-context", "1"}
	return run()
}
