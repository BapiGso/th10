package game

import (
	"bytes"
	"encoding/binary"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"th10/replay"
	"th10/zun"
)

func TestOriginalScoreFileRoundtrip(t *testing.T) {
	path := filepath.Join("..", ".tmp", "[th10]", "scoreth10.dat")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Skipf("read-only original score fixture missing: %v", err)
	}
	s, err := DecodeScoreFile(raw)
	if err != nil {
		t.Fatal(err)
	}
	if len(s.Rejected) != 0 {
		t.Fatalf("original file contains rejected sections: %v", s.Rejected)
	}
	encoded, err := EncodeScoreFile(s)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(raw, encoded) {
		t.Fatalf("original file roundtrip differs: %d vs %d bytes", len(raw), len(encoded))
	}
	viewBytes, err := saveView(s).nativeBytes()
	if err != nil || !bytes.Equal(raw, viewBytes) {
		t.Fatalf("original typed-view roundtrip differs: %v", err)
	}
	t.Logf("original scoreth10.dat: %d bytes identical, all CR/ST sections accepted", len(raw))
}

func scoreTestContainer(s *ScoreFile, payload []byte) []byte {
	compressed := replay.CompressLZSS(payload)
	replay.EncryptBlocks(compressed, scoreKeyInc, 0x10, scoreKey)
	header := s.Header
	binary.LittleEndian.PutUint32(header[4:], uint32(scoreHeaderSize+len(compressed)))
	binary.LittleEndian.PutUint32(header[0x10:], uint32(len(compressed)))
	binary.LittleEndian.PutUint32(header[0x14:], uint32(len(payload)))
	return append(header[:], compressed...)
}

func TestScoreFileByteRoundtrip(t *testing.T) {
	s := DefaultScoreFile(0x4321)
	s.Header[0xa], s.Header[0xb] = 0x12, 0x34
	for shot := range s.Blocks {
		// Keep opaque bytes, the ranking name's NUL tail, and its padding.
		s.Blocks[shot][0x4ba] = byte(0xa0 + shot)
		copy(s.Blocks[shot][0x16:0x1f], "A\x00TAIL!!\x00")
		s.Blocks[shot][0x1f] = byte(0xd0 + shot)
	}
	s.Stats[0x27] = 0xaa
	a, err := EncodeScoreFile(s)
	if err != nil {
		t.Fatal(err)
	}
	if string(a[:4]) != "TH10" || binary.LittleEndian.Uint32(a[4:]) != uint32(len(a)) || binary.LittleEndian.Uint32(a[0x14:]) != 0x1dcac {
		t.Fatal("native header/layout differs")
	}
	decoded, err := DecodeScoreFile(a)
	if err != nil {
		t.Fatal(err)
	}
	b, err := EncodeScoreFile(decoded)
	if err != nil || !bytes.Equal(a, b) {
		t.Fatalf("file roundtrip differs: %v", err)
	}
	// Typed views must not destroy name tails, padding, ST flags or records.
	view := saveView(decoded)
	c, err := view.nativeBytes()
	if err != nil || !bytes.Equal(a, c) {
		t.Fatalf("typed-view roundtrip differs: %v", err)
	}
}

func TestScoreOriginalCryptoAndDefaults(t *testing.T) {
	data := make([]byte, 16)
	for i := range data {
		data[i] = byte(i)
	}
	// 0x42b333: AL=ac, increment=35, block=10. Source order for inverse
	// permutation is 15,13,...,1,14,12,...,0; the key advances each byte.
	want := []byte{0xa3, 0xec, 0x1d, 0x42, 0x87, 0xb0, 0xe9, 0x1e, 0x5a, 0x85, 0xb4, 0xfb, 0x2e, 0x59, 0x90, 0xc7}
	replay.EncryptBlocks(data, scoreKeyInc, 16, scoreKey)
	if !bytes.Equal(data, want) {
		t.Fatalf("original cipher vector = %x, want %x", data, want)
	}
	replay.DecryptBlocks(data, scoreKeyInc, 16, scoreKey)
	for i, b := range data {
		if b != byte(i) {
			t.Fatal("inverse cipher differs")
		}
	}
	section := make([]byte, 12)
	copy(section[8:], []byte{1, 2, 0x80, 0xff})
	if scoreChecksum(section) != 386 {
		t.Fatal("checksum did not sum unsigned bytes from +8")
	}
	s := DefaultScoreFile(0)
	table := s.rankings(0, 0)
	if table[0].Score != 1000000 || table[9].Score != 100000 || table[0].Name != "--------" || table[0].Stage != 1 {
		t.Fatal("FUN_0042acb0 default ranking differs")
	}
	h := s.history()
	for _, tc := range []struct{ id, diff int }{{0, 2}, {49, 3}, {50, 1}, {96, 3}, {97, 4}, {109, 4}} {
		if h.Shots[0][tc.id].Reserved != [2]uint32{uint32(tc.id), uint32(tc.diff)} {
			t.Fatalf("PE 0x4743c0 metadata differs at card %d", tc.id)
		}
	}
	rng := &zun.RNG{}
	for at := 0x48; at < scoreStatsSize; at += 4 {
		n := rng.Raw()
		if binary.LittleEndian.Uint16(s.Stats[at:]) != uint16(n>>16) || binary.LittleEndian.Uint16(s.Stats[at+2:]) != uint16(n) {
			t.Fatal("FUN_0042add0 single-step ST words differ")
		}
	}
	if scoreStartupSeed(0) != rng.State {
		t.Fatal("startup main seed did not include ST initialization")
	}
}

