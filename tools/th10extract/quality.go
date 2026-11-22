// Quality gates apply only to web color data, never to desktop pixels or alpha.
package main

import (
	"fmt"
	"math"

	"th10/anm"
)

// Web quality is measured after conversion back to the original THTX format.
// Alpha is exact; hidden RGB is included to bound bilinear-filter fringes.
const (
	MinVisiblePSNR = 34.0
	MinAllPSNR     = 32.0
	MinEdgePSNR    = 36.0
)

type TextureQuality struct {
	VisiblePSNR float64
	AllPSNR     float64
	EdgePSNR    float64
	EdgeSamples int
}

func (q TextureQuality) Acceptable() bool {
	return q.VisiblePSNR >= MinVisiblePSNR && q.AllPSNR >= MinAllPSNR && q.EdgePSNR >= MinEdgePSNR
}

// CompareTextureQuality checks alpha and RGB, including colors under zero
// alpha. Edge samples simulate a bilinear half-texel sample followed by alpha
// compositing, not an alpha-masked comparison that discards neighboring RGB.
func CompareTextureQuality(a, b *anm.Texture) (TextureQuality, error) {
	var q TextureQuality
	if a.Format != b.Format || a.W != b.W || a.H != b.H || a.W <= 0 || a.H <= 0 || anm.BytesPerPixel(a.Format) == 0 || len(a.Pixels) != len(b.Pixels) || a.W*a.H*anm.BytesPerPixel(a.Format) > len(a.Pixels) {
		return q, fmt.Errorf("anm: quality comparison layout mismatch")
	}
	x, y := a.Image(), b.Image()
	allError, visibleError, edgeError := 0.0, 0.0, 0.0
	visible, edge := 0, 0
	for at := 0; at < len(x.Pix); at += 4 {
		if x.Pix[at+3] != y.Pix[at+3] {
			return q, fmt.Errorf("anm: web texture changed alpha")
		}
		for c := range 3 {
			d := float64(x.Pix[at+c]) - float64(y.Pix[at+c])
			allError += d * d
			if x.Pix[at+3] != 0 {
				visibleError += d * d
				visible++
			}
		}
	}
	for row := 0; row < a.H-1; row++ {
		for col := 0; col < a.W-1; col++ {
			at := x.PixOffset(col, row)
			corners := [4]int{at, at + 4, at + x.Stride, at + x.Stride + 4}
			minAlpha, maxAlpha, alpha := 255, 0, 0
			for _, p := range corners {
				v := int(x.Pix[p+3])
				minAlpha, maxAlpha, alpha = min(minAlpha, v), max(maxAlpha, v), alpha+v
			}
			if minAlpha == maxAlpha {
				continue
			}
			for c := range 3 {
				d := 0.0
				for _, p := range corners {
					d += float64(x.Pix[p+c]) - float64(y.Pix[p+c])
				}
				d *= 0.25 * float64(alpha) / (4 * 255)
				edgeError += d * d
				edge++
			}
		}
	}
	q.VisiblePSNR, q.AllPSNR = pixelPSNR(visibleError, visible), pixelPSNR(allError, len(x.Pix)/4*3)
	q.EdgePSNR, q.EdgeSamples = pixelPSNR(edgeError, edge), edge/3
	return q, nil
}

func pixelPSNR(errorSum float64, samples int) float64 {
	if errorSum == 0 || samples == 0 {
		return math.Inf(1)
	}
	return 10 * math.Log10(255*255*float64(samples)/errorSum)
}
