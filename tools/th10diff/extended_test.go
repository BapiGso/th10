package main

import (
	"bytes"
	"math"
	"strings"
	"testing"

	"th10/trace"
)

func extendedTestFrame() *trace.Frame {
	return &trace.Frame{Index: 1, PY: 400, PLife: 2, PPower: 500, PFaith: 50000, PScore: 123450, PState: 1, Items: 1,
		Ext: &trace.Extended{HiScore: 10000000, ShownScore: 123450, FaithHold: 130, FaithRate: 18, Rank: -512, PlayerIY: 40000,
			StateClock: 12, StateClockF: 12.25, InvincF: 6.5, Focus: 1, BombClock: 17, BombClockF: 17.25, ItemCalc: 1, Lasers: 1,
			Spell: &trace.SpellRow{ID: 93, Clock: 300, ClockF: 300.25, Bonus: 1234560, Initial: 1300000, Duration: 1800, Active: true, Capture: true}},
		I: []trace.ItemRow{{Slot: 2197, Type: 8, State: 5, X: 10, Y: 20, Clock: 3, ClockF: 3.25}},
		L: []trace.LaserRow{{ID: 8, Kind: 1, Type: 31, Color: 2, State: 3, X: 10, Y: 20, Angle: .5, Length: 80, Width: 16, Clock: 3, ClockF: 3.25}},
	}
}

func copyExtended(f *trace.Frame) *trace.Frame {
	c := *f
	x, s := *f.Ext, *f.Ext.Spell
	c.Ext, x.Spell = &x, &s
	c.I, c.L = append([]trace.ItemRow(nil), f.I...), append([]trace.LaserRow(nil), f.L...)
	return &c
}

func frameText(t *testing.T, f *trace.Frame) string {
	t.Helper()
	var b bytes.Buffer
	w := trace.NewWriter(&b)
	w.WriteFrame(f)
	if err := w.Flush(); err != nil {
		t.Fatal(err)
	}
	return b.String()
}

func extendedDiffer() *differ {
	return &differ{scope: "ext", eps: 1e-3 + posUnit + slack, angEps: 1e-4 + angUnit + slack}
}

func TestExtendedIntegerFieldsAreExact(t *testing.T) {
	cases := map[string]func(*trace.Frame){
		"life": func(f *trace.Frame) { f.PLife++ }, "power": func(f *trace.Frame) { f.PPower++ },
		"faith": func(f *trace.Frame) { f.PFaith++ }, "score": func(f *trace.Frame) { f.PScore++ },
		"player state": func(f *trace.Frame) { f.PState++ }, "invinc": func(f *trace.Frame) { f.PInvinc++ },
		"bomb timer": func(f *trace.Frame) { f.PBombTimer++ }, "player bullet count": func(f *trace.Frame) { f.PBullets++ },
		"hiscore": func(f *trace.Frame) { f.Ext.HiScore++ }, "hiscore continues": func(f *trace.Frame) { f.Ext.HiScoreContinues++ },
		"HUD score": func(f *trace.Frame) { f.Ext.ShownScore++ }, "faith hold": func(f *trace.Frame) { f.Ext.FaithHold++ },
		"faith rate": func(f *trace.Frame) { f.Ext.FaithRate++ }, "rank": func(f *trace.Frame) { f.Ext.Rank++ },
		"player fixed X": func(f *trace.Frame) { f.Ext.PlayerIX++ }, "player fixed Y": func(f *trace.Frame) { f.Ext.PlayerIY++ },
		"player timer": func(f *trace.Frame) { f.Ext.StateClock++ }, "focus": func(f *trace.Frame) { f.Ext.Focus++ },
		"bomb raw timer": func(f *trace.Frame) { f.Ext.BombClock++ }, "bomb active": func(f *trace.Frame) { f.Ext.BombActive = true },
		"item count": func(f *trace.Frame) { f.Items++ }, "item calc count": func(f *trace.Frame) { f.Ext.ItemCalc++ },
		"item[2197] slot": func(f *trace.Frame) { f.I[0].Slot-- }, "item[2197] type": func(f *trace.Frame) { f.I[0].Type++ },
		"item[2197] state": func(f *trace.Frame) { f.I[0].State++ }, "item[2197] clock": func(f *trace.Frame) { f.I[0].Clock++ },
		"laser count": func(f *trace.Frame) { f.Ext.Lasers++ }, "laser[0] ID": func(f *trace.Frame) { f.L[0].ID++ },
		"laser[0] kind": func(f *trace.Frame) { f.L[0].Kind = 0 }, "laser[0] type": func(f *trace.Frame) { f.L[0].Type++ },
		"laser[0] color": func(f *trace.Frame) { f.L[0].Color++ }, "laser[0] state": func(f *trace.Frame) { f.L[0].State++ },
		"laser[0] clock": func(f *trace.Frame) { f.L[0].Clock++ }, "spell ID": func(f *trace.Frame) { f.Ext.Spell.ID++ },
		"spell clock": func(f *trace.Frame) { f.Ext.Spell.Clock++ }, "spell bonus": func(f *trace.Frame) { f.Ext.Spell.Bonus++ },
		"spell initial bonus": func(f *trace.Frame) { f.Ext.Spell.Initial++ }, "spell duration": func(f *trace.Frame) { f.Ext.Spell.Duration++ },
		"spell active/capture/survival": func(f *trace.Frame) { f.Ext.Spell.Capture = false },
	}
	for field, change := range cases {
		t.Run(field, func(t *testing.T) {
			a := extendedTestFrame()
			b := copyExtended(a)
			change(b)
			if why := extendedDiffer().diffFrame(a, b); !strings.HasPrefix(why, field) {
				t.Fatalf("integer difference ignored/misreported: %q", why)
			}
		})
	}
}

