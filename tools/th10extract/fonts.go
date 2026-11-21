package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"go/format"
	"image"
	"image/color"
	"os"
	"path/filepath"
	"sort"
	"strconv"

	gtfont "github.com/go-text/typesetting/font"
	"golang.org/x/image/font"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/math/fixed"
)

type fontProduct struct {
	Name          string         `json:"name"`
	SourceSHA256  string         `json:"source_sha256"`
	SubsetSHA256  string         `json:"subset_sha256"`
	OriginalBytes int            `json:"original_bytes"`
	SubsetBytes   int            `json:"subset_bytes"`
	Charset       int            `json:"charset"`
	KeptGlyphs    int            `json:"kept_glyphs"`
	Missing       string         `json:"missing"`
	GlyphHashes   map[int]string `json:"glyph_sha256"`
}

type fontPixelCase struct {
	Size  int    `json:"size"`
	Right bool   `json:"right"`
	Color uint32 `json:"color"`
	Hash  string `json:"sha256"`
}

type fontPixelReference struct {
	Name  string          `json:"name"`
	Text  string          `json:"text"`
	Cases []fontPixelCase `json:"cases"`
}

type sfntTables struct {
	version []byte
	tables  map[string][]byte
	glyphs  [][]byte
}

func readFontTables(data []byte) (*sfntTables, error) {
	base := 0
	if len(data) >= 16 && string(data[:4]) == "ttcf" {
		base = int(binary.BigEndian.Uint32(data[12:]))
	}
	if base < 0 || base+12 > len(data) {
		return nil, fmt.Errorf("font: invalid SFNT header")
	}
	f := &sfntTables{version: bytes.Clone(data[base : base+4]), tables: map[string][]byte{}}
	count := int(binary.BigEndian.Uint16(data[base+4:]))
	for i := range count {
		at := base + 12 + i*16
		if at+16 > len(data) {
			return nil, fmt.Errorf("font: invalid table directory")
		}
		off, size := uint64(binary.BigEndian.Uint32(data[at+8:])), uint64(binary.BigEndian.Uint32(data[at+12:]))
		if off+size > uint64(len(data)) {
			return nil, fmt.Errorf("font: invalid table bounds")
		}
		f.tables[string(data[at:at+4])] = bytes.Clone(data[int(off):int(off+size)])
	}
	head, maxp, loca, glyf := f.tables["head"], f.tables["maxp"], f.tables["loca"], f.tables["glyf"]
	if len(head) < 54 || len(maxp) < 6 || len(loca) == 0 || len(glyf) == 0 {
		return nil, fmt.Errorf("font: needs TrueType outlines")
	}
	n := int(binary.BigEndian.Uint16(maxp[4:]))
	long := binary.BigEndian.Uint16(head[50:]) == 1
	stride := 2
	if long {
		stride = 4
	}
	if len(loca) < (n+1)*stride {
		return nil, fmt.Errorf("font: truncated loca")
	}
	offset := func(i int) uint64 {
		if long {
			return uint64(binary.BigEndian.Uint32(loca[i*4:]))
		}
		return uint64(binary.BigEndian.Uint16(loca[i*2:])) * 2
	}
	for i := range n {
		start, end := offset(i), offset(i+1)
		if start > end || end > uint64(len(glyf)) {
			return nil, fmt.Errorf("font: invalid glyph bounds")
		}
		f.glyphs = append(f.glyphs, glyf[int(start):int(end)])
	}
	return f, nil
}

type glyphRule struct{ inputs, outputs []int }

