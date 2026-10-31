// Command th10diff compares two per-frame traces (see package trace) and
// reports the first divergence with enough context to locate the cause.
//
// Usage:
//
//	th10diff -a original.trace -b replica.trace [-eps 0.001] [-context 3]
//	         [-shift 0] [-scope all|motion]
//
// Frames are matched by index (A's index plus -shift), so frames the sampler
// dropped are skipped instead of misaligning the rest. Comparison ends when
// the reference runs out; the replica ending first is a divergence.
//
// -scope motion compares only the player, enemy and bullet positions, enemy
// HP, angles, speeds and counts: what the ECL and the RNG decide on their own
// in a replay that never shoots.
//
// Exit code is 0 when the traces match within the epsilon budget, 1 when they
// diverge, and 2 on usage/IO errors.
package main

import (
	"flag"
	"fmt"
	"math"
	"os"

	"th10/trace"
)

func main() {
	os.Exit(run())
}

// diffFlags holds parsed options so run() can be invoked repeatedly in tests.
type diffFlags struct {
	pathA   string
	pathB   string
	eps     float64
	angEps  float64
	context int
	maxRows int
	shift   int
	scope   string
}

func parseFlags() (diffFlags, bool) {
	fs := flag.NewFlagSet("th10diff", flag.ContinueOnError)
	var f diffFlags
	fs.StringVar(&f.pathA, "a", "", "trace A (original/reference)")
	fs.StringVar(&f.pathB, "b", "", "trace B (replica)")
	fs.Float64Var(&f.eps, "eps", 1e-3, "position epsilon in pixels, on top of the trace's rounding")
	fs.Float64Var(&f.angEps, "ang-eps", 1e-4, "angle epsilon in radians, on top of the trace's rounding")
	fs.IntVar(&f.context, "context", 3, "frames of context to print around the divergence")
	fs.IntVar(&f.maxRows, "max-rows", 8, "max entity rows to show per side")
	fs.IntVar(&f.shift, "shift", 0, "added to A's frame indices before matching")
	fs.StringVar(&f.scope, "scope", "all", "fields to compare: all or motion")
	if err := fs.Parse(os.Args[1:]); err != nil {
		return f, false
	}
	return f, f.pathA != "" && f.pathB != "" && (f.scope == "all" || f.scope == "motion")
}

func run() int {
	opts, ok := parseFlags()
	if !ok {
		flag.Usage()
		return 2
	}
	fa, err := os.Open(opts.pathA)
	if err != nil {
		fmt.Fprintln(os.Stderr, "th10diff:", err)
		return 2
	}
	defer fa.Close()
	fb, err := os.Open(opts.pathB)
	if err != nil {
		fmt.Fprintln(os.Stderr, "th10diff:", err)
		return 2
	}
	defer fb.Close()

	ra, rb := trace.NewReader(fa), trace.NewReader(fb)
	diff := &differ{eps: opts.eps + posUnit + slack, angEps: opts.angEps + angUnit + slack,
		maxRows: opts.maxRows, context: opts.context, shift: opts.shift, motion: opts.scope == "motion"}
	code := diff.compare(ra, rb)
	if code == 0 {
		fmt.Printf("MATCH: %d frames compared, all within eps (pos %.4g px, ang %.4g rad)\n",
			diff.frames, opts.eps, opts.angEps)
		if diff.gaps > 0 {
			fmt.Printf("  %d replica frames had no reference sample\n", diff.gaps)
		}
		if diff.tail > 0 {
			fmt.Printf("  reference ended; %d further replica frames not compared\n", diff.tail)
		}
	}
	return code
}

// The trace prints positions and speeds to 3 decimals and angles to 5, so two
// values within the budget can print up to one unit further apart (one ulp
// apart on a rounding boundary prints a unit apart). slack absorbs the error
// of parsing the decimals back.
const (
	posUnit = 1e-3
	angUnit = 1e-5
	slack   = 1e-9
)

type differ struct {
	eps     float64 // budgets including posUnit/angUnit
	angEps  float64
	maxRows int
	context int
	shift   int
	motion  bool // -scope motion
	frames  int
	gaps    int            // B frames with no A frame at that index
	tail    int            // B frames after A ended
	history []*trace.Frame // ring of recent A frames for context printing
}

