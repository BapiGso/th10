package render

// text/v2 font loading used only by the font subset tests. The runtime no
// longer draws through text/v2 (go-text/typesetting is kept out of the WASM
// build); the tests still check that subset fonts keep text/v2 metrics.

import (
	"bytes"
	"fmt"
	"sync"
	"th10/assets"

	ebitext "github.com/hajimehoshi/ebiten/v2/text/v2"
)

var (
	ttcSourceCache sync.Map // path -> *ebitext.GoTextFaceSource
	textFaceCache  sync.Map // "path@size" -> ebitext.Face
)

// LoadTextFace reads an embedded TTC/TTF and returns a text/v2 Face.
func LoadTextFace(path string, size float64) ebitext.Face {
	key := fmt.Sprintf("embed:%s@%.2f", path, size)
	if cached, ok := textFaceCache.Load(key); ok {
		return cached.(ebitext.Face)
	}
	src := loadEmbedSource(path)
	if src == nil {
		return nil
	}
	face := &ebitext.GoTextFace{Source: src, Size: size}
	if actual, loaded := textFaceCache.LoadOrStore(key, face); loaded {
		return actual.(ebitext.Face)
	}
	return face
}

func loadEmbedSource(path string) *ebitext.GoTextFaceSource {
	cacheKey := "embed:" + path
	if cached, ok := ttcSourceCache.Load(cacheKey); ok {
		return cached.(*ebitext.GoTextFaceSource)
	}
	data, err := assets.Assets.ReadFile(path)
	if err != nil {
		return nil
	}
	src := parseFontBytes(data)
	if src == nil {
		return nil
	}
	ttcSourceCache.Store(cacheKey, src)
	return src
}

func parseFontBytes(data []byte) *ebitext.GoTextFaceSource {
	// Prefer the first face of a collection, then a single-face font.
	sources, err := ebitext.NewGoTextFaceSourcesFromCollection(bytes.NewReader(data))
	if err == nil && len(sources) > 0 {
		return sources[0]
	}
	single, err := ebitext.NewGoTextFaceSource(bytes.NewReader(data))
	if err != nil {
		return nil
	}
	return single
}