func coverageGlyphs(data []byte, off int) ([]int, error) {
	if off < 0 || off+4 > len(data) {
		return nil, fmt.Errorf("font: invalid GSUB coverage")
	}
	kind, count := int(binary.BigEndian.Uint16(data[off:])), int(binary.BigEndian.Uint16(data[off+2:]))
	var out []int
	switch kind {
	case 1:
		if off+4+count*2 > len(data) {
			return nil, fmt.Errorf("font: short GSUB coverage")
		}
		for i := range count {
			out = append(out, int(binary.BigEndian.Uint16(data[off+4+i*2:])))
		}
	case 2:
		if off+4+count*6 > len(data) {
			return nil, fmt.Errorf("font: short GSUB range")
		}
		for i := range count {
			at := off + 4 + i*6
			start, end := int(binary.BigEndian.Uint16(data[at:])), int(binary.BigEndian.Uint16(data[at+2:]))
			if start > end {
				return nil, fmt.Errorf("font: invalid GSUB range")
			}
			for g := start; g <= end; g++ {
				out = append(out, g)
			}
		}
	default:
		return nil, fmt.Errorf("font: unsupported GSUB coverage %d", kind)
	}
	return out, nil
}

func substitutionRules(data []byte) ([]glyphRule, error) {
	if len(data) == 0 {
		return nil, nil
	}
	if len(data) < 10 {
		return nil, fmt.Errorf("font: short GSUB header")
	}
	list := int(binary.BigEndian.Uint16(data[8:]))
	if list+2 > len(data) {
		return nil, fmt.Errorf("font: invalid GSUB lookup list")
	}
	count := int(binary.BigEndian.Uint16(data[list:]))
	if list+2+count*2 > len(data) {
		return nil, fmt.Errorf("font: short GSUB lookup list")
	}
	var rules []glyphRule
	var parse func(int, int, int) error
	parse = func(kind, base, depth int) error {
		if base < 0 || base+6 > len(data) || depth > 2 {
			return fmt.Errorf("font: invalid GSUB subtable")
		}
		if kind == 7 {
			if base+8 > len(data) {
				return fmt.Errorf("font: short GSUB extension")
			}
			return parse(int(binary.BigEndian.Uint16(data[base+2:])), base+int(binary.BigEndian.Uint32(data[base+4:])), depth+1)
		}
		// Contextual lookups only call other lookups. Their output glyphs are
		// covered by processing every substitution-producing lookup below.
		if kind == 5 || kind == 6 {
			return nil
		}
		coverage, err := coverageGlyphs(data, base+int(binary.BigEndian.Uint16(data[base+2:])))
		if err != nil {
			return err
		}
		format := int(binary.BigEndian.Uint16(data[base:]))
		if kind == 1 && format == 1 {
			delta := int(int16(binary.BigEndian.Uint16(data[base+4:])))
			for _, g := range coverage {
				rules = append(rules, glyphRule{[]int{g}, []int{(g + delta) & 0xffff}})
			}
			return nil
		}
		at := base + 4
		if kind == 8 {
			for range 2 {
				if at+2 > len(data) {
					return fmt.Errorf("font: short reverse GSUB")
				}
				n := int(binary.BigEndian.Uint16(data[at:]))
				at += 2 + n*2
			}
		}
		if at+2 > len(data) {
			return fmt.Errorf("font: short GSUB substitution count")
		}
		n := int(binary.BigEndian.Uint16(data[at:]))
		at += 2
		if n != len(coverage) || at+n*2 > len(data) {
			return fmt.Errorf("font: GSUB coverage/count mismatch")
		}
		for i, input := range coverage {
			value := int(binary.BigEndian.Uint16(data[at+i*2:]))
			if kind == 1 || kind == 8 {
				rules = append(rules, glyphRule{[]int{input}, []int{value}})
				continue
			}
			if kind != 2 && kind != 3 && kind != 4 {
				return fmt.Errorf("font: unsupported GSUB lookup %d", kind)
			}
			sequence := base + value
			if sequence+2 > len(data) {
				return fmt.Errorf("font: invalid GSUB sequence")
			}
			length := int(binary.BigEndian.Uint16(data[sequence:]))
			if sequence+2+length*2 > len(data) {
				return fmt.Errorf("font: short GSUB sequence")
			}
			var outputs []int
			for j := range length {
				g := int(binary.BigEndian.Uint16(data[sequence+2+j*2:]))
				if kind != 4 {
					outputs = append(outputs, g)
					continue
				}
				ligature := sequence + g
				if ligature+4 > len(data) {
					return fmt.Errorf("font: invalid ligature")
				}
				parts := int(binary.BigEndian.Uint16(data[ligature+2:]))
				if parts < 1 || ligature+4+(parts-1)*2 > len(data) {
					return fmt.Errorf("font: short ligature")
				}
				inputs := []int{input}
				for k := 0; k < parts-1; k++ {
					inputs = append(inputs, int(binary.BigEndian.Uint16(data[ligature+4+k*2:])))
				}
				rules = append(rules, glyphRule{inputs, []int{int(binary.BigEndian.Uint16(data[ligature:]))}})
			}
			if kind != 4 {
				rules = append(rules, glyphRule{[]int{input}, outputs})
			}
		}
		return nil
	}
	for i := range count {
		base := list + int(binary.BigEndian.Uint16(data[list+2+i*2:]))
		if base+6 > len(data) {
			return nil, fmt.Errorf("font: invalid GSUB lookup")
		}
		kind, n := int(binary.BigEndian.Uint16(data[base:])), int(binary.BigEndian.Uint16(data[base+4:]))
		if base+6+n*2 > len(data) {
			return nil, fmt.Errorf("font: short GSUB lookup")
		}
		for j := range n {
			if err := parse(kind, base+int(binary.BigEndian.Uint16(data[base+6+j*2:])), 0); err != nil {
				return nil, err
			}
		}
	}
	return rules, nil
}

