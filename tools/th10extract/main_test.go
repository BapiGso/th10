package main

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"th10/datfs"
)

func TestExtractOriginalAndPresentationArchives(t *testing.T) {
	dir, err := datfs.Locate()
	if err != nil {
		t.Skip(err)
	}
	for _, archive := range []string{"th10.dat", "th10c.dat"} {
		t.Run(archive, func(t *testing.T) {
			out := t.TempDir()
			name := "e00.msg"
			if archive == "th10.dat" {
				err = runArchive(dir, archive, out, false, false, 4, []string{name})
			} else {
				name = "e00.mst"
				err = runArchive(dir, archive, out, false, false, 4, []string{name})
			}
			if err != nil {
				t.Fatal(err)
			}
			d, err := datfs.Open(filepath.Join(dir, archive))
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() { _ = d.Close() })
			want, err := d.ReadFile(name)
			if err != nil {
				t.Fatal(err)
			}
			got, err := os.ReadFile(filepath.Join(out, "msg", name))
			if err != nil || !bytes.Equal(got, want) {
				t.Fatal("extractor changed the archive's raw resource bytes")
			}
		})
	}
	if err := runArchive(dir, "th10.exe", t.TempDir(), false, false, 4, nil); err == nil {
		t.Fatal("non-resource archive accepted")
	}
}
