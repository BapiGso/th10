package main

import (
	"bytes"
	"fmt"
	"image"
	"image/draw"
	"image/jpeg"
	"image/png"
	"io"
	"path/filepath"

	"th10/anm"
	"th10/gamedata"
)

// bootImages are the loading screen game.html draws while the WASM is still
// downloading, cut from the game's own ANMs at build time: sig.anm scripts
// 0/1 (sprites 0 and 1 side by side, 512+128 by 480) and ascii.anm's loading
// sheet, whose sprites 232-234 ("少女祈祷中", "NowLoading...", the leaf)
// loading.mjs draws as ascii.anm scripts 3-6 do.
var bootImages = []string{"sig.jpg", "loading.png"}

func buildBootImages(out string) error {
	sig, err := gamedata.ANM("sig.anm")
	if err != nil {
		return err
	}
	screen := image.NewNRGBA(image.Rect(0, 0, 640, 480))
	for id, x := range []int{0, 512} { // sig scripts 0 and 1: op 48 x = 0 / 512, top-left anchor
		if err := blit(screen, sig, id, x); err != nil {
			return err
		}
	}
	var sigJPEG bytes.Buffer
	if err := jpeg.Encode(&sigJPEG, screen, &jpeg.Options{Quality: 88}); err != nil {
		return err
	}
	ascii, err := gamedata.ANM("ascii.anm")
	if err != nil {
		return err
	}
	sheet, err := spriteSheet(ascii, 232, 233, 234)
	if err != nil {
		return err
	}
	var sheetPNG bytes.Buffer
	if err := (&png.Encoder{CompressionLevel: png.BestCompression}).Encode(&sheetPNG, sheet); err != nil {
		return err
	}
	for name, data := range map[string][]byte{"sig.jpg": sigJPEG.Bytes(), "loading.png": sheetPNG.Bytes()} {
		if err := atomicFile(filepath.Join(out, name), func(w io.Writer) error { _, err := w.Write(data); return err }); err != nil {
			return err
		}
	}
	return nil
}

func entryImage(f *anm.File, sprite int) (*image.NRGBA, anm.Sprite, error) {
	s, ok := f.Sprites[sprite]
	if !ok || s.Entry < 0 || s.Entry >= len(f.Entries) || f.Entries[s.Entry].Texture == nil {
		return nil, s, fmt.Errorf("sprite %d has no texture", sprite)
	}
	return f.Entries[s.Entry].Texture.Image(), s, nil
}

func blit(dst *image.NRGBA, f *anm.File, sprite, x int) error {
	img, s, err := entryImage(f, sprite)
	if err != nil {
		return err
	}
	r := image.Rect(int(s.X), int(s.Y), int(s.X+s.W), int(s.Y+s.H))
	draw.Draw(dst, image.Rect(x, 0, x+r.Dx(), r.Dy()), img, r.Min, draw.Src)
	return nil
}

// spriteSheet returns the texture all the given sprites share; loading.mjs
// keeps their rects (checked by TestBootImages).
func spriteSheet(f *anm.File, sprites ...int) (*image.NRGBA, error) {
	var sheet *image.NRGBA
	entry := -1
	for _, id := range sprites {
		img, s, err := entryImage(f, id)
		if err != nil {
			return nil, err
		}
		if entry >= 0 && s.Entry != entry {
			return nil, fmt.Errorf("sprite %d is on another texture", id)
		}
		sheet, entry = img, s.Entry
	}
	return sheet, nil
}