func TestScoreSectionChecksAndBounds(t *testing.T) {
	s := DefaultScoreFile(123)
	s.Blocks[0][0x900], s.Blocks[1][0x900] = 0x55, 0x66
	payload := encodeScorePayload(s)
	payload[scoreBlockSize+0x900] ^= 1 // deliberately leave CR 1's old checksum
	d, err := DecodeScoreFile(scoreTestContainer(s, payload))
	if err != nil || len(d.Rejected) != 1 || d.Blocks[0][0x900] != 0x55 || d.Blocks[1][0x900] != 0 {
		t.Fatalf("bad CR was not skipped independently: %v", err)
	}
	payload = encodeScorePayload(s)
	payload[2] = 1 // original CR version is exactly 0
	d, err = DecodeScoreFile(scoreTestContainer(s, payload))
	if err != nil || len(d.Rejected) != 1 || d.Blocks[0][0x900] != 0 || d.Blocks[1][0x900] != 0x66 {
		t.Fatal("nonzero section version was not skipped")
	}
	payload = encodeScorePayload(s)
	binary.LittleEndian.PutUint32(payload[0xc:], 7)
	binary.LittleEndian.PutUint32(payload[4:], scoreChecksum(payload[:scoreBlockSize]))
	if _, err := DecodeScoreFile(scoreTestContainer(s, payload)); err == nil {
		t.Fatal("out-of-range CR index was accepted")
	}
	valid, err := EncodeScoreFile(s)
	if err != nil {
		t.Fatal(err)
	}
	for _, at := range []int{0x10, 0x14} {
		bad := bytes.Clone(valid)
		binary.LittleEndian.PutUint32(bad[at:], 0xffffffff)
		if _, err := DecodeScoreFile(bad); err == nil {
			t.Fatalf("u32 length overflow at %#x accepted", at)
		}
	}
	if _, err := DecodeScoreFile(valid[:len(valid)-1]); err == nil {
		t.Fatal("truncated compressed body accepted")
	}
}

func TestScoreSharedSpellHistoryAndProgress(t *testing.T) {
	store := NewMemoryScoreStore()
	g := &Game{Scores: store}
	g.LoadProgress()
	a := g.NewState(1, 2, DiffNormal)
	a.Stage = 1
	g.BeginRun(a)
	a.RecordSpellDeclaration(109, "\x90\x4d\x95\x84")
	a.RecordSpellCapture(109)
	a.RecordPlayFrame()
	b := g.NewState(1, 2, DiffExtra)
	if a.SpellHistory != b.SpellHistory || a.SpellHistory != g.SpellHistory {
		t.Fatal("history is not application-owned across runs")
	}
	b.RecordSpellDeclaration(109, "\x90\x4d\x95\x84")
	a.Practice(3)
	a.Score = 1234560
	a.RecordStageClear()
	g.RecordPracticeScore(a)
	if err := g.SaveProgress(); err != nil {
		t.Fatal(err)
	}
	fresh := &Game{Scores: store}
	fresh.LoadProgress()
	c := fresh.NewState(1, 2, DiffNormal)
	if captures, attempts := c.SpellCounts(109); captures != 1 || attempts != 2 || !reflect.DeepEqual(g.SpellHistory, fresh.SpellHistory) {
		t.Fatal("spell history did not survive reload")
	}
	p := fresh.Save.Progress[5]
	if p.Runs != 1 || p.Frames != 1 || p.Stages[1][2].Score != 123456 || p.Stages[1][2].Available != 1 || p.Stages[1][2].Cleared != 1 {
		t.Fatalf("native counters/stage flags differ: %+v", p)
	}
	c.ScoreProgress[5].Frames = 215999999
	c.RecordPlayFrame()
	if c.ScoreProgress[5].Frames != 215999999 {
		t.Fatal("play-time counter did not saturate")
	}
}

func TestScoreReplayFlagsArePrivateAndRankingMovesRaw(t *testing.T) {
	store := NewMemoryScoreStore()
	g := &Game{Scores: store}
	g.LoadProgress()
	s := g.NewState(0, 0, DiffNormal)
	s.Stage, s.StageMode, s.ReplayMode = 3, 1, 1
	g.BindState(s)
	want := g.Save.Progress
	s.RecordStageClear()
	s.RecordPlayFrame()
	if g.Save.Progress != want || s.ScoreProgress[0].Stages[1][2].Cleared != 1 || s.ScoreProgress[0].Frames != 0 {
		t.Fatal("replay stage flags were lost or wrote into the saved view")
	}
	table := g.ScoreTable(s)
	table[0].Raw[0xf] = 0xab
	table[0].Raw[6], table[0].Raw[7], table[0].Raw[8] = 'A', 0, 'Z'
	table[0].Name = "A"
	s.Score = 20000000
	if g.InsertRanking(s, false, 456, 1) != 0 {
		t.Fatal("new score did not insert at the top")
	}
	if err := g.SaveProgress(); err != nil {
		t.Fatal(err)
	}
	raw, err := store.Read()
	if err != nil {
		t.Fatal(err)
	}
	native, err := DecodeScoreFile(raw)
	if err != nil {
		t.Fatal(err)
	}
	r := native.rankings(0, DiffNormal)[1]
	if r.Raw[0xf] != 0xab || r.Raw[8] != 'Z' || r.Name != "A" {
		t.Fatal("ranking shift did not retain all six original dwords")
	}
}
