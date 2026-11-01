package main

import (
	"fmt"
	"image"
	"image/color"
	"math"
	"regexp"
	"strconv"
)

// Frame geometry. Every original, remake and mask image must be exactly this size.
const (
	frameW = 640
	frameH = 480
)

// rect is a half-open pixel rectangle [x0, x1) x [y0, y1).
type rect struct{ x0, y0, x1, y1 int }

// The comparison regions. field is x 32..415 and y 16..463 inclusive; side is
// the right-hand panel.
var (
	regionAll   = rect{0, 0, frameW, frameH}
	regionField = rect{32, 16, 416, 464}
	regionSide  = rect{416, 0, frameW, frameH}
)

// File name patterns. Anything else in a directory is ignored.
var (
	origNameRE  = regexp.MustCompile(`^f(\d+)-(\d+)\.png$`)
	frameNameRE = regexp.MustCompile(`^f(\d+)\.png$`)
	maskNameRE  = regexp.MustCompile(`^m(\d+)\.png$`)
)

// parseOrigName parses an original shot name f<before>-<after>.png.
func parseOrigName(name string) (before, after int, ok bool) {
	m := origNameRE.FindStringSubmatch(name)
	if m == nil {
		return 0, 0, false
	}
	b, errB := strconv.Atoi(m[1])
	a, errA := strconv.Atoi(m[2])
	if errB != nil || errA != nil {
		return 0, 0, false
	}
	return b, a, true
}

// parseFrameName parses a remake frame name f<N>.png.
func parseFrameName(name string) (n int, ok bool) {
	return parseFrameNumber(frameNameRE, name)
}

// parseMaskName parses a remake mask name m<N>.png.
func parseMaskName(name string) (n int, ok bool) {
	return parseFrameNumber(maskNameRE, name)
}

func parseFrameNumber(re *regexp.Regexp, name string) (int, bool) {
	m := re.FindStringSubmatch(name)
	if m == nil {
		return 0, false
	}
	n, err := strconv.Atoi(m[1])
	if err != nil {
		return 0, false
	}
	return n, true
}

// checkFrame returns an error unless img covers exactly the frame at the origin.
func checkFrame(img image.Image) error {
	if b := img.Bounds(); b != image.Rect(0, 0, frameW, frameH) {
		return fmt.Errorf("%dx%d, want %dx%d", b.Dx(), b.Dy(), frameW, frameH)
	}
	return nil
}

// pixelAt returns the 8-bit R, G, B and alpha of (x, y) with the colour
// un-premultiplied. The decoder's own types take a fast path; anything else goes
// through the generic colour model.
func pixelAt(img image.Image, x, y int) (r, g, b, a uint8) {
	switch m := img.(type) {
	case *image.NRGBA:
		i := m.PixOffset(x, y)
		p := m.Pix[i : i+4 : i+4]
		return p[0], p[1], p[2], p[3]
	case *image.RGBA:
		i := m.PixOffset(x, y)
		p := m.Pix[i : i+4 : i+4]
		if p[3] == 0xff {
			return p[0], p[1], p[2], 0xff
		}
	}
	c := color.NRGBAModel.Convert(img.At(x, y)).(color.NRGBA)
	return c.R, c.G, c.B, c.A
}

// sqDiffSum returns the sum of squared R, G and B differences between a and b
// over the pixels of r that keep accepts (nil accepts all), and how many pixels
// were counted. Both images must cover r.
func sqDiffSum(a, b image.Image, r rect, keep func(x, y int) bool) (sum int64, pixels int) {
	for y := r.y0; y < r.y1; y++ {
		for x := r.x0; x < r.x1; x++ {
			if keep != nil && !keep(x, y) {
				continue
			}
			ar, ag, ab, _ := pixelAt(a, x, y)
			br, bg, bb, _ := pixelAt(b, x, y)
			dr := int64(ar) - int64(br)
			dg := int64(ag) - int64(bg)
			db := int64(ab) - int64(bb)
			sum += dr*dr + dg*dg + db*db
			pixels++
		}
	}
	return sum, pixels
}

// mse converts a sum from sqDiffSum into the mean squared difference per channel,
// in 8-bit units. ok is false when no pixels were counted.
func mse(sum int64, pixels int) (m float64, ok bool) {
	if pixels == 0 {
		return 0, false
	}
	return float64(sum) / float64(3*pixels), true
}

// psnrFromMSE returns 10*log10(255^2/mse) in dB, which is +Inf when mse is 0.
func psnrFromMSE(m float64) float64 {
	if m == 0 {
		return math.Inf(1)
	}
	return 10 * math.Log10(255*255/m)
}

// score is a PSNR in dB. OK is false when the region had nothing to compare,
// which prints as "-". DB is +Inf for identical pixels.
type score struct {
	DB float64
	OK bool
}

// regionScore returns the PSNR of a against b over the pixels of r that keep
// accepts (nil accepts all).
func regionScore(a, b image.Image, r rect, keep func(x, y int) bool) score {
	sum, n := sqDiffSum(a, b, r, keep)
	m, ok := mse(sum, n)
	if !ok {
		return score{}
	}
	return score{DB: psnrFromMSE(m), OK: true}
}

// candidate is one remake frame F+off considered as the match for a shot.
type candidate struct {
	off int
	img image.Image
}

// bestOffset returns the candidate with the lowest whole-image MSE against orig.
// Ties go to the smallest |off|, then to the smaller off. ok is false when cands
// is empty. The comparison uses the raw squared-difference sums, which order the
// same way as the MSE because every candidate covers the same pixels.
func bestOffset(orig image.Image, cands []candidate) (best candidate, ok bool) {
	var bestSum int64
	for _, c := range cands {
		sum, _ := sqDiffSum(orig, c.img, regionAll, nil)
		if !ok || sum < bestSum || (sum == bestSum && preferOffset(c.off, best.off)) {
			best, bestSum, ok = c, sum, true
		}
	}
	return best, ok
}

// preferOffset reports whether offset o wins a tie against offset than: the
// smaller |o| first, then the smaller o.
func preferOffset(o, than int) bool {
	ao, at := o, than
	if ao < 0 {
		ao = -ao
	}
	if at < 0 {
		at = -at
	}
	if ao != at {
		return ao < at
	}
	return o < than
}

// heatmap returns the per-pixel difference of a against b: the largest absolute
// difference over R, G and B, times 4, clamped to 255. b must cover a's bounds.
func heatmap(a, b image.Image) *image.Gray {
	bounds := a.Bounds()
	out := image.NewGray(bounds)
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			ar, ag, ab, _ := pixelAt(a, x, y)
			br, bg, bb, _ := pixelAt(b, x, y)
			d := absDiff8(ar, br)
			if v := absDiff8(ag, bg); v > d {
				d = v
			}
			if v := absDiff8(ab, bb); v > d {
				d = v
			}
			v := d * 4
			if v > 255 {
				v = 255
			}
			out.Pix[out.PixOffset(x, y)] = uint8(v)
		}
	}
	return out
}

func absDiff8(a, b uint8) int {
	if a > b {
		return int(a - b)
	}
	return int(b - a)
}
