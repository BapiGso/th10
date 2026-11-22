package main

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	gtfont "github.com/go-text/typesetting/font"
)

func testOriginalFontDirectory(t *testing.T) string {
	t.Helper()
	if p := os.Getenv("TH10_FONT_SOURCE"); p != "" {
		return p
	}
	p, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for {
		candidate := filepath.Join(p, ".tmp", "codex", "base", "assets", "font")
		if _, err := os.Stat(filepath.Join(candidate, "DFYuGaSo-W3.ttc")); err == nil {
			return candidate
		}
		parent := filepath.Dir(p)
		if parent == p {
			t.Skip("set TH10_FONT_SOURCE to check original glyph programs")
		}
		p = parent
	}
}

func TestFontSubsetKeepsGlyphProgramsAndMetrics(t *testing.T) {
	source := testOriginalFontDirectory(t)
	root := filepath.Join("..", "..", "assets")
	corpus, err := collectFontCharacters(root)
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"DFYuGaSo-W3.ttc", "DFYuGaSo-W7.ttc"} {
		t.Run(name, func(t *testing.T) {
			raw, err := os.ReadFile(filepath.Join(source, name))
			if err != nil {
				t.Fatal(err)
			}
			text := corpus.Text
			data, p, err := subsetFont(raw, text)
			if err != nil {
				t.Fatal(err)
			}
			stored, err := os.ReadFile(filepath.Join(root, "font", name))
			if err != nil || !bytes.Equal(data, stored) {
				t.Fatal("stored font is not reproducible from its original")
			}
			again, _, err := subsetFont(raw, text)
			if err != nil || !bytes.Equal(data, again) {
				t.Fatal("font subset is nondeterministic")
			}
			if tableChecksum(data) != 0xb1b0afba {
				t.Fatal("invalid SFNT checksum adjustment")
			}
			oldTables, err := readFontTables(raw)
			if err != nil {
				t.Fatal(err)
			}
			newTables, err := readFontTables(data)
			if err != nil {
				t.Fatal(err)
			}
			for tag, want := range oldTables.tables {
				if tag == "glyf" || tag == "loca" || tag == "cmap" || tag == "head" || tag == "DSIG" {
					continue
				}
				if !bytes.Equal(want, newTables.tables[tag]) {
					t.Errorf("changed metrics/hinting/layout table %q", tag)
				}
			}
			for g, want := range oldTables.glyphs {
				if _, kept := p.GlyphHashes[g]; kept {
					if !bytes.Equal(want, newTables.glyphs[g]) {
						t.Errorf("glyph %d: changed outline or hint program", g)
					}
				} else if len(newTables.glyphs[g]) != 0 {
					t.Errorf("unused glyph %d retained an outline", g)
				}
			}
			oldFaces, err := gtfont.ParseTTC(bytes.NewReader(raw))
			if err != nil {
				t.Fatal(err)
			}
			newFaces, err := gtfont.ParseTTC(bytes.NewReader(data))
			if err != nil || len(newFaces) != 1 {
				t.Fatal("subset did not retain exactly the used face")
			}
			for _, r := range text {
				old, _ := oldFaces[0].NominalGlyph(r)
				got, _ := newFaces[0].NominalGlyph(r)
				if old != got {
					t.Errorf("U+%04X: changed glyph index %d -> %d", r, old, got)
				}
			}
			if g, _ := newFaces[0].NominalGlyph(0x10ffff); g != 0 {
				t.Fatal("unknown character did not fall back to .notdef")
			}
		})
	}
}
