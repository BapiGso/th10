// Package replay parses TH10 replay files (".rpy", magic "t10r"), the official
// deterministic input streams that drive the differential harness.
//
// Format, reverse-engineered from th10.exe (FUN_00428f60 writer, FUN_0042a200
// loader, FUN_0042a8a0 record append, FUN_004297d0 per-frame consumer):
//
//	header (36 B):
//	  +0x00 char[4]  "t10r"
//	  +0x04 u32      version (=5)
//	  +0x0c u32      offset of the trailer (file size - 200)
//	  +0x1c u32      compressed payload length
//	  +0x20 u32      decompressed payload length
//	  +0x24 ...      compressed payload (2x XOR transform + ZUN LZSS)
//	  tail 200 B     plaintext metadata (player name, version, date, chara, rank, score)
//
// The decompressed payload starts with a 100-byte global header, then up to six
// chapter blocks, each 0x1c4-byte header + N 6-byte input records. A record is
// {u16 keys, u16 x, u16 y} where x/y carry stage-relative player position for
// the chapter's deterministic playback.
package replay

import (
	"encoding/binary"
	"errors"
	"fmt"
	"strings"

	"golang.org/x/text/encoding/japanese"
)

// Magic is the 4-byte file signature ("t10r" little-endian dword 0x72303174).
const Magic = "t10r"

// Version is the only replay format version TH10 writes.
const Version = 5

// trailerSize is the plaintext metadata block at the end of every .rpy.
const trailerSize = 200

// globalHeaderSize is the fixed prefix of the decompressed payload.
const globalHeaderSize = 100

// chapterHeaderSize is the per-chapter header length.
const chapterHeaderSize = 0x1c4

// recordSize is the per-frame input record length.
const recordSize = 6

// MaxChapters is the number of chapters (stage runs) a replay may contain.
const MaxChapters = 6

// Key bits in a record's key word. The replay key word is masked with 0x01f7
// by FUN_004297d0, so only these bits are ever recorded. Directions use a
// two-of-four encoding: 0x10/0x20 mark the horizontal/vertical axis as active,
// and 0x80/0x40 carry the up-or-left / down-or-right sign.
const (
	KeyShot  uint16 = 0x001 // Z: fire (also advances dialog)
	KeyBomb  uint16 = 0x002 // X: spirit attack / deathbomb (FUN_00425730)
	KeyFocus uint16 = 0x004 // Shift: focused movement
	KeySkip  uint16 = 0x100 // Ctrl: skip dialog / fast-forward

	axisHorizontal uint16 = 0x010 // left or right held
	axisVertical   uint16 = 0x020 // up or down held
	signDownRight  uint16 = 0x040 // down (with vertical) or right (with horizontal)
	signUpLeft     uint16 = 0x080 // up (with vertical) or left (with horizontal)
)

// Direction bit combinations for synthetic input streams and tests.
func KeyUpBit() uint16    { return axisVertical | signUpLeft }
func KeyDownBit() uint16  { return axisVertical | signDownRight }
func KeyLeftBit() uint16  { return axisHorizontal | signUpLeft }
func KeyRightBit() uint16 { return axisHorizontal | signDownRight }

// Record is one frame of recorded input.
type Record struct {
	Keys uint16 // key bitmask (KeyShot, KeyFocus, ...); use the direction helpers
	Aux1 uint16 // DAT_00474e62 (raw menu keycodes; semantics not fully decoded)
	Aux2 uint16 // DAT_00474e64
}

// Up reports whether the up direction was held this frame.
func (r Record) Up() bool { return r.Keys&axisVertical != 0 && r.Keys&signUpLeft != 0 }

// Down reports whether the down direction was held this frame.
func (r Record) Down() bool { return r.Keys&axisVertical != 0 && r.Keys&signDownRight != 0 }

// Left reports whether the left direction was held this frame.
func (r Record) Left() bool { return r.Keys&axisHorizontal != 0 && r.Keys&signUpLeft != 0 }

// Right reports whether the right direction was held this frame.
func (r Record) Right() bool { return r.Keys&axisHorizontal != 0 && r.Keys&signDownRight != 0 }

// Chapter is one stage run inside a replay.
//
// Field offsets are from the 0x1c4-byte block header; the loader FUN_00428f60
// restores the RNG seed from +2 and the frame clock from +0x18.
type Chapter struct {
	Index   uint16 // +0x00: chapter/stage index (1..7)
	Seed    uint16 // +0x02: RNG seed captured at chapter start (DAT_004918b0)
	Frame   int32  // +0x18: starting frame clock (0 for a fresh chapter)
	Rank    int32  // +0x20: dynamic rank (DAT_00474c98) at chapter start
	Records []Record
	Raw     []byte // full 0x1c4-byte header, for fields not yet decoded
}

// slowRateStride is how often the game appends a slow-rate sample byte while
// recording: FUN_004297d0 writes one byte every 0x1e frames.
const slowRateStride = 30

// slowRateSamples returns the trailing sample count for a chapter with n records.
func slowRateSamples(n int) int { return (n + slowRateStride - 1) / slowRateStride }

