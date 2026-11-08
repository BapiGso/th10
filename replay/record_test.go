package replay

import (
	"bytes"
	"encoding/binary"
	"testing"
)

func TestRecorderChunkAndSaveCount(t *testing.T) {
	r := NewRecorder(GlobalHeader{Character: 1, ShotType: 2, Difficulty: 3})
	if err := r.BeginChapter(Chapter{Index: 1, Seed: 0x1234, Faith: 5000, Lives: 2}); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 3601; i++ {
		r.Append(Record{Keys: uint16(i), Aux1: 1, Aux2: 2}, byte(i/30))
	}
	ch := &r.Replay().Chapters[0]
	if len(ch.SlowRates) != 121 || ch.SlowRates[120] != 120 {
		t.Fatalf("samples at chunk boundary: %v", ch.SlowRates)
	}
	r.PrepareSave()
	if ch.FrameCount != 3601 {
		t.Fatal("first save count")
	}
	r.PrepareSave()
	if ch.FrameCount != 7202 {
		t.Fatal("original repeat-save count behavior lost")
	}
	h, err := ChapterHeader(*ch)
	if err != nil || binary.LittleEndian.Uint32(h[8:]) != 3601*6+121 {
		t.Fatal("chunk body size")
	}
}

func TestStreamPrimeAndSamples(t *testing.T) {
	ch := Chapter{Records: make([]Record, 31), SlowRates: []byte{60, 59}}
	ch.Records[0] = Record{Keys: KeyShot | KeyLeft, Aux1: KeyShot}
	s := NewStream(ch)
	if s.Prime().Keys != KeyLeft {
		t.Fatal("held keys were not primed")
	}
	for i := 0; i < 31; i++ {
		rec, rate, ok := s.Next()
		if !ok || rec != ch.Records[i] || rate != ch.SlowRates[i/30] {
			t.Fatalf("record %d: %+v rate %d", i, rec, rate)
		}
	}
	if _, _, ok := s.Next(); ok || !s.Done() || s.Position() != 31 {
		t.Fatal("cursor did not finish")
	}
}

func TestRecorderRestartsChapter(t *testing.T) {
	r := NewRecorder(GlobalHeader{})
	if err := r.BeginChapter(Chapter{Index: 8}); err == nil {
		t.Fatal("accepted stage 8")
	}
	ch := Chapter{Index: 1, Seed: 1, Raw: make([]byte, chapterHeaderSize)}
	ch.Raw[0x40] = 99
	if err := r.BeginChapter(ch); err != nil {
		t.Fatal(err)
	}
	r.Append(Record{Keys: 1}, 60)
	if err := r.BeginChapter(ch); err != nil {
		t.Fatal(err)
	}
	if len(r.Replay().Chapters) != 1 || len(r.Replay().Chapters[0].Records) != 0 {
		t.Fatal("restart appended a duplicate chapter")
	}
	ch.Raw[0x40] = 0
	if bytes.Equal(ch.Raw, r.Replay().Chapters[0].Raw) {
		t.Fatal("snapshot aliases caller storage")
	}
}
