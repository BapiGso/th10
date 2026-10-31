package main

import (
	"os"
	"path/filepath"
	"regexp"
	"slices"
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

// TestDiffBulletTriage: -bullets lists exactly the bullet rows that differ
// beyond the budget (indices 1 and 3 here), and when the counts differ it lists
// the extra rows of the longer side.
func TestDiffBulletTriage(t *testing.T) {
	dir := t.TempDir()
	frame := "F 1 p 224.000 416.000 life 2 power 0 faith 49999 score 0 graze 0 state 0 invinc 0 bomb 0 | e 0 b 5 i 0 pb 0\n" +
		"B 10.000 20.000 ang 1.00000 spd 2.000 type 1 color 2\n" +
		"B 30.000 40.000 ang 1.10000 spd 2.000 type 1 color 2\n" +
		"B 50.000 60.000 ang 1.20000 spd 2.000 type 1 color 2\n" +
		"B 70.000 80.000 ang 1.30000 spd 2.000 type 1 color 2\n" +
		"B 90.000 100.000 ang 1.40000 spd 2.000 type 1 color 2\n"
	a := writeTrace(t, dir, "a.txt", frame)
	b := writeTrace(t, dir, "b.txt", strings.NewReplacer(
		"B 30.000 40.000", "B 31.000 40.000", // bullet 1: x off by 1 px
		"ang 1.30000", "ang 1.30500", // bullet 3: angle off by 0.005 rad
	).Replace(frame))

	code, out := runDiffStdout(t, a, b, "-bullets")
	if code != 1 {
		t.Fatalf("divergent traces returned %d, want 1", code)
	}
	if !strings.Contains(out, "--- differing bullets at frame 1 (A count 5, B count 5) ---") {
		t.Fatalf("missing triage header:\n%s", out)
	}
	var got []string
	for _, m := range regexp.MustCompile(`(?m)^  bullet\[(\d+)\] A `).FindAllStringSubmatch(out, -1) {
		got = append(got, m[1])
	}
	if want := []string{"1", "3"}; !slices.Equal(got, want) {
		t.Fatalf("listed bullets %v, want %v\n%s", got, want, out)
	}

	// B drops its last row: the counts differ, so A's row 4 is listed as extra.
	short := strings.NewReplacer("b 5 i", "b 4 i").Replace(frame)
	short = short[:strings.LastIndex(short, "B 90.000")]
	_, out = runDiffStdout(t, a, writeTrace(t, dir, "c.txt", short), "-bullets")
	if !strings.Contains(out, "extra bullets beyond index 4: A 1, B 0") ||
		!strings.Contains(out, "A bullet[4] (90.000,100.000)") {
		t.Fatalf("extra-row listing missing:\n%s", out)
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

// runDiffStdout is runDiff with stdout captured into a file, for the report.
func runDiffStdout(t *testing.T, a, b string, extra ...string) (int, string) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "stdout.txt")
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	old := os.Stdout
	os.Stdout = f
	code := func() int {
		defer func() { os.Stdout = old }()
		return runDiff(t, a, b, extra...)
	}()
	out, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return code, string(out)
}
