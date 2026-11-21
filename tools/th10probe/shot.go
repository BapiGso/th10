package main

import (
	"fmt"
	"math"
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