func (d *differ) compare(ra, rb *trace.Reader) int {
	a, err := ra.Next()
	if err != nil {
		fmt.Fprintln(os.Stderr, "th10diff: read A:", err)
		return 2
	}
	b, err := rb.Next()
	if err != nil {
		fmt.Fprintln(os.Stderr, "th10diff: read B:", err)
		return 2
	}
	for {
		if a == nil {
			for b != nil {
				d.tail++
				if b, err = rb.Next(); err != nil {
					fmt.Fprintln(os.Stderr, "th10diff: read B:", err)
					return 2
				}
			}
			return 0
		}
		if b == nil {
			fmt.Printf("DIVERGE: trace B ended before frame %d; A continues\n", a.Index+d.shift)
			d.printFrame("A", a)
			return 1
		}
		switch ai := a.Index + d.shift; {
		case ai < b.Index: // before the replica's first frame
			if a, err = ra.Next(); err != nil {
				fmt.Fprintln(os.Stderr, "th10diff: read A:", err)
				return 2
			}
			continue
		case ai > b.Index: // the sampler dropped this frame
			d.gaps++
			if b, err = rb.Next(); err != nil {
				fmt.Fprintln(os.Stderr, "th10diff: read B:", err)
				return 2
			}
			continue
		}
		d.frames++
		if why := d.diffFrame(a, b); why != "" {
			d.report(why, a, b)
			return 1
		}
		d.history = append(d.history, a)
		if len(d.history) > d.context {
			d.history = d.history[1:]
		}
		if a, err = ra.Next(); err != nil {
			fmt.Fprintln(os.Stderr, "th10diff: read A:", err)
			return 2
		}
		if b, err = rb.Next(); err != nil {
			fmt.Fprintln(os.Stderr, "th10diff: read B:", err)
			return 2
		}
	}
}

// diffFrame returns a non-empty description when the frames disagree.
func (d *differ) diffFrame(a, b *trace.Frame) string {
	if !d.motion {
		if why := d.diffStats(a, b); why != "" {
			return why
		}
	}
	if math.Abs(a.PX-b.PX) > d.eps || math.Abs(a.PY-b.PY) > d.eps {
		return fmt.Sprintf("player pos (%.3f,%.3f) != (%.3f,%.3f)", a.PX, a.PY, b.PX, b.PY)
	}
	if a.Enemies != b.Enemies {
		return fmt.Sprintf("enemy count %d != %d", a.Enemies, b.Enemies)
	}
	if a.Bullets != b.Bullets {
		return fmt.Sprintf("bullet count %d != %d", a.Bullets, b.Bullets)
	}
	if !d.motion {
		if a.Items != b.Items {
			return fmt.Sprintf("item count %d != %d", a.Items, b.Items)
		}
		if a.PBullets != b.PBullets {
			return fmt.Sprintf("player bullet count %d != %d", a.PBullets, b.PBullets)
		}
	}
	if why := diffMSG(a.MSG, b.MSG); why != "" {
		return why
	}
	if why := d.diffEnemies(a.E, b.E); why != "" {
		return why
	}
	return d.diffBullets(a.B, b.B)
}

// diffMSG compares the dialogs: which entry is open and where it is. The
// dialog drives ECL 339, so its timing counts for -scope motion too.
func diffMSG(a, b *trace.MSGRow) string {
	if (a == nil) == (b == nil) && (a == nil || *a == *b) {
		return ""
	}
	return fmt.Sprintf("dialog %s != %s", msgString(a), msgString(b))
}

func msgString(m *trace.MSGRow) string {
	if m == nil {
		return "none"
	}
	return fmt.Sprintf("{entry %d clock %d wait %d signal %d at %d op %d}",
		m.ID, m.Clock, m.Wait, m.Signal, m.At, m.Op)
}

// diffStats compares the player's counters (out of -scope motion).
func (d *differ) diffStats(a, b *trace.Frame) string {
	if a.PLife != b.PLife {
		return fmt.Sprintf("life %d != %d", a.PLife, b.PLife)
	}
	if a.PPower != b.PPower {
		return fmt.Sprintf("power %d != %d", a.PPower, b.PPower)
	}
	if a.PScore != b.PScore {
		return fmt.Sprintf("score %d != %d", a.PScore, b.PScore)
	}
	if a.PGraze != b.PGraze {
		return fmt.Sprintf("graze %d != %d", a.PGraze, b.PGraze)
	}
	if a.PFaith != b.PFaith {
		return fmt.Sprintf("faith %d != %d", a.PFaith, b.PFaith)
	}
	if a.PState != b.PState {
		return fmt.Sprintf("player state %d != %d", a.PState, b.PState)
	}
	return ""
}

