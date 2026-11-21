// Web color encodings are chosen per texture by measured size and error.
package main

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/jpeg"
	"image/png"
	"math"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"th10/anm"

	"golang.org/x/image/font"
	"golang.org/x/image/font/basicfont"
	"golang.org/x/image/math/fixed"
)

func chooseWebTexture(t *anm.Texture, lossless []byte, p *textureProduct) ([]byte, error) {
	if t.W*t.H < 65536 || len(lossless) < 24<<10 || float64(len(lossless))/float64(len(t.Pixels)) < 0.07 || !handPaintedTexture(p.Name) {
		return lossless, nil
	}
	img := t.Image()
	colors := map[uint32]bool{}
	for at := 0; at < len(img.Pix); at += 4 {
		colors[binary.LittleEndian.Uint32(img.Pix[at:])] = true
		if len(colors) > 4096 {
			break
		}
	}
	if len(colors) <= 4096 || smoothGradient(img) {
		return lossless, nil
	}
	best := lossless
	for _, rgb := range []bool{false, true} {
		for _, quality := range []int{55, 65, 75, 85, 92, 97} {
			var blob []byte
			var err error
			kind := "jpeg-alpha"
			if rgb {
				blob, err = encodeRGBJPEGAlpha(t, quality)
				kind = "jpeg-rgb-alpha"
			} else {
				blob, err = encodeJPEGAlpha(t, quality)
			}
			if err != nil {
				return nil, err
			}
			if len(blob)*10 >= len(lossless)*9 || len(blob) >= len(best) {
				break
			}
			decoded, err := anm.DecodeTexture(blob)
			if err != nil {
				return nil, err
			}
			q, err := anm.CompareTextureQuality(t, decoded)
			if err != nil {
				return nil, err
			}
			if !q.Acceptable() {
				continue
			}
			p.Web, p.WebAs, p.JPEGQuality = len(blob), kind, quality
			p.VisiblePSNR, p.AllPSNR, p.EdgePSNR = finitePSNR(q.VisiblePSNR), finitePSNR(q.AllPSNR), finitePSNR(q.EdgePSNR)
			p.EdgeSamples = q.EdgeSamples
			best = blob
			break
		}
	}
	for _, bits := range [][3]int{{4, 4, 4}, {4, 5, 4}, {5, 5, 5}} {
		quantized := quantizeTexture(t, bits)
		q, err := anm.CompareTextureQuality(t, quantized)
		if err != nil {
			return nil, err
		}
		if !q.Acceptable() {
			continue
		}
		blob, codec, err := encodeLosslessTexture(quantized)
		if err != nil {
			return nil, err
		}
		if len(blob)*10 >= len(lossless)*9 || len(blob) >= len(best) {
			continue
		}
		p.Web, p.WebAs, p.JPEGQuality = len(blob), "quantized-"+codec, 0
		p.ColorBits = []int{bits[0], bits[1], bits[2]}
		p.VisiblePSNR, p.AllPSNR, p.EdgePSNR = finitePSNR(q.VisiblePSNR), finitePSNR(q.AllPSNR), finitePSNR(q.EdgePSNR)
		p.EdgeSamples, best = q.EdgeSamples, blob
	}
	return best, nil
}

func quantizeTexture(t *anm.Texture, bits [3]int) *anm.Texture {
	img := t.Image()
	for at := 0; at < len(img.Pix); at += 4 {
		for channel := range 3 {
			levels := (1 << bits[channel]) - 1
			value := (int(img.Pix[at+channel])*levels + 127) / 255
			img.Pix[at+channel] = byte((value*255 + levels/2) / levels)
		}
	}
	return &anm.Texture{Format: t.Format, W: t.W, H: t.H, Pixels: anm.ImagePixels(img, t.Format)}
}

func handPaintedTexture(name string) bool {
	for _, prefix := range []string{"background/", "face/", "ending/", "card/"} {
		if strings.HasPrefix(name, prefix) {
			return true
		}
	}
	// Menu label sheets, numbers and logos keep lossless sharp edges.
	for _, prefix := range []string{"title/title00", "title/select00", "title/sl_pl"} {
		if strings.HasPrefix(name, prefix) {
			return true
		}
	}
	return false
}

func smoothGradient(img *image.NRGBA) bool {
	errorSum, count := 0, 0
	for y := 1; y < img.Rect.Dy()-1; y++ {
		for x := 1; x < img.Rect.Dx()-1; x++ {
			at := img.PixOffset(x, y)
			for c := range 3 {
				errorSum += absInt(int(img.Pix[at-4+c]) - 2*int(img.Pix[at+c]) + int(img.Pix[at+4+c]))
				errorSum += absInt(int(img.Pix[at-img.Stride+c]) - 2*int(img.Pix[at+c]) + int(img.Pix[at+img.Stride+c]))
				count += 2
			}
		}
	}
	return count == 0 || float64(errorSum)/float64(count) < 1.5
}

