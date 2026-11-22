// The ANM texture packer: build-time counterpart of anm.Restore and
// anm.TextureBlob, which are the only parts the runtime needs.
package main

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"sort"

	"th10/anm"
)

// TextureID includes format and dimensions as well as every original pixel
// byte, including RGB under alpha zero. A web variant retains this source ID.
func TextureID(t *anm.Texture) string {
	h := sha256.New()
	_, _ = fmt.Fprintf(h, "%d/%d/%d/", t.Format, t.W, t.H)
	_, _ = h.Write(t.Pixels)
	return hex.EncodeToString(h.Sum(nil))
}

// Split removes only THTX pixels. ANP1 stores the original file length, then
// {offset, length, SHA-256} records and concatenated non-pixel byte ranges.
// It does not rebase or rewrite ANM headers, instructions or sprite tables.
func Split(data []byte) ([]byte, map[string]*anm.Texture, error) {
	ranges, err := anm.PixelRanges(data)
	if err != nil {
		return nil, nil, err
	}
	if len(data) > anm.MaxPackedBytes {
		return nil, nil, fmt.Errorf("anm: file too large")
	}
	textures := map[string]*anm.Texture{}
	metaSize, previous := len(data), 0
	for _, r := range ranges {
		if r.Offset < previous || r.Offset+len(r.Texture.Pixels) > len(data) {
			return nil, nil, fmt.Errorf("anm: overlapping pixel ranges")
		}
		metaSize -= len(r.Texture.Pixels)
		previous = r.Offset + len(r.Texture.Pixels)
	}
	out := make([]byte, 12+len(ranges)*40, 12+len(ranges)*40+metaSize)
	copy(out, "ANP1")
	binary.LittleEndian.PutUint32(out[4:], uint32(len(data)))
	binary.LittleEndian.PutUint32(out[8:], uint32(len(ranges)))
	previous = 0
	for i, r := range ranges {
		id := TextureID(r.Texture)
		hash, err := hex.DecodeString(id)
		if err != nil {
			return nil, nil, err
		}
		at := 12 + i*40
		binary.LittleEndian.PutUint32(out[at:], uint32(r.Offset))
		binary.LittleEndian.PutUint32(out[at+4:], uint32(len(r.Texture.Pixels)))
		copy(out[at+8:], hash)
		out = append(out, data[previous:r.Offset]...)
		previous = r.Offset + len(r.Texture.Pixels)
		textures[id] = r.Texture
	}
	out = append(out, data[previous:]...)
	return out, textures, nil
}

// BuildTexturePack stores blobs once, sorted by their source SHA-256. TPK1's
// index holds {SHA-256, absolute offset, length}; lookup does not decode any
// texture or build a startup map. It is also deterministic after regeneration.
func BuildTexturePack(blobs map[string][]byte) ([]byte, error) {
	ids := make([]string, 0, len(blobs))
	total := 8 + len(blobs)*40
	for id, blob := range blobs {
		ids = append(ids, id)
		total += len(blob)
	}
	if total > anm.MaxPackedBytes {
		return nil, fmt.Errorf("anm: texture pack too large")
	}
	sort.Strings(ids)
	out := make([]byte, 8+len(ids)*40, total)
	copy(out, "TPK1")
	binary.LittleEndian.PutUint32(out[4:], uint32(len(ids)))
	for i, id := range ids {
		hash, err := hex.DecodeString(id)
		if err != nil || len(hash) != 32 {
			return nil, fmt.Errorf("anm: invalid texture ID %q", id)
		}
		at, blob := 8+i*40, blobs[id]
		copy(out[at:], hash)
		binary.LittleEndian.PutUint32(out[at+32:], uint32(len(out)))
		binary.LittleEndian.PutUint32(out[at+36:], uint32(len(blob)))
		out = append(out, blob...)
	}
	return out, nil
}