func (d *differ) diffEnemies(ea, eb []trace.EnemyRow) string {
	n := min(len(ea), len(eb))
	for i := 0; i < n; i++ {
		a, b := &ea[i], &eb[i]
		if a.Active != b.Active {
			return fmt.Sprintf("enemy[%d] active %v != %v", i, a.Active, b.Active)
		}
		if a.HP != b.HP {
			return fmt.Sprintf("enemy[%d] hp %d != %d", i, a.HP, b.HP)
		}
		if math.Abs(a.X-b.X) > d.eps || math.Abs(a.Y-b.Y) > d.eps {
			return fmt.Sprintf("enemy[%d] pos (%.3f,%.3f) != (%.3f,%.3f)", i, a.X, a.Y, b.X, b.Y)
		}
	}
	if len(ea) != len(eb) {
		return fmt.Sprintf("enemy row count %d != %d", len(ea), len(eb))
	}
	return ""
}

func (d *differ) diffBullets(ba, bb []trace.BulletRow) string {
	n := min(len(ba), len(bb))
	for i := 0; i < n; i++ {
		a, b := &ba[i], &bb[i]
		if !d.motion && (a.Type != b.Type || a.Color != b.Color) {
			return fmt.Sprintf("bullet[%d] type/color (%d,%d) != (%d,%d)", i, a.Type, a.Color, b.Type, b.Color)
		}
		if math.Abs(a.X-b.X) > d.eps || math.Abs(a.Y-b.Y) > d.eps {
			return fmt.Sprintf("bullet[%d] pos (%.3f,%.3f) != (%.3f,%.3f)", i, a.X, a.Y, b.X, b.Y)
		}
		if math.Abs(a.Angle-b.Angle) > d.angEps {
			return fmt.Sprintf("bullet[%d] angle %.5f != %.5f", i, a.Angle, b.Angle)
		}
		if math.Abs(a.Speed-b.Speed) > d.eps {
			return fmt.Sprintf("bullet[%d] speed %.3f != %.3f", i, a.Speed, b.Speed)
		}
	}
	if len(ba) != len(bb) {
		return fmt.Sprintf("bullet row count %d != %d", len(ba), len(bb))
	}
	return ""
}

func (d *differ) report(why string, a, b *trace.Frame) {
	fmt.Printf("DIVERGE at frame %d: %s\n", a.Index, why)
	if d.context > 0 && len(d.history) > 0 {
		fmt.Println("\n--- preceding frames (A) ---")
		for _, f := range d.history {
			fmt.Printf("  frame %d: pos=(%.3f,%.3f) life=%d power=%d enemies=%d bullets=%d\n",
				f.Index, f.PX, f.PY, f.PLife, f.PPower, f.Enemies, f.Bullets)
		}
	}
	fmt.Println("\n--- A (reference) ---")
	d.printFrame("A", a)
	fmt.Println("\n--- B (replica) ---")
	d.printFrame("B", b)
}

func (d *differ) printFrame(tag string, f *trace.Frame) {
	if f == nil {
		fmt.Printf("  [%s] <none>\n", tag)
		return
	}
	fmt.Printf("  [%s] frame %d pos=(%.3f,%.3f) life=%d power=%d faith=%d score=%d graze=%d state=%d\n",
		tag, f.Index, f.PX, f.PY, f.PLife, f.PPower, f.PFaith, f.PScore, f.PGraze, f.PState)
	fmt.Printf("       enemies=%d bullets=%d items=%d playerBullets=%d\n",
		f.Enemies, f.Bullets, f.Items, f.PBullets)
	if f.MSG != nil {
		fmt.Printf("       dialog %s\n", msgString(f.MSG))
	}
	for i, e := range f.E {
		if i >= d.maxRows {
			fmt.Printf("       ... %d more enemies\n", len(f.E)-i)
			break
		}
		fmt.Printf("       E%d (%.2f,%.2f) hp=%d act=%v type=%d\n", i, e.X, e.Y, e.HP, e.Active, e.Type)
	}
	for i, bl := range f.B {
		if i >= d.maxRows {
			fmt.Printf("       ... %d more bullets\n", len(f.B)-i)
			break
		}
		fmt.Printf("       B%d (%.2f,%.2f) ang=%.4f spd=%.2f type=%d color=%d\n",
			i, bl.X, bl.Y, bl.Angle, bl.Speed, bl.Type, bl.Color)
	}
}
