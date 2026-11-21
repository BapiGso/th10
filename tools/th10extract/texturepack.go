package main

import (
	"bytes"
	"compress/flate"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"sort"

	"th10/anm"
	"th10/datfs"
)

type textureProduct struct {
	ID          string   `json:"id"`
	References  []string `json:"references"`
	Name        string   `json:"name"`
	Format      int      `json:"format"`
	Width       int      `json:"width"`
	Height      int      `json:"height"`
	RawBytes    int      `json:"raw_bytes"`
	Lossless    int      `json:"lossless_bytes"`
	LosslessAs  string   `json:"lossless_codec"`
	Web         int      `json:"web_bytes"`
	WebAs       string   `json:"web_codec"`
	JPEGQuality int      `json:"jpeg_quality,omitempty"`
	ColorBits   []int    `json:"color_bits,omitempty"`
	VisiblePSNR float64  `json:"visible_psnr,omitempty"`
	AllPSNR     float64  `json:"all_psnr,omitempty"`
	EdgePSNR    float64  `json:"edge_psnr,omitempty"`
	EdgeSamples int      `json:"edge_samples,omitempty"`
}

type textureManifest struct {
	Version      int               `json:"version"`
	LogicalBytes int               `json:"logical_bytes"`
	Lossless     int               `json:"lossless_pack_bytes"`
	Web          int               `json:"web_pack_bytes"`
	Textures     []*textureProduct `json:"textures"`
}

// packTextures reads the original archives, even when -out already contains
// generated files. Only ANMs already present in -out are regenerated. This
// preserves the chosen runtime resource set and makes reruns idempotent.
func packTextures(dir, out string, web bool, compare string) error {
	if dir == "" {
		var err error
		if dir, err = datfs.Locate(); err != nil {
			return err
		}
	}
	dats := map[string]*datfs.Archive{}
	for _, name := range []string{"th10.dat", "th10c.dat"} {
		d, err := datfs.Open(filepath.Join(dir, name))
		if err != nil {
			return err
		}
		defer func() { _ = d.Close() }()
		dats[name] = d
	}
	files, err := os.ReadDir(filepath.Join(out, "anm"))
	if err != nil {
		return err
	}
	lossless, webBlobs := map[string][]byte{}, map[string][]byte{}
	products, originals := map[string]*textureProduct{}, map[string]*anm.Texture{}
	index := map[string]string{}
	manifest := textureManifest{Version: 1}
	for _, file := range files {
		name, ext := file.Name(), filepath.Ext(file.Name())
		if file.IsDir() || (ext != ".anm" && ext != ".ant") {
			continue
		}
		archive := "th10.dat"
		if ext == ".ant" {
			archive = "th10c.dat"
		}
		raw, err := dats[archive].ReadFile(name)
		if err != nil {
			return err
		}
		f, err := anm.Load(raw)
		if err != nil {
			return err
		}
		packed, textures, err := anm.Split(raw)
		if err != nil {
			return fmt.Errorf("%s: %w", name, err)
		}
		for i, e := range f.Entries {
			if e.Texture == nil {
				continue
			}
			id := anm.TextureID(e.Texture)
			p := products[id]
			if p == nil {
				t := textures[id]
				blob, codec, err := encodeLosslessTexture(t)
				if err != nil {
					return fmt.Errorf("%s/%s: %w", name, e.Name, err)
				}
				lossless[id], webBlobs[id], originals[id] = blob, blob, t
				p = &textureProduct{
					ID: id, Name: e.Name, Format: t.Format, Width: t.W, Height: t.H,
					RawBytes: len(t.Pixels), Lossless: len(blob), LosslessAs: codec, Web: len(blob), WebAs: codec,
				}
				products[id] = p
			}
			p.References = append(p.References, fmt.Sprintf("%s#%d", name, i))
			if ext == ".anm" {
				if _, seen := index[e.Name]; !seen {
					index[e.Name] = name
				}
			}
		}
		if err := write(filepath.Join(out, "anm", name), packed); err != nil {
			return err
		}
		manifest.LogicalBytes += len(packed)
	}
	ids := make([]string, 0, len(products))
	for id := range products {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	for _, id := range ids {
		p := products[id]
		if web {
			blob, err := chooseWebTexture(originals[id], lossless[id], p)
			if err != nil {
				return err
			}
			webBlobs[id] = blob
		}
		manifest.Textures = append(manifest.Textures, p)
	}
	for _, variant := range []string{"lossless", "web"} {
		blobs := lossless
		if variant == "web" {
			blobs = webBlobs
		}
		pack, err := anm.BuildTexturePack(blobs)
		if err != nil {
			return err
		}
		if variant == "lossless" {
			manifest.Lossless = len(pack)
		} else {
			manifest.Web = len(pack)
		}
		if err := write(filepath.Join(out, "textures", variant+".tpk"), pack); err != nil {
			return err
		}
	}
	indexData, err := json.MarshalIndent(index, "", "  ")
	if err != nil {
		return err
	}
	if err := write(filepath.Join(out, "anm", "texture_index.json"), append(indexData, '\n')); err != nil {
		return err
	}
	data, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return err
	}
	if err := write(filepath.Join(filepath.Dir(filepath.Clean(out)), "docs", "research", "textures_manifest.json"), append(data, '\n')); err != nil {
		return err
	}
	if web && compare != "" {
		return writeTextureComparisons(compare, manifest.Textures, originals, webBlobs)
	}
	return nil
}