// Trailer is the plaintext metadata appended after the compressed payload.
type Trailer struct {
	User     string  // player name ("Name ZUN")
	Version  string  // "debug"
	Name     string  // replay description, Shift-JIS (from the leading USER block)
	Date     string  // e.g. "07/08/02 07:43"
	Chara    string  // e.g. "ReimuB"
	Rank     string  // e.g. "Lunatic"
	Stage    int     // "Stage 5"
	Score    int64   // "Score 1319078"
	SlowRate float64 // "Slow Rate 0.02"
	Raw      []byte
}

// Replay is a fully decoded replay file.
type Replay struct {
	Header   Header
	Global   GlobalHeader
	Chapters []Chapter
	Trailer  Trailer
}

// Header mirrors the on-disk 36-byte file header.
type Header struct {
	Version      uint32
	TrailerOff   uint32 // file offset of the 200-byte trailer
	Compressed   uint32 // compressed payload length
	Decompressed uint32 // decompressed payload length
}

// GlobalHeader is the 100-byte payload prefix. Field offsets are pinned by
// correlating all four official demos against their trailer text
// (Chara/Rank/Stage/Score); see replay_test.go.
type GlobalHeader struct {
	Character  uint32 // +0x50: 0=Reimu 1=Marisa
	ShotType   uint32 // +0x54: 0=A 1=B 2=C
	Difficulty uint32 // +0x58: 0=E 1=N 2=H 3=L 4=X
	Stage      uint32 // +0x5c: 1..6, 7=Extra
	Raw        []byte
}

// Parse decodes a .rpy file.
func Parse(data []byte) (*Replay, error) {
	if len(data) < 36+trailerSize {
		return nil, fmt.Errorf("replay: file too short (%d bytes)", len(data))
	}
	if string(data[0:4]) != Magic {
		return nil, fmt.Errorf("replay: bad magic %q", string(data[0:4]))
	}
	h := Header{
		Version:      binary.LittleEndian.Uint32(data[4:]),
		TrailerOff:   binary.LittleEndian.Uint32(data[0x0c:]),
		Compressed:   binary.LittleEndian.Uint32(data[0x1c:]),
		Decompressed: binary.LittleEndian.Uint32(data[0x20:]),
	}
	if h.Version != Version {
		return nil, fmt.Errorf("replay: unsupported version %d", h.Version)
	}
	if int(h.TrailerOff) > len(data) || int(h.Compressed) > len(data)-0x24 {
		return nil, errors.New("replay: header lengths exceed file size")
	}

	payload := make([]byte, h.Compressed)
	copy(payload, data[0x24:0x24+h.Compressed])
	// FUN_0042a200: two block-wise XOR passes, then ZUN LZSS.
	xorTransform(payload, 0xe1, 0x400, 0xaa)
	xorTransform(payload, 0x7a, 0x80, 0x3d)
	raw := lzssDecompress(payload, int(h.Decompressed))
	if len(raw) < globalHeaderSize {
		return nil, fmt.Errorf("replay: payload too short (%d bytes)", len(raw))
	}

	r := &Replay{Header: h}
	r.Global = parseGlobalHeader(raw[:globalHeaderSize])
	r.Chapters = parseChapters(raw[globalHeaderSize:])
	r.Trailer = parseTrailer(data[len(data)-trailerSize:])
	return r, nil
}

func parseGlobalHeader(b []byte) GlobalHeader {
	g := GlobalHeader{Raw: append([]byte(nil), b...)}
	g.Character = binary.LittleEndian.Uint32(b[0x50:])
	g.ShotType = binary.LittleEndian.Uint32(b[0x54:])
	g.Difficulty = binary.LittleEndian.Uint32(b[0x58:])
	g.Stage = binary.LittleEndian.Uint32(b[0x5c:])
	return g
}

func parseChapters(b []byte) []Chapter {
	var out []Chapter
	pos := 0
	for i := 0; i < MaxChapters; i++ {
		if pos+chapterHeaderSize > len(b) {
			break
		}
		idx := binary.LittleEndian.Uint16(b[pos:])
		seed := binary.LittleEndian.Uint16(b[pos+2:])
		count := int(binary.LittleEndian.Uint32(b[pos+4:]))
		if idx == 0 || count == 0 {
			break
		}
		recStart := pos + chapterHeaderSize
		if count > (len(b)-recStart)/recordSize {
			count = (len(b) - recStart) / recordSize
		}
		ch := Chapter{
			Index:   idx,
			Seed:    seed,
			Frame:   int32(binary.LittleEndian.Uint32(b[pos+0x18:])),
			Rank:    int32(binary.LittleEndian.Uint32(b[pos+0x20:])),
			Records: make([]Record, count),
			Raw:     append([]byte(nil), b[pos:pos+chapterHeaderSize]...),
		}
		for j := 0; j < count; j++ {
			q := recStart + j*recordSize
			ch.Records[j] = Record{
				Keys: binary.LittleEndian.Uint16(b[q:]),
				Aux1: binary.LittleEndian.Uint16(b[q+2:]),
				Aux2: binary.LittleEndian.Uint16(b[q+4:]),
			}
		}
		out = append(out, ch)
		// Each chapter is followed by its slow-rate samples (one per 30 frames).
		pos = recStart + count*recordSize + slowRateSamples(count)
	}
	return out
}

