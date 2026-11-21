//go:build js && th10web

package assets_test

import (
	"bytes"
	"encoding/json"
	"os"
	"path"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"th10/assets"
	"th10/gamedata"
)

// Run under tools/wasmexec/run.cjs with TH10_WASM_ASSET_DIR set to a
// wasmbuild dist directory: the real fetch path, pack checks and ANM restore.
func TestWebPacksServeEveryFile(t *testing.T) {
	dir := os.Getenv("TH10_WASM_ASSET_DIR")
	if dir == "" {
		t.Skip("TH10_WASM_ASSET_DIR not set")
	}
	var mu sync.Mutex
	var last assets.Progress
	if err := assets.Start(func(p assets.Progress) {
		mu.Lock()
		defer mu.Unlock()
		if p.Err != nil {
			t.Error(p.Err)
		}
		if p.LoadedPacks > last.LoadedPacks {
			last = p
		}
	}); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(dir, filepath.FromSlash(assets.ManifestPath)))
	if err != nil {
		t.Fatal(err)
	}
	var m assets.Manifest
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatal(err)
	}
	files := 0
	for _, p := range m.Packs {
		raw, err := os.ReadFile(filepath.Join(dir, "assets", p.File))
		if err != nil {
			t.Fatal(err)
		}
		entries, err := assets.DecodePack(raw)
		if err != nil {
			t.Fatal(err)
		}
		for _, e := range entries {
			if e.Kind != assets.PackFile {
				continue
			}
			files++
			got, err := assets.ReadFile(e.Name)
			if err != nil || !bytes.Equal(got, e.Data) {
				t.Fatalf("%s: %v", e.Name, err)
			}
			if ext := path.Ext(e.Name); ext == ".anm" || ext == ".ant" {
				if _, err := gamedata.ReadFile(e.Name); err != nil {
					t.Fatalf("restore %s: %v", e.Name, err)
				}
			}
		}
	}
	if _, err := assets.ReadFile("anm/missing.anm"); err == nil {
		t.Fatal("missing file read succeeded")
	}
	deadline := time.Now().Add(30 * time.Second)
	for {
		mu.Lock()
		done := last.LoadedPacks == len(m.Packs) && last.TotalPacks == len(m.Packs)
		mu.Unlock()
		if done {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("progress stopped at %+v", last)
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Logf("%d files in %d packs, %d bytes", files, len(m.Packs), last.TotalBytes)
}
