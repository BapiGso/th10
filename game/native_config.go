package game

import (
	"encoding/binary"
	"fmt"
	"math"
)

// NativeConfig is th10.cfg's exact 0x34-byte DAT_00491d48 structure.
// Unmodeled graphics/key fields remain raw; the host UI only edits volume.
type NativeConfig struct{ Raw [0x34]byte }

// DefaultNativeConfig is FUN_00418c40. Key defaults are DAT_00474e88..98
// as initialized in the PE; the two 600 words are at +0x16/+0x18.
func DefaultNativeConfig() *NativeConfig {
	c := &NativeConfig{}
	binary.LittleEndian.PutUint32(c.Raw[:], 0x100003)
	for i, key := range [...]uint16{0, 1, 2, 3, 0xffff, 0xffff, 0xffff, 0xffff, 4} {
		binary.LittleEndian.PutUint16(c.Raw[4+i*2:], key)
	}
	binary.LittleEndian.PutUint16(c.Raw[0x16:], 600)
	binary.LittleEndian.PutUint16(c.Raw[0x18:], 600)
	c.Raw[0x1b], c.Raw[0x1c], c.Raw[0x1f] = 1, 1, 2
	c.Raw[0x20], c.Raw[0x21] = 100, 80
	c.SetFlags(0x100)
	return c
}

// DecodeNativeConfig is FUN_00420870's size/version/graphics validation.
func DecodeNativeConfig(data []byte) (*NativeConfig, error) {
	if len(data) != 0x34 || binary.LittleEndian.Uint32(data) != 0x100003 {
		return nil, fmt.Errorf("game: th10.cfg must be 52 bytes, version 0x100003")
	}
	for i, limit := range [...]byte{2, 3, 2, 2, 3, 3} {
		if data[0x1a+i] >= limit {
			return nil, fmt.Errorf("game: invalid th10.cfg graphics option +%#x", 0x1a+i)
		}
	}
	c := &NativeConfig{}
	copy(c.Raw[:], data)
	return c, nil
}

func (c *NativeConfig) Flags() uint32         { return binary.LittleEndian.Uint32(c.Raw[0x30:]) }
func (c *NativeConfig) SetFlags(flags uint32) { binary.LittleEndian.PutUint32(c.Raw[0x30:], flags) }

// JoystickBindings is the native nine-word input table at cfg +4.
// Option's Key Config changes only indices 0..3/8; all other words survive.
func (c *NativeConfig) JoystickBindings() (keys [9]int16) {
	for i := range keys {
		keys[i] = int16(binary.LittleEndian.Uint16(c.Raw[4+i*2:]))
	}
	return keys
}

func (c *NativeConfig) SetJoystickBindings(keys [9]int16) {
	for _, i := range [...]int{0, 1, 2, 3, 8} { // FUN_0042f540, 0x42f75f..0x42f7d3
		binary.LittleEndian.PutUint16(c.Raw[4+i*2:], uint16(keys[i]))
	}
}

func (c *Config) NativeBytes() []byte {
	native := DefaultNativeConfig()
	if c != nil && c.Native != nil {
		*native = *c.Native
	}
	if c != nil {
		native.Raw[0x20] = byte(math.Round(min(max(c.BGMVolume, 0), 1) * 100))
		native.Raw[0x21] = byte(math.Round(min(max(c.SEVolume, 0), 1) * 100))
	}
	return append([]byte(nil), native.Raw[:]...)
}
