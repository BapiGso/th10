// Font subsetting must preserve the existing rasterizer's exact pixel output.
package render

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"image"
	"image/color"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"th10/gamedata"

	ebitext "github.com/hajimehoshi/ebiten/v2/text/v2"
	"golang.org/x/image/font"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/math/fixed"
)

type pixelFontReference struct {
	Name  string `json:"name"`
	Text  string `json:"text"`
	Cases []struct {
		Size  int    `json:"size"`
		Right bool   `json:"right"`
		Color uint32 `json:"color"`
		Hash  string `json:"sha256"`
	} `json:"cases"`
}

func originalFontsDirectory() string {
	if p := os.Getenv("TH10_FONT_SOURCE"); p != "" {
		return p
	}
	p, err := os.Getwd()
	if err != nil {
		return ""
	}
	for {
		candidate := filepath.Join(p, ".tmp", "codex", "base", "assets", "font")
		if _, err := os.Stat(filepath.Join(candidate, "DFYuGaSo-W3.ttc")); err == nil {
			return candidate
		}
		parent := filepath.Dir(p)
		if p == parent {
			return ""
		}
		p = parent
	}
}

func pixelReferences(t *testing.T) []pixelFontReference {
	t.Helper()
	var references []pixelFontReference
	if err := json.Unmarshal([]byte(fontPixelReferenceJSON), &references); err != nil {
		t.Fatal(err)
	}
	return references
}

// The independent reference hashes come from the unmodified original fonts.
// They keep all-glyph pixel checks available without shipping full TTCs.
func TestFontSubsetReferencePixels(t *testing.T) {
	for _, ref := range pixelReferences(t) {
		for _, c := range ref.Cases {
			style := TextStyle{Size: c.Size, Bold: strings.Contains(ref.Name, "W7"), Right: c.Right, Color: c.Color}
			h := sha256.New()
			runes := []rune(ref.Text)
			for start := 0; start < len(runes); start += 24 {
				img := RasterText(string(runes[start:min(start+24, len(runes))]), 1024, 64, style)
				_, _ = h.Write(img.Pix)
			}
			if got := hex.EncodeToString(h.Sum(nil)); got != c.Hash {
				t.Errorf("%s size=%d right=%t ink=%06x: pixels %s, original %s", ref.Name, c.Size, c.Right, c.Color, got, c.Hash)
			}
		}
	}
}

func TestFontSubsetDirectOriginalPixelComparison(t *testing.T) {
	dir := originalFontsDirectory()
	if dir == "" {
		t.Skip("set TH10_FONT_SOURCE to compare against original TTCs; reference pixel hashes still run")
	}
	for _, ref := range pixelReferences(t) {
		data, err := os.ReadFile(filepath.Join(dir, ref.Name))
		if err != nil {
			t.Fatal(err)
		}
		collection, err := opentype.ParseCollection(data)
		if err != nil {
			t.Fatal(err)
		}
		f, err := collection.Font(0)
		if err != nil {
			t.Fatal(err)
		}
		for _, c := range ref.Cases {
			face, err := opentype.NewFace(f, &opentype.FaceOptions{Size: float64(c.Size), DPI: 72, Hinting: font.HintingFull})
			if err != nil {
				t.Fatal(err)
			}
			style := TextStyle{Size: c.Size, Bold: strings.Contains(ref.Name, "W7"), Right: c.Right, Color: c.Color}
			runes := []rune(ref.Text)
			for start := 0; start < len(runes); start += 24 {
				text := string(runes[start:min(start+24, len(runes))])
				got := RasterText(text, 1024, 64, style)
				want := originalRasterForTest(text, face, style)
				if !bytes.Equal(got.Pix, want.Pix) {
					t.Errorf("%s size=%d right=%t ink=%06x character chunk %d: changed pixels", ref.Name, c.Size, c.Right, c.Color, start)
				}
			}
			_ = face.Close()
		}
	}
}

func originalRasterForTest(text string, face font.Face, style TextStyle) *image.NRGBA {
	img := image.NewNRGBA(image.Rect(0, 0, 1024, 64))
	x := 0
	if style.Right {
		x = 1024 - textByteWidth(text)*(style.Size-1)/2
	}
	y := face.Metrics().Ascent.Ceil()
	d := font.Drawer{Dst: img, Face: face, Src: image.NewUniform(color.NRGBA{A: 255}), Dot: fixed.P(x+1, y+1)}
	d.DrawString(text)
	d.Src = image.NewUniform(color.NRGBA{R: byte(style.Color >> 16), G: byte(style.Color >> 8), B: byte(style.Color), A: 255})
	d.Dot = fixed.P(x, y)
	d.DrawString(text)
	return img
}

func TestFontSubsetTextV2MetricsAndFallback(t *testing.T) {
	dir := originalFontsDirectory()
	for _, name := range []string{"Bimini Bold.ttf", "DFYuGaSo-W3.ttc", "DFYuGaSo-W7.ttc"} {
		for _, size := range []float64{16, 17, 20, 28} {
			face := LoadTextFace("font/"+name, size)
			if face == nil {
				t.Fatalf("%s failed to load", name)
			}
			if dir != "" {
				data, err := os.ReadFile(filepath.Join(dir, name))
				if err != nil {
					t.Fatal(err)
				}
				source := parseFontBytes(data)
				if source == nil {
					t.Fatal("original font failed to load")
				}
				old := &ebitext.GoTextFace{Source: source, Size: size}
				if face.Metrics() != old.Metrics() {
					t.Errorf("%s size %g: changed text/v2 metrics", name, size)
				}
				w, h := ebitext.Measure("AVATAR To Wa 0123456789", face, 0)
				x, y := ebitext.Measure("AVATAR To Wa 0123456789", old, 0)
				if w != x || h != y {
					t.Errorf("%s size %g: changed text/v2 kerning/placement", name, size)
				}
			}
			// Characters outside the fixed corpus resolve to the retained
			// .notdef; they must remain safe even for imported replay names.
			_, _ = ebitext.Measure(string(rune(0x10ffff)), face, 0)
		}
	}
	for _, bold := range []bool{false, true} {
		img := RasterText(string(rune(0x10ffff)), 64, 64, TextStyle{Size: 17, Bold: bold, Color: 0xffffff})
		if img.Bounds() != image.Rect(0, 0, 64, 64) {
			t.Fatal("unknown character changed the sprite size")
		}
	}
	if _, err := gamedata.ReadFile("font/DFYuGaSo-W3.ttc"); err != nil {
		t.Fatal(err)
	}
}
