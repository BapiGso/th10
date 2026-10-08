// Package trace defines the per-frame state dump shared by the headless
// simulator (sim) and the read-only original-process sampler (tools/th10probe).
// Both sides must emit the same schema so tools/th10diff can align them.
//
// The format is line-oriented text, one frame per block, so diffs are readable
// without tooling. Values are printed with fixed precision; the diff tool
// applies per-field epsilon budgets instead of comparing text.
package trace

import (
	"bufio"
	"fmt"
	"io"
	"strconv"
	"strings"
)

// FormatVersion is bumped when the schema changes; the diff tool refuses to
// compare traces of different versions.
const FormatVersion = 1

// Frame is one sampled frame of game state.
type Frame struct {
	Index int // frame number since run start (1-based)

	// Player
	PX, PY     float64
	PLife      int
	PPower     int // hundredths of displayed power (0..500)
	PFaith     int64
	PScore     int64
	PGraze     int
	PState     int
	PInvinc    int
	PBombTimer int

	// Entity counts
	Enemies  int
	Bullets  int
	Items    int
	PBullets int

	// Enemies (bounded sample, in pool order)
	E []EnemyRow
	// Enemy bullets (bounded sample, in pool order)
	B []BulletRow
}

// EnemyRow is one enemy's sampled state.
type EnemyRow struct {
	X, Y   float64
	HP     int
	Active bool
	Type   int
}

// BulletRow is one enemy bullet's sampled state.
type BulletRow struct {
	X, Y  float64
	Angle float64
	Speed float64
	Type  int
	Color int
}

// MaxEnemyRows and MaxBulletRows bound the per-frame sample so traces stay
// comparable in size between the simulator and the external sampler.
const (
	MaxEnemyRows  = 64
	MaxBulletRows = 256
)

// Writer emits trace text.
type Writer struct {
	w   *bufio.Writer
	err error
}

// NewWriter wraps w.
func NewWriter(w io.Writer) *Writer {
	return &Writer{w: bufio.NewWriter(w)}
}

// WriteFrame appends one frame block.
func (tw *Writer) WriteFrame(f *Frame) {
	if tw.err != nil {
		return
	}
	_, tw.err = fmt.Fprintf(tw.w,
		"F %d p %.3f %.3f life %d power %d faith %d score %d graze %d state %d invinc %d bomb %d | e %d b %d i %d pb %d\n",
		f.Index, f.PX, f.PY, f.PLife, f.PPower, f.PFaith, f.PScore, f.PGraze,
		f.PState, f.PInvinc, f.PBombTimer,
		f.Enemies, f.Bullets, f.Items, f.PBullets)
	if tw.err != nil {
		return
	}
	for i := range f.E {
		e := &f.E[i]
		if _, tw.err = fmt.Fprintf(tw.w, "E %.3f %.3f hp %d act %d type %d\n",
			e.X, e.Y, e.HP, boolInt(e.Active), e.Type); tw.err != nil {
			return
		}
	}
	for i := range f.B {
		b := &f.B[i]
		if _, tw.err = fmt.Fprintf(tw.w, "B %.3f %.3f ang %.5f spd %.3f type %d color %d\n",
			b.X, b.Y, b.Angle, b.Speed, b.Type, b.Color); tw.err != nil {
			return
		}
	}
}

// Flush flushes buffered output and returns the first write error.
func (tw *Writer) Flush() error {
	if tw.err != nil {
		return tw.err
	}
	return tw.w.Flush()
}

func boolInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

// Reader parses trace text.
type Reader struct {
	s    *bufio.Scanner
	err  error
	held *Frame // frame parsed from the current "F" line, not yet returned
	done bool
}

// NewReader wraps r.
func NewReader(r io.Reader) *Reader {
	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)
	return &Reader{s: sc}
}