func subsetFont(data []byte, text string) ([]byte, fontProduct, error) {
	p := fontProduct{OriginalBytes: len(data), Charset: len([]rune(text)), GlyphHashes: map[int]string{}}
	f, err := readFontTables(data)
	if err != nil {
		return nil, p, err
	}
	faces, err := gtfont.ParseTTC(bytes.NewReader(data))
	if err != nil {
		return nil, p, err
	}
	keep, mapping := map[int]bool{0: true}, map[rune]int{}
	for _, r := range text {
		g, ok := faces[0].NominalGlyph(r)
		if !ok || g == 0 {
			p.Missing += string(r)
			continue
		}
		keep[int(g)], mapping[r] = true, int(g)
	}
	rules, err := substitutionRules(f.tables["GSUB"])
	if err != nil {
		return nil, p, err
	}
	for changed := true; changed; {
		changed = false
		for _, rule := range rules {
			active := true
			for _, g := range rule.inputs {
				active = active && keep[g]
			}
			if active {
				for _, g := range rule.outputs {
					if !keep[g] {
						keep[g], changed = true, true
					}
				}
			}
		}
	}
	queue := make([]int, 0, len(keep))
	for g := range keep {
		queue = append(queue, g)
	}
	for at := 0; at < len(queue); at++ {
		g := queue[at]
		if g < 0 || g >= len(f.glyphs) {
			return nil, p, fmt.Errorf("font: dependency glyph out of range")
		}
		glyph := f.glyphs[g]
		if len(glyph) < 10 || int16(binary.BigEndian.Uint16(glyph)) >= 0 {
			continue
		}
		for pos := 10; ; {
			if pos+4 > len(glyph) {
				return nil, p, fmt.Errorf("font: short composite glyph")
			}
			flags, child := binary.BigEndian.Uint16(glyph[pos:]), int(binary.BigEndian.Uint16(glyph[pos+2:]))
			if !keep[child] {
				keep[child] = true
				queue = append(queue, child)
			}
			pos += 6
			if flags&1 != 0 {
				pos += 2
			}
			switch {
			case flags&8 != 0:
				pos += 2
			case flags&0x40 != 0:
				pos += 4
			case flags&0x80 != 0:
				pos += 8
			}
			if flags&0x20 == 0 {
				break
			}
		}
	}
	loca := make([]byte, (len(f.glyphs)+1)*4)
	var glyf []byte
	for g, glyph := range f.glyphs {
		binary.BigEndian.PutUint32(loca[g*4:], uint32(len(glyf)))
		if keep[g] {
			glyf = append(glyf, glyph...)
			hash := sha256.Sum256(glyph)
			p.GlyphHashes[g] = hex.EncodeToString(hash[:])
		}
	}
	binary.BigEndian.PutUint32(loca[len(f.glyphs)*4:], uint32(len(glyf)))
	f.tables["glyf"], f.tables["loca"], f.tables["cmap"] = glyf, loca, buildSubsetCmap(mapping)
	binary.BigEndian.PutUint16(f.tables["head"][50:], 1)
	delete(f.tables, "DSIG")
	out := buildSFNT(f)
	p.KeptGlyphs, p.SubsetBytes = len(keep), len(out)
	oldHash, newHash := sha256.Sum256(data), sha256.Sum256(out)
	p.SourceSHA256, p.SubsetSHA256 = hex.EncodeToString(oldHash[:]), hex.EncodeToString(newHash[:])
	return out, p, nil
}