func textureHeader(t *anm.Texture, codec byte, payload []byte) []byte {
	out := make([]byte, 16, 16+len(payload))
	copy(out, "TXP1")
	out[4], out[5] = codec, byte(t.Format)
	binary.LittleEndian.PutUint16(out[6:], uint16(t.W))
	binary.LittleEndian.PutUint16(out[8:], uint16(t.H))
	binary.LittleEndian.PutUint32(out[12:], uint32(len(t.Pixels)))
	return append(out, payload...)
}

func deflateBytes(data []byte) ([]byte, error) {
	var buf bytes.Buffer
	w, err := flate.NewWriter(&buf, flate.BestCompression)
	if err != nil {
		return nil, err
	}
	if _, err := w.Write(data); err != nil {
		return nil, err
	}
	if err := w.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func encodeLosslessTexture(t *anm.Texture) ([]byte, string, error) {
	raw, err := deflateBytes(t.Pixels)
	if err != nil {
		return nil, "", err
	}
	best, codec := textureHeader(t, 0, raw), "flate"
	try := func(candidate []byte, kind string) {
		if len(candidate) >= len(best) {
			return
		}
		decoded, err := anm.DecodeTexture(candidate)
		if err == nil && bytes.Equal(decoded.Pixels, t.Pixels) {
			best, codec = candidate, kind
		}
	}
	bpp := pixelBytes(t.Format)
	if t.W*t.H*bpp != len(t.Pixels) {
		return best, codec, nil
	}
	filtered, err := deflateBytes(filterPixels(t.Pixels, t.W*bpp, t.H, bpp))
	if err != nil {
		return nil, "", err
	}
	try(textureHeader(t, 1, filtered), "filtered-flate")
	img := t.Image()
	var buf bytes.Buffer
	encoder := png.Encoder{CompressionLevel: png.BestCompression}
	if err := encoder.Encode(&buf, img); err != nil {
		return nil, "", err
	}
	try(textureHeader(t, 2, buf.Bytes()), "png")
	palette, indices := paletteImage(img)
	if palette != nil {
		buf.Reset()
		if err := encoder.Encode(&buf, indices); err != nil {
			return nil, "", err
		}
		try(textureHeader(t, 2, buf.Bytes()), "palette-png")
	}
	decoded, err := anm.DecodeTexture(best)
	if err != nil || !bytes.Equal(decoded.Pixels, t.Pixels) {
		return nil, "", fmt.Errorf("lossless texture roundtrip failed")
	}
	return best, codec, nil
}

func paletteImage(img *image.NRGBA) (color.Palette, *image.Paletted) {
	colors := map[uint32]byte{}
	palette := color.Palette{}
	indices := image.NewPaletted(img.Rect, nil)
	for at := 0; at < len(img.Pix); at += 4 {
		key := binary.LittleEndian.Uint32(img.Pix[at:])
		index, ok := colors[key]
		if !ok {
			if len(palette) == 256 {
				return nil, nil
			}
			index = byte(len(palette))
			colors[key] = index
			palette = append(palette, color.NRGBA{R: img.Pix[at], G: img.Pix[at+1], B: img.Pix[at+2], A: img.Pix[at+3]})
		}
		indices.Pix[at/4] = index
	}
	indices.Palette = palette
	return palette, indices
}

func pixelBytes(format int) int {
	switch format {
	case anm.FormatBGRA8888:
		return 4
	case anm.FormatRGB565, anm.FormatARGB4444:
		return 2
	case anm.FormatGray8:
		return 1
	}
	return 0
}

func filterPixels(pixels []byte, stride, height, bpp int) []byte {
	out := make([]byte, (stride+1)*height)
	candidate := make([]byte, stride)
	for y := range height {
		row := pixels[y*stride : (y+1)*stride]
		bestCost := int(^uint(0) >> 1)
		for kind := byte(0); kind <= 4; kind++ {
			cost := 0
			for x, value := range row {
				var left, up, upperLeft, prediction byte
				if x >= bpp {
					left = row[x-bpp]
				}
				if y > 0 {
					up = pixels[(y-1)*stride+x]
					if x >= bpp {
						upperLeft = pixels[(y-1)*stride+x-bpp]
					}
				}
				switch kind {
				case 1:
					prediction = left
				case 2:
					prediction = up
				case 3:
					prediction = byte((int(left) + int(up)) / 2)
				case 4:
					prediction = predictPaeth(left, up, upperLeft)
				}
				candidate[x] = value - prediction
				cost += absInt(int(int8(candidate[x])))
			}
			if cost < bestCost {
				bestCost = cost
				out[y*(stride+1)] = kind
				copy(out[y*(stride+1)+1:(y+1)*(stride+1)], candidate)
			}
		}
	}
	return out
}

func predictPaeth(a, b, c byte) byte {
	p := int(a) + int(b) - int(c)
	pa, pb, pc := absInt(p-int(a)), absInt(p-int(b)), absInt(p-int(c))
	if pa <= pb && pa <= pc {
		return a
	}
	if pb <= pc {
		return b
	}
	return c
}

func absInt(n int) int {
	if n < 0 {
		return -n
	}
	return n
}
