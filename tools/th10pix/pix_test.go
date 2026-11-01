package main

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"math"
	"os"
	"path/filepath"
	"testing"
)

// fill returns a w x h opaque image with pixel (x, y) set to f(x, y).
func fill(w, h int, f func(x, y int) color.RGBA) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.SetRGBA(x, y, f(x, y))
		}
	}
	return img
}

// solid returns a full-size frame with every channel equal to v.
func solid(v uint8) *image.RGBA {
	return fill(frameW, frameH, func(x, y int) color.RGBA {
		return color.RGBA{v, v, v, 0xff}
	})
}

// gradient returns a full-size frame whose pixels differ from those of
// gradient(k) for every k that is not equal.
func gradient(k uint8) *image.RGBA {
	return fill(frameW, frameH, func(x, y int) color.RGBA {
		return color.RGBA{uint8(x) + k, uint8(y) + k, uint8(x^y) + k, 0xff}
	})
}

// clone returns a copy of src.
func clone(src *image.RGBA) *image.RGBA {
	dst := image.NewRGBA(src.Bounds())
	copy(dst.Pix, src.Pix)
	return dst
}

// addIn adds d to the R, G and B of the pixels of r, in place, with uint8 wrap.
func addIn(img *image.RGBA, r rect, d uint8) {
	for y := r.y0; y < r.y1; y++ {
		for x := r.x0; x < r.x1; x++ {
			i := img.PixOffset(x, y)
			for c := 0; c < 3; c++ {
				img.Pix[i+c] += d
			}
		}
	}
}

