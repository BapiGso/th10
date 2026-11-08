package trace

import (
	"strings"
	"testing"
)

func TestReaderRejectsTruncatedBulletRows(t *testing.T) {
	const header = "F 1 p 0.000 400.000 life 2 power 400 faith 50000 score 0 graze 0 state 1 invinc 0 bomb 0 | e 0 b 1 i 0 pb 0\n"
	for _, line := range []string{"B 25.", "B 72.812 33.664 ang -0.35903 spd 3.300", "B 1 2 ang 0 spd 1 type"} {
		r := NewReader(strings.NewReader(header + line))
		if _, err := r.Next(); err == nil || !strings.Contains(err.Error(), "short bullet row") {
			t.Fatalf("row %q: err=%v, want truncated-row error instead of panic", line, err)
		}
	}
	const valid = "B 72.812 33.664 ang -0.35903 spd 3.300 type 6 color 2\n"
	r := NewReader(strings.NewReader(header + valid))
	f, err := r.Next()
	if err != nil || f == nil || len(f.B) != 1 || f.B[0].Type != 6 || f.B[0].Color != 2 {
		t.Fatalf("valid row changed: frame=%v err=%v", f, err)
	}
}
