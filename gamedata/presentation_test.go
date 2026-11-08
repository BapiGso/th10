package gamedata

import (
	"bytes"
	"errors"
	"io/fs"
	"path"
	"path/filepath"
	"strings"
	"testing"

	"th10/datfs"
)

// Keep the patch data raw and independently verified, rather than replacing
// the original .msg/.ecl that existing logic/golden tests use.
func TestPresentationAssetsMatchTH10C(t *testing.T) {
	dir, err := datfs.Locate()
	if err != nil {
		t.Skip(err)
	}
	d, err := datfs.Open(filepath.Join(dir, "th10c.dat"))
	if errors.Is(err, fs.ErrNotExist) {
		t.Skip(err)
	}
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = d.Close() }()
	checked := 0
	for _, name := range d.Names() {
		folder := "misc"
		switch path.Ext(name) {
		case ".ant":
			folder = "anm"
		case ".mst":
			folder = "msg"
		case ".ect":
			folder = "ecl"
		default:
			if name != "musiccmtt.txt" {
				continue
			}
		}
		got, err := ReadFile(folder + "/" + name)
		if errors.Is(err, fs.ErrNotExist) {
			continue
		}
		if err != nil {
			t.Fatal(err)
		}
		want, err := d.ReadFile(name)
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(got, want) {
			t.Errorf("%s differs from th10c.dat", name)
		}
		checked++
	}
	if checked < 30 { // 8 ANTs, 14 MSTs, 7 ECTs and musiccmtt.txt
		t.Errorf("checked %d patch assets, want at least 30", checked)
	}
}

func TestPresentationANMUsesTraditionalTexture(t *testing.T) {
	f, err := PresentationANM("front.anm")
	if err != nil {
		t.Fatal(err)
	}
	data, err := ReadFile("anm/front.ant")
	if err != nil {
		t.Fatal(err)
	}
	if len(data) == 0 || len(f.Scripts) == 0 || !strings.HasPrefix(f.Entries[0].Name, "front/") {
		t.Fatal("traditional front.anm did not retain its ANM metadata")
	}
}
