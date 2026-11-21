//go:build !th10web

package assets

import (
	"embed"

	"th10/anm"
)

// GUI/MSG text uses the existing DFYuGaSo fonts with th10c.dat's traditional
// Chinese text. Menus keep their current font policy; numeric GUI sprites
// come from front.anm/ascii.anm rather than a font.
//
//go:embed anm/*.anm anm/*.ant anm/texture_index.json ecl/*.ecl ecl/*.ect msg/*.msg msg/*.mst sht/*.sht std/*.std "font/Bimini Bold.ttf" font/DFYuGaSo-W3.ttc font/DFYuGaSo-W7.ttc misc/demo*.rpy misc/musiccmt.txt misc/musiccmtt.txt wav/se/*.wav wav/bgm/*.ogg wav/bgm/thbgm.fmt
var Assets embed.FS

// ReadFile returns a private copy of an embedded runtime file.
func ReadFile(name string) ([]byte, error) { return Assets.ReadFile(name) }

// ReadTexture returns only the selected blob, without decoding other images.
func ReadTexture(id string) ([]byte, error) { return anm.TextureBlob(texturePack, id) }

// Start prepares resources before the game is created. Embedded builds
// have nothing to fetch.
func Start(func(Progress)) error { return nil }
