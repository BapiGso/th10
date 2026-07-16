package sprite

import (
	"bytes"
	"image"
	_ "image/png"

	"th10/anm"
	"th10/assets"

	"github.com/hajimehoshi/ebiten/v2"
)

// AnmSheet is a loaded .anm file plus its textures, resolving ANM scripts and
// sprite ids to drawable sub-images. It is the runtime side of the binary anm
// loader: the parser yields UV rects + script timelines, and this builds the
// ebiten sub-images from the already-extracted PNG textures.
type AnmSheet struct {
	file     *anm.File
	textures []*ebiten.Image       // per entry (texture)
	subCache map[int]*ebiten.Image // sprite id -> sub-image
}

// LoadAnmSheet parses an embedded .anm (e.g. "anm/enemy.anm") and loads each
// entry's texture from the matching extracted PNG ("anm/<entry name>").
func LoadAnmSheet(anmPath string) *AnmSheet {
	sh := &AnmSheet{subCache: map[int]*ebiten.Image{}}
	data, err := assets.Assets.ReadFile(anmPath)
	if err != nil {
		return sh
	}
	file, err := anm.Load(data)
	if err != nil {
		return sh
	}
	sh.file = file
	sh.textures = make([]*ebiten.Image, len(file.Entries))
	for i, e := range file.Entries {
		sh.textures[i] = loadTexturePNG("anm/" + e.Name)
	}
	return sh
}

func loadTexturePNG(path string) *ebiten.Image {
	data, err := assets.Assets.ReadFile(path)
	if err != nil {
		return nil
	}
	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil
	}
	return ebiten.NewImageFromImage(img)
}

// Sprite returns the sub-image for a sprite id (cached), or nil.
func (a *AnmSheet) Sprite(id int) *ebiten.Image {
	if a == nil || a.file == nil {
		return nil
	}
	if img, ok := a.subCache[id]; ok {
		return img
	}
	sp, ok := a.file.Sprites[id]
	if !ok || sp.Entry < 0 || sp.Entry >= len(a.textures) || a.textures[sp.Entry] == nil {
		return nil
	}
	rect := image.Rect(int(sp.X), int(sp.Y), int(sp.X+sp.W), int(sp.Y+sp.H))
	sub := a.textures[sp.Entry].SubImage(rect).(*ebiten.Image)
	a.subCache[id] = sub
	return sub
}

// ScriptSprite resolves an ANM script at the given age (frames) to its current
// sprite sub-image, or nil if the script shows none.
func (a *AnmSheet) ScriptSprite(scriptID, age int) *ebiten.Image {
	if a == nil || a.file == nil {
		return nil
	}
	sc := a.file.Scripts[scriptID]
	if sc == nil {
		return nil
	}
	id := sc.SpriteAt(age)
	if id < 0 {
		return nil
	}
	return a.Sprite(id)
}