// writePNG encodes img into dir/name.
func writePNG(t *testing.T, dir, name string, img image.Image) {
	t.Helper()
	f, err := os.Create(filepath.Join(dir, name))
	if err != nil {
		t.Fatal(err)
	}
	if err := png.Encode(f, img); err != nil {
		f.Close()
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestParseNames(t *testing.T) {
	b, a, ok := parseOrigName("f1200-1201.png")
	if !ok || b != 1200 || a != 1201 {
		t.Errorf("parseOrigName(f1200-1201.png) = %d, %d, %v; want 1200, 1201, true", b, a, ok)
	}
	if _, _, ok := parseOrigName("f1200.png"); ok {
		t.Error("parseOrigName(f1200.png) accepted a remake-style name")
	}
	if n, ok := parseFrameName("f1200.png"); !ok || n != 1200 {
		t.Errorf("parseFrameName(f1200.png) = %d, %v; want 1200, true", n, ok)
	}
	if _, ok := parseFrameName("f1200-1201.png"); ok {
		t.Error("parseFrameName(f1200-1201.png) accepted a shot name")
	}
	if n, ok := parseMaskName("m1200.png"); !ok || n != 1200 {
		t.Errorf("parseMaskName(m1200.png) = %d, %v; want 1200, true", n, ok)
	}
	if _, ok := parseMaskName("f1200.png"); ok {
		t.Error("parseMaskName(f1200.png) accepted a frame name")
	}
	for _, name := range []string{
		"notes.txt", "f1200-1201.jpg", "f-1200.png", "f1200-.png", "x1200-1201.png",
		"f1200-1201.png.bak", "F1200.png", "d1200.png", "f99999999999999999999999.png",
	} {
		if _, _, ok := parseOrigName(name); ok {
			t.Errorf("parseOrigName(%q) accepted an ignored name", name)
		}
		if _, ok := parseFrameName(name); ok {
			t.Errorf("parseFrameName(%q) accepted an ignored name", name)
		}
		if _, ok := parseMaskName(name); ok {
			t.Errorf("parseMaskName(%q) accepted an ignored name", name)
		}
	}
	if _, _, ok := parseOrigName("f99999999999999999999999-1.png"); ok {
		t.Error("parseOrigName accepted an overflowing frame number")
	}
}

func TestBestOffsetKnown(t *testing.T) {
	orig := gradient(0)
	cands := []candidate{
		{off: 0, img: gradient(2)},
		{off: -1, img: gradient(1)},
		{off: 1, img: orig},
	}
	best, ok := bestOffset(orig, cands)
	if !ok || best.off != 1 {
		t.Fatalf("bestOffset chose %d (ok %v), want +1", best.off, ok)
	}
	s := regionScore(orig, best.img, regionAll, nil)
	if !s.OK || !math.IsInf(s.DB, 1) {
		t.Errorf("PSNR of the exact match = %+v, want +Inf", s)
	}
	for _, c := range cands[:2] {
		if s := regionScore(orig, c.img, regionAll, nil); math.IsInf(s.DB, 1) {
			t.Errorf("candidate %+d is identical to the original, want a difference", c.off)
		}
	}
	if _, ok := bestOffset(orig, nil); ok {
		t.Error("bestOffset with no candidates reported a match")
	}
}

func TestPSNRKnown(t *testing.T) {
	// Every channel differs by exactly 5, so MSE is 25.
	s := regionScore(solid(100), solid(105), regionAll, nil)
	want := 10 * math.Log10(65025.0/25.0)
	if !s.OK || math.Abs(s.DB-want) > 1e-9 {
		t.Errorf("PSNR = %v, want %v", s.DB, want)
	}
	if math.Abs(want-34.1514) > 1e-4 {
		t.Errorf("reference PSNR %v is not about 34.1514", want)
	}
	if got := psnrFromMSE(0); !math.IsInf(got, 1) {
		t.Errorf("psnrFromMSE(0) = %v, want +Inf", got)
	}
}

func TestBestOffsetTies(t *testing.T) {
	orig := solid(100)
	cases := []struct {
		name  string
		cands []candidate
		want  int
	}{
		{"equal at +1 and -1 picks -1", []candidate{{1, solid(110)}, {-1, solid(110)}}, -1},
		{"equal at -1, 0 and +1 picks 0", []candidate{{-1, solid(110)}, {1, solid(110)}, {0, solid(110)}}, 0},
		{"0 strictly better than equal -1 and +1", []candidate{{-1, solid(110)}, {1, solid(110)}, {0, solid(105)}}, 0},
		{"equal at -2 and +1 picks +1", []candidate{{-2, solid(110)}, {1, solid(110)}}, 1},
		{"lower MSE beats smaller |off|", []candidate{{0, solid(120)}, {2, solid(101)}}, 2},
	}
	for _, c := range cases {
		best, ok := bestOffset(orig, c.cands)
		if !ok || best.off != c.want {
			t.Errorf("%s: chose %d (ok %v), want %d", c.name, best.off, ok, c.want)
		}
	}
}

func TestRegionSplit(t *testing.T) {
	orig := gradient(0)

	// (100,100) is inside the field, so side must stay identical.
	inField := clone(orig)
	addIn(inField, rect{100, 100, 101, 101}, 5)
	if s := regionScore(orig, inField, regionSide, nil); !s.OK || !math.IsInf(s.DB, 1) {
		t.Errorf("side PSNR with a field-only difference = %+v, want +Inf", s)
	}
	if s := regionScore(orig, inField, regionField, nil); !s.OK || math.IsInf(s.DB, 1) {
		t.Errorf("field PSNR with a field-only difference = %+v, want finite", s)
	}

	// (500,100) is in the side panel, so field must stay identical.
	inSide := clone(orig)
	addIn(inSide, rect{500, 100, 501, 101}, 5)
	if s := regionScore(orig, inSide, regionField, nil); !s.OK || !math.IsInf(s.DB, 1) {
		t.Errorf("field PSNR with a side-only difference = %+v, want +Inf", s)
	}
	if s := regionScore(orig, inSide, regionSide, nil); !s.OK || math.IsInf(s.DB, 1) {
		t.Errorf("side PSNR with a side-only difference = %+v, want finite", s)
	}
}

func TestMaskBackground(t *testing.T) {
	orig := gradient(0)
	fg := rect{200, 200, 300, 300}
	sim := clone(orig)
	addIn(sim, fg, 4)

	mask := image.NewNRGBA(image.Rect(0, 0, frameW, frameH))
	for y := fg.y0; y < fg.y1; y++ {
		for x := fg.x0; x < fg.x1; x++ {
			mask.SetNRGBA(x, y, color.NRGBA{0, 0, 0, 0xff})
		}
	}
	keepBG := func(x, y int) bool {
		_, _, _, a := pixelAt(mask, x, y)
		return a == 0
	}

	if s := regionScore(orig, sim, regionAll, keepBG); !s.OK || !math.IsInf(s.DB, 1) {
		t.Errorf("background PSNR = %+v, want +Inf", s)
	}
	if s := regionScore(orig, sim, regionAll, nil); !s.OK || math.IsInf(s.DB, 1) {
		t.Errorf("whole-image PSNR = %+v, want finite", s)
	}

	// A mask that is foreground everywhere leaves no background to score.
	allFG := image.NewNRGBA(image.Rect(0, 0, frameW, frameH))
	for i := 3; i < len(allFG.Pix); i += 4 {
		allFG.Pix[i] = 0xff
	}
	noBG := func(x, y int) bool {
		_, _, _, a := pixelAt(allFG, x, y)
		return a == 0
	}
	if s := regionScore(orig, sim, regionAll, noBG); s.OK {
		t.Errorf("background PSNR with no background pixels = %+v, want no score", s)
	}
}

func TestHeatmap(t *testing.T) {
	a := image.NewRGBA(image.Rect(0, 0, 4, 1))
	b := image.NewRGBA(image.Rect(0, 0, 4, 1))
	b.SetRGBA(1, 0, color.RGBA{2, 0, 7, 0xff})   // max channel difference 7 -> 28
	b.SetRGBA(2, 0, color.RGBA{100, 0, 0, 0xff}) // 100 -> 400, clamped to 255
	b.SetRGBA(3, 0, color.RGBA{0, 1, 0, 0xff})   // 1 -> 4
	h := heatmap(a, b)
	if h.Bounds() != a.Bounds() {
		t.Fatalf("heatmap bounds %v, want %v", h.Bounds(), a.Bounds())
	}
	want := []uint8{0, 28, 255, 4}
	for x, w := range want {
		if got := h.GrayAt(x, 0).Y; got != w {
			t.Errorf("heatmap at x=%d is %d, want %d", x, got, w)
		}
	}
}

func TestCompareDirsEndToEnd(t *testing.T) {
	root := t.TempDir()
	origDir := filepath.Join(root, "orig")
	simDir := filepath.Join(root, "sim")
	outDir := filepath.Join(root, "out")
	for _, d := range []string{origDir, simDir} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}

	base := gradient(0)
	// Original shots. f1200-1201 matches the remake at +1 except for a
	// foreground block, so its whole-image PSNR is finite and its background is
	// identical. f1300-1301 matches exactly at +1 and has no mask. f5000-5001 has
	// no remake frame near it. f300-301 is the wrong size. f2000-2001 has a
	// remake candidate of the wrong size.
	writePNG(t, origDir, "f1200-1201.png", base)
	writePNG(t, origDir, "f1300-1301.png", base)
	writePNG(t, origDir, "f5000-5001.png", base)
	writePNG(t, origDir, "f300-301.png", image.NewRGBA(image.Rect(0, 0, 320, 240)))
	writePNG(t, origDir, "f2000-2001.png", base)
	writePNG(t, origDir, "f1200.png", base) // a remake-style name: not an original shot
	writePNG(t, origDir, "f1200-1201.jpg.bak", base)
	if err := os.WriteFile(filepath.Join(origDir, "notes.txt"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Remake frames. f1199, f1200 and f1202 differ from the original everywhere;
	// f1201 differs only inside the mask's foreground block.
	far := clone(base)
	addIn(far, regionAll, 20)
	near := clone(base)
	fg := rect{200, 200, 300, 300}
	addIn(near, fg, 3)
	writePNG(t, simDir, "f1199.png", far)
	writePNG(t, simDir, "f1200.png", far)
	writePNG(t, simDir, "f1201.png", near)
	writePNG(t, simDir, "f1202.png", far)
	mask := image.NewNRGBA(image.Rect(0, 0, frameW, frameH))
	for y := fg.y0; y < fg.y1; y++ {
		for x := fg.x0; x < fg.x1; x++ {
			mask.SetNRGBA(x, y, color.NRGBA{0, 0, 0, 0xff})
		}
	}
	writePNG(t, simDir, "m1201.png", mask)
	writePNG(t, simDir, "f1300.png", far)
	writePNG(t, simDir, "f1301.png", base)
	writePNG(t, simDir, "f2000.png", base)
	writePNG(t, simDir, "f2001.png", image.NewRGBA(image.Rect(0, 0, 320, 240)))
	if err := os.WriteFile(filepath.Join(simDir, "readme.txt"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	results, err := compareDirs(origDir, simDir, outDir, 2)
	if err != nil {
		t.Fatalf("compareDirs: %v", err)
	}
	if len(results) != 5 {
		t.Fatalf("got %d shots, want 5", len(results))
	}
	wantOrder := []int{300, 1200, 1300, 2000, 5000}
	for i, r := range results {
		if r.before != wantOrder[i] {
			t.Errorf("shot %d has before %d, want %d", i, r.before, wantOrder[i])
		}
	}

	// f300-301: the original is the wrong size.
	if r := results[0]; r.err == nil || r.chosen {
		t.Errorf("f300-301: err %v chosen %v, want a size error and no match", r.err, r.chosen)
	}

	// f1200-1201: picks +1 out of four candidates. The difference lies inside
	// the field, so whole image and field are finite, side is identical, and the
	// background is identical.
	if r := results[1]; r.err != nil || !r.chosen || r.off != 1 || r.cands != 4 {
		t.Errorf("f1200-1201: err %v chosen %v off %d cands %d, want match +1 among 4",
			r.err, r.chosen, r.off, r.cands)
	}
	if r := results[1]; !r.all.OK || math.IsInf(r.all.DB, 1) ||
		!r.field.OK || math.IsInf(r.field.DB, 1) || !r.side.OK || !math.IsInf(r.side.DB, 1) {
		t.Errorf("f1200-1201: regions %+v %+v %+v, want finite, finite, +Inf", r.all, r.field, r.side)
	}
	if r := results[1]; !r.bg.OK || !math.IsInf(r.bg.DB, 1) {
		t.Errorf("f1200-1201: background %+v, want +Inf", r.bg)
	}

	// f1300-1301: exact match at +1, and no mask, so no background score.
	if r := results[2]; r.err != nil || !r.chosen || r.off != 1 || r.cands != 2 {
		t.Errorf("f1300-1301: err %v chosen %v off %d cands %d, want match +1 among 2",
			r.err, r.chosen, r.off, r.cands)
	}
	if r := results[2]; !r.all.OK || !math.IsInf(r.all.DB, 1) || r.bg.OK {
		t.Errorf("f1300-1301: all %+v bg %+v, want all +Inf and no background", r.all, r.bg)
	}

	// f2000-2001: the candidate at +1 is the wrong size.
	if r := results[3]; r.err == nil || r.chosen {
		t.Errorf("f2000-2001: err %v chosen %v, want a size error and no match", r.err, r.chosen)
	}

	// f5000-5001: nothing near 5000 in the remake.
	if r := results[4]; !r.missing || r.cands != 0 || r.err != nil || r.chosen {
		t.Errorf("f5000-5001: missing %v cands %d err %v chosen %v, want missing",
			r.missing, r.cands, r.err, r.chosen)
	}

	for _, name := range []string{"d1200.png", "d1300.png"} {
		f, err := os.Open(filepath.Join(outDir, name))
		if err != nil {
			t.Errorf("heatmap %s not written: %v", name, err)
			continue
		}
		img, err := png.Decode(f)
		f.Close()
		if err != nil {
			t.Errorf("heatmap %s does not decode: %v", name, err)
			continue
		}
		if img.Bounds() != image.Rect(0, 0, frameW, frameH) {
			t.Errorf("heatmap %s has bounds %v", name, img.Bounds())
		}
	}
}

func TestRunExitCodes(t *testing.T) {
	empty := t.TempDir()
	var out, errOut bytes.Buffer

	usage := [][]string{
		nil,
		{"-orig", empty},
		{"-orig", empty, "-sim", empty, "-window", "-1"},
		{"-orig", filepath.Join(empty, "missing"), "-sim", empty},
		{"-orig", empty, "-sim", empty}, // no shots in the original directory
		{"-bogus"},
	}
	for _, args := range usage {
		out.Reset()
		errOut.Reset()
		if code := run(args, &out, &errOut); code != 2 {
			t.Errorf("run(%q) = %d, want 2", args, code)
		}
	}

	orig := t.TempDir()
	sim := t.TempDir()
	writePNG(t, orig, "f10-11.png", solid(7))
	out.Reset()
	if code := run([]string{"-orig", orig, "-sim", sim}, &out, &errOut); code != 0 {
		t.Fatalf("run with a missing remake = %d, want 0", code)
	}
	if out.Len() == 0 {
		t.Error("run printed no report")
	}
}
