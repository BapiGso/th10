package anmdraw

import (
	"image"
	"image/color"
	"testing"

	"th10/anm"
	"th10/gamedata"
)

// thtx returns a w x h THTX block of format f whose pixel i has every
// byte equal to i+1 (wrapping), so each texel is distinguishable.
func thtx(f, w, h int) *anm.Texture {
	bpp := map[int]int{anm.FormatBGRA8888: 4, anm.FormatRGB565: 2, anm.FormatARGB4444: 2}[f]
	px := make([]byte, w*h*bpp)
	for i := range px {
		px[i] = byte(i/bpp + 1)
	}
	return &anm.Texture{Format: f, W: w, H: h, Pixels: px}
}

// TestEntryImage checks an entry's texels: its THTX block at the top left
// of a Width x Height texture, zero elsewhere, and zero means opaque black
// in R5G6B5 (format 3), which has no alpha.
func TestEntryImage(t *testing.T) {
	for _, tc := range []struct {
		name    string
		e       anm.Entry
		padding color.NRGBA
	}{
		{"bgra smaller", anm.Entry{Width: 8, Height: 4, Format: 1, Texture: thtx(1, 3, 2)}, color.NRGBA{}},
		{"argb4444 smaller", anm.Entry{Width: 4, Height: 4, Format: 5, Texture: thtx(5, 2, 3)}, color.NRGBA{}},
		{"rgb565 smaller", anm.Entry{Width: 4, Height: 8, Format: 3, Texture: thtx(3, 4, 5)}, color.NRGBA{A: 0xff}},
		{"bgra full", anm.Entry{Width: 4, Height: 2, Format: 1, Texture: thtx(1, 4, 2)}, color.NRGBA{}},
		{"@ bgra", anm.Entry{Name: "@", Width: 8, Height: 8, Format: 1}, color.NRGBA{}},
		{"@ rgb565", anm.Entry{Name: "@", Width: 8, Height: 8, Format: 3}, color.NRGBA{A: 0xff}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := entryImage(&tc.e)
			if got.Bounds() != image.Rect(0, 0, tc.e.Width, tc.e.Height) {
				t.Fatalf("bounds %v, want %dx%d", got.Bounds(), tc.e.Width, tc.e.Height)
			}
			var src *image.NRGBA
			if tc.e.Texture != nil {
				src = tc.e.Texture.Image()
			}
			for y := range tc.e.Height {
				for x := range tc.e.Width {
					want := tc.padding
					if src != nil && x < tc.e.Texture.W && y < tc.e.Texture.H {
						want = src.NRGBAAt(x, y)
					}
					if c := got.NRGBAAt(x, y); c != want {
						t.Fatalf("(%d, %d) = %v, want %v", x, y, c, want)
					}
				}
			}
		})
	}
}

// TestNewSheet checks the runtime records against FUN_00447470 and
// FUN_00447940's arithmetic (m4_r6.md R6-A.2), and that entries sharing a
// name get textures of their own.
func TestNewSheet(t *testing.T) {
	f := &anm.File{
		Entries: []anm.Entry{
			{Name: "a.png", Width: 256, Height: 128, Format: 1, Texture: thtx(1, 4, 4)},
			{Name: "a.png", Width: 64, Height: 32, Format: 5, Texture: thtx(5, 4, 4)},
		},
		Sprites: map[int]anm.Sprite{
			0: {ID: 0, X: 1.1, Y: 2.7, W: 3.3, H: 77.9, Entry: 0},
			1: {ID: 1, X: 0, Y: 0, W: 256, H: 128, Entry: 0},
			2: {ID: 2, X: 13.25, Y: 0.1, W: 50.7, H: 31.9, Entry: 1},
		},
	}
	s := NewSheet(f)
	if len(s.Sprites) != 3 {
		t.Fatalf("%d sprites, want 3", len(s.Sprites))
	}
	t0, t1 := s.Sprites[0].Tex, s.Sprites[2].Tex
	if t0 == nil || t1 == nil || t0 == t1 || s.Sprites[1].Tex != t0 {
		t.Fatalf("textures %p %p %p: want one per entry", t0, s.Sprites[1].Tex, t1)
	}
	if t0.W != 256 || t0.H != 128 || t1.W != 64 || t1.H != 32 {
		t.Fatalf("texture sizes %dx%d, %dx%d", t0.W, t0.H, t1.W, t1.H)
	}
	for id, sp := range f.Sprites {
		e := f.Entries[sp.Entry]
		wd, hd := float32(e.Width), float32(e.Height) // the texture's size; D3DX pads nothing
		f1, f2 := wd/float32(e.Width), hd/float32(e.Height)
		x0, y0 := sp.X*f1, sp.Y*f2
		x1, y1 := (sp.W+sp.X)*f1, (sp.H+sp.Y)*f2
		want := Sprite{
			Tex: s.Sprites[id].Tex,
			U0:  x0 / wd, V0: y0 / hd, U1: x1 / wd, V1: y1 / hd,
			W: (x1 - x0) / f1, H: (y1 - y0) / f2,
		}
		if got := s.Sprites[id]; got != want {
			t.Errorf("sprite %d: %+v, want %+v", id, got, want)
		}
	}
}

// TestNewSheetShipped builds the sheet of every embedded .anm: one record
// per sprite id, every one on a texture of its entry's size.
func TestNewSheetShipped(t *testing.T) {
	for _, name := range []string{"bullet.anm", "enemy.anm", "pl00.anm", "stgenm01.anm", "stgenm07.anm"} {
		f, err := gamedata.ANM(name)
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		s := NewSheet(f)
		if len(s.Sprites) != len(f.Sprites) {
			t.Errorf("%s: %d records, want %d", name, len(s.Sprites), len(f.Sprites))
			continue
		}
		for id, sp := range f.Sprites {
			e := f.Entries[sp.Entry]
			r := s.Sprites[id]
			if r.Tex == nil || r.Tex.W != e.Width || r.Tex.H != e.Height {
				t.Errorf("%s sprite %d: texture %+v, want %dx%d", name, id, r.Tex, e.Width, e.Height)
			}
			if r.W != sp.W || r.H != sp.H {
				t.Errorf("%s sprite %d: size %gx%g, want %gx%g", name, id, r.W, r.H, sp.W, sp.H)
			}
		}
	}
}
