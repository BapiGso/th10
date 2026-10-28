package sprite

import (
	"image"

	"th10/anm"
	"th10/gamedata"

	"github.com/hajimehoshi/ebiten/v2"
)

// AnmSheet is a loaded .anm file plus its textures, resolving ANM scripts and
// sprite ids to drawable sub-images. It is the runtime side of the binary anm
// loader: the parser yields UV rects, script timelines and THTX pixels, and
// this builds the ebiten sub-images from them.
type AnmSheet struct {
	file     *anm.File
	textures []*ebiten.Image       // per entry (texture)
	subCache map[int]*ebiten.Image // sprite id -> sub-image
}

// LoadAnmSheet parses an archived .anm (e.g. "anm/enemy.anm") and builds each
// entry's texture from its THTX pixels. Entries sharing a texture name are
// tiles of one picture and share one assembled image.
func LoadAnmSheet(anmPath string) *AnmSheet {
	sh := &AnmSheet{subCache: map[int]*ebiten.Image{}}
	data, err := gamedata.ReadFile(anmPath)
	if err != nil {
		return sh
	}
	file, err := anm.Load(data)
	if err != nil {
		return sh
	}
	return NewAnmSheet(file)
}

// NewAnmSheet builds the textures of a parsed .anm.
func NewAnmSheet(file *anm.File) *AnmSheet {
	sh := &AnmSheet{subCache: map[int]*ebiten.Image{}, file: file}
	sh.textures = make([]*ebiten.Image, len(file.Entries))
	byName := map[string]*ebiten.Image{}
	for i, e := range file.Entries {
		img, ok := byName[e.Name]
		if !ok {
			if pix := file.NamedImage(e.Name); pix != nil {
				img = ebiten.NewImageFromImage(pix)
			}
			byName[e.Name] = img
		}
		sh.textures[i] = img
	}
	return sh
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