func buildSubsetCmap(mapping map[rune]int) []byte {
	runes := make([]rune, 0, len(mapping))
	for r := range mapping {
		runes = append(runes, r)
	}
	sort.Slice(runes, func(i, j int) bool { return runes[i] < runes[j] })
	type group struct{ start, end, glyph uint32 }
	var groups []group
	for _, r := range runes {
		g := uint32(mapping[r])
		if len(groups) > 0 {
			last := &groups[len(groups)-1]
			if uint32(r) == last.end+1 && g == last.glyph+last.end-last.start+1 {
				last.end = uint32(r)
				continue
			}
		}
		groups = append(groups, group{uint32(r), uint32(r), g})
	}
	out := make([]byte, 20+16+len(groups)*12)
	binary.BigEndian.PutUint16(out[2:], 2)
	binary.BigEndian.PutUint16(out[6:], 4) // Unicode full repertoire
	binary.BigEndian.PutUint32(out[8:], 20)
	binary.BigEndian.PutUint16(out[12:], 3)
	binary.BigEndian.PutUint16(out[14:], 10) // Windows UCS-4
	binary.BigEndian.PutUint32(out[16:], 20)
	binary.BigEndian.PutUint16(out[20:], 12)
	binary.BigEndian.PutUint32(out[24:], uint32(len(out)-20))
	binary.BigEndian.PutUint32(out[32:], uint32(len(groups)))
	for i, g := range groups {
		at := 36 + i*12
		binary.BigEndian.PutUint32(out[at:], g.start)
		binary.BigEndian.PutUint32(out[at+4:], g.end)
		binary.BigEndian.PutUint32(out[at+8:], g.glyph)
	}
	return out
}

func tableChecksum(data []byte) uint32 {
	var sum uint32
	for at := 0; at < len(data); at += 4 {
		var word [4]byte
		copy(word[:], data[at:min(at+4, len(data))])
		sum += binary.BigEndian.Uint32(word[:])
	}
	return sum
}

func buildSFNT(f *sfntTables) []byte {
	tags := make([]string, 0, len(f.tables))
	for tag := range f.tables {
		tags = append(tags, tag)
	}
	sort.Strings(tags)
	binary.BigEndian.PutUint32(f.tables["head"][8:], 0)
	out := make([]byte, 12+len(tags)*16)
	copy(out, f.version)
	binary.BigEndian.PutUint16(out[4:], uint16(len(tags)))
	power, selector := 1, 0
	for power*2 <= len(tags) {
		power *= 2
		selector++
	}
	binary.BigEndian.PutUint16(out[6:], uint16(power*16))
	binary.BigEndian.PutUint16(out[8:], uint16(selector))
	binary.BigEndian.PutUint16(out[10:], uint16(len(tags)*16-power*16))
	headOffset := 0
	for i, tag := range tags {
		data, at := f.tables[tag], 12+i*16
		copy(out[at:], tag)
		binary.BigEndian.PutUint32(out[at+4:], tableChecksum(data))
		binary.BigEndian.PutUint32(out[at+8:], uint32(len(out)))
		binary.BigEndian.PutUint32(out[at+12:], uint32(len(data)))
		if tag == "head" {
			headOffset = len(out)
		}
		out = append(out, data...)
		for len(out)%4 != 0 {
			out = append(out, 0)
		}
	}
	binary.BigEndian.PutUint32(out[headOffset+8:], 0xb1b0afba-tableChecksum(out))
	return out
}

