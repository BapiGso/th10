package main

import (
	"fmt"
	"math"

	"th10/trace"
)

func validScope(scope string) bool {
	return scope == "all" || scope == "motion" || needsExtension(scope)
}

func needsExtension(scope string) bool {
	switch scope {
	case "ext", "stats", "items", "lasers", "spell":
		return true
	}
	return false
}

func (d *differ) diffExtended(a, b *trace.Frame, scope string) string {
	if a.Ext == nil || b.Ext == nil {
		return "extended fields missing on one side (use -ext on both producers)"
	}
	if scope == "ext" || scope == "stats" {
		if why := d.diffExtendedStats(a, b); why != "" {
			return why
		}
	}
	if scope == "ext" || scope == "items" {
		if why := d.diffItems(a, b); why != "" {
			return why
		}
	}
	if scope == "ext" || scope == "lasers" {
		if why := d.diffLasers(a, b); why != "" {
			return why
		}
	}
	if scope == "ext" || scope == "spell" {
		return d.diffSpell(a.Ext.Spell, b.Ext.Spell)
	}
	return ""
}

type integerField struct {
	name string
	a, b int64
}
type floatField struct {
	name      string
	a, b, eps float64
}

func diffIntegers(fields ...integerField) string {
	for _, f := range fields {
		if f.a != f.b {
			return fmt.Sprintf("%s %d != %d", f.name, f.a, f.b)
		}
	}
	return ""
}

func diffFloats(fields ...floatField) string {
	for _, f := range fields {
		if math.IsNaN(f.a) || math.IsNaN(f.b) || math.IsInf(f.a, 0) || math.IsInf(f.b, 0) || math.Abs(f.a-f.b) > f.eps {
			return fmt.Sprintf("%s %.9g != %.9g", f.name, f.a, f.b)
		}
	}
	return ""
}

func (d *differ) diffExtendedStats(a, b *trace.Frame) string {
	if why := d.diffStats(a, b); why != "" {
		return why
	}
	x, y := a.Ext, b.Ext
	if why := diffIntegers(
		integerField{"invinc", int64(a.PInvinc), int64(b.PInvinc)}, integerField{"bomb timer", int64(a.PBombTimer), int64(b.PBombTimer)},
		integerField{"hiscore", x.HiScore, y.HiScore}, integerField{"hiscore continues", int64(x.HiScoreContinues), int64(y.HiScoreContinues)},
		integerField{"HUD score", x.ShownScore, y.ShownScore}, integerField{"faith hold", int64(x.FaithHold), int64(y.FaithHold)},
		integerField{"faith rate", int64(x.FaithRate), int64(y.FaithRate)}, integerField{"rank", int64(x.Rank), int64(y.Rank)},
		integerField{"player fixed X", int64(x.PlayerIX), int64(y.PlayerIX)}, integerField{"player fixed Y", int64(x.PlayerIY), int64(y.PlayerIY)},
		integerField{"player timer", int64(x.StateClock), int64(y.StateClock)}, integerField{"focus", int64(x.Focus), int64(y.Focus)},
		integerField{"bomb raw timer", int64(x.BombClock), int64(y.BombClock)}, integerField{"player bullet count", int64(a.PBullets), int64(b.PBullets)},
	); why != "" {
		return why
	}
	if x.BombActive != y.BombActive {
		return fmt.Sprintf("bomb active %v != %v", x.BombActive, y.BombActive)
	}
	return diffFloats(floatField{"player timer F", x.StateClockF, y.StateClockF, d.eps},
		floatField{"invinc F", x.InvincF, y.InvincF, d.eps}, floatField{"bomb timer F", x.BombClockF, y.BombClockF, d.eps})
}

