package title

import (
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"main/assets"
)

type sig struct {
	img *ebiten.Image
}

func (s sig) init() {
	s.img, _, _ = ebitenutil.NewImageFromFileSystem(assets.Assets, "anm/title/title00s.png")
}

func (s sig) Update() error {
	return nil
}

func (s sig) Render() (*ebiten.Image, *ebiten.DrawImageOptions) {
	return s.img, &ebiten.DrawImageOptions{}
}
