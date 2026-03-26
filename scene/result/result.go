package result

import (
	"github.com/hajimehoshi/ebiten/v2"
	"th10/game"
)

// Result 结算场景（显示分数、符卡获取情况等）
type Result struct{}

func New() *Result { return &Result{} }

func (r *Result) Update(g *game.Game) error {
	// TODO: 显示结算信息，按键返回 title
	return nil
}

func (r *Result) Draw(screen *ebiten.Image) {
	// TODO: 绘制结算画面
}
