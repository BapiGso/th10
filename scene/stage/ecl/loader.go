package ecl

// Binary ECL (TH10 "SCPT" V2) loader.
//
// This parses the raw compiled stage*.ecl bytes that th10.exe itself loads,
// rather than thtk's decompiled text. The container layout and the
// param_mask argument model are taken verbatim from thtk's thecl10.c
// (th10_open / th10_instr_t) and confirmed against the in-binary resolver
// FUN_0044fdb0. See docs/ecl_truth.md.
//
// On-disk layout (little-endian):
//
//	SCPT header (36B): magic[4] u16=1 u16 include_len u32 include_off(=36)
//	                   u32 zero u32 sub_count u32 zero[4]
//	ANIM list @include_off: magic[4] u32 count, then count NUL strings, pad to 4
//	ECLI list:              magic[4] u32 count, then count NUL strings, pad to 4
//	u32 sub_offsets[sub_count]   (absolute file offsets)
//	sub name table:         sub_count NUL strings
//	per sub @sub_offsets[i]: ECLH header (16B): magic[4] u32 data_off(=16) u32 zero[2]
//	                         then instruction stream until next sub / EOF
//	instr (16B header): u32 time u16 id u16 size u16 param_mask u8 rank_mask
//	                    u8 param_count u32 zero, then param data (size-16 bytes)

import (
	"encoding/binary"
	"errors"
	"fmt"
	"math"
)

// Program is a parsed ECL file.
type Program struct {
	AnimNames []string
	EcliNames []string
	Subs      []*Sub
	SubByName map[string]*Sub
}

// Sub is one ECL subroutine (a "void Name(...)" in thtk text).
type Sub struct {
	Name   string
	Instrs []Instr
	// offsetIndex maps an instruction's byte offset (relative to the sub's
	// ECLH header) to its index in Instrs, for resolving 'o' jump targets.
	offsetIndex map[int32]int
}

// Instr is a single decoded instruction.
type Instr struct {
	Time     int32
	Opcode   uint16
	RankMask uint8 // bits: 1111LHNE (bit0=Easy,1=Normal,2=Hard,3=Lunatic)
	Offset   int32 // byte offset of this instr within the sub (incl 16B ECLH header)
	Params   []Param
}

// Param is one decoded argument.
type Param struct {
	Type  byte   // 'S' int, 'f' float, 'm'/'x' string, 'o' offset, 't' time, 'D' typed
	Raw   uint32 // raw 4-byte value (low 4 bytes for 'D')
	IsVar bool   // param_mask bit: the value is a variable / stack reference
	Str   string // decoded string for 'm'/'x'
	DFrom byte   // 'D' only: source type 'i'/'f'
	DTo   byte   // 'D' only: dest type 'i'/'f'
}

// Int returns the raw value reinterpreted as a signed 32-bit int.
func (p Param) Int() int32 { return int32(p.Raw) }

// Float returns the raw value reinterpreted as a float32 bit pattern.
func (p Param) Float() float32 { return math.Float32frombits(p.Raw) }

// VarID returns the variable/stack id a param references (only meaningful when
// IsVar). Float-typed params encode the id as a float bit pattern: a local
// var at byte offset 12 is stored as 12.0f, the special "aim" var as -9998.0f.
// Int-typed params store the id directly. id>=0 = local frame byte offset,
// id==-1 = pop data stack, id<-1 = special/global variable.
func (p Param) VarID() int32 {
	if p.Type == 'f' {
		return int32(p.Float())
	}
	return p.Int()
}

// rankAllows reports whether this instruction runs at the given difficulty
// index (0=Easy..3=Lunatic, 4=Extra treated as Lunatic for the 4-bit mask).
func (in Instr) rankAllows(diff int) bool {
	bit := uint8(1) << uint(rankBitFor(diff))
	return in.RankMask&bit != 0
}

func rankBitFor(diff int) int {
	switch {
	case diff <= 0:
		return 0 // Easy
	case diff == 1:
		return 1 // Normal
	case diff == 2:
		return 2 // Hard
	default:
		return 3 // Lunatic/Extra
	}
}

var (
	errBadMagic = errors.New("ecl: bad magic")
	errTooShort = errors.New("ecl: truncated")
)

// LoadProgram parses a raw TH10 ECL byte slice.
func LoadProgram(data []byte) (*Program, error) {
	if len(data) < 36 {
		return nil, errTooShort
	}
	if string(data[0:4]) != "SCPT" {
		return nil, fmt.Errorf("%w: want SCPT got %q", errBadMagic, data[0:4])
	}
	includeOff := binary.LittleEndian.Uint32(data[8:12])
	subCount := binary.LittleEndian.Uint32(data[16:20])

	prog := &Program{SubByName: map[string]*Sub{}}

	// ANIM list.
	off := int(includeOff)
	names, off, err := readList(data, off, "ANIM")
	if err != nil {
		return nil, err
	}
	prog.AnimNames = names
	off = align4(off)

	// ECLI list.
	names, off, err = readList(data, off, "ECLI")
	if err != nil {
		return nil, err
	}
	prog.EcliNames = names
	off = align4(off)

	// Sub offset table.
	if off+int(subCount)*4 > len(data) {
		return nil, errTooShort
	}
	subOffsets := make([]uint32, subCount)
	for i := range subOffsets {
		subOffsets[i] = binary.LittleEndian.Uint32(data[off : off+4])
		off += 4
	}

	// Sub name table (one NUL string per sub, in order).
	subNames := make([]string, subCount)
	for i := range subNames {
		s, n := readCString(data, off)
		subNames[i] = s
		off += n
	}

	for i := uint32(0); i < subCount; i++ {
		end := len(data)
		if i+1 < subCount {
			end = int(subOffsets[i+1])
		}
		sub, err := parseSub(data, int(subOffsets[i]), end, subNames[i])
		if err != nil {
			return nil, fmt.Errorf("sub %q: %w", subNames[i], err)
		}
		prog.Subs = append(prog.Subs, sub)
		prog.SubByName[sub.Name] = sub
	}
	return prog, nil
}

