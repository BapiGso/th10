package sht

import (
	"fmt"
	"testing"

	"th10/gamedata"
)

func TestAllTH10ShotCallbackTables(t *testing.T) {
	seenInit, seenUpdate, seenKind := map[uint32]bool{}, map[uint32]bool{}, map[int]bool{}
	for character := range 2 {
		for shot := range 3 {
			name := fmt.Sprintf("sht/pl%02d%c.sht", character, 'a'+shot)
			data, err := gamedata.ReadFile(name)
			if err != nil {
				t.Fatal(err)
			}
			f, err := Parse(data)
			if err != nil {
				t.Fatal(err)
			}
			if err := f.ValidateTH10(); err != nil {
				t.Fatalf("%s: %v", name, err)
			}
			for _, group := range f.Groups {
				for _, s := range group {
					seenInit[s.Callbacks[0]], seenUpdate[s.Callbacks[1]], seenKind[s.Kind] = true, true, true
					if s.Callbacks[2] != 0 || s.Callbacks[3] != 0 {
						t.Fatal("a shipped SHT uses a nonzero index into the .bss callback placeholders")
					}
				}
			}
		}
	}
	if len(seenInit) != 2 || !seenInit[0] || !seenInit[1] || len(seenUpdate) != 3 ||
		!seenUpdate[0] || !seenUpdate[1] || !seenUpdate[2] || len(seenKind) != 3 ||
		!seenKind[0] || !seenKind[2] || !seenKind[3] {
		t.Fatalf("tables: init=%v update=%v kind=%v", seenInit, seenUpdate, seenKind)
	}
}

func TestRejectUnknownTH10ShotSemantics(t *testing.T) {
	base := Shot{Source: 1, Kind: 0}
	for _, s := range []Shot{
		{Source: 1, Kind: 1},
		{Source: 1, Callbacks: [4]uint32{2, 0, 0, 0}},
		{Source: 1, Callbacks: [4]uint32{0, 3, 0, 0}},
		{Source: 1, Callbacks: [4]uint32{0, 0, 1, 0}},
		{Source: 1, Callbacks: [4]uint32{0, 0, 0, 1}},
		{Source: 0, Kind: 3, Callbacks: [4]uint32{0, 2, 0, 0}},
		{Source: 2},
	} {
		f := &File{}
		f.Groups[1] = []Shot{s}
		if err := f.ValidateTH10(); err == nil {
			t.Fatalf("accepted unsupported record %+v", s)
		}
	}
	f := &File{}
	f.Groups[1] = []Shot{base}
	if err := f.ValidateTH10(); err != nil {
		t.Fatal(err)
	}
}
