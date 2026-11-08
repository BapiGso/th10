package render

// The portable approximation of FUN_004479d0/00447a50/00447ae0: rasterize
// embedded text into an ANM sprite's texture, leaving its VM timeline,
// position, anchors, colour/alpha interpolation and draw path untouched.
// Rasterization is CPU-only, so the same VMs work in the recording-device
// smoke tests and WASM. It is not a pixel-exact reproduction of GDI/D3DX.

import (
	"image"
	"image/color"
	"maps"
	"sync"

	"th10/anm"
	"th10/gamedata"

	"golang.org/x/image/font"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/math/fixed"
)

// TextStyle models the GUI's font height, ink and horizontal placement.
// Right is FUN_00447ae0; false is FUN_00447a50. Size defaults to 17, as
// the latter's right-aligned sibling does when VM+0x3a0 is zero.
type TextStyle struct {
	Size  int
	Bold  bool
	Right bool
	Color uint32 // RGB, independent of the VM's interpolated colour/alpha
}

type vmTextKey struct {
	file   *anm.File
	sprite int
	text   string
	style  TextStyle
}

var vmTextCache = struct {
	sync.Mutex
	files   map[vmTextKey]*anm.File
	sources map[*anm.File]*anm.File
}{files: map[vmTextKey]*anm.File{}, sources: map[*anm.File]*anm.File{}}

// WriteVMText replaces only vm's texture data. A private file keeps each
// text sprite separate from other lines sharing text.anm's @ texture. Cached
// immutable files are reused, rather than accumulating a new GPU sheet each
// time the same embedded line is shown. The sprite's dimensions stay exact.
func WriteVMText(vm *anm.VM, text string, style TextStyle) {
	if vm == nil || vm.File == nil || vm.Sprite < 0 {
		return
	}
	if style.Size <= 0 {
		style.Size = 17
	}
	vmTextCache.Lock()
	defer vmTextCache.Unlock()
	source := vm.File
	if f := vmTextCache.sources[source]; f != nil {
		source = f
	}
	key := vmTextKey{source, vm.Sprite, text, style}
	f := vmTextCache.files[key]
	if f == nil {
		sp, ok := source.Sprites[vm.Sprite]
		if !ok {
			return
		}
		w, h := int(sp.W), int(sp.H)
		img := RasterText(text, w, h, style)
		pixels := make([]byte, len(img.Pix))
		for i := 0; i < len(pixels); i += 4 {
			pixels[i], pixels[i+1], pixels[i+2], pixels[i+3] = img.Pix[i+2], img.Pix[i+1], img.Pix[i], img.Pix[i+3]
		}
		entry := anm.Entry{
			Name: "@", Width: w, Height: h, Format: anm.FormatBGRA8888,
			Texture: &anm.Texture{Format: anm.FormatBGRA8888, W: w, H: h, Pixels: pixels},
		}
		f = &anm.File{Entries: []anm.Entry{entry}, Sprites: maps.Clone(source.Sprites), Scripts: source.Scripts}
		// NewSheet indexes every sprite, even though this file is for one VM.
		for id, spr := range f.Sprites {
			spr.Entry = 0
			f.Sprites[id] = spr
		}
		sp.Entry, sp.X, sp.Y = 0, 0, 0
		f.Sprites[vm.Sprite] = sp
		vmTextCache.files[key], vmTextCache.sources[f] = f, source
	}
	vm.File = f
	vm.Flags |= anm.FlagVisible // FUN_00447a50 0x447ac5, FUN_00447ae0 0x447b94
}

var rasterFaces = struct {
	sync.Mutex
	fonts [2]*opentype.Font
	faces map[[2]int]font.Face
}{faces: map[[2]int]font.Face{}}

// RasterText uses the existing DFYuGaSo fonts, not system font files. The
// GDI texture writer draws an offset black shadow, then the coloured glyphs.
// Its font-derived placement is necessarily approximate, but it does not
// substitute a screen overlay or change the ANM sprite size and timeline.
func RasterText(s string, w, h int, style TextStyle) *image.NRGBA {
	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	if s == "" || w <= 0 || h <= 0 {
		return img
	}
	if style.Size <= 0 {
		style.Size = 17
	}
	rasterFaces.Lock()
	defer rasterFaces.Unlock()
	weight := 0
	name := "font/DFYuGaSo-W3.ttc"
	if style.Bold {
		weight, name = 1, "font/DFYuGaSo-W7.ttc"
	}
	if rasterFaces.fonts[weight] == nil {
		data, err := gamedata.ReadFile(name)
		if err != nil {
			panic("render: embedded game font: " + err.Error())
		}
		collection, err := opentype.ParseCollection(data)
		if err != nil {
			panic("render: game font collection: " + err.Error())
		}
		f, err := collection.Font(0)
		if err != nil {
			panic("render: game font: " + err.Error())
		}
		rasterFaces.fonts[weight] = f
	}
	key := [2]int{weight, style.Size}
	face := rasterFaces.faces[key]
	if face == nil {
		f, err := opentype.NewFace(rasterFaces.fonts[weight], &opentype.FaceOptions{
			Size: float64(style.Size), DPI: 72, Hinting: font.HintingFull,
		})
		if err != nil {
			panic("render: game font face: " + err.Error())
		}
		rasterFaces.faces[key], face = f, f
	}
	d := font.Drawer{Dst: img, Face: face}
	x := 0
	if style.Right {
		// FUN_00447ae0 0x447b52..0x447b75 computes the byte-count width:
		// half-width ASCII, full-width CJK, font height minus one.
		x = w - textByteWidth(s)*(style.Size-1)/2
	}
	y := face.Metrics().Ascent.Ceil()
	d.Src, d.Dot = image.NewUniform(color.NRGBA{A: 255}), fixed.P(x+1, y+1)
	d.DrawString(s)
	c := style.Color
	d.Src, d.Dot = image.NewUniform(color.NRGBA{R: byte(c >> 16), G: byte(c >> 8), B: byte(c), A: 255}), fixed.P(x, y)
	d.DrawString(s)
	return img
}

// textByteWidth is the SJIS/GBK character-width convention used by the
// original texture writer's strlen, not the UTF-8 byte count in Go.
func textByteWidth(s string) int {
	n := 0
	for _, r := range s {
		if r < 0x80 {
			n++
		} else {
			n += 2
		}
	}
	return n
}
