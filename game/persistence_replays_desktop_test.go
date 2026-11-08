//go:build !js && !wasm

package game

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
	"th10/replay"
)

func TestDesktopReplayStoreOriginalFilenameAndOverwrite(t *testing.T) {
	root := t.TempDir()
	s := &fileReplayStore{dir: filepath.Join(root, "replay")}
	g := &Game{Replays: s}
	r := storedReplay()
	if err := g.SaveReplay(5, r); err != nil {
		t.Fatal(err)
	}
	want, err := replay.Encode(r)
	if err != nil {
		t.Fatal(err)
	}
	file := filepath.Join(root, "replay", "th10_05.rpy")
	got, err := os.ReadFile(file)
	if err != nil || !bytes.Equal(got, want) {
		t.Fatal("wrong path or file format")
	}
	r.Global.User = "NEXT"
	if err := g.SaveReplay(5, r); err != nil {
		t.Fatal("overwrite failed:", err)
	}
	roundtrip, err := (&Game{Replays: &fileReplayStore{dir: s.dir}}).LoadReplay(5)
	if err != nil || roundtrip.Global.User != "NEXT" {
		t.Fatal("fresh store did not read overwritten slot")
	}
	files, err := os.ReadDir(s.dir)
	if err != nil || len(files) != 1 || files[0].Name() != "th10_05.rpy" {
		t.Fatal("temporary files leaked")
	}
}

func TestDesktopReplayStoreRespectsDataDirectory(t *testing.T) {
	root := t.TempDir()
	old := resolvedDataDir
	resolvedDataDir = ""
	t.Cleanup(func() { resolvedDataDir = old })
	t.Setenv("TH10_DATA_DIR", root)
	g := &Game{}
	if err := g.SaveReplay(1, storedReplay()); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(root, "replay", "th10_01.rpy")); err != nil {
		t.Fatal(err)
	}
}

func TestDesktopReplayStoreReadFailureDoesNotTruncate(t *testing.T) {
	root := t.TempDir()
	s := &fileReplayStore{dir: root}
	if err := os.Mkdir(filepath.Join(root, "th10_01.rpy"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := s.Write(1, []byte("data")); err == nil {
		t.Fatal("overwrote a directory")
	}
	files, err := os.ReadDir(root)
	if err != nil || len(files) != 1 || !files[0].IsDir() {
		t.Fatal("failure leaked a temporary file")
	}
}