// Next returns the next frame, or nil at end of input.
//
// Entity rows (E/B lines) belong to the most recent "F" line, so the frame is
// only complete once the next "F" line (or EOF) is reached. The parser
// therefore holds one frame back and flushes it when the boundary appears.
func (tr *Reader) Next() (*Frame, error) {
	for {
		if !tr.s.Scan() {
			if err := tr.s.Err(); err != nil {
				return nil, err
			}
			if tr.done {
				return nil, nil
			}
			tr.done = true
			return tr.held, nil
		}
		line := tr.s.Text()
		if line == "" {
			continue
		}
		fields := strings.Fields(line)
		switch fields[0] {
		case "F":
			prev := tr.held
			f := &Frame{}
			if err := parseFrameLine(f, fields); err != nil {
				return nil, err
			}
			tr.held = f
			if prev != nil {
				return prev, nil
			}
		case "E":
			if tr.held == nil {
				continue
			}
			var e EnemyRow
			if err := parseEnemyLine(&e, fields); err != nil {
				return nil, err
			}
			tr.held.E = append(tr.held.E, e)
		case "B":
			if tr.held == nil {
				continue
			}
			var b BulletRow
			if err := parseBulletLine(&b, fields); err != nil {
				return nil, err
			}
			tr.held.B = append(tr.held.B, b)
		}
	}
}

func parseFrameLine(f *Frame, t []string) error {
	// F <idx> p <x> <y> life <n> power <n> faith <n> score <n> graze <n>
	//   state <n> invinc <n> bomb <n> | e <n> b <n> i <n> pb <n>
	var err error
	get := func(i *int) string { v := t[*i]; *i++; return v }
	i := 1
	nextInt := func() int {
		v, e := strconv.Atoi(get(&i))
		if e != nil && err == nil {
			err = e
		}
		return v
	}
	nextF := func() float64 {
		v, e := strconv.ParseFloat(get(&i), 64)
		if e != nil && err == nil {
			err = e
		}
		return v
	}
	f.Index = nextInt()
	// "p"
	i++
	f.PX, f.PY = nextF(), nextF()
	i++ // "life"
	f.PLife = nextInt()
	i++ // "power"
	f.PPower = nextInt()
	i++ // "faith"
	f.PFaith = int64(nextInt())
	i++ // "score"
	f.PScore = int64(nextInt())
	i++ // "graze"
	f.PGraze = nextInt()
	i++ // "state"
	f.PState = nextInt()
	i++ // "invinc"
	f.PInvinc = nextInt()
	i++ // "bomb"
	f.PBombTimer = nextInt()
	i++ // "|"
	i++ // "e"
	f.Enemies = nextInt()
	i++ // "b"
	f.Bullets = nextInt()
	i++ // "i"
	f.Items = nextInt()
	i++ // "pb"
	f.PBullets = nextInt()
	return err
}

func parseEnemyLine(e *EnemyRow, t []string) error {
	// E <x> <y> hp <n> act <n> type <n>
	var err error
	var act int
	parseAt := func(idx int, dst any) {
		switch d := dst.(type) {
		case *float64:
			v, e := strconv.ParseFloat(t[idx], 64)
			if e != nil && err == nil {
				err = e
			}
			*d = v
		case *int:
			v, e := strconv.Atoi(t[idx])
			if e != nil && err == nil {
				err = e
			}
			*d = v
		}
	}
	parseAt(1, &e.X)
	parseAt(2, &e.Y)
	parseAt(4, &e.HP)
	parseAt(6, &act)
	parseAt(8, &e.Type)
	e.Active = act != 0
	return err
}

func parseBulletLine(b *BulletRow, t []string) error {
	// B <x> <y> ang <f> spd <f> type <n> color <n>
	var err error
	parseAt := func(idx int, dst any) {
		switch d := dst.(type) {
		case *float64:
			v, e := strconv.ParseFloat(t[idx], 64)
			if e != nil && err == nil {
				err = e
			}
			*d = v
		case *int:
			v, e := strconv.Atoi(t[idx])
			if e != nil && err == nil {
				err = e
			}
			*d = v
		}
	}
	parseAt(1, &b.X)
	parseAt(2, &b.Y)
	parseAt(4, &b.Angle)
	parseAt(6, &b.Speed)
	parseAt(8, &b.Type)
	parseAt(10, &b.Color)
	return err
}
