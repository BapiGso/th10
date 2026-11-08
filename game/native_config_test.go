package game

import (
	"bytes"
	"encoding/binary"
	"os"
	"path/filepath"
	"testing"
)

func TestNativeConfigOriginalFileRoundtrip(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", ".tmp", "[th10]", "th10.cfg"))
	if err != nil {
		t.Skipf("original config unavailable: %v", err)
	}
	c, err := DecodeNativeConfig(data)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(c.Raw[:], data) {
		t.Fatal("native config bytes changed")
	}
	if c.Flags() != binary.LittleEndian.Uint32(data[0x30:]) {
		t.Fatal("cfg +0x30 not decoded")
	}
	host := &Config{Native: c, BGMVolume: float64(c.Raw[0x20]) / 100, SEVolume: float64(c.Raw[0x21]) / 100}
	if !bytes.Equal(host.NativeBytes(), data) {
		t.Fatal("host save is not byte-exact for original config")
	}
}

func TestNativeConfigDefaultsAndValidation(t *testing.T) {
	c := DefaultNativeConfig()
	if binary.LittleEndian.Uint32(c.Raw[:]) != 0x100003 || c.Flags() != 0x100 || c.Raw[0x20] != 100 || c.Raw[0x21] != 80 || binary.LittleEndian.Uint16(c.Raw[0x16:]) != 600 {
		t.Fatal("FUN_00418c40 defaults differ")
	}
	for _, size := range []int{0, 4, 51, 53} {
		if _, err := DecodeNativeConfig(make([]byte, size)); err == nil {
			t.Fatal("invalid size accepted")
		}
	}
	bad := c.Raw
	bad[0x1d] = 2
	if _, err := DecodeNativeConfig(bad[:]); err == nil {
		t.Fatal("invalid window setting accepted")
	}
}
