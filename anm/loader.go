// Package anm parses TH10 ".anm" sprite/animation containers straight from the
// compiled bytes the game loads (the same binary-direct approach as the ECL VM).
//
// An .anm file is a chain of entries; each entry carries one texture, a sprite
// table (UV rects into that texture), and a set of scripts. A script is a tiny
// clock-gated bytecode (interpreted by th10.exe's FUN_0043ee30) whose ins_3
// "show sprite" / ins_4 "loop" timeline drives an animation. ECL opcodes
// anmSetMain/anmSelect/anmPlay reference these scripts by id.
//
// We only need the sprite table + scripts here: the texture pixels already live
// as extracted PNGs under assets/anm/<name>, so the THTX blob is skipped.
//
// On-disk layout (little-endian), per entry, base = entry offset:
//
//	header (0x40):
//	  0x00 u32 numSprites   0x04 u32 numScripts  0x08 u32 zero
//	  0x0c u32 width        0x10 u32 height       0x14 u32 format
//	  0x18 u32 zero/colorkey 0x1c u32 nameOffset
//	  0x20 u32 x  0x24 u32 y  0x28 u32 version  0x2c u32 memPriority
//	  0x30 u32 thtxOffset   0x34 u16 hasData,u16 lowRes  0x38 u32 nextOffset
//	  0x3c u32 zero
//	@0x40                 u32 spriteOffsets[numSprites]   (entry-relative)
//	@0x40+numSprites*4    {u32 id; u32 offset}[numScripts] (offset entry-relative)
//	@nameOffset           NUL texture name
//	sprite struct (20B):  u32 id; f32 x; f32 y; f32 w; f32 h
//	script bytecode:      instr* until next script / name / THTX
//	instr (8B header):    u16 opcode; u16 size; u16 time; u16 paramMask ; params[size-8]
package anm

import (
	"encoding/binary"
	"errors"
	"fmt"
	"math"
)

// Sprite is one UV rectangle into an entry's texture.
type Sprite struct {
	ID    int
	X, Y  float32
	W, H  float32
	Entry int // index into File.Entries (which texture it cuts from)
}

// Instr is one decoded ANM script instruction.
type Instr struct {
	Time   int
	Op     int
	Mask   int
	Offset int    // byte offset of this instr within its script (for goto targets)
	Args   []byte // raw param bytes (size-8)
}

// Arg returns parameter word i as int32 (4-byte little-endian), or 0.
func (in Instr) Arg(i int) int32 {
	off := i * 4
	if off+4 > len(in.Args) {
		return 0
	}
	return int32(binary.LittleEndian.Uint32(in.Args[off : off+4]))
}

// Script is one animation routine (a sequence of timed instructions).
type Script struct {
	ID     int
	Instrs []Instr
}

// Entry is one texture + its metadata.
type Entry struct {
	Name          string // texture path, e.g. "enemy/enemy.png"
	Width, Height int
	Format        int
}

// File is a parsed .anm: every entry's texture, plus the global sprite and
// script tables (ids are unique across the whole file).
type File struct {
	Entries []Entry
	Sprites map[int]Sprite
	Scripts map[int]*Script
}

var errTooShort = errors.New("anm: truncated")

const headerSize = 0x40

// Load parses a raw .anm byte slice.
func Load(data []byte) (*File, error) {
	f := &File{Sprites: map[int]Sprite{}, Scripts: map[int]*Script{}}
	base := 0
	for {
		next, err := f.parseEntry(data, base)
		if err != nil {
			return nil, err
		}
		if next == 0 {
			break
		}
		nb := base + next
		if nb <= base || nb >= len(data) {
			break
		}
		base = nb
	}
	return f, nil
}

func (f *File) parseEntry(data []byte, base int) (next int, err error) {
	if base+headerSize > len(data) {
		return 0, errTooShort
	}
	u32 := func(off int) uint32 { return binary.LittleEndian.Uint32(data[base+off : base+off+4]) }
	numSprites := int(u32(0x00))
	numScripts := int(u32(0x04))
	width := int(u32(0x0c))
	height := int(u32(0x10))
	format := int(u32(0x14))
	nameOff := int(u32(0x1c))
	nextOff := int(u32(0x38))

	entryIdx := len(f.Entries)
	f.Entries = append(f.Entries, Entry{
		Name:   readCString(data, base+nameOff),
		Width:  width,
		Height: height,
		Format: format,
	})

	// Sprite offset table, then sprite structs.
	tbl := base + headerSize
	for i := 0; i < numSprites; i++ {
		if tbl+4 > len(data) {
			return 0, errTooShort
		}
		so := base + int(binary.LittleEndian.Uint32(data[tbl:tbl+4]))
		tbl += 4
		if so+20 > len(data) {
			return 0, errTooShort
		}
		id := int(int32(binary.LittleEndian.Uint32(data[so : so+4])))
		f.Sprites[id] = Sprite{
			ID:    id,
			X:     f32(data, so+4),
			Y:     f32(data, so+8),
			W:     f32(data, so+12),
			H:     f32(data, so+16),
			Entry: entryIdx,
		}
	}

	// Script table: {id, offset} pairs, then bytecode at each offset.
	for i := 0; i < numScripts; i++ {
		if tbl+8 > len(data) {
			return 0, errTooShort
		}
		id := int(int32(binary.LittleEndian.Uint32(data[tbl : tbl+4])))
		so := base + int(binary.LittleEndian.Uint32(data[tbl+4:tbl+8]))
		tbl += 8
		instrs, perr := parseScript(data, so)
		if perr != nil {
			return 0, fmt.Errorf("anm script %d: %w", id, perr)
		}
		f.Scripts[id] = &Script{ID: id, Instrs: instrs}
	}
	return nextOff, nil
}

// parseScript reads instructions until an end marker (ins_0) or a malformed
// header (which signals we've run into the next section).
func parseScript(data []byte, pos int) ([]Instr, error) {
	var out []Instr
	start := pos
	for pos+8 <= len(data) {
		op := int(binary.LittleEndian.Uint16(data[pos : pos+2]))
		size := int(binary.LittleEndian.Uint16(data[pos+2 : pos+4]))
		time := int(int16(binary.LittleEndian.Uint16(data[pos+4 : pos+6])))
		mask := int(binary.LittleEndian.Uint16(data[pos+6 : pos+8]))
		if size < 8 || pos+size > len(data) {
			break
		}
		args := append([]byte(nil), data[pos+8:pos+size]...)
		out = append(out, Instr{Time: time, Op: op, Mask: mask, Offset: pos - start, Args: args})
		pos += size
		if op == opEnd { // ins_0: end of script
			break
		}
	}
	return out, nil
}

func f32(data []byte, off int) float32 {
	if off+4 > len(data) {
		return 0
	}
	return math.Float32frombits(binary.LittleEndian.Uint32(data[off : off+4]))
}

func readCString(data []byte, off int) string {
	if off < 0 || off >= len(data) {
		return ""
	}
	i := off
	for i < len(data) && data[i] != 0 {
		i++
	}
	return string(data[off:i])
}
