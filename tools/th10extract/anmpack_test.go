package main

import (
	"bytes"
	"compress/flate"
	"encoding/binary"
	"testing"

	"th10/anm"
)

func packedFixture(t *testing.T) ([]byte, []byte, string) {
	t.Helper()
	raw := make([]byte, 72+16+24)
	binary.LittleEndian.PutUint32(raw[0x0c:], 3)
	binary.LittleEndian.PutUint32(raw[0x10:], 2)
	binary.LittleEndian.PutUint32(raw[0x14:], anm.FormatBGRA8888)
	binary.LittleEndian.PutUint32(raw[0x1c:], 64)
	binary.LittleEndian.PutUint32(raw[0x30:], 72)
	binary.LittleEndian.PutUint16(raw[0x34:], 1)
	copy(raw[64:], "test")
	copy(raw[72:], "THTX")
	binary.LittleEndian.PutUint16(raw[78:], anm.FormatBGRA8888)
	binary.LittleEndian.PutUint16(raw[80:], 3)
	binary.LittleEndian.PutUint16(raw[82:], 2)
	binary.LittleEndian.PutUint32(raw[84:], 24)
	for i := range 24 {
		raw[88+i] = byte(i * 13) // nonzero RGB under zero alpha must survive
	}
	raw[91] = 0
	var compressed bytes.Buffer
	w, err := flate.NewWriter(&compressed, flate.BestCompression)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := w.Write(raw[88:]); err != nil {
		t.Fatal(err)
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	blob := make([]byte, 16)
	copy(blob, "TXP1")
	blob[5] = anm.FormatBGRA8888
	binary.LittleEndian.PutUint16(blob[6:], 3)
	binary.LittleEndian.PutUint16(blob[8:], 2)
	binary.LittleEndian.PutUint32(blob[12:], 24)
	blob = append(blob, compressed.Bytes()...)
	f, err := anm.Load(raw)
	if err != nil {
		t.Fatal(err)
	}
	return raw, blob, TextureID(f.Entries[0].Texture)
}

func TestPackedANMRoundtripAndDeterminism(t *testing.T) {
	raw, blob, id := packedFixture(t)
	packed, textures, err := Split(raw)
	if err != nil {
		t.Fatal(err)
	}
	if len(textures) != 1 || textures[id] == nil || !anm.IsPacked(packed) {
		t.Fatal("split lost its texture reference")
	}
	pack, err := BuildTexturePack(map[string][]byte{id: blob})
	if err != nil {
		t.Fatal(err)
	}
	got, err := anm.Restore(packed, func(id string) ([]byte, error) { return anm.TextureBlob(pack, id) })
	if err != nil || !bytes.Equal(got, raw) {
		t.Fatalf("restored ANM is not byte-exact: %v", err)
	}
	again, _, err := Split(raw)
	if err != nil || !bytes.Equal(again, packed) {
		t.Fatal("ANM split is nondeterministic")
	}
	again, err = BuildTexturePack(map[string][]byte{id: blob})
	if err != nil || !bytes.Equal(again, pack) {
		t.Fatal("texture pack is nondeterministic")
	}
}

func TestPackedRejectsTruncationAndChangedLayout(t *testing.T) {
	raw, blob, _ := packedFixture(t)
	for n := 0; n < len(blob); n++ {
		if _, err := anm.DecodeTexture(blob[:n]); err == nil {
			t.Fatalf("accepted truncated texture at %d", n)
		}
	}
	packed, _, err := Split(raw)
	if err != nil {
		t.Fatal(err)
	}
	bad := bytes.Clone(blob)
	bad[5] = anm.FormatRGB565
	if _, err := anm.Restore(packed, func(string) ([]byte, error) { return bad, nil }); err == nil {
		t.Fatal("accepted a texture with a changed THTX format")
	}
	for _, n := range []int{0, 11, 12, 51, len(packed) - 1} {
		if _, err := anm.Restore(packed[:n], func(string) ([]byte, error) { return blob, nil }); err == nil {
			t.Fatalf("accepted truncated packed ANM at %d", n)
		}
	}
	bad = bytes.Clone(blob)
	bad[4] = 255
	if _, err := anm.DecodeTexture(bad); err == nil {
		t.Fatal("accepted unknown codec")
	}
	bad = bytes.Clone(blob)
	for i := 6; i < 10; i++ {
		bad[i] = 255
	}
	if _, err := anm.DecodeTexture(bad); err == nil {
		t.Fatal("accepted oversized dimensions")
	}
}
