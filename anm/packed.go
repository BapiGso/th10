// Packed resources preserve raw ANM offsets and use standard-library codecs.
package anm

import (
	"bytes"
	"compress/flate"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"io"
	"sort"
)

const (
	textureHeaderSize = 16
	maxPackedBytes    = 128 << 20
)

// PixelRange locates one complete THTX pixel block in the original ANM.
// Every byte outside these ranges remains verbatim in the packed container.
type PixelRange struct {
	Offset  int
	Texture *Texture
}

// PixelRanges validates a raw ANM and locates its pixel blocks in file order.
func PixelRanges(data []byte) ([]PixelRange, error) {
	if _, err := Load(data); err != nil {
		return nil, err
	}
	var ranges []PixelRange
	for base := 0; ; {
		thtx := int(binary.LittleEndian.Uint32(data[base+0x30:]))
		hasData := binary.LittleEndian.Uint16(data[base+0x34:]) != 0
		if hasData && thtx != 0 {
			t, err := parseTHTX(data, base+thtx)
			if err != nil {
				return nil, err
			}
			ranges = append(ranges, PixelRange{base + thtx + 16, t})
		}
		next := int(binary.LittleEndian.Uint32(data[base+0x38:]))
		if next == 0 {
			break
		}
		if next <= 0 || base+next >= len(data) {
			return nil, fmt.Errorf("anm: invalid entry chain")
		}
		base += next
	}
	return ranges, nil
}

// TextureID includes format and dimensions as well as every original pixel
// byte, including RGB under alpha zero. A web variant retains this source ID.
func TextureID(t *Texture) string {
	h := sha256.New()
	_, _ = fmt.Fprintf(h, "%d/%d/%d/", t.Format, t.W, t.H)
	_, _ = h.Write(t.Pixels)
	return hex.EncodeToString(h.Sum(nil))
}

// Split removes only THTX pixels. ANP1 stores the original file length, then
// {offset, length, SHA-256} records and concatenated non-pixel byte ranges.
// It does not rebase or rewrite ANM headers, instructions or sprite tables.
func Split(data []byte) ([]byte, map[string]*Texture, error) {
	ranges, err := PixelRanges(data)
	if err != nil {
		return nil, nil, err
	}
	if len(data) > maxPackedBytes {
		return nil, nil, fmt.Errorf("anm: file too large")
	}
	textures := map[string]*Texture{}
	metaSize, previous := len(data), 0
	for _, r := range ranges {
		if r.Offset < previous || r.Offset+len(r.Texture.Pixels) > len(data) {
			return nil, nil, fmt.Errorf("anm: overlapping pixel ranges")
		}
		metaSize -= len(r.Texture.Pixels)
		previous = r.Offset + len(r.Texture.Pixels)
	}
	out := make([]byte, 12+len(ranges)*40, 12+len(ranges)*40+metaSize)
	copy(out, "ANP1")
	binary.LittleEndian.PutUint32(out[4:], uint32(len(data)))
	binary.LittleEndian.PutUint32(out[8:], uint32(len(ranges)))
	previous = 0
	for i, r := range ranges {
		id := TextureID(r.Texture)
		hash, err := hex.DecodeString(id)
		if err != nil {
			return nil, nil, err
		}
		at := 12 + i*40
		binary.LittleEndian.PutUint32(out[at:], uint32(r.Offset))
		binary.LittleEndian.PutUint32(out[at+4:], uint32(len(r.Texture.Pixels)))
		copy(out[at+8:], hash)
		out = append(out, data[previous:r.Offset]...)
		previous = r.Offset + len(r.Texture.Pixels)
		textures[id] = r.Texture
	}
	out = append(out, data[previous:]...)
	return out, textures, nil
}

// IsPacked reports whether data is a split ANM container, not a raw ANM.
func IsPacked(data []byte) bool { return len(data) >= 4 && string(data[:4]) == "ANP1" }

