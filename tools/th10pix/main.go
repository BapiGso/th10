// Command th10pix compares screenshots of the original game with screenshots of
// the remake: the L3 visual diff (docs/research/m4_render.md, section 3).
//
// Usage:
//
//	th10pix -orig DIR -sim DIR [-out DIR] [-window 2]
//
// -orig holds the original shots f<before>-<after>.png written by th10probe.
// -sim holds the remake frames f<N>.png and, optionally, foreground masks
// m<N>.png, where a pixel is foreground when the mask alpha is nonzero. For each
// original shot the remake frame numbered within -window of its before frame,
// with the lowest whole-image MSE, is the match. The report gives PSNR for the
// whole image, the field and side regions, and the background (the pixels the
// match's mask leaves as background).
//
// -out writes d<before>.png for each shot: a grayscale heatmap of the difference
// (largest channel difference times 4, clamped). If two shots share a before
// frame, the later one is written as d<before>-<after>.png so that nothing is
// overwritten.
//
// The tool only reports. It exits 0 after a report whatever the quality, and 2
// on usage errors, unreadable directories, or an original directory with no
// shots. A shot that cannot be compared (missing candidates, wrong image size)
// is reported on its own line and the run goes on.
package main

import (
	"errors"
	"flag"
	"fmt"
	"image"
	"image/png"
	"io"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// maxWindow bounds -window; each shot tests 2*window+1 remake frames.
const maxWindow = 1000

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

// run parses args, runs the comparison and prints the report to stdout. It
// returns the process exit code.
func run(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("th10pix", flag.ContinueOnError)
	fs.SetOutput(stderr)
	var origDir, simDir, outDir string
	var window int
	fs.StringVar(&origDir, "orig", "", "directory of original shots f<before>-<after>.png")
	fs.StringVar(&simDir, "sim", "", "directory of remake frames f<N>.png and masks m<N>.png")
	fs.StringVar(&outDir, "out", "", "if set, write heatmaps d<before>.png into this directory")
	fs.IntVar(&window, "window", 2, "candidate remake offsets -K..+K around each shot's frame")
	fs.Usage = func() {
		fmt.Fprintln(stderr, "usage: th10pix -orig DIR -sim DIR [-out DIR] [-window 2]")
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		return 2
	}
	if fs.NArg() > 0 || origDir == "" || simDir == "" {
		fs.Usage()
		return 2
	}
	if window < 0 || window > maxWindow {
		fmt.Fprintf(stderr, "th10pix: -window must be 0..%d\n", maxWindow)
		return 2
	}
	results, err := compareDirs(origDir, simDir, outDir, window)
	if err != nil {
		fmt.Fprintln(stderr, "th10pix:", err)
		return 2
	}
	printReport(stdout, results, window)
	return 0
}

// compareDirs compares every original shot in origDir with the remake frames in
// simDir. It returns one result per shot, sorted by before then after, and writes
// each shot's heatmap into outDir when outDir is not empty. The error covers only
// unreadable directories and an original directory with no shots; per-shot
// problems are recorded in the results.
func compareDirs(origDir, simDir, outDir string, window int) ([]shotResult, error) {
	shots, err := listShots(origDir)
	if err != nil {
		return nil, err
	}
	if len(shots) == 0 {
		return nil, fmt.Errorf("no f<before>-<after>.png shots in %s", origDir)
	}
	rd, err := scanRemake(simDir)
	if err != nil {
		return nil, err
	}
	if outDir != "" {
		if err := os.MkdirAll(outDir, 0o755); err != nil {
			return nil, err
		}
	}
	results := make([]shotResult, 0, len(shots))
	seen := map[int]bool{}
	for _, s := range shots {
		heat := ""
		if outDir != "" {
			heat = fmt.Sprintf("d%d.png", s.before)
			if seen[s.before] {
				heat = fmt.Sprintf("d%d-%d.png", s.before, s.after)
			}
		}
		seen[s.before] = true
		results = append(results, compareShot(s, rd, window, outDir, heat))
	}
	return results, nil
}

// origShot is one original screenshot on disk.
type origShot struct {
	before, after int
	path          string
}

// listShots returns the original shots in dir, sorted by before then after.
// Files with other names are ignored.
func listShots(dir string) ([]origShot, error) {
	ents, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var shots []origShot
	for _, e := range ents {
		if e.IsDir() {
			continue
		}
		if b, a, ok := parseOrigName(e.Name()); ok {
			shots = append(shots, origShot{before: b, after: a, path: filepath.Join(dir, e.Name())})
		}
	}
	sort.Slice(shots, func(i, j int) bool {
		if shots[i].before != shots[j].before {
			return shots[i].before < shots[j].before
		}
		return shots[i].after < shots[j].after
	})
	return shots, nil
}

// remakeDir indexes the remake's frames and masks by frame number.
type remakeDir struct {
	frames map[int]string // f<N>.png
	masks  map[int]string // m<N>.png
}

// scanRemake indexes the frame and mask files in dir. Other files are ignored.
func scanRemake(dir string) (remakeDir, error) {
	ents, err := os.ReadDir(dir)
	if err != nil {
		return remakeDir{}, err
	}
	rd := remakeDir{frames: map[int]string{}, masks: map[int]string{}}
	for _, e := range ents {
		if e.IsDir() {
			continue
		}
		if n, ok := parseFrameName(e.Name()); ok {
			rd.frames[n] = filepath.Join(dir, e.Name())
		}
		if n, ok := parseMaskName(e.Name()); ok {
			rd.masks[n] = filepath.Join(dir, e.Name())
		}
	}
	return rd, nil
}

// shotResult is the report for one original shot.
type shotResult struct {
	before, after int
	cands         int   // remake frames that exist in before-K..before+K
	missing       bool  // no candidate frame exists at all
	chosen        bool  // a match was picked; off and the scores are valid
	off           int   // offset of the chosen match
	all, field    score // PSNR of the chosen match, per region
	side, bg      score
	err           error // the shot could not be fully compared
}

// compareShot picks the best remake match for one original shot and scores it.
// Problems are recorded in the result, not returned, so the run continues.
func compareShot(s origShot, rd remakeDir, window int, outDir, heatName string) shotResult {
	res := shotResult{before: s.before, after: s.after}
	orig, err := loadFrame(s.path)
	if err != nil {
		res.err = err
		return res
	}
	var offs []int
	for o := -window; o <= window; o++ {
		if _, ok := rd.frames[s.before+o]; ok {
			offs = append(offs, o)
		}
	}
	res.cands = len(offs)
	if res.cands == 0 {
		res.missing = true
		return res
	}

	cands := make([]candidate, 0, len(offs))
	for _, o := range offs {
		img, err := loadFrame(rd.frames[s.before+o])
		if err != nil {
			res.err = err
			return res
		}
		cands = append(cands, candidate{off: o, img: img})
	}
	best, _ := bestOffset(orig, cands)
	res.chosen = true
	res.off = best.off

	var mask image.Image
	if mp, ok := rd.masks[s.before+best.off]; ok {
		mask, err = loadFrame(mp)
		if err != nil {
			res.err = err
			return res
		}
	}
	res.all = regionScore(orig, best.img, regionAll, nil)
	res.field = regionScore(orig, best.img, regionField, nil)
	res.side = regionScore(orig, best.img, regionSide, nil)
	if mask != nil {
		res.bg = regionScore(orig, best.img, regionAll, func(x, y int) bool {
			_, _, _, a := pixelAt(mask, x, y)
			return a == 0
		})
	}
	if outDir != "" {
		if err := writeHeatmap(filepath.Join(outDir, heatName), heatmap(orig, best.img)); err != nil {
			res.err = err
		}
	}
	return res
}

// loadFrame decodes a PNG and checks that it is frameW x frameH. Size errors name
// the file.
func loadFrame(path string) (image.Image, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	img, err := png.Decode(f)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", filepath.Base(path), err)
	}
	if err := checkFrame(img); err != nil {
		return nil, fmt.Errorf("%s: %w", filepath.Base(path), err)
	}
	return img, nil
}

