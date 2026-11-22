//go:build !js && !wasm

package game

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestScoreDesktopNativeBytesAndAtomicFailure(t *testing.T) {
	dir := t.TempDir()
	s := binaryScoreStore{store: dirStore(dir)}
	raw, err := EncodeScoreFile(DefaultScoreFile(0))
	if err != nil {
		t.Fatal(err)
	}
	if err := s.Write(raw); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "score.dat")
	onDisk, err := os.ReadFile(path)
	if err != nil || !bytes.Equal(onDisk, raw) {
		t.Fatal("desktop did not store the native bytes")
	}
	fresh := binaryScoreStore{store: dirStore(dir)}
	loaded, err := fresh.Read()
	if err != nil || !bytes.Equal(loaded, raw) {
		t.Fatal("new desktop backend did not read the same bytes")
	}
	if err := s.Write(make([]byte, maxScoreBytes+1)); err == nil {
		t.Fatal("oversized desktop score accepted")
	}
	onDisk, err = os.ReadFile(path)
	if err != nil || !bytes.Equal(onDisk, raw) {
		t.Fatal("failed write changed the existing score")
	}
	blockedDir := t.TempDir()
	blocked := binaryScoreStore{store: dirStore(blockedDir)}
	if err := os.Mkdir(filepath.Join(blockedDir, "score.dat"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := blocked.Write(raw); err == nil {
		t.Fatal("write replaced a directory")
	}
	entries, err := os.ReadDir(blockedDir)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), ".th10-") {
			t.Fatal("failed atomic save left a temporary file")
		}
	}
}

func TestScoreDesktopUsesUserDataDirectory(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("TH10_DATA_DIR", dir)
	prior := resolvedDataDir
	resolvedDataDir = ""
	t.Cleanup(func() { resolvedDataDir = prior })
	g := &Game{}
	g.LoadProgress()
	if err := g.SaveProgress(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, "score.dat")); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, "save.json")); !os.IsNotExist(err) {
		t.Fatal("new progress still writes JSON")
	}
}