// Restore reassembles a split ANM using the selected build's texture blobs.
// Its result is byte-identical to the source on desktop. The js build only
// changes bytes inside THTX pixel ranges, never the format or dimensions.
func Restore(data []byte, readTexture func(string) ([]byte, error)) ([]byte, error) {
	if len(data) < 12 || !IsPacked(data) {
		return nil, fmt.Errorf("anm: invalid packed header")
	}
	size, count := uint64(binary.LittleEndian.Uint32(data[4:])), uint64(binary.LittleEndian.Uint32(data[8:]))
	if size < headerSize || size > maxPackedBytes || count > uint64((len(data)-12)/40) {
		return nil, fmt.Errorf("anm: invalid packed size")
	}
	meta, previous, pixelBytes := 12+int(count)*40, 0, uint64(0)
	for i := 0; i < int(count); i++ {
		at := 12 + i*40
		off, n := uint64(binary.LittleEndian.Uint32(data[at:])), uint64(binary.LittleEndian.Uint32(data[at+4:]))
		if off < uint64(previous) || off < 16 || n == 0 || off+n > size {
			return nil, fmt.Errorf("anm: invalid packed range")
		}
		pixelBytes += n
		previous = int(off + n)
	}
	if uint64(len(data)-meta)+pixelBytes != size {
		return nil, fmt.Errorf("anm: packed metadata length mismatch")
	}
	previous = 0
	out := make([]byte, int(size))
	// Copy metadata before decoding, so the original THTX headers can check
	// the blobs. Bounds checks also reject overlaps and extra container bytes.
	for i := 0; i < int(count); i++ {
		at := 12 + i*40
		off, n := uint64(binary.LittleEndian.Uint32(data[at:])), uint64(binary.LittleEndian.Uint32(data[at+4:]))
		if off < uint64(previous) || off < 16 || n == 0 || off+n > size {
			return nil, fmt.Errorf("anm: invalid packed range")
		}
		length := int(off) - previous
		if meta+length > len(data) {
			return nil, errTooShort
		}
		copy(out[previous:int(off)], data[meta:meta+length])
		meta, previous = meta+length, int(off+n)
	}
	if len(data)-meta != len(out)-previous {
		return nil, fmt.Errorf("anm: packed metadata length mismatch")
	}
	copy(out[previous:], data[meta:])
	for i := 0; i < int(count); i++ {
		at := 12 + i*40
		off, n := int(binary.LittleEndian.Uint32(data[at:])), int(binary.LittleEndian.Uint32(data[at+4:]))
		original, err := parseTHTX(out, off-16)
		if err != nil || len(original.Pixels) != n {
			return nil, fmt.Errorf("anm: packed range is not a THTX pixel block")
		}
		id := hex.EncodeToString(data[at+8 : at+40])
		blob, err := readTexture(id)
		if err != nil {
			return nil, fmt.Errorf("anm texture %s: %w", id, err)
		}
		texture, err := DecodeTexture(blob)
		if err != nil {
			return nil, fmt.Errorf("anm texture %s: %w", id, err)
		}
		if texture.Format != original.Format || texture.W != original.W || texture.H != original.H || len(texture.Pixels) != n {
			return nil, fmt.Errorf("anm: packed texture layout mismatch")
		}
		copy(out[off:off+n], texture.Pixels)
	}
	return out, nil
}

