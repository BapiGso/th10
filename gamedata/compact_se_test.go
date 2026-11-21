package gamedata

import (
	"bytes"
	"testing"

	"th10/assets"
)

func TestCompactSERejectsTruncation(t *testing.T) {
	data, err := assets.Assets.ReadFile("wav/se/se_gun00.wav")
	if err != nil {
		t.Fatal(err)
	}
	if len(data) < 8 || string(data[:4]) != "ZSE1" {
		t.Skip("this sound is stored as raw WAV")
	}
	for _, size := range []int{0, 4, 7, 8, len(data) - 1} {
		if _, err := compactWAV(data[:size]); err == nil {
			t.Fatalf("accepted truncated SE at %d", size)
		}
	}
	bad := bytes.Clone(data)
	for i := 4; i < 8; i++ {
		bad[i] = 255
	}
	if _, err := compactWAV(bad); err == nil {
		t.Fatal("accepted unbounded SE output")
	}
}
