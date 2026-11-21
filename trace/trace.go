// Package trace defines the per-frame state dump shared by the headless
// simulator (sim) and the read-only original-process sampler (tools/th10probe).
// Both sides must emit the same schema so tools/th10diff can align them.
//
// The format is line-oriented text, one frame per block, so diffs are readable
// without tooling. Values are printed with fixed precision; the diff tool
// applies per-field epsilon budgets instead of comparing text.
//
// Positions are in the original's game space: origin at the playfield's
// top-centre, x right, y down, in pixels (ECL coordinates). The replica's
// screen space is that plus (game.FieldLeft+192, game.FieldTop).
package trace

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
)

// FormatVersion is bumped when the schema changes; the diff tool refuses to
// compare traces of different versions.
const FormatVersion = 2 // 2: game-space positions, manager-order enemies

// Frame is one sampled frame of game state.
type Frame struct {
	Index int // frame number since run start (1-based)
	// Half is present only when the sampler could not confirm a complete
	// frame. Such a block is diagnostic data, never a comparison sample.
	Half *HalfRow

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

	// Enemies (bounded sample) in the enemy manager's list order, which
	// includes the stage's hidden "main" enemy at the head
	E []EnemyRow
	// Enemy bullets (bounded sample, in pool slot order)
	B []BulletRow
	// The playing .msg entry, nil without one
	MSG *MSGRow
	// Ext and the full item/laser rows are opt-in; nil preserves v2 bytes.
	Ext *Extended
	I   []ItemRow
	L   []LaserRow

	extCommitted bool
	extSpell     bool
	extItems     int
	extLasers    int
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

// MSGRow is the playing dialog (FUN_00415e90's object): the entry ECL 338
// opened, its clock (+0x1c), op 10's timer (+0x30), op 11's signal (+0x78)
// and the time and opcode of the instruction it is at (+0x5c).
type MSGRow struct {
	ID, Clock, Wait, Signal int
	At, Op                  int
}

// MaxEnemyRows and MaxBulletRows bound the per-frame sample so traces stay
// comparable in size between the simulator and the external sampler.
const (
	MaxEnemyRows  = 64
	MaxBulletRows = 256
)

// Writer emits trace text.
type Writer struct {
	w       *bufio.Writer
	err     error
	direct  io.Writer
	block   bytes.Buffer
	capture *CaptureHeader
	frames  int
	last    int
	ended   bool
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
	if tw.ended {
		tw.err = fmt.Errorf("trace: frame after capture footer")
		return
	}
	if tw.capture != nil {
		if err := validateCaptureInput(f, tw.capture, tw.last); err != nil {
			tw.err = err
			return
		}
	}
	if tw.direct != nil {
		tw.block.Reset()
		tw.w.Reset(&tw.block)
	}
	_, tw.err = fmt.Fprintf(tw.w,
		"F %d p %.3f %.3f life %d power %d faith %d score %d graze %d state %d invinc %d bomb %d | e %d b %d i %d pb %d\n",
		f.Index, f.PX, f.PY, f.PLife, f.PPower, f.PFaith, f.PScore, f.PGraze,
		f.PState, f.PInvinc, f.PBombTimer,
		f.Enemies, f.Bullets, f.Items, f.PBullets)
	if tw.err != nil {
		return
	}
	if h := f.Half; h != nil {
		if _, tw.err = fmt.Fprintf(tw.w, "H %s attempts %d before %d after %d\n",
			h.Reason, h.Attempts, h.Before, h.After); tw.err != nil {
			return
		}
	}
	if f.Ext != nil {
		if tw.err = writeExtendedHeader(tw.w, f); tw.err != nil {
			return
		}
	}
	if m := f.MSG; m != nil {
		if _, tw.err = fmt.Fprintf(tw.w, "M %d clock %d wait %d signal %d at %d op %d\n",
			m.ID, m.Clock, m.Wait, m.Signal, m.At, m.Op); tw.err != nil {
			return
		}
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
	if f.Ext != nil {
		tw.err = writeExtendedRows(tw.w, f)
	} else if tw.capture != nil {
		tw.err = writeCommit(tw.w, f)
	}
	if tw.direct != nil && tw.err == nil {
		if tw.err = tw.w.Flush(); tw.err == nil {
			tw.err = tw.commit(tw.block.Bytes())
		}
	}
	if tw.err == nil {
		tw.frames++
		tw.last = f.Index
	}
}

// Flush flushes buffered output and returns the first write error.
func (tw *Writer) Flush() error {
	err := tw.w.Flush()
	if f, ok := tw.direct.(interface{ Flush() error }); ok {
		err = errors.Join(err, f.Flush())
	}
	tw.err = errors.Join(tw.err, err)
	return tw.err
}

func boolInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

// Reader parses trace text.
type Reader struct {
	s            *bufio.Scanner
	held         *Frame // frame parsed from the current "F" line, not yet returned
	done         bool
	capture      *CaptureHeader
	frames, last int
	endSeen      bool
	finalErr     error
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
			if err := tr.validateFrame(tr.held); err != nil {
				return nil, err
			}
			if tr.capture != nil && !tr.endSeen {
				return nil, fmt.Errorf("trace: incomplete capture: missing Q footer")
			}
			if tr.finalErr != nil {
				return nil, tr.finalErr
			}
			return tr.held, nil
		}
		line := tr.s.Text()
		if line == "" {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) == 0 {
			continue
		}
		if tr.endSeen && !strings.HasPrefix(fields[0], "#") {
			return nil, fmt.Errorf("trace: data after capture footer")
		}
		if tr.held != nil && tr.held.extCommitted {
			switch fields[0] {
			case "E", "B", "M", "H":
				return nil, fmt.Errorf("trace: frame %d: row after Z commit", tr.held.Index)
			}
		}
		switch fields[0] {
		case "F":
			prev := tr.held
			if err := tr.validateFrame(prev); err != nil {
				return nil, err
			}
			f := &Frame{}
			if err := parseFrameLine(f, fields); err != nil {
				return nil, err
			}
			if tr.capture != nil && (f.Index <= tr.last || f.Index < 1) {
				return nil, fmt.Errorf("trace: non-increasing capture frame %d", f.Index)
			}
			tr.frames++
			tr.last = f.Index
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
				return nil, fmt.Errorf("trace: frame %d: %w", tr.held.Index, err)
			}
			tr.held.B = append(tr.held.B, b)
		case "M":
			if tr.held == nil {
				continue
			}
			var m MSGRow
			if err := parseMSGLine(&m, fields); err != nil {
				return nil, err
			}
			tr.held.MSG = &m
		case "H":
			if tr.held == nil || tr.held.Half != nil {
				return nil, fmt.Errorf("trace: unexpected H line")
			}
			h, err := parseHalfLine(fields)
			if err != nil {
				return nil, fmt.Errorf("trace: frame %d: %w", tr.held.Index, err)
			}
			tr.held.Half = h
		case "C":
			if tr.capture != nil || tr.frames != 0 {
				return nil, fmt.Errorf("trace: unexpected C header")
			}
			h, err := parseCaptureHeader(fields)
			if err != nil {
				return nil, err
			}
			tr.capture = h
		case "Q":
			if err := tr.readCaptureEnd(fields); err != nil {
				return nil, err
			}
			if tr.held != nil {
				f := tr.held
				tr.held = nil
				return f, nil
			}
		case "Z":
			if tr.capture != nil && tr.held != nil && tr.held.Ext == nil {
				if err := readCommit(tr.held, fields); err != nil {
					return nil, err
				}
			} else if err := readExtendedLine(tr.held, fields); err != nil {
				return nil, err
			}
		case "X", "I", "L", "S":
			if err := readExtendedLine(tr.held, fields); err != nil {
				return nil, err
			}
		}
	}
}

