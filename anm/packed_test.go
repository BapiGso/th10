package anm

import (
	"bytes"
	"image"
	"testing"
)

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
