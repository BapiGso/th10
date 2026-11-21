package main

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestReleaseFilesAndGzip(t *testing.T) {
	root := t.TempDir()
	glueDir := filepath.Join(root, "go", "lib", "wasm")
	if err := os.MkdirAll(glueDir, 0o755); err != nil {
		t.Fatal(err)
	}
	glue := []byte("test runtime glue\n")
	if err := os.WriteFile(filepath.Join(glueDir, "wasm_exec.js"), glue, 0o644); err != nil {
		t.Fatal(err)
	}
	wasm := []byte("\x00asm\x01\x00\x00\x00test payload")
	dir := filepath.Join(root, "dist")
	m, err := buildRelease(context.Background(), releaseOptions{Root: root, Out: dir, GoRoot: filepath.Join(root, "go"), GoVersion: "test"},
		func(_ context.Context, buildRoot, output string) error {
			if buildRoot != root {
				t.Fatal("builder changed the module root")
			}
			return os.WriteFile(output, wasm, 0o644)
		})
	if err != nil {
		t.Fatal(err)
	}
	if m.WASMBytes != int64(len(wasm)) || m.GzipBytes == 0 || len(m.WASMSHA256) != 64 || m.GoVersion != "test" {
		t.Fatalf("bad manifest: %+v", m)
	}
	names := append(append([]string{}, webNames...), "wasm_exec.js", "th10.wasm", "release.json")
	for _, name := range names {
		plain, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			t.Fatal(err)
		}
		compressed, err := os.ReadFile(filepath.Join(dir, name+".gz"))
		if err != nil {
			t.Fatal(err)
		}
		reader, err := gzip.NewReader(bytes.NewReader(compressed))
		if err != nil {
			t.Fatal(err)
		}
		decoded, err := io.ReadAll(reader)
		_ = reader.Close()
		if err != nil || !bytes.Equal(plain, decoded) {
			t.Fatalf("%s gzip differs: %v", name, err)
		}
	}
	runtime, _ := os.ReadFile(filepath.Join(dir, "wasm_exec.js"))
	if !bytes.Equal(runtime, glue) {
		t.Fatal("release did not copy the build toolchain's runtime")
	}
	data, _ := os.ReadFile(filepath.Join(dir, "release.json"))
	var decoded manifest
	if err := json.Unmarshal(data, &decoded); err != nil || decoded != *m {
		t.Fatal("release.json differs from the reported size/hash")
	}
}

func TestPreviewMIMEAndNegotiation(t *testing.T) {
	dir := t.TempDir()
	for name, data := range map[string][]byte{"index.html": []byte("<html>test</html>"), "th10.wasm": []byte("\x00asmtest"), "player.mjs": []byte("export {};\n")} {
		if err := os.WriteFile(filepath.Join(dir, name), data, 0o644); err != nil {
			t.Fatal(err)
		}
		if err := gzipFile(filepath.Join(dir, name)); err != nil {
			t.Fatal(err)
		}
	}
	handler, err := previewHandler(dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		path, accept, mime, encoding string
	}{
		{"/", "", "text/html", ""},
		{"/th10.wasm", "", "application/wasm", ""},
		{"/th10.wasm", "br, gzip", "application/wasm", "gzip"},
		{"/th10.wasm.gz", "", "application/wasm", "gzip"},
		{"/th10.wasm", "gzip;q=0, *;q=1", "application/wasm", ""},
		{"/player.mjs", "gzip", "text/javascript", "gzip"},
	} {
		t.Run(test.path+test.accept, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodGet, test.path, nil)
			request.Header.Set("Accept-Encoding", test.accept)
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, request)
			if response.Code != 200 || !strings.HasPrefix(response.Header().Get("Content-Type"), test.mime) || response.Header().Get("Content-Encoding") != test.encoding || response.Header().Get("Vary") != "Accept-Encoding" {
				t.Fatalf("status=%d headers=%v", response.Code, response.Header())
			}
			if test.encoding == "gzip" {
				reader, err := gzip.NewReader(response.Body)
				if err != nil {
					t.Fatal(err)
				}
				if _, err := io.ReadAll(reader); err != nil {
					t.Fatal(err)
				}
				_ = reader.Close()
			}
		})
	}
	for _, path := range []string{"/../go.mod", "/go.mod", "/web/player.mjs", "/th10.wasm/", "/foo.gz"} {
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, path, nil))
		if response.Code != 404 {
			t.Fatalf("exposed non-release path %q", path)
		}
	}
	request := httptest.NewRequest(http.MethodGet, "/th10.wasm", nil)
	request.Header.Set("Accept-Encoding", "gzip")
	request.Header.Set("Range", "bytes=0-3")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != 206 || response.Header().Get("Content-Encoding") != "" || response.Body.String() != "\x00asm" {
		t.Fatal("identity byte range was interpreted as gzip offsets")
	}
	response = httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodHead, "/th10.wasm.gz", nil))
	if response.Code != 200 || response.Body.Len() != 0 || response.Header().Get("Content-Encoding") != "gzip" || response.Header().Get("Content-Length") == "" {
		t.Fatal("HEAD lost gzip metadata")
	}
}

func TestAcceptsGzip(t *testing.T) {
	for header, want := range map[string]bool{"": false, "gzip": true, "GZip ; q=0.5": true, "br": false, "*": true, "*;q=0": false, "gzip;q=0,*": false, "gzip;q=bad": false, "gzip;q=NaN": false, "gzip;q=2": false} {
		if got := acceptsGzip(header); got != want {
			t.Errorf("%q: got %v, want %v", header, got, want)
		}
	}
}
