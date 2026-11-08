package game

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"math"
	"slices"

	"th10/replay"
)

const (
	scoreHeaderSize      = 0x18    // FUN_0042b030 / FUN_0042b1e0
	scoreBlockSize       = 0x437c  // CR section, FUN_0042acb0
	scoreStatsSize       = 0x448   // ST section, FUN_0042add0
	scoreHistoryAt       = 0x59c   // DAT_0047783c+0x5a4 minus the first CR's +8
	scoreProgressAt      = 0x4d4   // stage records at object+0x4dc, index stage+difficulty*6
	maxScoreBytes        = 2 << 20 // host bound; FUN_0042b1e0 allocates a 2 MiB scratch buffer
	scoreKeyInc     byte = 0x35    // 0x42b05b / 0x42b32f
	scoreKey        byte = 0xac    // 0x42b062 / 0x42b333
)

// ScoreFile is the complete native score object except its two host pointers.
// Seven CRs (six shots and the total block) precede one ST. Raw bytes retain
// unmodeled fields, padding, music flags and the ST's 512 random words.
type ScoreFile struct {
	Header [scoreHeaderSize]byte
	Blocks [7][scoreBlockSize]byte
	Stats  [scoreStatsSize]byte
	// FUN_0042b030 skips CR/ST sections with a bad version or checksum,
	// leaving that section's initialized defaults in place.
	Rejected []string
}

// DefaultScoreFile ports FUN_0042ae60 -> FUN_0042acb0 / FUN_0042add0.
// The injected seed is for the opaque ST words, not the presentation RNG.
func DefaultScoreFile(seed uint16) *ScoreFile {
	s := &ScoreFile{}
	copy(s.Header[:], "TH10")
	binary.LittleEndian.PutUint16(s.Header[8:], 3)       // 0x42b1c1
	binary.LittleEndian.PutUint32(s.Header[0xc:], 0x100) // 0x42b1c9
	for shot := range s.Blocks {
		b := s.Blocks[shot][:]
		copy(b, "CR")
		binary.LittleEndian.PutUint32(b[8:], scoreBlockSize)
		for difficulty := range 5 {
			for rank := range 10 {
				at := 0x10 + difficulty*0xf0 + rank*0x18
				binary.LittleEndian.PutUint32(b[at:], uint32(1000000-rank*100000)) // 0x42acf0
				b[at+4] = 1
				copy(b[at+6:at+15], "--------\x00") // PE 0x46e858
			}
		}
		// FUN_0042acb0 (0x42ad50..0x42adb9) initializes the two reserved
		// spell words to id/difficulty. PE 0x4743c0 holds this exact table.
		for id := range SpellCount {
			diff := byte(4)
			if id < 50 {
				diff = [...]byte{2, 3, 0, 1}[id%4]
			} else if id < 97 {
				diff = [...]byte{1, 2, 3, 0}[(id-50)%4]
			}
			at := scoreHistoryAt + id*0x90
			binary.LittleEndian.PutUint32(b[at+0x88:], uint32(id))
			binary.LittleEndian.PutUint32(b[at+0x8c:], uint32(diff))
		}
	}
	copy(s.Stats[:], "ST")
	binary.LittleEndian.PutUint32(s.Stats[8:], scoreStatsSize)
	copy(s.Stats[0xc:0x15], "        \x00") // PE 0x46e354
	for at := 0x48; at < scoreStatsSize; at += 2 {
		// FUN_0042add0 (0x42ae20..0x42ae59) consumes single 16-bit steps.
		w := (seed ^ 0x9630) - 0x6553
		seed = w<<2 | w>>14
		binary.LittleEndian.PutUint16(s.Stats[at:], seed)
	}
	return s
}

// scoreChecksum is FUN_0042ac90, inlined by 0x42b0c3 / 0x42b126 and
// 0x42b253 / 0x42b2c0. It sums unsigned bytes starting at +8, not the header.
func scoreChecksum(section []byte) uint32 {
	var sum uint32
	for _, b := range section[8:] {
		sum += uint32(b)
	}
	return sum
}

