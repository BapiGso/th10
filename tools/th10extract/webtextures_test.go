// Both texture variants share exactly the same logical ANM bytes.
package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"th10/anm"
)

func TestTextureVariantsMetadataAndQuality(t *testing.T) {
	root := filepath.Join("..", "..", "assets")
	lossless, err := os.ReadFile(filepath.Join(root, "textures", "lossless.tpk"))
	if err != nil {
		t.Fatal(err)
	}
	web, err := os.ReadFile(filepath.Join(root, "textures", "web.tpk"))
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join("..", "..", "docs", "research", "textures_manifest.json"))
	if err != nil {
		t.Fatal(err)
	}
	var manifest textureManifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		t.Fatal(err)
	}
	for _, p := range manifest.Textures {
		x, err := anm.TextureBlob(lossless, p.ID)
		if err != nil {
			t.Fatal(err)
		}
		y, err := anm.TextureBlob(web, p.ID)
		if err != nil {
			t.Fatal(err)
		}
		a, err := anm.DecodeTexture(x)
		if err != nil {
			t.Fatal(err)
		}
		b, err := anm.DecodeTexture(y)
		if err != nil {
			t.Fatal(err)
		}
		q, err := anm.CompareTextureQuality(a, b)
		if err != nil || !q.Acceptable() {
			t.Errorf("%s (%s): unacceptable web quality %+v: %v", p.Name, p.ID, q, err)
		}
		if p.JPEGQuality == 0 && len(p.ColorBits) == 0 && !bytes.Equal(a.Pixels, b.Pixels) {
			t.Errorf("%s: a lossless web image changed pixels", p.Name)
		}
	}
	files, err := os.ReadDir(filepath.Join(root, "anm"))
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range files {
		ext := filepath.Ext(entry.Name())
		if ext != ".anm" && ext != ".ant" {
			continue
		}
		data, err := os.ReadFile(filepath.Join(root, "anm", entry.Name()))
		if err != nil {
			t.Fatal(err)
		}
		a, err := anm.Restore(data, func(id string) ([]byte, error) { return anm.TextureBlob(lossless, id) })
		if err != nil {
			t.Fatal(err)
		}
		b, err := anm.Restore(data, func(id string) ([]byte, error) { return anm.TextureBlob(web, id) })
		if err != nil {
			t.Fatal(err)
		}
		x, err := anm.Load(a)
		if err != nil {
			t.Fatal(err)
		}
		y, err := anm.Load(b)
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(x.Sprites, y.Sprites) || !reflect.DeepEqual(x.Scripts, y.Scripts) || !reflect.DeepEqual(x.ScriptOrder, y.ScriptOrder) {
			t.Errorf("%s: web variant changed sprite table or bytecode", entry.Name())
		}
		ranges, err := anm.PixelRanges(a)
		if err != nil {
			t.Fatal(err)
		}
		for _, r := range ranges {
			clear(a[r.Offset : r.Offset+len(r.Texture.Pixels)])
			clear(b[r.Offset : r.Offset+len(r.Texture.Pixels)])
		}
		if !bytes.Equal(a, b) {
			t.Errorf("%s: web variant changed non-pixel bytes", entry.Name())
		}
	}
}

func TestJPEGAlphaDoesNotPremultiplyHiddenRGB(t *testing.T) {
	texture := &anm.Texture{Format: anm.FormatBGRA8888, W: 32, H: 32, Pixels: make([]byte, 32*32*4)}
	for at := 0; at < len(texture.Pixels); at += 4 {
		texture.Pixels[at], texture.Pixels[at+1], texture.Pixels[at+2], texture.Pixels[at+3] = 37, 72, 113, byte((at/4)%256)
	}
	blob, err := encodeJPEGAlpha(texture, 97)
	if err != nil {
		t.Fatal(err)
	}
	decoded, err := anm.DecodeTexture(blob)
	if err != nil {
		t.Fatal(err)
	}
	q, err := anm.CompareTextureQuality(texture, decoded)
	if err != nil || !q.Acceptable() {
		t.Fatalf("JPEG discarded or premultiplied hidden RGB: %+v %v", q, err)
	}
	blob, err = encodeRGBJPEGAlpha(texture, 97)
	if err != nil {
		t.Fatal(err)
	}
	decoded, err = anm.DecodeTexture(blob)
	if err != nil {
		t.Fatal(err)
	}
	q, err = anm.CompareTextureQuality(texture, decoded)
	if err != nil || !q.Acceptable() {
		t.Fatalf("RGB JPEG changed hidden RGB or alpha: %+v %v", q, err)
	}
}
