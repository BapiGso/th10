package main

import (
	"fmt"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

func previewHandler(dir string) (http.Handler, error) {
	dir, err := filepath.Abs(dir)
	if err != nil {
		return nil, err
	}
	if _, err := os.Stat(filepath.Join(dir, "index.html")); err != nil {
		return nil, fmt.Errorf("release is missing index.html: %w", err)
	}
	allowed := map[string]bool{"wasm_exec.js": true, "th10.wasm": true, "release.json": true}
	for _, name := range webNames {
		allowed[name] = true
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			w.Header().Set("Allow", "GET, HEAD")
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		name := strings.TrimPrefix(r.URL.Path, "/")
		if name == "" {
			name = "index.html"
		}
		plain := strings.TrimSuffix(name, ".gz")
		pack := isPackName(plain)
		// Only release artifacts are public; never expose directory listings or the module.
		if !pack && (!allowed[plain] || strings.ContainsAny(name, "/\\")) {
			http.NotFound(w, r)
			return
		}
		encoded := name != plain
		if !encoded && r.Header.Get("Range") == "" && acceptsGzip(r.Header.Get("Accept-Encoding")) {
			if info, err := os.Stat(filepath.Join(dir, name+".gz")); err == nil && info.Mode().IsRegular() {
				name += ".gz"
				encoded = true
			}
		}
		f, err := os.Open(filepath.Join(dir, name))
		if err != nil {
			http.NotFound(w, r)
			return
		}
		defer func() { _ = f.Close() }()
		info, err := f.Stat()
		if err != nil || !info.Mode().IsRegular() {
			http.NotFound(w, r)
			return
		}
		contentType := mime.TypeByExtension(filepath.Ext(plain))
		switch filepath.Ext(plain) {
		case ".wasm":
			contentType = "application/wasm"
		case ".js", ".mjs":
			contentType = "text/javascript; charset=utf-8"
		}
		w.Header().Set("Content-Type", contentType)
		w.Header().Set("X-Content-Type-Options", "nosniff")
		if strings.HasSuffix(plain, ".pak") {
			w.Header().Set("Cache-Control", "public, max-age=31536000, immutable") // content-hashed name
		} else {
			w.Header().Set("Cache-Control", "no-cache")
		}
		w.Header().Set("Vary", "Accept-Encoding")
		if encoded {
			w.Header().Set("Content-Encoding", "gzip")
			w.Header().Set("Content-Length", strconv.FormatInt(info.Size(), 10))
		}
		http.ServeContent(w, r, plain, info.ModTime(), f)
	}), nil
}

// isPackName accepts assets/manifest.json and assets/<pack>.<hash>.pak only.
func isPackName(name string) bool {
	rest, ok := strings.CutPrefix(name, "assets/")
	if !ok || rest == "" || strings.ContainsAny(rest, "/\\") || strings.HasPrefix(rest, ".") || strings.Contains(rest, "..") {
		return false
	}
	return rest == "manifest.json" || strings.HasSuffix(rest, ".pak")
}

func acceptsGzip(header string) bool {
	gzipQ, wildcardQ := -1.0, 0.0
	for _, part := range strings.Split(header, ",") {
		fields := strings.Split(part, ";")
		q := 1.0
		for _, option := range fields[1:] {
			name, value, found := strings.Cut(strings.TrimSpace(option), "=")
			if found && strings.EqualFold(name, "q") {
				parsed, err := strconv.ParseFloat(value, 64)
				if err != nil || !(parsed >= 0 && parsed <= 1) {
					q = 0
				} else {
					q = parsed
				}
			}
		}
		switch strings.ToLower(strings.TrimSpace(fields[0])) {
		case "gzip":
			gzipQ = q
		case "*":
			wildcardQ = q
		}
	}
	if gzipQ >= 0 {
		return gzipQ > 0
	}
	return wildcardQ > 0
}
