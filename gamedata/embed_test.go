package gamedata

import (
	"bytes"
	"errors"
	"fmt"
	"image"
	"image/draw"
	"io/fs"
	"path/filepath"
	"strings"
	"testing"

	"th10/anm"
	"th10/assets"
	"th10/datfs"
)

// notInDat are embedded pictures under anm/ that th10.dat does not carry.
var notInDat = map[string]bool{
	"ascii/leaf.png":   true,
	"loading/sigm.png": true,
}

// TestEmbeddedMatchesDat checks the embedded copies against the user's
// th10.dat: raw files byte for byte, decoded textures pixel for pixel (colour
// under zero alpha ignored). Skips without a TH10 install.
func TestEmbeddedMatchesDat(t *testing.T) {
	dir, err := datfs.Locate()
	if err != nil {
		t.Skip(err)
	}
	dat, err := datfs.Open(filepath.Join(dir, "th10.dat"))
	if err != nil {
		t.Fatal(err)
	}
	defer dat.Close()

	raw := 0
	for _, name := range dat.Names() {
		got, err := ReadFile(datfs.EmbedPath(name))
		if errors.Is(err, fs.ErrNotExist) {
			continue // not embedded yet
		} else if err != nil {
			t.Fatal(err)
		}
		want, err := dat.ReadFile(name)
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(got, want) {
			t.Errorf("%s differs from th10.dat:%s", datfs.EmbedPath(name), name)
		}
		raw++
	}

	textures := datTextures(t, dat)
	pics := 0
	err = fs.WalkDir(assets.Assets, "anm", func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(p, ".png") {
			return err
		}
		name := strings.TrimPrefix(p, "anm/")
		var want image.Image
		if f, ok := textures[name]; ok {
			want = f.NamedImage(name)
		} else if pieces, ok := composites[name]; ok {
			want, err = composite(dat, pieces)
			if err != nil {
				return err
			}
		} else {
			if !notInDat[name] {
				t.Errorf("%s: no texture of that name in th10.dat", p)
			}
			return nil
		}
		got, err := Image(p)
		if err != nil {
			return err
		}
		if msg := comparePixels(got, want); msg != "" {
			t.Errorf("%s: %s", p, msg)
		}
		pics++
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("checked %d raw files and %d decoded textures", raw, pics)
}

// datTextures maps each texture name to the first .anm in th10.dat (archive
// order) carrying its pixels.
func datTextures(t *testing.T, dat *datfs.Archive) map[string]*anm.File {
	m := map[string]*anm.File{}
	for _, name := range dat.Names() {
		if !strings.HasSuffix(name, ".anm") {
			continue
		}
		f, err := datANM(dat, name)
		if err != nil {
			t.Fatal(err)
		}
		for _, e := range f.Entries {
			if _, seen := m[e.Name]; !seen && e.Texture != nil {
				m[e.Name] = f
			}
		}
	}
	return m
}

func datANM(dat *datfs.Archive, name string) (*anm.File, error) {
	data, err := dat.ReadFile(name)
	if err != nil {
		return nil, err
	}
	return anm.Load(data)
}

// piece places sprite Sprite of the .anm File (Sprite < 0: the whole
// texture Tex) at (X, Y).
type piece struct {
	File, Tex string
	Sprite    int
	X, Y      int
}

// composites are embedded full-screen pictures the original draws as several
// ANM sprites; the 2D screens use them until the ANM VM draws the parts
// itself. Positions are the op 48 (pos) of the scripts named in the comments.
var composites = map[string][]piece{
	// front.anm scripts 2–5 (children of script 0): the frame around the field.
	"front/front00s.png": {
		{File: "front.anm", Sprite: 0, X: 0, Y: 0},
		{File: "front.anm", Sprite: 1, X: 416, Y: 0},
		{File: "front.anm", Sprite: 2, X: 32, Y: 0},
		{File: "front.anm", Sprite: 3, X: 32, Y: 464},
	},
	"title/title00s.png": {
		{File: "title.anm", Tex: "title/title00a.png", Sprite: -1},
		{File: "title.anm", Tex: "title/title00b.png", Sprite: -1, X: 512},
	},
	"title/select00s.png": {
		{File: "title.anm", Tex: "title/select00.png", Sprite: -1},
		{File: "title.anm", Tex: "title/select00b.png", Sprite: -1, X: 512},
	},
}

func composite(dat *datfs.Archive, pieces []piece) (image.Image, error) {
	out := image.NewNRGBA(image.Rect(0, 0, 640, 480))
	for _, pc := range pieces {
		f, err := datANM(dat, pc.File)
		if err != nil {
			return nil, err
		}
		var src image.Image
		var r image.Rectangle
		if pc.Sprite < 0 {
			tex := f.NamedImage(pc.Tex)
			if tex == nil {
				return nil, fmt.Errorf("%s has no %s", pc.File, pc.Tex)
			}
			src, r = tex, tex.Bounds()
		} else {
			sp, ok := f.Sprites[pc.Sprite]
			if !ok {
				return nil, fmt.Errorf("%s has no sprite %d", pc.File, pc.Sprite)
			}
			tex := f.NamedImage(f.Entries[sp.Entry].Name)
			if tex == nil {
				return nil, fmt.Errorf("%s sprite %d has no pixels", pc.File, pc.Sprite)
			}
			src, r = tex, image.Rect(int(sp.X), int(sp.Y), int(sp.X+sp.W), int(sp.Y+sp.H))
		}
		draw.Draw(out, r.Sub(r.Min).Add(image.Pt(pc.X, pc.Y)), src, r.Min, draw.Src)
	}
	return out, nil
}

// comparePixels reports how got differs from want, or "" if they match.
func comparePixels(got, want image.Image) string {
	if got.Bounds() != want.Bounds() {
		return fmt.Sprintf("bounds %v, th10.dat has %v", got.Bounds(), want.Bounds())
	}
	a, b := toNRGBA(got), toNRGBA(want)
	n := 0
	for i := 0; i < len(a.Pix); i += 4 {
		if a.Pix[i+3] == 0 && b.Pix[i+3] == 0 {
			continue
		}
		if !bytes.Equal(a.Pix[i:i+4], b.Pix[i:i+4]) {
			n++
		}
	}
	if n > 0 {
		return fmt.Sprintf("%d pixels differ from th10.dat", n)
	}
	return ""
}

func toNRGBA(img image.Image) *image.NRGBA {
	if n, ok := img.(*image.NRGBA); ok && n.Rect.Min == (image.Point{}) {
		return n
	}
	n := image.NewNRGBA(img.Bounds().Sub(img.Bounds().Min))
	draw.Draw(n, n.Rect, img, img.Bounds().Min, draw.Src)
	return n
}
