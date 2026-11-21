package trace

import (
	"fmt"
	"io"
	"strconv"
	"strings"
)

// ExtendedVersion is opt-in on X lines. The unextended v2 format is unchanged.
const ExtendedVersion = 1

const (
	MaxItemRows  = 150 + 2048
	MaxLaserRows = 256
)

// Extended contains native fields not reliably represented in old traces.
// Score, faith, power, lives, state and invincibility Cur remain on F.
type Extended struct {
	HiScore            int64
	HiScoreContinues   int
	ShownScore         int64 // GUI+0x9e78, stored value x10; not the actual score
	FaithHold          int
	FaithRate          int
	Rank               int
	PlayerIX, PlayerIY int32
	StateClock         int
	StateClockF        float64
	InvincF            float64
	Focus              int
	BombActive         bool
	BombClock          int
	BombClockF         float64
	ItemCalc           int // native manager Live, which excludes delayed items
	Lasers             int // linked objects, including pending StateDead objects
	Spell              *SpellRow
}

type ItemRow struct {
	Slot, Type, State, Clock int
	X, Y, ClockF             float64
}

type LaserRow struct {
	ID, Kind, Type, Color, State, Clock int
	X, Y, Angle, Length, Width, ClockF  float64
}

type SpellRow struct {
	ID, Clock, Initial, Duration int
	ClockF                       float64
	Bonus                        int64
	Active, Capture, Survival    bool
}

const extShape = "X _ hiscore _ hicont _ shown _ hold _ rate _ rank _ fixed _ _ timer _ _ invf _ focus _ bomb _ _ _ | itemcalc _ lasers _ irows _ lrows _ spell _"
const itemShape = "I _ type _ p _ _ state _ clock _ _"
const laserShape = "L _ kind _ type _ color _ p _ _ ang _ len _ width _ state _ clock _ _"
const spellShape = "S _ clock _ _ bonus _ initial _ duration _ active _ capture _ survival _"
const commitShape = "Z _ e _ b _ i _ l _ s _"

func writeExtendedHeader(w io.Writer, f *Frame) error {
	x := f.Ext
	_, err := fmt.Fprintf(w, "X %d hiscore %d hicont %d shown %d hold %d rate %d rank %d fixed %d %d timer %d %.3f invf %.3f focus %d bomb %d %d %.3f | itemcalc %d lasers %d irows %d lrows %d spell %d\n",
		ExtendedVersion, x.HiScore, x.HiScoreContinues, x.ShownScore, x.FaithHold, x.FaithRate, x.Rank,
		x.PlayerIX, x.PlayerIY, x.StateClock, x.StateClockF, x.InvincF, x.Focus,
		boolInt(x.BombActive), x.BombClock, x.BombClockF, x.ItemCalc, x.Lasers, len(f.I), len(f.L), boolInt(x.Spell != nil))
	return err
}

func writeExtendedRows(w io.Writer, f *Frame) error {
	if s := f.Ext.Spell; s != nil {
		if _, err := fmt.Fprintf(w, "S %d clock %d %.3f bonus %d initial %d duration %d active %d capture %d survival %d\n",
			s.ID, s.Clock, s.ClockF, s.Bonus, s.Initial, s.Duration, boolInt(s.Active), boolInt(s.Capture), boolInt(s.Survival)); err != nil {
			return err
		}
	}
	for _, i := range f.I {
		if _, err := fmt.Fprintf(w, "I %d type %d p %.3f %.3f state %d clock %d %.3f\n", i.Slot, i.Type, i.X, i.Y, i.State, i.Clock, i.ClockF); err != nil {
			return err
		}
	}
	for _, l := range f.L {
		if _, err := fmt.Fprintf(w, "L %d kind %d type %d color %d p %.3f %.3f ang %.5f len %.3f width %.3f state %d clock %d %.3f\n",
			l.ID, l.Kind, l.Type, l.Color, l.X, l.Y, l.Angle, l.Length, l.Width, l.State, l.Clock, l.ClockF); err != nil {
			return err
		}
	}
	return writeCommit(w, f)
}

func writeCommit(w io.Writer, f *Frame) error {
	spell := f.Ext != nil && f.Ext.Spell != nil
	_, err := fmt.Fprintf(w, "Z %d e %d b %d i %d l %d s %d\n", f.Index, len(f.E), len(f.B), len(f.I), len(f.L), boolInt(spell))
	return err
}

