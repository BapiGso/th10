package main

import (
	"bytes"
	"strings"
	"testing"

	"th10/trace"
)

func TestDiffReportsCaptureWithoutFooterAsIncomplete(t *testing.T) {
	var data bytes.Buffer
	w := trace.NewAtomicWriter(&data)
	if err := w.BeginCapture(trace.CaptureHeader{Stage: 1, Demo: -1, Records: 2}); err != nil {
		t.Fatal(err)
	}
	w.WriteFrame(&trace.Frame{Index: 1})
	w.WriteFrame(&trace.Frame{Index: 2})
	if err := w.Flush(); err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	a := writeTrace(t, dir, "a.txt", data.String())
	b := writeTrace(t, dir, "b.txt", data.String())
	code, out := runDiffStdout(t, a, b)
	if code != 2 || !strings.Contains(out, "INCOMPLETE:") || strings.Contains(out, "MATCH:") {
		t.Fatalf("truncated capture accepted: %d %s", code, out)
	}
}