func encodeScorePayload(s *ScoreFile) []byte {
	payload := make([]byte, 0, len(s.Blocks)*scoreBlockSize+scoreStatsSize)
	for index := range s.Blocks {
		b := s.Blocks[index]
		binary.LittleEndian.PutUint32(b[0xc:], uint32(index)) // FUN_0042b1e0, 0x42b248
		binary.LittleEndian.PutUint32(b[4:], scoreChecksum(b[:]))
		payload = append(payload, b[:]...)
	}
	stats := s.Stats
	binary.LittleEndian.PutUint32(stats[4:], scoreChecksum(stats[:]))
	return append(payload, stats[:]...)
}

// EncodeScoreFile is FUN_0042b1e0: CR/ST checksums, tree LZSS, then one
// XOR/permutation pass. Header +4 is the file size, not a checksum.
func EncodeScoreFile(s *ScoreFile) ([]byte, error) {
	if s == nil || string(s.Header[:4]) != "TH10" || binary.LittleEndian.Uint16(s.Header[8:]) != 3 {
		return nil, fmt.Errorf("game: invalid native score header")
	}
	for i := range s.Blocks {
		if string(s.Blocks[i][:2]) != "CR" || binary.LittleEndian.Uint32(s.Blocks[i][8:]) != scoreBlockSize {
			return nil, fmt.Errorf("game: invalid CR section %d", i)
		}
	}
	if string(s.Stats[:2]) != "ST" || binary.LittleEndian.Uint32(s.Stats[8:]) != scoreStatsSize {
		return nil, fmt.Errorf("game: invalid ST section")
	}
	payload := encodeScorePayload(s)
	compressed := replay.CompressLZSS(payload)
	replay.EncryptBlocks(compressed, scoreKeyInc, 0x10, scoreKey) // 0x42b335
	header := s.Header
	binary.LittleEndian.PutUint32(header[4:], uint32(scoreHeaderSize+len(compressed)))
	binary.LittleEndian.PutUint32(header[0x10:], uint32(len(compressed)))
	binary.LittleEndian.PutUint32(header[0x14:], uint32(len(payload)))
	return append(header[:], compressed...), nil
}

// DecodeScoreFile is FUN_0042b030. Bounds are checked before converting
// u32 lengths to int, including on js/wasm. Bad section checksums follow the
// binary's skip/default behavior; structural corruption returns an error.
func DecodeScoreFile(data []byte) (*ScoreFile, error) {
	return decodeScoreFile(data, 0)
}

func decodeScoreFile(data []byte, seed uint16) (*ScoreFile, error) {
	if len(data) < scoreHeaderSize || len(data) > maxScoreBytes || string(data[:4]) != "TH10" || binary.LittleEndian.Uint16(data[8:]) != 3 {
		return nil, fmt.Errorf("game: invalid native score header")
	}
	compressedLen := binary.LittleEndian.Uint32(data[0x10:])
	rawLen := binary.LittleEndian.Uint32(data[0x14:])
	if uint64(compressedLen) > uint64(len(data)-scoreHeaderSize) || rawLen > maxScoreBytes {
		return nil, fmt.Errorf("game: invalid native score lengths")
	}
	compressed := slices.Clone(data[scoreHeaderSize : scoreHeaderSize+int(compressedLen)])
	replay.DecryptBlocks(compressed, scoreKeyInc, 0x10, scoreKey) // 0x42b064
	payload, err := replay.DecompressLZSS(compressed, int(rawLen))
	if err != nil {
		return nil, fmt.Errorf("game: score decompression: %w", err)
	}
	s := DefaultScoreFile(seed)
	copy(s.Header[:], data[:scoreHeaderSize])
	for at := 0; at < len(payload); {
		if len(payload)-at < 12 {
			return nil, fmt.Errorf("game: truncated score section")
		}
		length := binary.LittleEndian.Uint32(payload[at+8:])
		if length < 12 || uint64(length) > uint64(len(payload)-at) {
			return nil, fmt.Errorf("game: invalid score section size %d", length)
		}
		section := payload[at : at+int(length)]
		tag := string(section[:2])
		if (tag != "CR" || length != scoreBlockSize) && (tag != "ST" || length != scoreStatsSize) {
			return nil, fmt.Errorf("game: invalid score section %q/%d", tag, length)
		}
		if binary.LittleEndian.Uint16(section[2:]) != 0 || binary.LittleEndian.Uint32(section[4:]) != scoreChecksum(section) {
			s.Rejected = append(s.Rejected, fmt.Sprintf("%s at %#x: version/checksum", tag, at))
		} else if tag == "CR" {
			index := binary.LittleEndian.Uint32(section[0xc:])
			if index >= uint32(len(s.Blocks)) {
				return nil, fmt.Errorf("game: invalid score shot %d", index)
			}
			copy(s.Blocks[index][:], section)
		} else {
			copy(s.Stats[:], section)
		}
		at += int(length)
	}
	return s, nil
}

