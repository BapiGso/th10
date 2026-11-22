// Transparent neighbors must not escape the web quality gate.
package main

import (
	"bytes"
	"testing"

	"th10/anm"
)

func TestQualityIncludesHiddenRGBAndFilterEdges(t *testing.T) {
	a := &anm.Texture{Format: anm.FormatBGRA8888, W: 2, H: 2, Pixels: make([]byte, 16)}
	a.Pixels[3] = 255
	b := *a
	b.Pixels = bytes.Clone(a.Pixels)
	for at := 4; at < 16; at += 4 {
		b.Pixels[at], b.Pixels[at+1], b.Pixels[at+2] = 255, 255, 255
	}
	q, err := CompareTextureQuality(a, &b)
	if err != nil || q.Acceptable() || q.EdgeSamples != 1 || q.AllPSNR >= MinAllPSNR || q.EdgePSNR >= MinEdgePSNR {
		t.Fatalf("hidden RGB/filter edge error escaped the gate: %+v %v", q, err)
	}
	b.Pixels[3] = 128
	if _, err := CompareTextureQuality(a, &b); err == nil {
		t.Fatal("quality gate accepted changed alpha")
	}
}
