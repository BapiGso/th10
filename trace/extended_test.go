package trace

import (
	"bytes"
	"reflect"
	"strings"
	"testing"
)

func extendedFixture() *Frame {
	return &Frame{Index: 7, PY: 400, PLife: 2, PPower: 500, PScore: 9999999990, PFaith: 999990, PState: 1, Items: 2,
		Ext: &Extended{HiScore: 10000000, HiScoreContinues: 2, ShownScore: 123450, FaithHold: 130, FaithRate: 18, Rank: -512,
			PlayerIX: -1855, PlayerIY: 42000, StateClock: 12, StateClockF: 12.25, InvincF: 6.5, Focus: 1,
			BombActive: true, BombClock: 17, BombClockF: 17.25, ItemCalc: 1, Lasers: 2,
			Spell: &SpellRow{ID: 93, Clock: 300, ClockF: 300.25, Bonus: 1234560, Initial: 1300000, Duration: 1800, Active: true, Capture: true}},
		I: []ItemRow{{Slot: 149, Type: 1, State: 1, X: -10, Y: 20, Clock: 1, ClockF: 1.25}, {Slot: 2197, Type: 8, State: 5, X: 1, Y: 2}},
		L: []LaserRow{{ID: 8, Kind: 0, Type: 31, Color: 2, State: 1, X: 10, Y: 20, Angle: .5, Length: 80, Width: 16, Clock: 3, ClockF: 3.25},
			{ID: 9, Kind: 1, Type: 6, Color: 7, State: 3, X: 30, Y: 40, Angle: 1, Length: 448, Width: 2}},
	}
}

func encodeFixture(t *testing.T, f *Frame) string {
	t.Helper()
	var out bytes.Buffer
	w := NewWriter(&out)
	w.WriteFrame(f)
	if err := w.Flush(); err != nil {
		t.Fatal(err)
	}
	return out.String()
}

func TestExtendedRoundTripAllFields(t *testing.T) {
	want := extendedFixture()
	text := encodeFixture(t, want)
	if !strings.Contains(text, "Z 7 e 0 b 0 i 2 l 2 s 1\n") {
		t.Fatalf("missing commit: %s", text)
	}
	got, err := NewReader(strings.NewReader(text)).Next()
	if err != nil {
		t.Fatal(err)
	}
	// Parser bookkeeping is intentionally not a producer field.
	got.extCommitted, got.extSpell, got.extItems, got.extLasers = false, false, 0, 0
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("round trip\ngot  %+v\nwant %+v", got, want)
	}
}

func TestExtendedReaderRejectsMissingRowsAndCommit(t *testing.T) {
	text := encodeFixture(t, extendedFixture())
	lines := strings.SplitAfter(text, "\n")
	for _, tag := range []string{"X ", "S ", "I 2197 ", "L 9 ", "Z "} {
		var broken strings.Builder
		for _, line := range lines {
			if !strings.HasPrefix(line, tag) {
				broken.WriteString(line)
			}
		}
		if _, err := NewReader(strings.NewReader(broken.String())).Next(); err == nil {
			t.Fatalf("accepted missing %q", tag)
		}
	}
	for _, broken := range []string{
		strings.Replace(text, "X 1 ", "X 2 ", 1), strings.Replace(text, "irows 2", "irows 2199", 1),
		strings.Replace(text, "L 9 kind 1", "L 9 kind 2", 1), strings.Replace(text, "I 2197", "I 149", 1),
		strings.Replace(text, "Z 7", "Z 8", 1), strings.Replace(text, "Z 7 e 0 b 0 i 2", "Z 7 e 0 b 0 i 1", 1),
		text + "H unstable attempts 2 before 7 after 7\n", text + "E 0 0 hp 1 act 1 type 0\n",
	} {
		if _, err := NewReader(strings.NewReader(broken)).Next(); err == nil {
			t.Fatalf("accepted malformed extension: %s", broken)
		}
	}
}

func TestMarkedPartialExtendedFrameIsDiagnostic(t *testing.T) {
	f := extendedFixture()
	f.Half = &HalfRow{Reason: "read-error", Attempts: 4, Before: 7, After: 7}
	f.I, f.L = f.I[:1], f.L[:1]
	got, err := NewReader(strings.NewReader(encodeFixture(t, f))).Next()
	if err != nil || got == nil || got.Half == nil {
		t.Fatalf("marked partial block: frame=%v err=%v", got, err)
	}
	f.Half = nil
	if _, err := NewReader(strings.NewReader(encodeFixture(t, f))).Next(); err == nil {
		t.Fatal("unmarked partial block accepted")
	}
}

func TestExtendedEntityRowsAreNotBoundedToOldBulletLimit(t *testing.T) {
	f := &Frame{Index: 1, Items: MaxItemRows, Ext: &Extended{Lasers: MaxLaserRows}}
	for slot := 0; slot < MaxItemRows; slot++ {
		f.I = append(f.I, ItemRow{Slot: slot, Type: 8, State: 5})
	}
	for id := 1; id <= MaxLaserRows; id++ {
		f.L = append(f.L, LaserRow{ID: id, Kind: id % 2, State: 2})
	}
	got, err := NewReader(strings.NewReader(encodeFixture(t, f))).Next()
	if err != nil || len(got.I) != MaxItemRows || len(got.L) != MaxLaserRows {
		t.Fatalf("full entities: frame=%v err=%v", got, err)
	}
}
