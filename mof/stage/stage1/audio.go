package title

import (
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/audio"
	"main/assets"
)

type bgm struct {
	img          *ebiten.Image
	options      *ebiten.DrawImageOptions
	audioContext *audio.Context
}

func (b *bgm) init() {
	file, err := assets.Assets.ReadFile("wav/bgm/01. 封印されし神々.wav")
	if err != nil {
		panic(err)
	}
	b.audioContext = audio.NewContext(44100)
	b.audioContext.NewPlayerFromBytes(file).Play()
	b.img = ebiten.NewImage(640, 480)
	b.options = &ebiten.DrawImageOptions{}
}

func (b *bgm) Update() error {
	return nil
}

func (b *bgm) Render() (*ebiten.Image, *ebiten.DrawImageOptions) {
	return b.img, b.options
}