func TestExtendedFloatsUseExistingBudgets(t *testing.T) {
	for _, field := range []struct {
		name  string
		get   func(*trace.Frame) *float64
		angle bool
	}{
		{"state F", func(f *trace.Frame) *float64 { return &f.Ext.StateClockF }, false},
		{"invinc F", func(f *trace.Frame) *float64 { return &f.Ext.InvincF }, false},
		{"bomb F", func(f *trace.Frame) *float64 { return &f.Ext.BombClockF }, false},
		{"item X", func(f *trace.Frame) *float64 { return &f.I[0].X }, false}, {"item Y", func(f *trace.Frame) *float64 { return &f.I[0].Y }, false},
		{"item F", func(f *trace.Frame) *float64 { return &f.I[0].ClockF }, false},
		{"laser X", func(f *trace.Frame) *float64 { return &f.L[0].X }, false}, {"laser Y", func(f *trace.Frame) *float64 { return &f.L[0].Y }, false},
		{"laser length", func(f *trace.Frame) *float64 { return &f.L[0].Length }, false}, {"laser width", func(f *trace.Frame) *float64 { return &f.L[0].Width }, false},
		{"laser angle", func(f *trace.Frame) *float64 { return &f.L[0].Angle }, true}, {"laser F", func(f *trace.Frame) *float64 { return &f.L[0].ClockF }, false},
		{"spell F", func(f *trace.Frame) *float64 { return &f.Ext.Spell.ClockF }, false},
	} {
		t.Run(field.name, func(t *testing.T) {
			inside, outside := .002, .0021
			if field.angle {
				inside, outside = .00011, .00012
			}
			a := extendedTestFrame()
			b := copyExtended(a)
			*field.get(b) += inside
			if why := extendedDiffer().diffFrame(a, b); why != "" {
				t.Fatalf("existing rounding budget rejected: %s", why)
			}
			b = copyExtended(a)
			*field.get(b) += outside
			if why := extendedDiffer().diffFrame(a, b); why == "" {
				t.Fatal("tolerance widened")
			}
		})
	}
	a := extendedTestFrame()
	b := copyExtended(a)
	b.L[0].Width = math.NaN()
	if extendedDiffer().diffFrame(a, b) == "" {
		t.Fatal("NaN is not a match")
	}
}

func TestExtendedScopesAndLegacyCompatibility(t *testing.T) {
	dir := t.TempDir()
	f := extendedTestFrame()
	changed := copyExtended(f)
	changed.Ext.Rank++
	a := writeTrace(t, dir, "a.txt", frameText(t, f))
	b := writeTrace(t, dir, "b.txt", frameText(t, changed))
	for _, scope := range []string{"all", "ext", "stats", "motion", "items", "lasers", "spell"} {
		want := 0
		if scope == "all" || scope == "ext" || scope == "stats" {
			want = 1
		}
		if code := runDiff(t, a, b, "-scope", scope); code != want {
			t.Fatalf("scope %s code=%d want %d", scope, code, want)
		}
	}
	legacy := copyExtended(f)
	legacy.Ext = nil
	legacy.I = nil
	legacy.L = nil
	b = writeTrace(t, dir, "legacy.txt", frameText(t, legacy))
	if code := runDiff(t, a, b, "-scope", "motion"); code != 0 {
		t.Fatalf("motion legacy compatibility: %d", code)
	}
	code, out := runDiffStdout(t, a, b, "-scope", "ext")
	if code != 2 || !strings.Contains(out, "requires -ext traces on both sides") {
		t.Fatalf("missing extension accepted: code=%d out=%s", code, out)
	}
}

func TestExtendedScopeSkipsMarkedPartialFrames(t *testing.T) {
	dir := t.TempDir()
	f := extendedTestFrame()
	half := copyExtended(f)
	half.Half = &trace.HalfRow{Reason: "unstable", Attempts: 4, Before: 1, After: 1}
	half.Ext.Rank++
	next := copyExtended(f)
	next.Index = 2
	a := writeTrace(t, dir, "a.txt", frameText(t, half)+frameText(t, next))
	b := writeTrace(t, dir, "b.txt", frameText(t, f)+frameText(t, next))
	code, out := runDiffStdout(t, a, b, "-scope", "ext")
	if code != 0 || !strings.Contains(out, "SKIP: 1 half-frame indices") || !strings.Contains(out, "MATCH: 1 frames compared") {
		t.Fatalf("ext half skip: code=%d out=%s", code, out)
	}
}
