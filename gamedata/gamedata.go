// Package gamedata is the single access point for the original game's
// resources. Logical bytes are unchanged; ANM pixels are restored lazily
// from the build-selected texture pack. Nothing is read from a TH10 install
// at runtime.
//
// Paths are assets-relative ("ecl/stage01.ecl", "anm/enemy.anm",
// "wav/se/se_gun00.wav"); datfs.EmbedPath maps a th10.dat name to its path.
package gamedata

import (
	"bytes"
	"encoding/json"
	"image"
	_ "image/png" // decoded textures and self-made pictures
	"path"
	"strings"
	"sync"

	"th10/anm"
	"th10/assets"
	"th10/datfs"
)

// ReadFile reads an embedded resource.
func ReadFile(name string) ([]byte, error) {
	if ext := path.Ext(name); ext == ".anm" || ext == ".ant" {
		data, err := readANM(name)
		return bytes.Clone(data), err // preserve embed.FS.ReadFile's private-copy contract
	}
	data, err := assets.ReadFile(name)
	if err == nil && path.Ext(name) == ".wav" && len(data) >= 4 && string(data[:4]) == "ZSE1" {
		return compactWAV(data)
	}
	return data, err
}

// ANM parses an embedded .anm by its th10.dat name ("enemy.anm").
func ANM(name string) (*anm.File, error) {
	data, err := readANM(datfs.EmbedPath(name))
	if err != nil {
		return nil, err
	}
	return anm.Load(bytes.Clone(data))
}

// Image decodes the picture at p. "anm/<texture name>" (for example
// "anm/background/stg1bg.png") is the decoded copy of that texture if one is
// embedded, else it is assembled from the THTX blocks of an embedded .anm.
func Image(p string) (image.Image, error) {
	data, err := ReadFile(p)
	if err == nil {
		img, _, err := image.Decode(bytes.NewReader(data))
		return img, err
	}
	if name, ok := strings.CutPrefix(p, "anm/"); ok {
		if file, ok := textureIndex()[name]; ok {
			f, ferr := ANM(file)
			if ferr != nil {
				return nil, ferr
			}
			if img := f.NamedImage(name); img != nil {
				return img, nil
			}
		}
	}
	return nil, err
}

type loadedANM struct {
	once sync.Once
	data []byte
	err  error
}

var anmFiles = &sync.Map{} // path -> *loadedANM; no ANMs decoded at startup

func readANM(name string) ([]byte, error) {
	value, _ := anmFiles.LoadOrStore(name, &loadedANM{})
	file := value.(*loadedANM)
	file.once.Do(func() {
		file.data, file.err = assets.ReadFile(name)
		if file.err == nil && anm.IsPacked(file.data) {
			file.data, file.err = anm.Restore(file.data, assets.ReadTexture)
		}
	})
	return file.data, file.err
}

var (
	texOnce  sync.Once
	texIndex map[string]string // texture name -> embedded .anm holding it
)

func textureIndex() map[string]string {
	texOnce.Do(func() {
		texIndex = map[string]string{}
		if data, err := assets.ReadFile("anm/texture_index.json"); err == nil {
			_ = json.Unmarshal(data, &texIndex)
		}
	})
	return texIndex
}
