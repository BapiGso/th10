package ending

import (
	"github.com/hajimehoshi/ebiten/v2"
	"th10/game"
)

// Ending 结局场景
type Ending struct{}

func New() *Ending { return &Ending{} }

func (e *Ending) Update(g *game.Game) error {
	// TODO: 播放结局立绘 + 文字
	return nil
}

func (e *Ending) Draw(screen *ebiten.Image) {
	// TODO: 绘制结局画面
}
