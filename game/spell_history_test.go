package game

import (
	"testing"
	"unsafe"
)

func TestSpellRecordLayout(t *testing.T) {
	var r SpellRecord
	if unsafe.Sizeof(r) != 0x90 || unsafe.Offsetof(r.Captures) != 0x80 || unsafe.Offsetof(r.Attempts) != 0x84 || unsafe.Offsetof(r.Reserved) != 0x88 {
		t.Fatalf("record size/offsets = %#x/%#x/%#x/%#x", unsafe.Sizeof(r), unsafe.Offsetof(r.Captures), unsafe.Offsetof(r.Attempts), unsafe.Offsetof(r.Reserved))
	}
}

func TestSpellHistoryShotAndTotal(t *testing.T) {
	h := &SpellHistory{}
	for char := range 2 {
		for shot := range 3 {
			s := NewState(char, shot, DiffNormal)
			s.SpellHistory = h
			s.RecordSpellDeclaration(109, "\x90\x4d\x95\x84")
			s.RecordSpellCapture(109)
			if caps, seen := s.SpellCounts(109); caps != 1 || seen != 1 {
				t.Errorf("char %d shot %d counts = %d/%d", char, shot, caps, seen)
			}
		}
	}
	if r := h.Total[109]; r.Captures != 6 || r.Attempts != 6 || string(r.Name[:5]) != "\x90\x4d\x95\x84\x00" {
		t.Fatalf("total = %+v", r)
	}
	if h.Total[108] != (SpellRecord{}) {
		t.Fatal("another card's record changed")
	}
}

func TestSpellHistoryModeCheck(t *testing.T) {
	// The two binary writers test exactly mode != 1 (ReplayMode).
	for _, replay := range []bool{false, true} {
		s := NewState(CharReimu, ShotA, DiffEasy)
		if replay {
			s.ReplayMode = 1
		}
		s.RecordSpellDeclaration(2, "card")
		s.RecordSpellCapture(2)
		if replay {
			if s.SpellHistory != nil {
				t.Fatal("replay writes allocated history")
			}
			continue
		}
		if caps, seen := s.SpellCounts(2); caps != 1 || seen != 1 {
			t.Errorf("replay %v counts = %d/%d", replay, caps, seen)
		}
	}
	// Disabling recording leaves existing records, including names, intact.
	s := NewState(CharReimu, ShotA, DiffEasy)
	s.RecordSpellDeclaration(2, "before")
	r := s.SpellHistory.Shots[0][2]
	s.ReplayMode = 1
	s.RecordSpellDeclaration(2, "after")
	s.RecordSpellCapture(2)
	if s.SpellHistory.Shots[0][2] != r || s.SpellHistory.Total[2] != r {
		t.Fatal("replay overwrote an existing record")
	}
}

func TestSpellHistoryCounterCapsAndNameTail(t *testing.T) {
	s := NewState(CharMarisa, ShotB, DiffHard)
	s.SpellHistory = &SpellHistory{}
	a := &s.SpellHistory.Shots[4][60]
	b := &s.SpellHistory.Total[60]
	*a = SpellRecord{Captures: 99998, Attempts: 99999, Reserved: [2]uint32{0x12345678, 0x87654321}}
	*b = SpellRecord{Captures: 99999, Attempts: 99998, Reserved: [2]uint32{1, 2}}
	copy(a.Name[:], "old name")
	s.RecordSpellDeclaration(60, "new\x00ignored")
	s.RecordSpellCapture(60)
	s.RecordSpellDeclaration(60, "new")
	s.RecordSpellCapture(60)
	if a.Attempts != 99999 || a.Captures != 99999 || b.Attempts != 99999 || b.Captures != 99999 {
		t.Fatalf("caps: shot=%+v total=%+v", a, b)
	}
	if string(a.Name[:8]) != "new\x00name" {
		t.Fatalf("name copy cleared the old tail: %q", a.Name[:8])
	}
	if a.Reserved != [2]uint32{0x12345678, 0x87654321} || b.Reserved != [2]uint32{1, 2} {
		t.Fatal("unknown record fields changed")
	}
}

func TestSpellHistoryInvalidIndices(t *testing.T) {
	for _, c := range []struct {
		char, shot int
		id         int32
	}{{-1, 0, 0}, {2, 0, 0}, {0, -1, 0}, {0, 3, 0}, {0, 0, -1}, {0, 0, 110}} {
		s := NewState(c.char, c.shot, DiffEasy)
		s.RecordSpellDeclaration(c.id, "invalid")
		s.RecordSpellCapture(c.id)
		if s.SpellHistory != nil {
			t.Errorf("invalid index %+v allocated history", c)
		}
	}
	var s *GameState
	s.RecordSpellDeclaration(0, "nil")
	s.RecordSpellCapture(0)
	if caps, seen := s.SpellCounts(0); caps != 0 || seen != 0 {
		t.Fatal("nil state's counts are nonzero")
	}
}