func parseFrameLine(f *Frame, t []string) error {
	// F <idx> p <x> <y> life <n> power <n> faith <n> score <n> graze <n>
	//   state <n> invinc <n> bomb <n> | e <n> b <n> i <n> pb <n>
	if len(t) < 30 {
		return fmt.Errorf("trace: short F line %q", strings.Join(t, " "))
	}
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
	nextInt64 := func() int64 {
		v, e := strconv.ParseInt(get(&i), 10, 64)
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
	f.PFaith = nextInt64()
	i++ // "score"
	f.PScore = nextInt64()
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
	if len(t) < 9 {
		return fmt.Errorf("trace: short E line %q", strings.Join(t, " "))
	}
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

func parseMSGLine(m *MSGRow, t []string) error {
	// M <id> clock <n> wait <n> signal <n> at <n> op <n>
	if len(t) < 12 {
		return fmt.Errorf("trace: short M line %q", strings.Join(t, " "))
	}
	var err error
	for i, dst := range []*int{&m.ID, &m.Clock, &m.Wait, &m.Signal, &m.At, &m.Op} {
		v, e := strconv.Atoi(t[1+2*i])
		if e != nil && err == nil {
			err = e
		}
		*dst = v
	}
	return err
}

func parseBulletLine(b *BulletRow, t []string) error {
	// B <x> <y> ang <f> spd <f> type <n> color <n>
	if len(t) < 11 {
		return fmt.Errorf("short bullet row: got %d fields, want at least 11", len(t))
	}
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