func readList(data []byte, off int, magic string) (names []string, next int, err error) {
	if off+8 > len(data) {
		return nil, off, errTooShort
	}
	if string(data[off:off+4]) != magic {
		return nil, off, fmt.Errorf("%w: want %s got %q", errBadMagic, magic, data[off:off+4])
	}
	count := binary.LittleEndian.Uint32(data[off+4 : off+8])
	off += 8
	for i := uint32(0); i < count; i++ {
		s, n := readCString(data, off)
		names = append(names, s)
		off += n
	}
	return names, off, nil
}

func parseSub(data []byte, base, end int, name string) (*Sub, error) {
	if base+16 > len(data) || base >= end {
		return nil, errTooShort
	}
	if string(data[base:base+4]) != "ECLH" {
		return nil, fmt.Errorf("%w: want ECLH got %q", errBadMagic, data[base:base+4])
	}
	sub := &Sub{Name: name, offsetIndex: map[int32]int{}}
	// Instructions start after the 16-byte ECLH header.
	pos := base + 16
	for pos+16 <= end {
		time := int32(binary.LittleEndian.Uint32(data[pos : pos+4]))
		id := binary.LittleEndian.Uint16(data[pos+4 : pos+6])
		size := int(binary.LittleEndian.Uint16(data[pos+6 : pos+8]))
		paramMask := binary.LittleEndian.Uint16(data[pos+8 : pos+10])
		rankMask := data[pos+10]
		// data[pos+11] = param_count, data[pos+12:16] = zero (TH13+ stack-ref count)
		if size < 16 || pos+size > end {
			return nil, fmt.Errorf("%s: bad instr size %d at %d", name, size, pos)
		}
		instrOff := int32(pos - base)
		params, perr := decodeParams(data[pos+16:pos+size], id, paramMask)
		if perr != nil {
			return nil, fmt.Errorf("%s op %d @%d: %w", name, id, pos, perr)
		}
		sub.offsetIndex[instrOff] = len(sub.Instrs)
		sub.Instrs = append(sub.Instrs, Instr{
			Time:     time,
			Opcode:   id,
			RankMask: rankMask,
			Offset:   instrOff,
			Params:   params,
		})
		pos += size
	}
	return sub, nil
}

// decodeParams walks the param blob per the opcode's signature, assigning
// one param_mask bit per parameter (LSB first), matching thecl10.c.
func decodeParams(blob []byte, opcode uint16, mask uint16) ([]Param, error) {
	format := th10Format(opcode)
	var params []Param
	fi := 0        // index into format
	pos := 0       // index into blob
	bit := uint(0) // param_mask bit
	next := func() byte {
		if fi >= len(format) {
			return 'S' // default to int (matches thtk "*S" fallback)
		}
		if format[fi] == '*' {
			return format[fi+1]
		}
		return format[fi]
	}
	advance := func() {
		if fi < len(format) && format[fi] != '*' {
			fi++
		}
	}
	for pos < len(blob) {
		t := next()
		p := Param{Type: t}
		if t == 'H' {
			bit++ // 'H' consumes an extra mask bit before its own
		}
		p.IsVar = mask&(1<<bit) != 0
		bit++
		switch t {
		case 'm', 'x':
			if pos+4 > len(blob) {
				return nil, errTooShort
			}
			n := int(binary.LittleEndian.Uint32(blob[pos : pos+4]))
			pos += 4
			if pos+n > len(blob) {
				return nil, errTooShort
			}
			raw := append([]byte(nil), blob[pos:pos+n]...)
			if t == 'x' {
				xorString(raw)
			}
			p.Str = trimNul(raw)
			pos += n
		case 'D', 'H':
			// thecl_sub_param_t: from(1) to(1) zero(2) val(4) = 8 bytes.
			if pos+8 > len(blob) {
				return nil, errTooShort
			}
			p.DFrom = typeChar(blob[pos])
			p.DTo = typeChar(blob[pos+1])
			p.Raw = binary.LittleEndian.Uint32(blob[pos+4 : pos+8])
			pos += 8
		default: // S, f, o, t and unknowns: 4-byte value
			if pos+4 > len(blob) {
				return nil, errTooShort
			}
			p.Raw = binary.LittleEndian.Uint32(blob[pos : pos+4])
			pos += 4
		}
		params = append(params, p)
		advance()
	}
	return params, nil
}

// typeChar maps the 'D' from/to nibble byte to 'i'/'f'.
func typeChar(b byte) byte {
	if b == 'f' {
		return 'f'
	}
	return 'i'
}

func xorString(b []byte) {
	// thtk util_xor(data, len, 0x77, 7, 16)
	key := byte(0x77)
	inc := byte(7)
	for i := range b {
		b[i] ^= key
		key += inc
		inc += 16
	}
}

func trimNul(b []byte) string {
	for i, c := range b {
		if c == 0 {
			return string(b[:i])
		}
	}
	return string(b)
}

func readCString(data []byte, off int) (string, int) {
	if off >= len(data) {
		return "", 0
	}
	i := off
	for i < len(data) && data[i] != 0 {
		i++
	}
	return string(data[off:i]), i - off + 1
}

func align4(off int) int {
	if off%4 != 0 {
		off += 4 - off%4
	}
	return off
}
