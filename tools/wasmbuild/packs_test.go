package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"th10/assets"
)

// Every embedded file reaches exactly one pack byte for byte, and every
// packed ANM finds all of its web textures in its own pack.
func TestBuildPacksCoversAssets(t *testing.T) {
	root, err := moduleRoot()
	if err != nil {
		t.Fatal(err)
	}
	out := t.TempDir()
	if err := os.WriteFile(filepath.Join(out, "index.html"), []byte("<html></html>"), 0o644); err != nil {
		t.Fatal(err)
	}
	infos, err := buildPacks(root, out)
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(out, "assets", "manifest.json"))
	if err != nil {
		t.Fatal(err)
	}
	var m assets.Manifest
	if err := json.Unmarshal(data, &m); err != nil || len(m.Packs) != len(infos) {
		t.Fatalf("manifest: %v, %d packs for %d infos", err, len(m.Packs), len(infos))
	}
	files := map[string][]byte{}
	textures := map[string]bool{}
	packOfTexture := map[string]map[string]bool{}
	for name, p := range m.Packs {
		raw, err := os.ReadFile(filepath.Join(out, "assets", p.File))
		if err != nil {
			t.Fatal(err)
		}
		sum := sha256.Sum256(raw)
		if len(raw) != p.Bytes || hex.EncodeToString(sum[:]) != p.SHA256 {
			t.Fatalf("%s: size or hash differs from manifest", name)
		}
		entries, err := assets.DecodePack(raw)
		if err != nil {
			t.Fatal(err)
		}
		packOfTexture[name] = map[string]bool{}
		for _, e := range entries {
			if e.Kind == assets.PackTexture {
				textures[e.Name] = true
				packOfTexture[name][e.Name] = true
				continue
			}
			if _, dup := files[e.Name]; dup || assets.PackOf(e.Name) != name {
				t.Fatalf("%s misplaced in %s", e.Name, name)
			}
			files[e.Name] = e.Data
		}
	}
	for name, data := range files {
		want, err := assets.Assets.ReadFile(name)
		if err != nil || string(want) != string(data) {
			t.Fatalf("%s differs from the embedded file", name)
		}
	}
	if len(textures) == 0 || len(packOfTexture[assets.BootPack]) == 0 {
		t.Fatal("no texture blobs packed")
	}

	handler, err := previewHandler(out)
	if err != nil {
		t.Fatal(err)
	}
	boot := m.Packs[assets.BootPack].File
	for path, want := range map[string]int{"/assets/manifest.json": 200, "/assets/" + boot: 200, "/assets/../release.json": 404,
		"/assets/x/y.pak": 404, "/assets/.pak": 404, "/assets/": 404, "/assets/foo.txt": 404} {
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, path, nil))
		if response.Code != want {
			t.Fatalf("%s: status %d, want %d", path, response.Code, want)
		}
	}
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/assets/"+boot, nil))
	if response.Header().Get("Cache-Control") != "public, max-age=31536000, immutable" {
		t.Fatal("content-hashed pack is not cacheable")
	}
}