func subsetFonts(root, source string) error {
	if source == "" {
		return fmt.Errorf("-fonts requires -font-source DIR with the original fonts")
	}
	characters, err := collectFontCharacters(root)
	if err != nil {
		return err
	}
	var products []fontProduct
	var references []fontPixelReference
	for _, name := range []string{"Bimini Bold.ttf", "DFYuGaSo-W3.ttc", "DFYuGaSo-W7.ttc"} {
		raw, err := os.ReadFile(filepath.Join(source, name))
		if err != nil {
			return err
		}
		text := characters.Text
		if name == "Bimini Bold.ttf" {
			text = characters.Sources["ASCII"]
		}
		data, product, err := subsetFont(raw, text)
		if err != nil {
			return fmt.Errorf("%s: %w", name, err)
		}
		product.Name = name
		if name != "Bimini Bold.ttf" {
			reference, err := originalFontPixels(raw, name, text)
			if err != nil {
				return err
			}
			references = append(references, reference)
		}
		if err := write(filepath.Join(root, "font", name), data); err != nil {
			return err
		}
		products = append(products, product)
	}
	project := filepath.Dir(filepath.Clean(root))
	data, err := json.MarshalIndent(products, "", "  ")
	if err != nil {
		return err
	}
	if err := write(filepath.Join(project, "docs", "research", "font_subset.json"), append(data, '\n')); err != nil {
		return err
	}
	data, err = json.Marshal(references)
	if err != nil {
		return err
	}
	fixture, err := format.Source([]byte("// Code generated by th10extract -fonts from the original fonts; DO NOT EDIT.\npackage render\n\nconst fontPixelReferenceJSON = " + strconv.QuoteToASCII(string(data)) + "\n"))
	if err != nil {
		return err
	}
	return write(filepath.Join(project, "render", "font_pixels_data_test.go"), fixture)
}

func originalFontPixels(data []byte, name, text string) (fontPixelReference, error) {
	ref := fontPixelReference{Name: name, Text: text}
	collection, err := opentype.ParseCollection(data)
	if err != nil {
		return ref, err
	}
	f, err := collection.Font(0)
	if err != nil {
		return ref, err
	}
	for _, size := range []int{16, 17, 20, 28} {
		face, err := opentype.NewFace(f, &opentype.FaceOptions{Size: float64(size), DPI: 72, Hinting: font.HintingFull})
		if err != nil {
			return ref, err
		}
		for _, right := range []bool{false, true} {
			for _, ink := range []uint32{0xffffff, 0xf8f08f, 0x88ccff} {
				c := fontPixelCase{Size: size, Right: right, Color: ink}
				h := sha256.New()
				runes := []rune(text)
				for start := 0; start < len(runes); start += 24 {
					img := referenceRaster(string(runes[start:min(start+24, len(runes))]), face, c)
					_, _ = h.Write(img.Pix)
				}
				c.Hash = hex.EncodeToString(h.Sum(nil))
				ref.Cases = append(ref.Cases, c)
			}
		}
		_ = face.Close()
	}
	return ref, nil
}

// This is the pre-subset RasterText algorithm, using an explicit original
// face rather than an embedded font or the runtime's font cache.
func referenceRaster(text string, face font.Face, c fontPixelCase) *image.NRGBA {
	img := image.NewNRGBA(image.Rect(0, 0, 1024, 64))
	x, width := 0, 0
	if c.Right {
		for _, r := range text {
			width++
			if r >= 0x80 {
				width++
			}
		}
		x = 1024 - width*(c.Size-1)/2
	}
	y := face.Metrics().Ascent.Ceil()
	d := font.Drawer{Dst: img, Face: face, Src: image.NewUniform(color.NRGBA{A: 255}), Dot: fixed.P(x+1, y+1)}
	d.DrawString(text)
	d.Src = image.NewUniform(color.NRGBA{R: byte(c.Color >> 16), G: byte(c.Color >> 8), B: byte(c.Color), A: 255})
	d.Dot = fixed.P(x, y)
	d.DrawString(text)
	return img
}