// DecodeTexture reads TXP1: codec, original THTX format, dimensions, raw
// length, then deflate, filtered deflate, PNG, or JPEG plus lossless alpha.
// All runtime codecs use the standard library and have bounded output.
func DecodeTexture(data []byte) (*Texture, error) {
	if len(data) < textureHeaderSize || string(data[:4]) != "TXP1" {
		return nil, fmt.Errorf("anm: invalid texture pack header")
	}
	t := &Texture{
		Format: int(data[5]), W: int(binary.LittleEndian.Uint16(data[6:])), H: int(binary.LittleEndian.Uint16(data[8:])),
	}
	size := uint64(binary.LittleEndian.Uint32(data[12:]))
	bpp := bytesPerPixel(t.Format)
	expectedSize := uint64(t.W) * uint64(t.H) * uint64(bpp)
	if bpp == 0 || t.W == 0 || t.H == 0 || expectedSize == 0 || size < expectedSize || size > maxPackedBytes {
		return nil, fmt.Errorf("anm: invalid texture pack layout")
	}
	expected := int(expectedSize)
	payload := data[textureHeaderSize:]
	var err error
	switch data[4] {
	case 0:
		t.Pixels, err = inflateExact(payload, int(size))
	case 1:
		if int(size) != expected {
			return nil, fmt.Errorf("anm: filtered texture has trailing pixels")
		}
		var filtered []byte
		filtered, err = inflateExact(payload, expected+t.H)
		if err == nil {
			t.Pixels, err = unfilterTexture(filtered, t.W*bpp, t.H, bpp)
		}
	case 2:
		if int(size) != expected {
			return nil, fmt.Errorf("anm: PNG texture has trailing pixels")
		}
		var cfg image.Config
		cfg, err = png.DecodeConfig(bytes.NewReader(payload))
		if err == nil && (cfg.Width != t.W || cfg.Height != t.H) {
			err = fmt.Errorf("anm: PNG dimensions mismatch")
		}
		if err == nil {
			var img image.Image
			img, err = png.Decode(bytes.NewReader(payload))
			if err == nil {
				t.Pixels = ImagePixels(img, t.Format)
			}
		}
	case 3:
		t.Pixels, err = decodeJPEGAlpha(payload, t)
	case 4:
		t.Pixels, err = decodeRGBJPEGAlpha(payload, t)
	default:
		err = fmt.Errorf("anm: unknown texture pack codec %d", data[4])
	}
	if err != nil {
		return nil, err
	}
	if len(t.Pixels) != int(size) {
		return nil, fmt.Errorf("anm: decoded texture length mismatch")
	}
	return t, nil
}

func inflateExact(data []byte, size int) ([]byte, error) {
	r := flate.NewReader(bytes.NewReader(data))
	defer func() { _ = r.Close() }()
	out, err := io.ReadAll(io.LimitReader(r, int64(size)+1))
	if err != nil {
		return nil, err
	}
	if len(out) != size {
		return nil, fmt.Errorf("anm: inflated length %d, want %d", len(out), size)
	}
	return out, nil
}

func paeth(a, b, c byte) byte {
	p := int(a) + int(b) - int(c)
	pa, pb, pc := abs(p-int(a)), abs(p-int(b)), abs(p-int(c))
	if pa <= pb && pa <= pc {
		return a
	}
	if pb <= pc {
		return b
	}
	return c
}

func abs(n int) int {
	if n < 0 {
		return -n
	}
	return n
}

func unfilterTexture(filtered []byte, stride, height, bpp int) ([]byte, error) {
	out := make([]byte, stride*height)
	for y := range height {
		kind := filtered[y*(stride+1)]
		if kind > 4 {
			return nil, fmt.Errorf("anm: invalid pixel filter %d", kind)
		}
		row := out[y*stride : (y+1)*stride]
		copy(row, filtered[y*(stride+1)+1:(y+1)*(stride+1)])
		for x := range stride {
			var left, up, upperLeft byte
			if x >= bpp {
				left = row[x-bpp]
			}
			if y > 0 {
				up = out[(y-1)*stride+x]
				if x >= bpp {
					upperLeft = out[(y-1)*stride+x-bpp]
				}
			}
			switch kind {
			case 1:
				row[x] += left
			case 2:
				row[x] += up
			case 3:
				row[x] += byte((int(left) + int(up)) / 2)
			case 4:
				row[x] += paeth(left, up, upperLeft)
			}
		}
	}
	return out, nil
}

// ImagePixels reverses Texture.Image's channel packing. Its callers only
// pass an image with the original dimensions. Narrow formats use the high
// bits, reversing D3D's bit replication exactly for lossless images.
func ImagePixels(img image.Image, format int) []byte {
	return imagePixels(img, format, false)
}