// parseShape checks both field count and labels before indexing a row.
// Integer counters never pass through float or architecture-sized int64 casts.
func parseShape(t []string, shape string, values ...any) error {
	parts := strings.Fields(shape)
	if len(t) != len(parts) {
		return fmt.Errorf("short/malformed %s line: got %d fields, want %d", parts[0], len(t), len(parts))
	}
	n := 0
	for i, label := range parts {
		if label != "_" {
			if t[i] != label {
				return fmt.Errorf("%s line: want label %q, got %q", parts[0], label, t[i])
			}
			continue
		}
		var err error
		switch dst := values[n].(type) {
		case *string:
			*dst = t[i]
		case *int:
			*dst, err = strconv.Atoi(t[i])
		case *int32:
			var v int64
			v, err = strconv.ParseInt(t[i], 10, 32)
			*dst = int32(v)
		case *int64:
			*dst, err = strconv.ParseInt(t[i], 10, 64)
		case *float64:
			*dst, err = strconv.ParseFloat(t[i], 64)
		case *bool:
			var v int
			v, err = strconv.Atoi(t[i])
			if err == nil && v != 0 && v != 1 {
				err = fmt.Errorf("boolean value %d", v)
			}
			*dst = v == 1
		default:
			return fmt.Errorf("invalid parser destination")
		}
		if err != nil {
			return fmt.Errorf("%s line: %w", parts[0], err)
		}
		n++
	}
	return nil
}

func readExtendedLine(f *Frame, t []string) error {
	if f == nil {
		return fmt.Errorf("trace: %s line without a frame", t[0])
	}
	if f.extCommitted {
		return fmt.Errorf("trace: frame %d: row after Z commit", f.Index)
	}
	fail := func(err error) error { return fmt.Errorf("trace: frame %d: %w", f.Index, err) }
	if t[0] == "X" {
		if f.Ext != nil {
			return fail(fmt.Errorf("duplicate X line"))
		}
		x := &Extended{}
		var version int
		if err := parseShape(t, extShape, &version, &x.HiScore, &x.HiScoreContinues, &x.ShownScore,
			&x.FaithHold, &x.FaithRate, &x.Rank, &x.PlayerIX, &x.PlayerIY,
			&x.StateClock, &x.StateClockF, &x.InvincF, &x.Focus, &x.BombActive, &x.BombClock, &x.BombClockF,
			&x.ItemCalc, &x.Lasers, &f.extItems, &f.extLasers, &f.extSpell); err != nil {
			return fail(err)
		}
		if version != ExtendedVersion {
			return fail(fmt.Errorf("unsupported X version %d", version))
		}
		if f.extItems < 0 || f.extItems > MaxItemRows || f.extLasers < 0 || f.extLasers > MaxLaserRows {
			return fail(fmt.Errorf("extension row count out of range"))
		}
		f.Ext = x
		return nil
	}
	if f.Ext == nil {
		return fail(fmt.Errorf("%s line without X extension", t[0]))
	}
	switch t[0] {
	case "I":
		var v ItemRow
		if err := parseShape(t, itemShape, &v.Slot, &v.Type, &v.X, &v.Y, &v.State, &v.Clock, &v.ClockF); err != nil {
			return fail(err)
		}
		if len(f.I) >= f.extItems || v.Slot < 0 || v.Slot >= MaxItemRows || len(f.I) > 0 && v.Slot <= f.I[len(f.I)-1].Slot {
			return fail(fmt.Errorf("invalid item slot/order/count"))
		}
		f.I = append(f.I, v)
	case "L":
		var v LaserRow
		if err := parseShape(t, laserShape, &v.ID, &v.Kind, &v.Type, &v.Color, &v.X, &v.Y, &v.Angle,
			&v.Length, &v.Width, &v.State, &v.Clock, &v.ClockF); err != nil {
			return fail(err)
		}
		if len(f.L) >= f.extLasers || v.Kind < 0 || v.Kind > 1 {
			return fail(fmt.Errorf("invalid laser kind/count"))
		}
		f.L = append(f.L, v)
	case "S":
		if !f.extSpell || f.Ext.Spell != nil {
			return fail(fmt.Errorf("unexpected S line"))
		}
		var v SpellRow
		if err := parseShape(t, spellShape, &v.ID, &v.Clock, &v.ClockF, &v.Bonus, &v.Initial, &v.Duration,
			&v.Active, &v.Capture, &v.Survival); err != nil {
			return fail(err)
		}
		f.Ext.Spell = &v
	case "Z":
		var index, e, b, items, lasers, spell int
		if err := parseShape(t, commitShape, &index, &e, &b, &items, &lasers, &spell); err != nil {
			return fail(err)
		}
		if index != f.Index || e != len(f.E) || b != len(f.B) || items != len(f.I) || lasers != len(f.L) || spell != boolInt(f.Ext.Spell != nil) {
			return fail(fmt.Errorf("z commit row counts do not match"))
		}
		f.extCommitted = true
		return validateExtended(f)
	}
	return nil
}

func validateExtended(f *Frame) error {
	if f == nil || f.Ext == nil {
		return nil
	}
	fail := func(why string) error { return fmt.Errorf("trace: frame %d: incomplete extension: %s", f.Index, why) }
	if !f.extCommitted {
		return fail("missing Z commit")
	}
	if len(f.I) != f.extItems || len(f.L) != f.extLasers || (f.Ext.Spell != nil) != f.extSpell {
		return fail("missing extended rows")
	}
	if f.Half == nil && (f.Items != len(f.I) || f.Ext.Lasers != len(f.L) || len(f.E) != min(f.Enemies, MaxEnemyRows) || len(f.B) != min(f.Bullets, MaxBulletRows)) {
		return fail("declared entity counts do not match rows")
	}
	return nil
}
