//go:build !th10web

package assets

import "th10/anm"

// ReadFile returns a private copy of an embedded runtime file.
func ReadFile(name string) ([]byte, error) { return Assets.ReadFile(name) }

// ReadTexture returns only the selected blob, without decoding other images.
func ReadTexture(id string) ([]byte, error) { return anm.TextureBlob(texturePack, id) }

// Start prepares resources before the game is created. Embedded builds
// have nothing to fetch.
func Start(func(Progress)) error { return nil }