// writeHeatmap encodes img as a PNG at path.
func writeHeatmap(path string, img image.Image) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	if err := png.Encode(f, img); err != nil {
		f.Close()
		return err
	}
	return f.Close()
}

// printReport writes one line per shot, then the summary.
func printReport(w io.Writer, results []shotResult, window int) {
	fmt.Fprintf(w, "%6s %6s %4s %5s %7s %7s %7s %7s  %s\n",
		"frame", "after", "off", "cands", "all", "field", "side", "bg", "note")
	for _, r := range results {
		line := fmt.Sprintf("%6d %6d %4s %5d %7s %7s %7s %7s  %s",
			r.before, r.after, offText(r), r.cands,
			scoreText(r.all), scoreText(r.field), scoreText(r.side), scoreText(r.bg),
			noteText(r, window))
		fmt.Fprintln(w, strings.TrimRight(line, " "))
	}

	var compared, missing, failed int
	offsets := map[int]int{}
	var means [4]regionMean // all, field, side, bg
	for _, r := range results {
		if r.missing {
			missing++
		}
		if r.err != nil {
			failed++
		}
		if !r.chosen {
			continue
		}
		compared++
		offsets[r.off]++
		for i, s := range []score{r.all, r.field, r.side, r.bg} {
			means[i].add(s)
		}
	}

	fmt.Fprintf(w, "\nshots %d  compared %d  without candidates %d  errors %d\n",
		len(results), compared, missing, failed)

	if len(offsets) > 0 {
		keys := make([]int, 0, len(offsets))
		maxN := 0
		for o, n := range offsets {
			keys = append(keys, o)
			maxN = max(maxN, n)
		}
		sort.Ints(keys)
		fmt.Fprintln(w, "chosen offsets:")
		for _, o := range keys {
			n := offsets[o]
			bar := strings.Repeat("#", (n*40+maxN-1)/maxN)
			fmt.Fprintf(w, "  %4s %5d  %s\n", offsetString(o), n, bar)
		}
		if len(keys) > 1 {
			fmt.Fprintf(w, "warning: chosen offset is not constant (%d distinct); check -window and the frame alignment\n", len(keys))
		}
	}

	fmt.Fprintln(w, "mean PSNR over finite values (dB):")
	fmt.Fprintf(w, "  %-6s %8s %7s %5s %5s\n", "region", "mean", "finite", "inf", "n/a")
	for i, name := range []string{"all", "field", "side", "bg"} {
		m := means[i]
		fmt.Fprintf(w, "  %-6s %8s %7d %5d %5d\n", name, m.mean(), m.finite, m.inf, m.na)
	}
}