func imagePixels(img image.Image, format int, nearest bool) []byte {
	b := img.Bounds()
	bpp := bytesPerPixel(format)
	out := make([]byte, b.Dx()*b.Dy()*bpp)
	nrgba, direct := img.(*image.NRGBA)
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			var c color.NRGBA
			if direct {
				i := nrgba.PixOffset(x, y)
				c = color.NRGBA{R: nrgba.Pix[i], G: nrgba.Pix[i+1], B: nrgba.Pix[i+2], A: nrgba.Pix[i+3]}
			} else {
				c = color.NRGBAModel.Convert(img.At(x, y)).(color.NRGBA)
			}
			at := ((y-b.Min.Y)*b.Dx() + x - b.Min.X) * bpp
			switch format {
			case FormatBGRA8888:
				out[at], out[at+1], out[at+2], out[at+3] = c.B, c.G, c.R, c.A
			case FormatRGB565:
				r, g, b := c.R>>3, c.G>>2, c.B>>3
				if nearest {
					// JPEG colors use the nearest representable value, not
					// truncation at an expanded narrow-channel boundary.
					r = byte((int(c.R)*31 + 127) / 255)
					g = byte((int(c.G)*63 + 127) / 255)
					b = byte((int(c.B)*31 + 127) / 255)
				}
				v := uint16(r)<<11 | uint16(g)<<5 | uint16(b)
				binary.LittleEndian.PutUint16(out[at:], v)
			case FormatARGB4444:
				r, g, b := c.R>>4, c.G>>4, c.B>>4
				if nearest {
					r, g, b = byte((int(c.R)+8)/17), byte((int(c.G)+8)/17), byte((int(c.B)+8)/17)
				}
				out[at], out[at+1] = g<<4|b, c.A&0xf0|r
			case FormatGray8:
				out[at] = c.R
			}
		}
	}
	return out
}

func decodeJPEGAlpha(payload []byte, t *Texture) ([]byte, error) {
	if len(payload) < 5 {
		return nil, errTooShort
	}
	n := uint64(binary.LittleEndian.Uint32(payload))
	if n > uint64(len(payload)-5) {
		return nil, errTooShort
	}
	jpegData := payload[5 : 5+int(n)]
	cfg, err := jpeg.DecodeConfig(bytes.NewReader(jpegData))
	if err != nil {
		return nil, err
	}
	if cfg.Width != t.W || cfg.Height != t.H {
		return nil, fmt.Errorf("anm: JPEG dimensions mismatch")
	}
	img, err := jpeg.Decode(bytes.NewReader(jpegData))
	if err != nil {
		return nil, err
	}
	var alpha []byte
	switch payload[4] {
	case 0:
		if len(payload) != 5+int(n) {
			return nil, fmt.Errorf("anm: unexpected opaque JPEG data")
		}
	case 1:
		alpha, err = inflateExact(payload[5+int(n):], t.W*t.H)
		if err != nil {
			return nil, err
		}
	default:
		return nil, fmt.Errorf("anm: invalid JPEG alpha codec")
	}
	nrgba := image.NewNRGBA(image.Rect(0, 0, t.W, t.H))
	for y := range t.H {
		for x := range t.W {
			c := color.NRGBAModel.Convert(img.At(x, y)).(color.NRGBA)
			at := nrgba.PixOffset(x, y)
			nrgba.Pix[at], nrgba.Pix[at+1], nrgba.Pix[at+2], nrgba.Pix[at+3] = c.R, c.G, c.B, 255
			if alpha != nil {
				nrgba.Pix[at+3] = alpha[y*t.W+x]
			}
		}
	}
	return imagePixels(nrgba, t.Format, true), nil
}

