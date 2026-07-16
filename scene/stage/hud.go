package stage

// 关卡 HUD：右侧状态栏 + 左下角信仰值。
//
// 整体由 ebitenui 的 widget 树排版，调样式只动这个文件里的：
//   - 字体：LoadHUDFace（Bimini Bold，分数/数字）
//   - 颜色：从 sceneui.DefaultTheme() 取，统一改主题即全局生效
//   - 行间距：RowLayoutOpts.Spacing
//   - 锚点位置：AnchorLayoutData.Padding
//
// 不再需要算硬坐标。

import (
	"fmt"
	"image/color"
	"strings"
	"th10/game"
	"th10/render"
	sceneui "th10/scene/ui"

	"github.com/ebitenui/ebitenui"
	"github.com/ebitenui/ebitenui/widget"
	"github.com/hajimehoshi/ebiten/v2"
	ebitext "github.com/hajimehoshi/ebiten/v2/text/v2"
)

type hud struct {
	state *game.GameState
	ui    *ebitenui.UI

	hiScore *widget.Text
	score   *widget.Text
	player  *widget.Text
	bomb    *widget.Text
	power   *widget.Text
	graze   *widget.Text
	point   *widget.Text
	faith   *widget.Text
}

func newHud(state *game.GameState) *hud {
	theme := sceneui.DefaultTheme()
	bodyFace := render.LoadHUDFace(16)
	hintFace := render.LoadHUDFace(14)

	h := &hud{state: state}
	h.hiScore = hudLabel(bodyFace, theme.HighColor)
	h.score = hudLabel(bodyFace, theme.TextColor)
	h.player = hudLabel(bodyFace, theme.TextColor)
	h.bomb = hudLabel(bodyFace, theme.TextColor)
	h.power = hudLabel(bodyFace, theme.TextColor)
	h.graze = hudLabel(bodyFace, theme.TextColor)
	h.point = hudLabel(bodyFace, theme.TextColor)
	h.faith = hudLabel(hintFace, theme.HighColor)

	root := widget.NewContainer(
		widget.ContainerOpts.Layout(widget.NewAnchorLayout()),
	)

	// 右侧状态栏：从 (FieldRight+16, 16) 开始，垂直堆叠。
	right := anchoredContainer(
		widget.NewRowLayout(
			widget.RowLayoutOpts.Direction(widget.DirectionVertical),
			widget.RowLayoutOpts.Spacing(4),
		),
		&widget.Insets{Left: game.FieldRight + 16, Top: 16},
	)
	for _, t := range []*widget.Text{
		h.hiScore, h.score,
		h.player, h.bomb,
		h.power, h.graze, h.point,
	} {
		right.AddChild(t)
	}
	root.AddChild(right)

	// Faith 在游戏区域左下角。
	faithBox := anchoredContainer(
		widget.NewAnchorLayout(),
		&widget.Insets{Left: game.FieldLeft + 6, Top: game.FieldBottom - 18},
	)
	faithBox.AddChild(h.faith)
	root.AddChild(faithBox)

	h.ui = &ebitenui.UI{Container: root}
	return h
}

func hudLabel(face ebitext.Face, col color.Color) *widget.Text {
	return widget.NewText(widget.TextOpts.Text("", &face, col))
}

// anchoredContainer 在 root 的 AnchorLayout 里左上角对齐 + Padding 控制偏移，
// 等价于 CSS 的 `position: absolute; left: padding.Left; top: padding.Top;`
func anchoredContainer(layout widget.Layouter, padding *widget.Insets) *widget.Container {
	return widget.NewContainer(
		widget.ContainerOpts.Layout(layout),
		widget.ContainerOpts.WidgetOpts(
			widget.WidgetOpts.LayoutData(widget.AnchorLayoutData{
				HorizontalPosition: widget.AnchorLayoutPositionStart,
				VerticalPosition:   widget.AnchorLayoutPositionStart,
				Padding:            padding,
			}),
		),
	)
}

func (h *hud) refresh() {
	st := h.state
	h.hiScore.Label = fmt.Sprintf("HiScore %012d", st.HiScore)
	h.score.Label = fmt.Sprintf("Score   %012d", st.Score)
	h.player.Label = "Player  " + strings.Repeat("*", max(st.Life, 0))
	h.bomb.Label = "Bomb    " + strings.Repeat("#", max(st.Bomb, 0))
	h.power.Label = fmt.Sprintf("Power   %d.%02d / %d.%02d",
		st.Power/100, st.Power%100, st.MaxPower/100, st.MaxPower%100)
	h.graze.Label = fmt.Sprintf("Graze   %d", st.Graze)
	h.point.Label = fmt.Sprintf("Point   %d", st.Point)
	h.faith.Label = fmt.Sprintf("Faith %d", st.Faith)
	h.ui.Update()
}

func (h *hud) draw(screen *ebiten.Image) {
	h.ui.Draw(screen)
}
