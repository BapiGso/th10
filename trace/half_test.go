package trace

import (
	"bytes"
	"reflect"
	"strings"
	"testing"
)

const legacyFrame = "F 1 p 0.000 400.000 life 2 power 500 faith 50000 score 9999999990 graze 0 state 1 invinc 0 bomb 0 | e 0 b 0 i 0 pb 0\n"

func TestHalfFrameRoundTrip(t *testing.T) {
	f := &Frame{Index: 1, PY: 400, PLife: 2, PPower: 500, PFaith: 50000, PScore: 9999999990, PState: 1,
		Half: &HalfRow{Reason: "unstable", Attempts: 4, Before: 1, After: 2}}
	var out bytes.Buffer
	w := NewWriter(&out)
	w.WriteFrame(f)
	if err := w.Flush(); err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(out.String(), legacyFrame+"H unstable attempts 4 before 1 after 2\n") {
		t.Fatalf("unexpected encoding: %q", out.String())
	}
	got, err := NewReader(&out).Next()
	if err != nil || !reflect.DeepEqual(got, f) {
		t.Fatalf("round trip: frame=%+v err=%v", got, err)
	}
}

func TestLegacyFrameEncodingUnchanged(t *testing.T) {
	r := NewReader(strings.NewReader(legacyFrame))
	f, err := r.Next()
	if err != nil || f == nil || f.Half != nil {
		t.Fatalf("legacy read: frame=%v err=%v", f, err)
	}
	var out bytes.Buffer
	w := NewWriter(&out)
	w.WriteFrame(f)
	if err := w.Flush(); err != nil || out.String() != legacyFrame {
		t.Fatalf("legacy encoding changed: %q err=%v", out.String(), err)
	}
}

func TestReaderRejectsMalformedHalfMarker(t *testing.T) {
	for _, marker := range []string{
		"H unstable attempts 2 before 1 after\n", "H unstable attempts -1 before 1 after 1\n",
		"H unstable attempts x before 1 after 1\n", "H unstable attempts 2 before 1 after 1 extra\n",
		"H unstable attempts 2 before 1 after 1\nH unstable attempts 2 before 1 after 1\n",
	} {
		if _, err := NewReader(strings.NewReader(legacyFrame + marker)).Next(); err == nil {
			t.Fatalf("accepted marker %q", marker)
		}
	}
}