// Separate grayscale JPEGs retain full-resolution R/G/B. Standard color
// JPEG's 4:2:0 chroma sampling fails the gate on many sharp painted textures.
func decodeRGBJPEGAlpha(payload []byte, t *Texture) ([]byte, error) {
	if len(payload) < 13 {
		return nil, errTooShort
	}
	out := image.NewNRGBA(image.Rect(0, 0, t.W, t.H))
	at := uint64(13)
	for channel := range 3 {
		n := uint64(binary.LittleEndian.Uint32(payload[channel*4:]))
		if at+n > uint64(len(payload)) {
			return nil, errTooShort
		}
		data := payload[int(at):int(at+n)]
		cfg, err := jpeg.DecodeConfig(bytes.NewReader(data))
		if err != nil {
			return nil, err
		}
		if cfg.Width != t.W || cfg.Height != t.H {
			return nil, fmt.Errorf("anm: RGB JPEG dimensions mismatch")
		}
		img, err := jpeg.Decode(bytes.NewReader(data))
		if err != nil {
			return nil, err
		}
		gray, ok := img.(*image.Gray)
		if !ok {
			return nil, fmt.Errorf("anm: RGB JPEG channel is not grayscale")
		}
		for y := range t.H {
			for x := range t.W {
				out.Pix[out.PixOffset(x, y)+channel] = gray.Pix[gray.PixOffset(x, y)]
			}
		}
		at += n
	}
	var alpha []byte
	switch payload[12] {
	case 0:
		if int(at) != len(payload) {
			return nil, fmt.Errorf("anm: unexpected RGB JPEG data")
		}
	case 1:
		var err error
		alpha, err = inflateExact(payload[int(at):], t.W*t.H)
		if err != nil {
			return nil, err
		}
	default:
		return nil, fmt.Errorf("anm: invalid RGB JPEG alpha codec")
	}
	for i := 3; i < len(out.Pix); i += 4 {
		out.Pix[i] = 255
		if alpha != nil {
			out.Pix[i] = alpha[i/4]
		}
	}
	return imagePixels(out, t.Format, true), nil
}

// BuildTexturePack stores blobs once, sorted by their source SHA-256. TPK1's
// index holds {SHA-256, absolute offset, length}; lookup does not decode any
// texture or build a startup map. It is also deterministic after regeneration.
func BuildTexturePack(blobs map[string][]byte) ([]byte, error) {
	ids := make([]string, 0, len(blobs))
	total := 8 + len(blobs)*40
	for id, blob := range blobs {
		ids = append(ids, id)
		total += len(blob)
	}
	if total > maxPackedBytes {
		return nil, fmt.Errorf("anm: texture pack too large")
	}
	sort.Strings(ids)
	out := make([]byte, 8+len(ids)*40, total)
	copy(out, "TPK1")
	binary.LittleEndian.PutUint32(out[4:], uint32(len(ids)))
	for i, id := range ids {
		hash, err := hex.DecodeString(id)
		if err != nil || len(hash) != 32 {
			return nil, fmt.Errorf("anm: invalid texture ID %q", id)
		}
		at, blob := 8+i*40, blobs[id]
		copy(out[at:], hash)
		binary.LittleEndian.PutUint32(out[at+32:], uint32(len(out)))
		binary.LittleEndian.PutUint32(out[at+36:], uint32(len(blob)))
		out = append(out, blob...)
	}
	return out, nil
}

// TextureBlob finds an immutable blob without copying the containing pack.
func TextureBlob(data []byte, id string) ([]byte, error) {
	hash, err := hex.DecodeString(id)
	if err != nil || len(hash) != 32 || len(data) < 8 || string(data[:4]) != "TPK1" {
		return nil, fmt.Errorf("anm: invalid texture pack or ID")
	}
	count := uint64(binary.LittleEndian.Uint32(data[4:]))
	if count > uint64((len(data)-8)/40) {
		return nil, fmt.Errorf("anm: truncated texture pack index")
	}
	i := sort.Search(int(count), func(i int) bool {
		return bytes.Compare(data[8+i*40:8+i*40+32], hash) >= 0
	})
	if i == int(count) || !bytes.Equal(data[8+i*40:8+i*40+32], hash) {
		return nil, fmt.Errorf("anm: missing texture %s", id)
	}
	at := 8 + i*40
	off, n := uint64(binary.LittleEndian.Uint32(data[at+32:])), uint64(binary.LittleEndian.Uint32(data[at+36:]))
	if off < 8+count*40 || n < textureHeaderSize || off+n > uint64(len(data)) {
		return nil, fmt.Errorf("anm: invalid texture pack offset")
	}
	return data[int(off):int(off+n)], nil
}