// regionMean accumulates one region's PSNR values for the summary.
type regionMean struct {
	sum             float64
	finite, inf, na int
}

func (m *regionMean) add(s score) {
	switch {
	case !s.OK:
		m.na++
	case math.IsInf(s.DB, 1):
		m.inf++
	default:
		m.sum += s.DB
		m.finite++
	}
}

// mean returns the mean of the finite values, or "-" when there are none.
func (m *regionMean) mean() string {
	if m.finite == 0 {
		return "-"
	}
	return fmt.Sprintf("%.2f", m.sum/float64(m.finite))
}

// offsetString formats a match offset as +1, -2 or 0.
func offsetString(o int) string {
	if o == 0 {
		return "0"
	}
	return fmt.Sprintf("%+d", o)
}

// offText is the off column: "-" when no match was chosen.
func offText(r shotResult) string {
	if !r.chosen {
		return "-"
	}
	return offsetString(r.off)
}

// scoreText formats a PSNR for a table cell: "-" when there was nothing to
// compare, "inf" for identical pixels, otherwise two decimals.
func scoreText(s score) string {
	switch {
	case !s.OK:
		return "-"
	case math.IsInf(s.DB, 1):
		return "inf"
	default:
		return fmt.Sprintf("%.2f", s.DB)
	}
}

// noteText explains a shot that has no scores.
func noteText(r shotResult, window int) string {
	switch {
	case r.err != nil:
		return "error: " + r.err.Error()
	case r.missing:
		return fmt.Sprintf("missing: no remake frame in %d..%d", r.before-window, r.before+window)
	}
	return ""
}
