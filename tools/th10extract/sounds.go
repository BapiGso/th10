// Audio products are regenerated from original PCM, never from lossy files.
package main

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"th10/datfs"
)

type seProduct struct {
	Name       string `json:"name"`
	Original   int    `json:"original_bytes"`
	Stored     int    `json:"stored_bytes"`
	Compressed bool   `json:"compressed"`
}

func packSounds(dir, out string) error {
	if dir == "" {
		var err error
		if dir, err = datfs.Locate(); err != nil {
			return err
		}
	}
	d, err := datfs.Open(filepath.Join(dir, "th10.dat"))
	if err != nil {
		return err
	}
	defer func() { _ = d.Close() }()
	files, err := os.ReadDir(filepath.Join(out, "wav", "se"))
	if err != nil {
		return err
	}
	var products []seProduct
	for _, file := range files {
		if file.IsDir() || filepath.Ext(file.Name()) != ".wav" {
			continue
		}
		raw, err := d.ReadFile(file.Name())
		if err != nil {
			return err
		}
		compressed, err := deflateBytes(raw)
		if err != nil {
			return err
		}
		blob := make([]byte, 8, 8+len(compressed))
		copy(blob, "ZSE1")
		binary.LittleEndian.PutUint32(blob[4:], uint32(len(raw)))
		blob = append(blob, compressed...)
		p := seProduct{Name: file.Name(), Original: len(raw), Stored: len(blob), Compressed: true}
		if len(blob) >= len(raw) {
			blob, p.Stored, p.Compressed = raw, len(raw), false
		}
		if err := write(filepath.Join(out, "wav", "se", p.Name), blob); err != nil {
			return err
		}
		products = append(products, p)
	}
	data, err := json.MarshalIndent(products, "", "  ")
	if err != nil {
		return err
	}
	return write(filepath.Join(filepath.Dir(filepath.Clean(out)), "docs", "research", "se_manifest.json"), append(data, '\n'))
}

func encodeOneBGM(dir, archive, out string, quality float64, index int) error {
	if archive != "th10.dat" && archive != "th10c.dat" {
		return fmt.Errorf("unsupported resource archive %q", archive)
	}
	if dir == "" {
		var err error
		if dir, err = datfs.Locate(); err != nil {
			return err
		}
	}
	d, err := datfs.Open(filepath.Join(dir, archive))
	if err != nil {
		return err
	}
	defer func() { _ = d.Close() }()
	return encodeBGMSelection(d, dir, out, quality, index)
}
