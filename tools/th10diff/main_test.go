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

// TestDiffPrintRounding: floats an ulp apart on a rounding boundary print a
// unit apart, which the budget allows on top; a larger gap still diverges.
func TestDiffPrintRounding(t *testing.T) {
	dir := t.TempDir()
	a := writeTrace(t, dir, "a.txt", baseFrame)
	for name, tc := range map[string]struct {
		x    string
		want int
	}{"b.txt": {"224.001", 0}, "c.txt": {"223.999", 0}, "d.txt": {"224.003", 1}} {
		b := writeTrace(t, dir, name, strings.Replace(baseFrame, "F 2 p 224.000", "F 2 p "+tc.x, 1))
		if code := runDiff(t, a, b); code != tc.want {
			t.Errorf("x %s returned %d, want %d", tc.x, code, tc.want)
		}
	}
}

func TestDiffSkipsSamplerGaps(t *testing.T) {
	dir := t.TempDir()
	frame3 := strings.ReplaceAll(strings.SplitAfter(baseFrame, "\n")[1], "F 2 ", "F 3 ")
	a := writeTrace(t, dir, "a.txt", strings.SplitAfter(baseFrame, "\n")[0]+frame3)
	b := writeTrace(t, dir, "b.txt", baseFrame+frame3)
	if code := runDiff(t, a, b); code != 0 {
		t.Fatalf("dropped reference frame returned %d, want 0", code)
	}
}

func TestDiffReferenceEndsFirst(t *testing.T) {
	dir := t.TempDir()
	a := writeTrace(t, dir, "a.txt", strings.SplitAfter(baseFrame, "\n")[0])
	b := writeTrace(t, dir, "b.txt", baseFrame)
	if code := runDiff(t, a, b); code != 0 {
		t.Fatalf("shorter reference returned %d, want 0", code)
	}
}

func TestDiffShift(t *testing.T) {
	dir := t.TempDir()
	a := writeTrace(t, dir, "a.txt", strings.NewReplacer("F 1 ", "F 0 ", "F 2 ", "F 1 ").Replace(baseFrame))
	b := writeTrace(t, dir, "b.txt", baseFrame)
	if code := runDiff(t, a, b, "-shift", "1"); code != 0 {
		t.Fatalf("shifted reference returned %d, want 0", code)
	}
}

func TestDiffMotionScope(t *testing.T) {
	dir := t.TempDir()
	a := writeTrace(t, dir, "a.txt", baseFrame)
	b := writeTrace(t, dir, "b.txt", strings.Replace(baseFrame, "faith 49998", "faith 50000", 1))
	if code := runDiff(t, a, b); code != 1 {
		t.Fatalf("faith divergence returned %d, want 1", code)
	}
	if code := runDiff(t, a, b, "-scope", "motion"); code != 0 {
		t.Fatalf("faith divergence under -scope motion returned %d, want 0", code)
	}
}

// TestDiffDialog: the dialog's timing is compared under both scopes, as it
// drives ECL 339.
func TestDiffDialog(t *testing.T) {
	dir := t.TempDir()
	msg := func(clock string) string {
		return strings.Replace(baseFrame, "pb 0\nF 2", "pb 0\nM 1 clock "+clock+" wait 0 signal 0 at 0 op 10\nF 2", 1)
	}
	a := writeTrace(t, dir, "a.txt", msg("3"))
	if code := runDiff(t, a, writeTrace(t, dir, "b.txt", msg("3"))); code != 0 {
		t.Fatalf("identical dialogs returned %d, want 0", code)
	}
	for name, body := range map[string]string{"c.txt": msg("4"), "d.txt": baseFrame} {
		if code := runDiff(t, a, writeTrace(t, dir, name, body), "-scope", "motion"); code != 1 {
			t.Fatalf("%s: dialog divergence returned %d, want 1", name, code)
		}
	}
}

// runDiff invokes the command's run() with flags parsed from args.
func runDiff(t *testing.T, a, b string, extra ...string) int {
	t.Helper()
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()
	os.Args = append([]string{"th10diff", "-a", a, "-b", b, "-context", "1"}, extra...)
	return run()
}
