package main

import (
	"bytes"
	"fmt"
	"testing"

	"th10/anm"
)

func TestLosslessTexturePixelFormats(t *testing.T) {
	for _, format := range []int{anm.FormatBGRA8888, anm.FormatRGB565, anm.FormatARGB4444, anm.FormatGray8} {
		for _, palette := range []bool{false, true} {
			t.Run(fmt.Sprintf("format%d/palette%t", format, palette), func(t *testing.T) {
				texture := &anm.Texture{Format: format, W: 17, H: 13}
				texture.Pixels = make([]byte, texture.W*texture.H*anm.BytesPerPixel(format))
				for i := range texture.Pixels {
					texture.Pixels[i] = byte(i*19 + i/7)
					if palette {
						texture.Pixels[i] %= 4
					}
				}
				if format == anm.FormatBGRA8888 {
					for i := 3; i < len(texture.Pixels); i += 16 {
						texture.Pixels[i] = 0
					}
				}
				blob, codec, err := encodeLosslessTexture(texture)
				if err != nil {
					t.Fatal(err)
				}
				decoded, err := anm.DecodeTexture(blob)
				if err != nil || !bytes.Equal(decoded.Pixels, texture.Pixels) {
					t.Fatalf("%s changed raw pixel bytes: %v", codec, err)
				}
				again, _, err := encodeLosslessTexture(texture)
				if err != nil || !bytes.Equal(blob, again) {
					t.Fatal("lossless encode is nondeterministic")
				}
			})
		}
	}
}
