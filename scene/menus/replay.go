package menus

import (
	"image/color"
	"th10/game"
	"th10/input"
	sceneui "th10/scene/ui"

	"github.com/ebitenui/ebitenui"
	"github.com/ebitenui/ebitenui/widget"
	"github.com/hajimehoshi/ebiten/v2"
)

type Replay struct {
	ui   *ebitenui.UI
	back game.Scene
}

func NewReplay(back game.Scene) *Replay {
	s := &Replay{back: back}
	s.build()
	return s
}

func (s *Replay) build() {
	theme := sceneui.DefaultTheme()
	root := sceneui.NewRoot(theme, widget.NewAnchorLayout())

	panel := widget.NewContainer(
		widget.ContainerOpts.BackgroundImage(theme.Panel),
		widget.ContainerOpts.Layout(widget.NewRowLayout(
			widget.RowLayoutOpts.Direction(widget.DirectionVertical),
			widget.RowLayoutOpts.Spacing(14),
			widget.RowLayoutOpts.Padding(widget.NewInsetsSimple(36)),
		)),
		widget.ContainerOpts.WidgetOpts(
			widget.WidgetOpts.LayoutData(widget.AnchorLayoutData{
				HorizontalPosition: widget.AnchorLayoutPositionCenter,
				VerticalPosition:   widget.AnchorLayoutPositionCenter,
			}),
			widget.WidgetOpts.MinSize(460, 220),
		),
	)
	root.AddChild(panel)

	panel.AddChild(sceneui.Label("Replay", theme.TitleFace, theme.HighColor))
	panel.AddChild(sceneui.Label("当前还没有可用的回放文件。", theme.ItemFace, theme.TextColor))
	panel.AddChild(sceneui.Label("本轮改动先补完了存档、流程和可玩机体。", theme.HintFace, theme.HintColor))
	panel.AddChild(sceneui.Label("Press Z or X to return", theme.HintFace, theme.HintColor))

	s.ui = &ebitenui.UI{Container: root}
}

func (s *Replay) Update(g *game.Game) error {
	if input.Global.JustPressed(input.KeyOk) || input.Global.JustPressed(input.KeyCancel) {
		g.GoTo(s.back)
	}
	s.ui.Update()
	return nil
}

func (s *Replay) Draw(screen *ebiten.Image) {
	screen.Fill(color.RGBA{14, 14, 22, 255})
	s.ui.Draw(screen)
}