func parseTrailer(b []byte) Trailer {
	t := Trailer{Raw: append([]byte(nil), b...)}
	// The trailer is plain text: "USER <name>\0 ... \r\nName ...\r\nDate ...\r\n..."
	text := decodeTrailerText(b)
	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(line)
		switch {
		case strings.HasPrefix(line, "USER "):
			// The USER block carries the replay description in Shift-JIS.
			t.Name = strings.TrimSpace(line[5:])
		case strings.HasPrefix(line, "Version "):
			t.Version = strings.TrimSpace(line[8:])
		case strings.HasPrefix(line, "Name "):
			t.User = strings.TrimSpace(line[5:])
		case strings.HasPrefix(line, "Date "):
			t.Date = strings.TrimSpace(line[5:])
		case strings.HasPrefix(line, "Chara "):
			t.Chara = strings.TrimSpace(line[6:])
		case strings.HasPrefix(line, "Rank "):
			t.Rank = strings.TrimSpace(line[5:])
		case strings.HasPrefix(line, "Stage "):
			fmt.Sscanf(line[6:], "%d", &t.Stage)
		case strings.HasPrefix(line, "Score "):
			fmt.Sscanf(line[6:], "%d", &t.Score)
		case strings.HasPrefix(line, "Slow Rate "):
			fmt.Sscanf(line[10:], "%f", &t.SlowRate)
		}
	}
	return t
}

// decodeTrailerText renders the plaintext trailer as UTF-8. The trailer holds
// an ASCII key/value block (Version/Name/Date/Chara/Rank/Stage/Score/Slow Rate)
// preceded by a short binary prefix; NUL bytes are mapped to newlines so the
// key/value lines stay separable. The replay display name is Shift-JIS, so the
// buffer runs through the Shift-JIS decoder (ASCII passes through unchanged).
func decodeTrailerText(b []byte) string {
	clean := make([]byte, len(b))
	for i, c := range b {
		if c == 0 {
			clean[i] = '\n'
		} else {
			clean[i] = c
		}
	}
	s, err := japanese.ShiftJIS.NewDecoder().String(string(clean))
	if err != nil {
		return string(clean)
	}
	return s
}

// xorTransform ports FUN_0044b0d0. The source is read sequentially from a copy
// while writes walk odd block positions descending, then even ones descending,
// XORing with a key byte that advances by keyInc per write.
func xorTransform(data []byte, keyInc byte, modulus int, initKey byte) {
	n := len(data)
	if n == 0 {
		return
	}
	rem := n % modulus
	corr := 0
	if rem < modulus/4 {
		corr = rem
	}
	toDo := n - ((n & 1) + corr)
	if toDo <= 0 {
		return
	}
	src := make([]byte, n)
	copy(src, data)
	key := initKey
	pos, read := 0, 0
	for toDo > 0 {
		size := modulus
		if toDo < size {
			size = toDo
		}
		for i := size - 1; i >= 0; i -= 2 {
			data[pos+i] = src[read] ^ key
			key += keyInc
			read++
		}
		for i := size - 2; i >= 0; i -= 2 {
			data[pos+i] = src[read] ^ key
			key += keyInc
			read++
		}
		pos += size
		toDo -= size
	}
}

// lzssDecompress ports FUN_00435dc0: ZUN's LZSS with an 8 KB ring window.
// Flags are read MSB-first; a 1 bit emits one literal byte, a 0 bit emits a
// 13-bit window offset and a 4-bit length (copy = length+3 bytes). Offset 0
// terminates the stream.
func lzssDecompress(src []byte, outLen int) []byte {
	out := make([]byte, 0, outLen)
	var window [0x2000]byte
	winPos := 1
	bitPos := byte(0x80)
	var cur byte
	pos := 0
	nextBit := func() int {
		if bitPos == 0x80 {
			if pos < len(src) {
				cur = src[pos]
				pos++
			} else {
				cur = 0
			}
		}
		b := int(cur&bitPos) >> 0
		if cur&bitPos == 0 {
			b = 0
		} else {
			b = 1
		}
		bitPos >>= 1
		if bitPos == 0 {
			bitPos = 0x80
		}
		return b
	}
	for len(out) < outLen {
		if nextBit() != 0 {
			b := byte(0)
			for i := 0; i < 8; i++ {
				b = b<<1 | byte(nextBit())
			}
			out = append(out, b)
			window[winPos] = b
			winPos = (winPos + 1) & 0x1fff
			continue
		}
		off := 0
		for i := 0; i < 13; i++ {
			off = off<<1 | nextBit()
		}
		if off == 0 {
			break
		}
		ln := 0
		for i := 0; i < 4; i++ {
			ln = ln<<1 | nextBit()
		}
		for i := 0; i <= ln+2 && len(out) < outLen; i++ {
			b := window[(off+i)&0x1fff]
			out = append(out, b)
			window[winPos] = b
			winPos = (winPos + 1) & 0x1fff
		}
	}
	return out
}
