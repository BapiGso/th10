//go:build !js && !wasm

package game

import (
	"encoding/binary"
	"os"
	"path/filepath"
	"testing"
)

func TestNativeConfigStoragePreservesOpaqueFields(t *testing.T) {
	root := t.TempDir()
	old := resolvedDataDir
	resolvedDataDir = ""
	t.Cleanup(func() { resolvedDataDir = old })
	t.Setenv("TH10_DATA_DIR", root)
	c := DefaultNativeConfig()
	c.SetFlags(0x380)
	c.Raw[0x2f] = 0xab
	if err := os.WriteFile(filepath.Join(root, "th10.cfg"), c.Raw[:], 0o644); err != nil {
		t.Fatal(err)
	}
	loaded := LoadConfig()
	if loaded.Native.Flags() != 0x380 || loaded.BGMVolume != 1 || loaded.SEVolume != 0.8 {
		t.Fatal("native config not loaded")
	}
	loaded.BGMVolume, loaded.SEVolume = 0.37, 0.61
	g := &Game{Config: loaded}
	if err := g.SaveConfig(); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(root, "th10.cfg"))
	if err != nil {
		t.Fatal(err)
	}
	if len(data) != 52 || data[0x20] != 37 || data[0x21] != 61 || data[0x2f] != 0xab || binary.LittleEndian.Uint32(data[0x30:]) != 0x380 {
		t.Fatal("opaque fields or volume bytes changed incorrectly")
	}
}
