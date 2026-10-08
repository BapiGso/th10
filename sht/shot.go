package sht

import (
	"encoding/binary"
	"fmt"
	"math"
)

// Offset is an option position relative to the player, in 0.01 pixel units.
type Offset struct{ X, Y int }

// Shot is one 0x34-byte record. Callback indices remain indices, never addresses.
type Shot struct {
	Interval, Delay                   int
	Damage                            int
	X, Y, Width, Height, Angle, Speed float64
	Source, Kind                      int
	Script, HitScript, Sound          int
	Callbacks                         [4]uint32
}

type File struct {
	Movement Movement
	Options  [2][10]Offset // unfocused/focused, triangular layouts for 1..4 options
	Groups   [10][]Shot    // power 0..4, then focused power 0..4
}

// Parse reads TH10's version-3 SHT container. It does not implement arbitrary
// shot callbacks; each runtime must explicitly validate the callbacks it uses.
func Parse(data []byte) (*File, error) {
	movement, err := ParseMovement(data)
	if err != nil {
		return nil, err
	}
	if len(data) < 0x160 {
		return nil, fmt.Errorf("SHT option/group table truncated")
	}
	if count := binary.LittleEndian.Uint16(data[2:]); count != 10 {
		return nil, fmt.Errorf("unsupported TH10 SHT group count %d", count)
	}
	f := &File{Movement: movement}
	readFloat := func(at int) (float64, error) {
		v := float64(math.Float32frombits(binary.LittleEndian.Uint32(data[at:])))
		if math.IsNaN(v) || math.IsInf(v, 0) {
			return 0, fmt.Errorf("non-finite SHT float at %#x", at)
		}
		return v, nil
	}
	for mode := range f.Options {
		for i := range f.Options[mode] {
			at := 0x20 + mode*0x78 + i*12
			x, err := readFloat(at)
			if err != nil {
				return nil, err
			}
			y, err := readFloat(at + 4)
			if err != nil {
				return nil, err
			}
			if math.Abs(x) > 10000 || math.Abs(y) > 10000 {
				return nil, fmt.Errorf("SHT option outside supported range")
			}
			f.Options[mode][i] = Offset{int(x * 100), int(y * 100)}
		}
	}
	for group := range f.Groups {
		at := uint64(binary.LittleEndian.Uint32(data[0x110+group*8:]))
		if at < 0x160 || at+4 > uint64(len(data)) {
			return nil, fmt.Errorf("SHT group %d offset out of bounds", group)
		}
		for {
			if at+4 > uint64(len(data)) {
				return nil, fmt.Errorf("SHT group %d missing terminator", group)
			}
			if int8(data[at]) < 0 {
				break
			}
			if at+0x34 > uint64(len(data)) {
				return nil, fmt.Errorf("SHT group %d truncated shot", group)
			}
			o := int(at)
			s := Shot{Interval: int(data[o]), Delay: int(data[o+1]), Damage: int(int16(binary.LittleEndian.Uint16(data[o+2:]))),
				Source: int(int8(data[o+0x1c])), Kind: int(data[o+0x1d]),
				Script:    5 + int(int16(binary.LittleEndian.Uint16(data[o+0x1e:]))),
				HitScript: 5 + int(int16(binary.LittleEndian.Uint16(data[o+0x20:]))),
				Sound:     int(int16(binary.LittleEndian.Uint16(data[o+0x22:])))}
			if s.Interval == 0 || s.Delay >= s.Interval || s.Damage < 0 || s.Source < 0 || s.Source > 4 {
				return nil, fmt.Errorf("invalid SHT shot at %#x", at)
			}
			for i, dst := range []*float64{&s.X, &s.Y, &s.Width, &s.Height, &s.Angle, &s.Speed} {
				v, err := readFloat(o + 4 + i*4)
				if err != nil {
					return nil, err
				}
				*dst = v
			}
			if s.Width <= 0 || s.Height <= 0 || s.Speed < 0 {
				return nil, fmt.Errorf("invalid SHT shot geometry at %#x", at)
			}
			for i := range s.Callbacks {
				s.Callbacks[i] = binary.LittleEndian.Uint32(data[o+0x24+i*4:])
			}
			f.Groups[group] = append(f.Groups[group], s)
			at += 0x34
		}
	}
	return f, nil
}

func PowerLevel(power int) int { return max(0, min(power/100, 4)) }

func (f *File) Shots(power int, focused bool) []Shot {
	group := PowerLevel(power)
	if focused {
		group += 5
	}
	return f.Groups[group]
}

func (f *File) OptionOffsets(power int, focused bool) []Offset {
	count := PowerLevel(power)
	mode := 0
	if focused {
		mode = 1
	}
	start := count * (count - 1) / 2
	return f.Options[mode][start : start+count]
}

// ValidateReimuA rejects unsupported semantics instead of silently emitting
// straight bullets when a different SHT file or callback is wired by mistake.
func (f *File) ValidateReimuA() error {
	for g, shots := range f.Groups {
		for _, s := range shots {
			want := [4]uint32{}
			if s.Source != 0 {
				want = [4]uint32{1, 1, 0, 0}
			}
			if s.Kind != 0 || s.Callbacks != want || s.Source > g%5 {
				return fmt.Errorf("unsupported Reimu A shot semantics in group %d", g)
			}
		}
	}
	return nil
}