func (d *differ) diffItems(a, b *trace.Frame) string {
	if why := diffIntegers(integerField{"item count", int64(a.Items), int64(b.Items)},
		integerField{"item calc count", int64(a.Ext.ItemCalc), int64(b.Ext.ItemCalc)},
		integerField{"item row count", int64(len(a.I)), int64(len(b.I))}); why != "" {
		return why
	}
	for i, x := range a.I {
		y := b.I[i]
		prefix := fmt.Sprintf("item[%d] ", x.Slot)
		if why := diffIntegers(integerField{prefix + "slot", int64(x.Slot), int64(y.Slot)},
			integerField{prefix + "type", int64(x.Type), int64(y.Type)}, integerField{prefix + "state", int64(x.State), int64(y.State)},
			integerField{prefix + "clock", int64(x.Clock), int64(y.Clock)}); why != "" {
			return why
		}
		if why := diffFloats(floatField{prefix + "X", x.X, y.X, d.eps}, floatField{prefix + "Y", x.Y, y.Y, d.eps},
			floatField{prefix + "clock F", x.ClockF, y.ClockF, d.eps}); why != "" {
			return why
		}
	}
	return ""
}

func (d *differ) diffLasers(a, b *trace.Frame) string {
	if why := diffIntegers(integerField{"laser count", int64(a.Ext.Lasers), int64(b.Ext.Lasers)},
		integerField{"laser row count", int64(len(a.L)), int64(len(b.L))}); why != "" {
		return why
	}
	for i, x := range a.L {
		y := b.L[i]
		prefix := fmt.Sprintf("laser[%d] ", i)
		if why := diffIntegers(integerField{prefix + "ID", int64(x.ID), int64(y.ID)}, integerField{prefix + "kind", int64(x.Kind), int64(y.Kind)},
			integerField{prefix + "type", int64(x.Type), int64(y.Type)}, integerField{prefix + "color", int64(x.Color), int64(y.Color)},
			integerField{prefix + "state", int64(x.State), int64(y.State)}, integerField{prefix + "clock", int64(x.Clock), int64(y.Clock)}); why != "" {
			return why
		}
		if why := diffFloats(floatField{prefix + "X", x.X, y.X, d.eps}, floatField{prefix + "Y", x.Y, y.Y, d.eps},
			floatField{prefix + "angle", x.Angle, y.Angle, d.angEps}, floatField{prefix + "length", x.Length, y.Length, d.eps},
			floatField{prefix + "width", x.Width, y.Width, d.eps}, floatField{prefix + "clock F", x.ClockF, y.ClockF, d.eps}); why != "" {
			return why
		}
	}
	return ""
}

func (d *differ) diffSpell(a, b *trace.SpellRow) string {
	if (a == nil) != (b == nil) {
		return "spell manager presence differs"
	}
	if a == nil {
		return ""
	}
	if why := diffIntegers(integerField{"spell ID", int64(a.ID), int64(b.ID)}, integerField{"spell clock", int64(a.Clock), int64(b.Clock)},
		integerField{"spell bonus", a.Bonus, b.Bonus}, integerField{"spell initial bonus", int64(a.Initial), int64(b.Initial)},
		integerField{"spell duration", int64(a.Duration), int64(b.Duration)}); why != "" {
		return why
	}
	if a.Active != b.Active || a.Capture != b.Capture || a.Survival != b.Survival {
		return fmt.Sprintf("spell active/capture/survival (%v,%v,%v) != (%v,%v,%v)", a.Active, a.Capture, a.Survival, b.Active, b.Capture, b.Survival)
	}
	return diffFloats(floatField{"spell clock F", a.ClockF, b.ClockF, d.eps})
}

func (d *differ) printExtended(f *trace.Frame) {
	x := f.Ext
	fmt.Printf("       ext hiscore=%d shown=%d hold=%d rate=%d rank=%d fixed=(%d,%d) timer=%d/%.3f invf=%.3f focus=%d\n",
		x.HiScore, x.ShownScore, x.FaithHold, x.FaithRate, x.Rank, x.PlayerIX, x.PlayerIY, x.StateClock, x.StateClockF, x.InvincF, x.Focus)
	fmt.Printf("       bomb=%v/%d/%.3f itemCalc=%d lasers=%d\n", x.BombActive, x.BombClock, x.BombClockF, x.ItemCalc, x.Lasers)
	if x.Spell != nil {
		fmt.Printf("       spell %+v\n", *x.Spell)
	}
	for i, row := range f.I {
		if i >= d.maxRows {
			break
		}
		fmt.Printf("       I %+v\n", row)
	}
	for i, row := range f.L {
		if i >= d.maxRows {
			break
		}
		fmt.Printf("       L %+v\n", row)
	}
}
