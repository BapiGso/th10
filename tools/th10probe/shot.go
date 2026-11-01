package main

import (
	"fmt"
	"image"
	"image/png"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

// maxShotFrames bounds how many frames one -shot-frames list may name.
const maxShotFrames = 10000

// parseFrameList parses a -shot-frames value: comma-separated frame numbers and
// inclusive ranges, e.g. "1200,1300-1302" gives [1200 1300 1301 1302]. The result
// is sorted and free of duplicates.
func parseFrameList(s string) ([]int32, error) {
	var out []int32
	for _, item := range strings.Split(s, ",") {
		item = strings.TrimSpace(item)
		loS, hiS, isRange := strings.Cut(item, "-")
		if !isRange {
			hiS = loS
		}
		lo, errLo := strconv.Atoi(strings.TrimSpace(loS))
		hi, errHi := strconv.Atoi(strings.TrimSpace(hiS))
		if errLo != nil || errHi != nil || lo < 0 || hi < lo || hi > math.MaxInt32 {
			return nil, fmt.Errorf("bad frame item %q (want N or LO-HI with 0 <= LO <= HI)", item)
		}
		if len(out)+hi-lo+1 > maxShotFrames {
			return nil, fmt.Errorf("more than %d frames listed", maxShotFrames)
		}
		for f := lo; f <= hi; f++ {
			out = append(out, int32(f))
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
	n := 0
	for i, f := range out {
		if i == 0 || f != out[n-1] {
			out[n] = f
			n++
		}
	}
	return out[:n], nil
}

// shotJob is one captured client area waiting to be written. before and after
// are the frame counter read just before and just after the screen copy.
type shotJob struct {
	img           *image.RGBA
	before, after int32
}

// shotWriter encodes and writes captures on its own goroutine, so the sampling
// loop only pays for the screen copy.
type shotWriter struct {
	dir  string
	jobs chan shotJob
	done chan struct{}
}

func newShotWriter(dir string) *shotWriter {
	w := &shotWriter{dir: dir, jobs: make(chan shotJob, 8), done: make(chan struct{})}
	go w.loop()
	return w
}

// Queue hands a capture to the writer. It blocks only when eight captures are
// already waiting.
func (w *shotWriter) Queue(j shotJob) {
	w.jobs <- j
}

// Close writes everything still queued and returns once every file is on disk.
func (w *shotWriter) Close() {
	close(w.jobs)
	<-w.done
}

func (w *shotWriter) loop() {
	defer close(w.done)
	for j := range w.jobs {
		path := filepath.Join(w.dir, fmt.Sprintf("f%d-%d.png", j.before, j.after))
		if err := writePNG(path, j.img); err != nil {
			fmt.Fprintln(os.Stderr, "th10probe: shot:", err)
			continue
		}
		fmt.Fprintf(os.Stderr, "shot f%d-%d -> %s\n", j.before, j.after, path)
	}
}

func writePNG(path string, img *image.RGBA) error {
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
