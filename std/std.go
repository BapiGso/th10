// Package std runs a stage's .std background script: the script VM
// FUN_00403c80 as driven by the STD calc FUN_00402720, whose camera ECL
// 270/271 enemies read through the copy DAT_00491d7c, and the stage's ANM
// objects (the quads' VMs and op 14's slots). The fog tween and drawing are
// M4 (docs/research/m4_render.md S2, S3).
package std

import (
	"encoding/binary"
	"fmt"
	"math"
)

// File is a parsed .std (v10 header, thtk formats_v1). The offsets in the
// field comments are file offsets.
type File struct {
	ANM     string  // +0x10, anm_name: the stage's background .anm
	Entries []Entry // the table at +0x90 (one u32 offset each), in file order
	Faces   []Face  // from faces_offset (+0x04) up to the 0xffff terminator
	Script  []byte  // from script_offset (+0x08) to the end; jump offsets index it
}

// Entry is one object of the stage: a 0x1c-byte header, then its quads and a
// 0x0004ffff terminator. Its offset is in the table at +0x90.
type Entry struct {
	ID      uint16     // +0x00, the entry's index (thtk writes it that way)
	Unknown uint16     // +0x02
	Pos     [3]float32 // +0x04, x y z
	Size    [3]float32 // +0x10, width height depth
	Quads   []Quad
}

// Quad is one sprite of an entry: a 0x1c-byte record, the first at +0x1c of
// the entry. FUN_00403990 steps through an entry's quads the same way.
type Quad struct {
	Type        uint16     // +0x00, 0 in all seven stages
	RecSize     uint16     // +0x02, always 0x1c
	ScriptIndex uint16     // +0x04, script in the stage's .anm that draws it
	Padding     uint16     // +0x06, 0 in every TH10 file; FUN_00403990 uses it as a quad VM index
	Pos         [3]float32 // +0x08, x y z
	Size        [2]float32 // +0x14, width height
}

// Face places an entry on the stage: a 0x10-byte record from faces_offset.
type Face struct {
	Entry   uint16     // +0x00, index into Entries
	Unknown uint16     // +0x02, 256 in every TH10 file
	Pos     [3]float32 // +0x04, x y z
}

const (
	headerSize   = 0x90       // std_header_10_t; the entry offset table follows it
	entryRecSize = 0x1c       // std_entry_header_t
	quadRecSize  = 0x1c       // std_object_t
	faceRecSize  = 0x10       // std_object_instance_t
	quadEnd      = 0x0004ffff // u32 that ends an entry's quads (thtk writes it so)
	faceEnd      = 0xffff     // object id that ends the face list
)

// Load parses a .std. Every read is bounds-checked, and the header's counts
// must agree with what the entries and faces hold.
func Load(data []byte) (*File, error) {
	if len(data) < headerSize {
		return nil, fmt.Errorf("std: %d bytes, header needs %d", len(data), headerSize)
	}
	off := binary.LittleEndian.Uint32(data[8:])
	if int64(off) > int64(len(data)) {
		return nil, fmt.Errorf("std: script offset %#x past the end (%d bytes)", off, len(data))
	}
	name := data[0x10:headerSize]
	for i, c := range name {
		if c == 0 {
			name = name[:i]
			break
		}
	}
	f := &File{ANM: string(name), Script: data[off:]}
	if err := f.parseObjects(data, int(off)); err != nil {
		return nil, fmt.Errorf("std: %w", err)
	}
	return f, nil
}