func encodeJPEGAlpha(t *anm.Texture, quality int) ([]byte, error) {
	img := t.Image()
	rgb := image.NewRGBA(img.Rect)
	alpha, opaque := make([]byte, t.W*t.H), true
	for at := 0; at < len(img.Pix); at += 4 {
		// JPEG receives opaque RGB, so hidden and semi-transparent colors
		// are not premultiplied or discarded by the color-model conversion.
		copy(rgb.Pix[at:at+3], img.Pix[at:at+3])
		rgb.Pix[at+3], alpha[at/4] = 255, img.Pix[at+3]
		opaque = opaque && img.Pix[at+3] == 255
	}
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, rgb, &jpeg.Options{Quality: quality}); err != nil {
		return nil, err
	}
	payload := make([]byte, 5, 5+buf.Len())
	binary.LittleEndian.PutUint32(payload, uint32(buf.Len()))
	payload = append(payload, buf.Bytes()...)
	if !opaque {
		data, err := deflateBytes(alpha)
		if err != nil {
			return nil, err
		}
		payload[4] = 1
		payload = append(payload, data...)
	}
	return textureHeader(t, 3, payload), nil
}

func encodeRGBJPEGAlpha(t *anm.Texture, quality int) ([]byte, error) {
	img := t.Image()
	payload := make([]byte, 13)
	for channel := range 3 {
		plane := image.NewGray(img.Rect)
		for i := range plane.Pix {
			plane.Pix[i] = img.Pix[i*4+channel]
		}
		var buf bytes.Buffer
		if err := jpeg.Encode(&buf, plane, &jpeg.Options{Quality: quality}); err != nil {
			return nil, err
		}
		binary.LittleEndian.PutUint32(payload[channel*4:], uint32(buf.Len()))
		payload = append(payload, buf.Bytes()...)
	}
	alpha, opaque := make([]byte, t.W*t.H), true
	for i := range alpha {
		alpha[i] = img.Pix[i*4+3]
		opaque = opaque && alpha[i] == 255
	}
	if !opaque {
		data, err := deflateBytes(alpha)
		if err != nil {
			return nil, err
		}
		payload[12] = 1
		payload = append(payload, data...)
	}
	return textureHeader(t, 4, payload), nil
}

func finitePSNR(value float64) float64 {
	if math.IsInf(value, 1) || value > 100 {
		return 100
	}
	return value
}

func writeTextureComparisons(root string, products []*textureProduct, originals map[string]*anm.Texture, blobs map[string][]byte) error {
	var worst []*textureProduct
	for _, p := range products {
		if p.JPEGQuality > 0 || len(p.ColorBits) > 0 {
			worst = append(worst, p)
		}
	}
	sort.Slice(worst, func(i, j int) bool {
		if worst[i].VisiblePSNR == worst[j].VisiblePSNR {
			return worst[i].ID < worst[j].ID
		}
		return worst[i].VisiblePSNR < worst[j].VisiblePSNR
	})
	if len(worst) > 20 {
		worst = worst[:20]
	}
	type comparison struct {
		File    string          `json:"file"`
		Texture *textureProduct `json:"texture"`
	}
	var index []comparison
	for i, p := range worst {
		lossless := originals[p.ID].Image()
		decoded, err := anm.DecodeTexture(blobs[p.ID])
		if err != nil {
			return err
		}
		web := decoded.Image()
		w, h, gap := lossless.Rect.Dx(), lossless.Rect.Dy(), 8
		out := image.NewNRGBA(image.Rect(0, 0, w*3+gap*2, h+24))
		for y := range out.Rect.Dy() {
			for x := range out.Rect.Dx() {
				shade := byte(40)
				if y >= 24 {
					shade = 144
					if ((x/12)+(y/12))%2 == 0 {
						shade = 176
					}
				}
				out.SetNRGBA(x, y, color.NRGBA{R: shade, G: shade, B: shade, A: 255})
			}
		}
		draw.Draw(out, image.Rect(0, 24, w, h+24), lossless, image.Point{}, draw.Over)
		draw.Draw(out, image.Rect(w+gap, 24, w*2+gap, h+24), web, image.Point{}, draw.Over)
		for y := range h {
			for x := range w {
				at := lossless.PixOffset(x, y)
				out.SetNRGBA(w*2+gap*2+x, 24+y, color.NRGBA{
					R: byte(min(255, absInt(int(lossless.Pix[at])-int(web.Pix[at]))*8)),
					G: byte(min(255, absInt(int(lossless.Pix[at+1])-int(web.Pix[at+1]))*8)),
					B: byte(min(255, absInt(int(lossless.Pix[at+2])-int(web.Pix[at+2]))*8)), A: 255,
				})
			}
		}
		d := font.Drawer{Dst: out, Src: image.White, Face: basicfont.Face7x13}
		for panel, label := range []string{"LOSSLESS", "WEB", "RGB DIFF x8 (includes hidden RGB)"} {
			d.Dot = fixed.P(panel*(w+gap)+4, 16)
			d.DrawString(label)
		}
		var buf bytes.Buffer
		if err := png.Encode(&buf, out); err != nil {
			return err
		}
		name := fmt.Sprintf("%02d_%s_%s.png", i+1, strings.TrimSuffix(path.Base(p.Name), ".png"), p.ID[:12])
		if err := write(filepath.Join(root, name), buf.Bytes()); err != nil {
			return err
		}
		index = append(index, comparison{name, p})
	}
	data, err := json.MarshalIndent(index, "", "  ")
	if err != nil {
		return err
	}
	return write(filepath.Join(root, "index.json"), append(data, '\n'))
}
