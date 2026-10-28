// Package gamedata is the single access point for the original game's
// resources. They are embedded (package assets) as unmodified copies of
// th10.dat files plus BGM encoded from thbgm.dat, added a few at a time with
// tools/th10extract as features need them. Nothing is read from a TH10
// install at runtime.
//
// Paths are assets-relative ("ecl/stage01.ecl", "anm/enemy.anm",
// "wav/se/se_gun00.wav"); datfs.EmbedPath maps a th10.dat name to its path.
package gamedata

import (
	"bytes"
	"image"
	_ "image/png" // decoded textures and self-made pictures
	"io/fs"
	"path"
	"strings"
	"sync"

	"th10/anm"
	"th10/assets"
	"th10/datfs"
)

// ReadFile reads an embedded resource.
func ReadFile(name string) ([]byte, error) {
	return assets.Assets.ReadFile(name)
}

// ANM parses an embedded .anm by its th10.dat name ("enemy.anm").
func ANM(name string) (*anm.File, error) {
	data, err := ReadFile(datfs.EmbedPath(name))
	if err != nil {
		return nil, err
	}
	return anm.Load(data)
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

var (
	texOnce  sync.Once
	texIndex map[string]string // texture name -> embedded .anm holding it
)

func textureIndex() map[string]string {
	texOnce.Do(func() {
		texIndex = map[string]string{}
		files, _ := fs.Glob(assets.Assets, "anm/*.anm")
		for _, p := range files {
			name := path.Base(p)
			f, err := ANM(name)
			if err != nil {
				continue
			}
			for _, e := range f.Entries {
				if _, seen := texIndex[e.Name]; !seen && e.Texture != nil {
					texIndex[e.Name] = name
				}
			}
		}
	})
	return texIndex
}
