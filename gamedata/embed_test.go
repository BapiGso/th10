package gamedata

import (
	"bytes"
	"errors"
	"io/fs"
	"path/filepath"
	"strings"
	"testing"

	"th10/anm"
	"th10/assets"
	"th10/datfs"
)

// TestEmbeddedMatchesDat checks the embedded copies against the user's
// th10.dat: logical bytes and desktop THTX pixels byte for byte, including
// colour under zero alpha. Packed resources are restored before comparison.
// Skips without a TH10 install.
func TestEmbeddedMatchesDat(t *testing.T) {
	dir, err := datfs.Locate()
	if err != nil {
		t.Skip(err)
	}
	dat, err := datfs.Open(filepath.Join(dir, "th10.dat"))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = dat.Close() }()

	raw := 0
	for _, name := range dat.Names() {
		got, err := ReadFile(datfs.EmbedPath(name))
		if errors.Is(err, fs.ErrNotExist) {
			continue // not embedded yet
		} else if err != nil {
			t.Fatal(err)
		}
		want, err := dat.ReadFile(name)
		if err != nil {
			t.Fatal(err)
		}
		if strings.HasSuffix(name, ".anm") {
			compareArchiveANM(t, name, got, want)
		} else if !bytes.Equal(got, want) {
			t.Errorf("%s differs from th10.dat:%s", datfs.EmbedPath(name), name)
		}
		raw++
	}

	t.Logf("checked %d raw files", raw)
}

// compareArchiveANM verifies every non-pixel byte in both builds and every
// raw pixel byte on desktop. This includes unused padding and script labels,
// not only the interpreted VM instructions or the visible RGBA pixels.
func compareArchiveANM(t *testing.T, name string, got, want []byte) {
	t.Helper()
	a, err := anm.PixelRanges(got)
	if err != nil {
		t.Fatal(err)
	}
	b, err := anm.PixelRanges(want)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != len(want) || len(a) != len(b) {
		t.Fatalf("%s: restored ANM layout differs from archive", name)
	}
	x, y := bytes.Clone(got), bytes.Clone(want)
	for i, r := range a {
		p, q := r.Texture, b[i].Texture
		if r.Offset != b[i].Offset || p.W != q.W || p.H != q.H || p.Format != q.Format || len(p.Pixels) != len(q.Pixels) {
			t.Fatalf("%s texture %d: dimensions/format/range changed", name, i)
		}
		if assets.TextureVariant == "lossless" && !bytes.Equal(p.Pixels, q.Pixels) {
			t.Errorf("%s texture %d: lossless THTX pixels differ", name, i)
		}
		clear(x[r.Offset : r.Offset+len(p.Pixels)])
		clear(y[b[i].Offset : b[i].Offset+len(q.Pixels)])
	}
	if !bytes.Equal(x, y) {
		t.Errorf("%s: ANM non-pixel bytes differ from archive", name)
	}
}
