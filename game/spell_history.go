package game

// Spell history at DAT_0047783c, written by FUN_00409280 (declarations)
// and FUN_00409c00 (captures). This is only the in-memory record table;
// its on-disk encoding belongs to the save layer.

import "strings"

const (
	SpellCount = 110 // ECL card ids 0x00-0x6d; each record is 0x90 bytes
	spellShots = 6   // DAT_00474c68*3 + DAT_00474c6c
	spellCap   = 99999
)

// SpellRecord is one 0x90-byte record. Name contains the decrypted ECL
// name's Shift-JIS bytes, not the UTF-8 name the GUI displays. Unknown
// fields are retained for the save layer, not rewritten by spell logic.
type SpellRecord struct {
	Name     [0x80]byte // +0x00
	Captures int32      // +0x80 (save +0x624 for shot 0/card 0)
	Attempts int32      // +0x84 (save +0x628 for shot 0/card 0)
	Reserved [2]uint32  // +0x88, +0x8c: not written by these two functions
}

// SpellHistory keeps the seven record arrays, omitting unrelated save
// fields between them. Shots[character*3+shot][id] is at save offset
// 0x5a4 + (character*3+shot)*0x437c + id*0x90; Total[id] at
// 0x19a8c + id*0x90. Do not serialize the Go table as the entire save.
type SpellHistory struct {
	Shots [spellShots][SpellCount]SpellRecord
	Total [SpellCount]SpellRecord
}

func (s *GameState) spellRecords(id int32, create bool) (*SpellRecord, *SpellRecord) {
	if id < 0 || id >= SpellCount || s.Character < 0 || s.Character > 1 || s.ShotType < 0 || s.ShotType > 2 {
		return nil, nil
	}
	if s.SpellHistory == nil {
		if !create {
			return nil, nil
		}
		s.SpellHistory = &SpellHistory{}
	}
	return &s.SpellHistory.Shots[s.Character*3+s.ShotType][id], &s.SpellHistory.Total[id]
}

// declare copies through the NUL byte, leaving the old name's tail alone,
// as FUN_00409280's two byte-copy loops do. Embedded ECL names fit in 0x80.
func (r *SpellRecord) declare(name string) {
	if at := strings.IndexByte(name, 0); at >= 0 {
		name = name[:at]
	}
	n := copy(r.Name[:len(r.Name)-1], name)
	r.Name[n] = 0
	if r.Attempts < spellCap {
		r.Attempts++
	}
}

// RecordSpellDeclaration is the history part of FUN_00409280. The binary
// checks mode != 1, independently of the card's capture flag.
func (s *GameState) RecordSpellDeclaration(id int32, name string) {
	if s == nil || s.ReplayMode == 1 {
		return
	}
	shot, total := s.spellRecords(id, true)
	if shot != nil {
		shot.declare(name)
		total.declare(name)
	}
}

// RecordSpellCapture is the history part of FUN_00409c00. Both counters
// saturate separately at 99999; neither name nor attempts are changed.
func (s *GameState) RecordSpellCapture(id int32) {
	if s == nil || s.ReplayMode == 1 {
		return
	}
	shot, total := s.spellRecords(id, true)
	if shot != nil {
		if shot.Captures < spellCap {
			shot.Captures++
		}
		if total.Captures < spellCap {
			total.Captures++
		}
	}
}

// SpellCounts supplies the shot-specific counts read by FUN_00408d60's
// GUI tail. The display caps each at 99, without changing these counters.
func (s *GameState) SpellCounts(id int32) (captures, attempts int32) {
	if s == nil {
		return 0, 0
	}
	if shot, _ := s.spellRecords(id, false); shot != nil {
		return shot.Captures, shot.Attempts
	}
	return 0, 0
}
