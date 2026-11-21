// Transparent neighbors must not escape the web quality gate.
package anm

import (
	"bytes"
	"image"
	"testing"
)

func TestQualityIncludesHiddenRGBAndFilterEdges(t *testing.T) {
	a := &Texture{Format: FormatBGRA8888, W: 2, H: 2, Pixels: make([]byte, 16)}
	a.Pixels[3] = 255
	b := *a
	b.Pixels = bytes.Clone(a.Pixels)
	for at := 4; at < 16; at += 4 {
		b.Pixels[at], b.Pixels[at+1], b.Pixels[at+2] = 255, 255, 255
	}
	q, err := CompareTextureQuality(a, &b)
	if err != nil || q.Acceptable() || q.EdgeSamples != 1 || q.AllPSNR >= MinAllPSNR || q.EdgePSNR >= MinEdgePSNR {
		t.Fatalf("hidden RGB/filter edge error escaped the gate: %+v %v", q, err)
	}
	b.Pixels[3] = 128
	if _, err := CompareTextureQuality(a, &b); err == nil {
		t.Fatal("quality gate accepted changed alpha")
	}
}

func TestLossyPackingUsesNearestNarrowChannel(t *testing.T) {
	img := image.NewNRGBA(image.Rect(0, 0, 1, 1))
	img.Pix[0], img.Pix[3] = 7, 255
	if got := imagePixels(img, FormatRGB565, true); !bytes.Equal(got, []byte{0, 8}) {
		t.Fatalf("near 5-bit channel rounded down: %v", got)
	}
	img.Pix[0] = 14
	if got := imagePixels(img, FormatARGB4444, true); !bytes.Equal(got, []byte{0, 0xf1}) {
		t.Fatalf("near 4-bit channel rounded down: %v", got)
	}
}
