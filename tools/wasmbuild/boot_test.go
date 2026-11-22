package main

import (
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"th10/gamedata"
)

// The loading screen images decode at their game sizes, and loading.mjs
// cuts the same rects as ascii.anm's sprites 232-234.
func TestBootImages(t *testing.T) {
	out := t.TempDir()
	if err := buildBootImages(out); err != nil {
		t.Fatal(err)
	}
	for name, size := range map[string]image.Point{"sig.jpg": {640, 480}, "loading.png": {128, 128}} {
		f, err := os.Open(filepath.Join(out, name))
		if err != nil {
			t.Fatal(err)
		}
		c, _, err := image.DecodeConfig(f)
		_ = f.Close()
		if err != nil || c.Width != size.X || c.Height != size.Y {
			t.Fatalf("%s: %v %dx%d", name, err, c.Width, c.Height)
		}
	}
	js, err := webFiles.ReadFile("web/loading.mjs")
	if err != nil {
		t.Fatal(err)
	}
	ascii, err := gamedata.ANM("ascii.anm")
	if err != nil {
		t.Fatal(err)
	}
	for name, id := range map[string]int{"shoujo": 232, "now": 233, "leaf": 234} {
		s := ascii.Sprites[id]
		want := fmt.Sprintf("%s: [%d, %d, %d, %d]", name, int(s.X), int(s.Y), int(s.W), int(s.H))
		if !strings.Contains(string(js), want) {
			t.Fatalf("loading.mjs lacks %q", want)
		}
	}
}
