package anm

import (
	"encoding/binary"
	"fmt"
	"image"
)

// THTX pixel formats (thtk extlib/thtypes/anm_types.h). 16-bit formats are
// little-endian.
const (
	FormatBGRA8888 = 1 // bytes B, G, R, A
	FormatRGB565   = 3
	FormatARGB4444 = 5 // bytes GB, AR
	FormatGray8    = 7
)

// Texture is an entry's raw THTX pixel block. W×H may be smaller than the
// entry's (power-of-two) Width×Height.
type Texture struct {
	Format int
	W, H   int
	Pixels []byte // aliases the .anm data
}

func parseTHTX(data []byte, off int) (*Texture, error) {
	if off < 0 || off+16 > len(data) || string(data[off:off+4]) != "THTX" {
		return nil, fmt.Errorf("anm: no THTX block at 0x%x", off)
	}
	t := &Texture{
		Format: int(binary.LittleEndian.Uint16(data[off+6:])),
		W:      int(binary.LittleEndian.Uint16(data[off+8:])),
		H:      int(binary.LittleEndian.Uint16(data[off+10:])),
	}
	size := int(binary.LittleEndian.Uint32(data[off+12:]))
	if off+16+size > len(data) {
		return nil, errTooShort
	}
	t.Pixels = data[off+16 : off+16+size]
	if bpp := bytesPerPixel(t.Format); bpp == 0 {
		return nil, fmt.Errorf("anm: unknown THTX format %d", t.Format)
	} else if t.W*t.H*bpp > size {
		return nil, errTooShort
	}
	return t, nil
}

func bytesPerPixel(format int) int {
	switch format {
	case FormatBGRA8888:
		return 4
	case FormatRGB565, FormatARGB4444:
		return 2
	case FormatGray8:
		return 1
	}
	return 0
}

// Image converts the pixels to straight-alpha RGBA. Narrow channels are
// widened by bit replication (5-bit v -> v<<3|v>>2, 4-bit v -> v*17), the
// same expansion D3D9 applies when sampling. FormatGray8 becomes opaque gray,
// as thanm extracts it; how th10.exe maps it to a D3D format is unverified.
func (t *Texture) Image() *image.NRGBA {
	img := image.NewNRGBA(image.Rect(0, 0, t.W, t.H))
	n := t.W * t.H
	px := img.Pix
	src := t.Pixels
	switch t.Format {
	case FormatBGRA8888:
		for i := range n {
			px[i*4+0] = src[i*4+2]
			px[i*4+1] = src[i*4+1]
			px[i*4+2] = src[i*4+0]
			px[i*4+3] = src[i*4+3]
		}
	case FormatRGB565:
		for i := range n {
			v := binary.LittleEndian.Uint16(src[i*2:])
			r := byte(v >> 11 & 0x1f)
			g := byte(v >> 5 & 0x3f)
			b := byte(v & 0x1f)
			px[i*4+0] = r<<3 | r>>2
			px[i*4+1] = g<<2 | g>>4
			px[i*4+2] = b<<3 | b>>2
			px[i*4+3] = 0xff
		}
	case FormatARGB4444:
		for i := range n {
			gb, ar := src[i*2], src[i*2+1]
			px[i*4+0] = (ar & 0x0f) * 17
			px[i*4+1] = (gb >> 4) * 17
			px[i*4+2] = (gb & 0x0f) * 17
			px[i*4+3] = (ar >> 4) * 17
		}
	case FormatGray8:
		for i := range n {
			v := src[i]
			px[i*4+0], px[i*4+1], px[i*4+2], px[i*4+3] = v, v, v, 0xff
		}
	}
	return img
}

// NamedImage assembles the picture called name from every entry that carries
// it, placing each tile at its (X, Y). Returns nil if no entry has pixels.
func (f *File) NamedImage(name string) *image.NRGBA {
	w, h := 0, 0
	for _, e := range f.Entries {
		if e.Name == name && e.Texture != nil {
			w = max(w, e.X+e.Texture.W)
			h = max(h, e.Y+e.Texture.H)
		}
	}
	if w == 0 || h == 0 {
		return nil
	}
	out := image.NewNRGBA(image.Rect(0, 0, w, h))
	for _, e := range f.Entries {
		if e.Name != name || e.Texture == nil {
			continue
		}
		tile := e.Texture.Image()
		for y := 0; y < e.Texture.H; y++ {
			copy(out.Pix[out.PixOffset(e.X, e.Y+y):], tile.Pix[tile.PixOffset(0, y):tile.PixOffset(e.Texture.W, y)])
		}
	}
	return out
}
