package trace

import (
	"fmt"
	"strconv"
)

// HalfRow records a failed consistency check without giving diagnostic
// memory contents the status of a completed game frame.
type HalfRow struct {
	Reason   string
	Attempts int
	Before   int
	After    int
}

func parseHalfLine(t []string) (*HalfRow, error) {
	if len(t) != 8 || t[2] != "attempts" || t[4] != "before" || t[6] != "after" {
		return nil, fmt.Errorf("malformed H line")
	}
	h := &HalfRow{Reason: t[1]}
	for i, dst := range []*int{&h.Attempts, &h.Before, &h.After} {
		v, err := strconv.Atoi(t[3+2*i])
		if err != nil {
			return nil, fmt.Errorf("h line: %w", err)
		}
		*dst = v
	}
	if h.Attempts < 0 {
		return nil, fmt.Errorf("negative H attempts")
	}
	return h, nil
}