// parseObjects fills in Entries and Faces. The layout is thstd.c's
// std_read_file; FUN_00403850 reads the same header fields (nb_objects at +0,
// nb_faces at +2, faces_offset at +4, script_offset at +8).
func (f *File) parseObjects(data []byte, script int) error {
	nObjects := int(u16le(data[0:]))
	nQuads := int(u16le(data[2:]))
	facesOff := int(binary.LittleEndian.Uint32(data[4:]))

	if err := need(data, headerSize, 4*nObjects, "entry table"); err != nil {
		return err
	}
	quads := 0
	for i := range nObjects {
		off := int(binary.LittleEndian.Uint32(data[headerSize+4*i:]))
		e, err := parseEntry(data, off)
		if err != nil {
			return fmt.Errorf("entry %d: %w", i, err)
		}
		quads += len(e.Quads)
		f.Entries = append(f.Entries, e)
	}
	// nb_faces counts quads, not faces: the game sizes its quad VM array
	// from it (FUN_00403850).
	if quads != nQuads {
		return fmt.Errorf("header counts %d quads, entries hold %d", nQuads, quads)
	}

	faces, err := parseFaces(data, facesOff, nObjects, script)
	if err != nil {
		return err
	}
	f.Faces = faces
	return nil
}

// parseEntry reads the entry at off and its quads.
func parseEntry(data []byte, off int) (Entry, error) {
	if err := need(data, off, entryRecSize, "entry header"); err != nil {
		return Entry{}, err
	}
	e := Entry{
		ID:      u16le(data[off:]),
		Unknown: u16le(data[off+2:]),
		Pos:     vec3le(data[off+4:]),
		Size:    vec3le(data[off+16:]),
	}
	for q := off + entryRecSize; ; q += quadRecSize {
		if err := need(data, q, 4, "quad list"); err != nil {
			return Entry{}, err
		}
		if binary.LittleEndian.Uint32(data[q:]) == quadEnd {
			return e, nil
		}
		if err := need(data, q, quadRecSize, "quad"); err != nil {
			return Entry{}, err
		}
		// FUN_00403990 steps by this size field. thstd.c's record is 0x1c,
		// and a file with another size is not one we parse.
		if rs := u16le(data[q+2:]); rs != quadRecSize {
			return Entry{}, fmt.Errorf("quad at %#x has record size %#x, want %#x", q, rs, quadRecSize)
		}
		e.Quads = append(e.Quads, Quad{
			Type:        u16le(data[q:]),
			RecSize:     u16le(data[q+2:]),
			ScriptIndex: u16le(data[q+4:]),
			Padding:     u16le(data[q+6:]),
			Pos:         vec3le(data[q+8:]),
			Size:        [2]float32{f32le(data[q+20:]), f32le(data[q+24:])},
		})
	}
}

// parseFaces reads the face list at off up to its 0xffff object id. thstd.c
// writes a 16-byte terminator (four 0xffffffff words) after the list and the
// script after that, so the list must end exactly at script.
func parseFaces(data []byte, off, nObjects, script int) ([]Face, error) {
	var faces []Face
	for ; ; off += faceRecSize {
		if err := need(data, off, faceRecSize, "face list"); err != nil {
			return nil, err
		}
		id := u16le(data[off:])
		if id == faceEnd {
			break
		}
		if int(id) >= nObjects {
			return nil, fmt.Errorf("face %d names entry %d, file has %d", len(faces), id, nObjects)
		}
		faces = append(faces, Face{
			Entry:   id,
			Unknown: u16le(data[off+2:]),
			Pos:     vec3le(data[off+4:]),
		})
	}
	if off+faceRecSize != script {
		return nil, fmt.Errorf("face terminator at %#x, but script starts at %#x", off, script)
	}
	return faces, nil
}

// need fails unless the n bytes at off lie inside data.
func need(data []byte, off, n int, what string) error {
	if off < 0 || off > len(data) || n > len(data)-off {
		return fmt.Errorf("%s at %#x needs %d bytes, file has %d", what, off, n, len(data))
	}
	return nil
}

func u16le(b []byte) uint16 { return binary.LittleEndian.Uint16(b) }

func f32le(b []byte) float32 { return math.Float32frombits(binary.LittleEndian.Uint32(b)) }

func vec3le(b []byte) [3]float32 {
	return [3]float32{f32le(b), f32le(b[4:]), f32le(b[8:])}
}
