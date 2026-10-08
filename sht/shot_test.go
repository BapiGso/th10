package sht

import (
	"encoding/binary"
	"testing"
	"th10/assets"
)

func reimuData(t *testing.T) []byte {
	t.Helper()
	d, err := assets.Assets.ReadFile("sht/pl00a.sht")
	if err != nil {
		t.Fatal(err)
	}
	return d
}

func TestOriginalReimuAShotGroups(t *testing.T) {
	f, err := Parse(reimuData(t))
	if err != nil {
		t.Fatal(err)
	}
	if err = f.ValidateReimuA(); err != nil {
		t.Fatal(err)
	}
	mainDamage := []int{15, 14, 13, 11, 12}
	for level := 0; level < 5; level++ {
		for _, focus := range []bool{false, true} {
			shots := f.Shots(level*100, focus)
			if len(shots) != level+2 {
				t.Fatalf("level %d count %d", level, len(shots))
			}
			for i, s := range shots {
				if i < 2 {
					if s.Damage != mainDamage[level] || s.Speed != 24 || s.Width != 18 || s.Height != 48 || s.Interval != 3 || s.Source != 0 || s.Script != 5 {
						t.Fatalf("main %+v", s)
					}
				} else {
					if s.Source != i-1 || s.Speed != 12 || s.Interval != 5 || s.Script != 7 || s.Callbacks != [4]uint32{1, 1, 0, 0} {
						t.Fatalf("homing %+v", s)
					}
				}
			}
		}
	}
	for i, s := range f.Shots(500, false) {
		if s != f.Shots(400, false)[i] {
			t.Fatal("5.00 must use 4-option group")
		}
	}
}

func TestOriginalOptionLayouts(t *testing.T) {
	f, err := Parse(reimuData(t))
	if err != nil {
		t.Fatal(err)
	}
	want := []Offset{{-3200, 800}, {3200, 800}, {-1600, 3200}, {1600, 3200}}
	for i, o := range f.OptionOffsets(400, false) {
		if o != want[i] {
			t.Fatalf("option %d: %+v", i, o)
		}
	}
	focused := f.OptionOffsets(100, true)
	if len(focused) != 1 || focused[0] != (Offset{0, 2000}) {
		t.Fatal(focused)
	}
	if len(f.OptionOffsets(0, false)) != 0 {
		t.Fatal("options at zero power")
	}
}

func TestMalformedShots(t *testing.T) {
	data := reimuData(t)
	for _, n := range []int{32, 0x10f, 0x15f, len(data) - 1} {
		if _, err := Parse(data[:n]); err == nil {
			t.Fatalf("accepted truncated file %d", n)
		}
	}
	for _, tc := range []struct {
		at    int
		value uint32
	}{
		{0x110, 0xffffffff}, {0x110, 0x20}, {0x160, 0}, {0x160, 0x000f0303}, {0x17c, 5}, {0x16c, 0x7fc00000},
	} {
		bad := append([]byte(nil), data...)
		binary.LittleEndian.PutUint32(bad[tc.at:], tc.value)
		if _, err := Parse(bad); err == nil {
			t.Fatalf("accepted bad record %#x=%#x", tc.at, tc.value)
		}
	}
	bad := append([]byte(nil), data...)
	binary.LittleEndian.PutUint32(bad[0x234+0x28:], 2)
	f, err := Parse(bad)
	if err != nil {
		t.Fatal(err)
	}
	if f.ValidateReimuA() == nil {
		t.Fatal("unsupported callback silently accepted")
	}
}
