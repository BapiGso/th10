package main

import (
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

type releaseOptions struct {
	Root, Out, GoRoot, GoVersion string
}

type manifest struct {
	WASMBytes  int64  `json:"wasmBytes"`
	GzipBytes  int64  `json:"gzipBytes"`
	WASMSHA256 string `json:"wasmSHA256"`
	GoVersion  string `json:"goVersion"`
}

type wasmBuilder func(context.Context, string, string) error

func buildRelease(ctx context.Context, options releaseOptions, build wasmBuilder) (*manifest, error) {
	if err := os.MkdirAll(options.Out, 0o755); err != nil {
		return nil, err
	}
	glue, err := runtimeGlue(options.GoRoot)
	if err != nil {
		return nil, err
	}
	wasm, err := os.CreateTemp(options.Out, ".th10-build-*.wasm")
	if err != nil {
		return nil, err
	}
	name := wasm.Name()
	defer func() { _ = os.Remove(name) }()
	if err := wasm.Close(); err != nil {
		return nil, err
	}
	if err := build(ctx, options.Root, name); err != nil {
		return nil, err
	}
	if err := os.Rename(name, filepath.Join(options.Out, "th10.wasm")); err != nil {
		return nil, err
	}
	for _, name := range webNames {
		data, err := webFiles.ReadFile("web/" + name)
		if err != nil {
			return nil, err
		}
		if err := atomicFile(filepath.Join(options.Out, name), func(w io.Writer) error {
			_, err := w.Write(data)
			return err
		}); err != nil {
			return nil, err
		}
	}
	if err := buildBootImages(options.Out); err != nil {
		return nil, err
	}
	if err := copyFile(glue, filepath.Join(options.Out, "wasm_exec.js")); err != nil {
		return nil, err
	}
	names := append(append([]string{}, webNames...), "wasm_exec.js", "th10.wasm")
	for _, name := range names {
		if err := gzipFile(filepath.Join(options.Out, name)); err != nil {
			return nil, err
		}
	}
	m, err := releaseManifest(options.Out, options.GoVersion)
	if err != nil {
		return nil, err
	}
	// Publish the length/hash last; fetch progress counts decoded, not gzip wire bytes.
	if err := atomicFile(filepath.Join(options.Out, "release.json"), func(w io.Writer) error {
		return json.NewEncoder(w).Encode(m)
	}); err != nil {
		return nil, err
	}
	if err := gzipFile(filepath.Join(options.Out, "release.json")); err != nil {
		return nil, err
	}
	return m, nil
}

func runtimeGlue(goroot string) (string, error) {
	for _, dir := range []string{"lib", "misc"} {
		path := filepath.Join(goroot, dir, "wasm", "wasm_exec.js")
		if info, err := os.Stat(path); err == nil && info.Mode().IsRegular() {
			return path, nil
		}
	}
	return "", fmt.Errorf("wasm_exec.js not found in GOROOT %q", goroot)
}

func releaseManifest(dir, version string) (*manifest, error) {
	f, err := os.Open(filepath.Join(dir, "th10.wasm"))
	if err != nil {
		return nil, err
	}
	defer func() { _ = f.Close() }()
	hash := sha256.New()
	bytes, err := io.Copy(hash, f)
	if err != nil {
		return nil, err
	}
	gz, err := os.Stat(filepath.Join(dir, "th10.wasm.gz"))
	if err != nil {
		return nil, err
	}
	return &manifest{WASMBytes: bytes, GzipBytes: gz.Size(), WASMSHA256: hex.EncodeToString(hash.Sum(nil)), GoVersion: version}, nil
}

func atomicFile(path string, write func(io.Writer) error) error {
	f, err := os.CreateTemp(filepath.Dir(path), "."+filepath.Base(path)+"-*")
	if err != nil {
		return err
	}
	defer func() { _ = f.Close(); _ = os.Remove(f.Name()) }()
	if err := write(f); err != nil {
		return err
	}
	if err := f.Sync(); err != nil {
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	return os.Rename(f.Name(), path)
}

func copyFile(source, destination string) error {
	f, err := os.Open(source)
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()
	return atomicFile(destination, func(w io.Writer) error {
		_, err := io.Copy(w, f)
		return err
	})
}

func gzipFile(path string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()
	return atomicFile(path+".gz", func(w io.Writer) error {
		z, err := gzip.NewWriterLevel(w, gzip.BestCompression)
		if err != nil {
			return err
		}
		_, copyErr := io.Copy(z, f)
		closeErr := z.Close()
		if copyErr != nil {
			return copyErr
		}
		return closeErr
	})
}
