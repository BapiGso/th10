// Package std runs a stage's .std background script. So far it ports the
// camera half: the script VM FUN_00403c80 as driven by the STD calc
// FUN_00402720, which ECL 270/271 enemies read through the camera copy
// DAT_00491d7c. The faces, quads, their ANM VMs, fog tweens and rendering
// are M4.
package std

import (
	"encoding/binary"
	"fmt"
)

// File is a parsed .std (v10 header, thtk formats_v1).
type File struct {
	ANM    string // anm_name: the stage's background .anm
	Script []byte // from script_offset to the end; jump offsets index it
}

const headerSize = 0x90

// Load parses a .std file.
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
	return &File{ANM: string(name), Script: data[off:]}, nil
}
