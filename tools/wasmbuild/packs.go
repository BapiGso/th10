package main

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"slices"
	"sort"
	"strings"

	"th10/anm"
	"th10/assets"
)

type packInfo struct {
	Name      string
	File      string
	Bytes     int
	GzipBytes int64 // 0 when gzip saves under 5% and no .gz copy is published
}

// buildPacks writes the th10web release's resources: every embedded runtime
// file, byte for byte, grouped by assets.PackOf, plus the web texture blobs
// each ANM references. File names carry a content hash so browsers may cache
// packs indefinitely; manifest.json is the only mutable name.
//
// Without story (the th10nostory release) dialogue and ending resources are
// left out, and the text of the stage .msg scripts that remain is blanked.
func buildPacks(root, out string, story bool) ([]packInfo, error) {
	webPack, err := os.ReadFile(filepath.Join(root, "assets", "textures", "web.tpk"))
	if err != nil {
		return nil, err
	}
	type pack struct {
		entries []assets.PackEntry
		seen    map[string]bool
	}
	packs := map[string]*pack{}
	add := func(name string, e assets.PackEntry) {
		p := packs[name]
		if p == nil {
			p = &pack{seen: map[string]bool{}}
			packs[name] = p
		}
		key := fmt.Sprintf("%d:%s", e.Kind, e.Name)
		if !p.seen[key] {
			p.seen[key] = true
			p.entries = append(p.entries, e)
		}
	}
	err = fs.WalkDir(assets.Assets, ".", func(name string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		if !story && storyOnly(name) {
			return nil
		}
		data, err := assets.Assets.ReadFile(name)
		if err != nil {
			return err
		}
		if !story && strings.HasPrefix(name, "msg/") {
			if data, err = blankMSGText(data); err != nil {
				return fmt.Errorf("%s: %w", name, err)
			}
		}
		packName := assets.PackOf(name)
		add(packName, assets.PackEntry{Kind: assets.PackFile, Name: name, Data: data})
		if ext := path.Ext(name); (ext == ".anm" || ext == ".ant") && anm.IsPacked(data) {
			// Restore asks for exactly the texture IDs this ANM references.
			_, err := anm.Restore(data, func(id string) ([]byte, error) {
				blob, err := anm.TextureBlob(webPack, id)
				if err == nil {
					add(packName, assets.PackEntry{Kind: assets.PackTexture, Name: id, Data: blob})
				}
				return blob, err
			})
			if err != nil {
				return fmt.Errorf("%s: %w", name, err)
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	dir := filepath.Join(out, "assets")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	stale, err := filepath.Glob(filepath.Join(dir, "*.pak*"))
	if err != nil {
		return nil, err
	}
	for _, name := range stale {
		if err := os.Remove(name); err != nil {
			return nil, err
		}
	}
	names := make([]string, 0, len(packs))
	for name := range packs {
		names = append(names, name)
	}
	sort.Strings(names)
	manifest := assets.Manifest{Packs: map[string]assets.ManifestPack{}}
	var infos []packInfo
	for _, name := range names {
		entries := packs[name].entries
		sort.Slice(entries, func(i, j int) bool {
			if entries[i].Kind != entries[j].Kind {
				return entries[i].Kind < entries[j].Kind
			}
			return entries[i].Name < entries[j].Name
		})
		data, err := assets.EncodePack(entries)
		if err != nil {
			return nil, err
		}
		sum := sha256.Sum256(data)
		hash := hex.EncodeToString(sum[:])
		file := name + "." + hash[:16] + ".pak"
		target := filepath.Join(dir, file)
		if err := atomicFile(target, func(w io.Writer) error { _, err := w.Write(data); return err }); err != nil {
			return nil, err
		}
		if err := gzipFile(target); err != nil {
			return nil, err
		}
		gz, err := os.Stat(target + ".gz")
		if err != nil {
			return nil, err
		}
		gzipBytes := gz.Size()
		if gzipBytes > int64(len(data))*95/100 { // Ogg packs: serve plain bytes instead
			if err := os.Remove(target + ".gz"); err != nil {
				return nil, err
			}
			gzipBytes = 0
		}
		manifest.Packs[name] = assets.ManifestPack{File: file, Bytes: len(data), SHA256: hash}
		infos = append(infos, packInfo{Name: name, File: file, Bytes: len(data), GzipBytes: gzipBytes})
	}
	for _, name := range assets.PackOrder() {
		if _, ok := manifest.Packs[name]; !ok && !strings.HasPrefix(name, "bgm-") && (story || name != "ending") {
			return nil, fmt.Errorf("prefetch order names pack %q, but no runtime file maps to it", name)
		}
	}
	if err := atomicFile(filepath.Join(dir, "manifest.json"), func(w io.Writer) error {
		e := json.NewEncoder(w)
		e.SetIndent("", "  ")
		return e.Encode(manifest)
	}); err != nil {
		return nil, err
	}
	return infos, gzipFile(filepath.Join(dir, "manifest.json"))
}

// storyOnly reports the resources a th10nostory build never reads: ending and
// staff-roll ANM/MSG and the localized .mst texts. Portrait sheets stay: pl0N
// is also the player and stgenm0N the spell cut-ins.
func storyOnly(name string) bool {
	base := path.Base(name)
	stem := strings.TrimSuffix(base, path.Ext(base))
	switch {
	case path.Ext(name) == ".mst":
		return true
	case strings.HasPrefix(name, "msg/") || strings.HasPrefix(name, "anm/"):
		return stem == "staff" || len(stem) == 3 && stem[0] == 'e' && stem[1] >= '0' && stem[1] <= '9'
	}
	return false
}

// blankMSGText rewrites a stage .msg so its text ops (14, 15, 16) decrypt to
// NULs: the same layout and offsets, with no dialogue in the bytes.
func blankMSGText(data []byte) ([]byte, error) {
	if len(data) < 4 {
		return nil, errors.New("short msg")
	}
	out := slices.Clone(data)
	count := int(binary.LittleEndian.Uint32(out))
	if count <= 0 || 4+count*8 > len(out) {
		return nil, errors.New("bad msg header")
	}
	for i := range count {
		p := int(binary.LittleEndian.Uint32(out[4+i*8:]))
		end := len(out)
		if i+1 < count {
			end = int(binary.LittleEndian.Uint32(out[4+(i+1)*8:]))
		}
		if p > end || end > len(out) {
			return nil, errors.New("bad msg offsets")
		}
		for p+4 <= end {
			op, n := out[p+2], int(out[p+3])
			args := out[p+4 : min(p+4+n, end)]
			if op >= 14 && op <= 16 {
				key, step := byte(0x77), byte(7) // thtk util_xor(0x77, 7, 0x10)
				for j := range args {
					args[j] = key
					key += step
					step += 0x10
				}
			}
			p += 4 + n
		}
	}
	return out, nil
}
