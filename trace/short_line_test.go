package trace

import (
	"strings"
	"testing"
)

func TestReaderTruncatedRowsReturnErrors(t *testing.T) {
	header := "F 6360 p 0 400 life 0 power 0 faith 50000 score 0 graze 0 state 1 invinc 0 bomb 0 | e 1 b 1 i 0 pb 0\n"
	for _, line := range []string{"F 6360 p", "E 0 0 hp 1 act", "B 483.200 139.284 ang 2.64680 spd", "M 0 clock"} {
		t.Run(line[:1], func(t *testing.T) {
			r := NewReader(strings.NewReader(header + line))
			f, err := r.Next()
			if err == nil || f != nil {
				t.Fatalf("truncated row returned frame=%v err=%v", f, err)
			}
		})
	}
}

func TestReaderWhitespaceOnlyLine(t *testing.T) {
	r := NewReader(strings.NewReader(" \t \r\n"))
	if f, err := r.Next(); err != nil || f != nil {
		t.Fatalf("whitespace returned frame=%v err=%v", f, err)
	}
}
