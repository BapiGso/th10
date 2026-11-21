package render

import (
	"image"
	"image/color"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"golang.org/x/image/font"
	"golang.org/x/image/font/basicfont"
	"golang.org/x/image/math/fixed"
)

// debugText caches the last overlay string. Debug overlays replace
// ebitenutil.DebugPrintAt, whose package also links net/http into the WASM
// build; they are host diagnostics, not original GUI.
type debugText struct {
	text string
	img  *ebiten.Image
}

var debugTexts = map[[2]int]*debugText{}

// DebugPrintAt draws s with a white 7x13 bitmap font and a black shadow, top-left at (x, y).
func DebugPrintAt(screen *ebiten.Image, s string, x, y int) {
	if screen == nil || s == "" {
		return
	}
	key := [2]int{x, y}
	d := debugTexts[key]
	if d == nil {
		d = &debugText{}
		debugTexts[key] = d
	}
	if d.img == nil || d.text != s {
		face := basicfont.Face7x13
		lines := strings.Split(s, "\n")
		w := 0
		for _, line := range lines {
			w = max(w, font.MeasureString(face, line).Ceil())
		}
		img := image.NewNRGBA(image.Rect(0, 0, w+1, len(lines)*16+1))
		dr := font.Drawer{Dst: img, Face: face}
		for i, line := range lines {
			for _, pass := range [2]struct {
				off int
				c   color.NRGBA
			}{{1, color.NRGBA{A: 255}}, {0, color.NRGBA{255, 255, 255, 255}}} {
				dr.Src = image.NewUniform(pass.c)
				dr.Dot = fixed.P(pass.off, i*16+12+pass.off)
				dr.DrawString(line)
			}
		}
		if d.img != nil {
			d.img.Deallocate()
		}
		d.text, d.img = s, ebiten.NewImageFromImage(img)
	}
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(float64(x), float64(y))
	screen.DrawImage(d.img, op)
}