// FUN_0042ae60 always constructs the ST before loading sections, consuming
// 512 main-generator steps even when the file contains a complete ST.
func scoreStartupSeed(seed uint16) uint16 {
	for range 512 {
		w := (seed ^ 0x9630) - 0x6553
		seed = w<<2 | w>>14
	}
	return seed
}

func scoreName(raw []byte) string {
	if at := bytes.IndexByte(raw, 0); at >= 0 {
		raw = raw[:at]
	}
	return string(raw)
}

func decodeScoreEntry(b []byte) ScoreEntry {
	r := ScoreEntry{
		Score: int32(binary.LittleEndian.Uint32(b)), Stage: b[4], Continues: b[5],
		Name: scoreName(b[6:15]), Timestamp: int32(binary.LittleEndian.Uint32(b[0x10:])),
		SlowRate: math.Float32frombits(binary.LittleEndian.Uint32(b[0x14:])),
	}
	copy(r.Raw[:], b)
	return r
}

func encodeScoreEntry(r ScoreEntry, b []byte) {
	copy(b, r.Raw[:])
	binary.LittleEndian.PutUint32(b, uint32(r.Score))
	b[4], b[5] = r.Stage, r.Continues
	if scoreName(b[6:15]) != r.Name {
		clear(b[6:15])
		copy(b[6:14], r.Name)
	}
	binary.LittleEndian.PutUint32(b[0x10:], uint32(r.Timestamp))
	binary.LittleEndian.PutUint32(b[0x14:], math.Float32bits(r.SlowRate))
}

func (s *ScoreFile) rankings(shot, difficulty int) (table [10]ScoreEntry) {
	for i := range table {
		at := 0x10 + difficulty*0xf0 + i*0x18
		table[i] = decodeScoreEntry(s.Blocks[shot][at : at+0x18])
	}
	return table
}

func (s *ScoreFile) setRankings(shot, difficulty int, table [10]ScoreEntry) {
	for i, r := range table {
		at := 0x10 + difficulty*0xf0 + i*0x18
		encodeScoreEntry(r, s.Blocks[shot][at:at+0x18])
	}
}

func (s *ScoreFile) history() *SpellHistory {
	h := &SpellHistory{}
	for shot := range s.Blocks {
		for id := range SpellCount {
			b := s.Blocks[shot][scoreHistoryAt+id*0x90:]
			r := &h.Total[id]
			if shot < 6 {
				r = &h.Shots[shot][id]
			}
			copy(r.Name[:], b[:0x80])
			r.Captures, r.Attempts = int32(binary.LittleEndian.Uint32(b[0x80:])), int32(binary.LittleEndian.Uint32(b[0x84:]))
			r.Reserved = [2]uint32{binary.LittleEndian.Uint32(b[0x88:]), binary.LittleEndian.Uint32(b[0x8c:])}
		}
	}
	return h
}

func (s *ScoreFile) setHistory(h *SpellHistory) {
	if h == nil {
		return
	}
	for shot := range s.Blocks {
		for id := range SpellCount {
			b := s.Blocks[shot][scoreHistoryAt+id*0x90:]
			r := &h.Total[id]
			if shot < 6 {
				r = &h.Shots[shot][id]
			}
			copy(b[:0x80], r.Name[:])
			binary.LittleEndian.PutUint32(b[0x80:], uint32(r.Captures))
			binary.LittleEndian.PutUint32(b[0x84:], uint32(r.Attempts))
			binary.LittleEndian.PutUint32(b[0x88:], r.Reserved[0])
			binary.LittleEndian.PutUint32(b[0x8c:], r.Reserved[1])
		}
	}
}
