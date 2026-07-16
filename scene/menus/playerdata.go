package menus

import (
	"fmt"
	"image/color"
	"th10/game"
	"th10/input"
	sceneui "th10/scene/ui"

	"github.com/ebitenui/ebitenui"
	"github.com/ebitenui/ebitenui/widget"
	"github.com/hajimehoshi/ebiten/v2"
)

type PlayerData struct {
	ui   *ebitenui.UI
	save game.SaveData
	back game.Scene
}

func NewPlayerData(save *game.SaveData, back game.Scene) *PlayerData {
	var snap game.SaveData
	if save != nil {
		snap = *save
	}
	s := &PlayerData{save: snap, back: back}
	s.build()
	return s
}

func (s *PlayerData) build() {
	theme := sceneui.DefaultTheme()
	root := sceneui.NewRoot(theme, widget.NewAnchorLayout())

	panel := widget.NewContainer(
		widget.ContainerOpts.BackgroundImage(theme.Panel),
		widget.ContainerOpts.Layout(widget.NewGridLayout(
			widget.GridLayoutOpts.Columns(2),
			widget.GridLayoutOpts.Spacing(24, 10),
			widget.GridLayoutOpts.Padding(widget.NewInsetsSimple(32)),
		)),
		widget.ContainerOpts.WidgetOpts(
			widget.WidgetOpts.LayoutData(widget.AnchorLayoutData{
				HorizontalPosition: widget.AnchorLayoutPositionCenter,
				VerticalPosition:   widget.AnchorLayoutPositionCenter,
			}),
			widget.WidgetOpts.MinSize(500, 360),
		),
	)
	root.AddChild(panel)

	title := sceneui.Label("Player Data", theme.TitleFace, theme.HighColor)
	panel.AddChild(title)
	panel.AddChild(sceneui.Label(" ", theme.TitleFace, theme.HighColor)) // spacer for grid

	kvs := [][2]string{
		{"HiScore", fmt.Sprintf("%012d", s.save.HighScore)},
		{"Total Runs", fmt.Sprintf("%d", s.save.TotalRuns)},
		{"Total Clears", fmt.Sprintf("%d", s.save.TotalClears)},
		{"Extra Clears", fmt.Sprintf("%d", s.save.TotalExtraClears)},
		{"Game Overs", fmt.Sprintf("%d", s.save.TotalGameOvers)},
		{"Last Shot", fmt.Sprintf("%s / %s", game.CharacterName(s.save.LastCharacter), game.ShotName(s.save.LastCharacter, s.save.LastShotType))},
	}
	if s.save.ExtraUnlocked {
		kvs = append(kvs, [2]string{"Extra Stage", "Unlocked"})
	} else {
		kvs = append(kvs, [2]string{"Extra Stage", "Locked"})
	}
	for _, kv := range kvs {
		panel.AddChild(sceneui.Label(kv[0], theme.ItemFace, theme.TextColor))
		panel.AddChild(sceneui.Label(kv[1], theme.ItemFace, theme.HighColor))
	}

	panel.AddChild(sceneui.Label("Press Z or X to return", theme.HintFace, theme.HintColor))
	panel.AddChild(sceneui.Label(" ", theme.HintFace, theme.HintColor))

	s.ui = &ebitenui.UI{Container: root}
}

func (s *PlayerData) Update(g *game.Game) error {
	if input.Global.JustPressed(input.KeyOk) || input.Global.JustPressed(input.KeyCancel) {
		g.GoTo(s.back)
	}
	s.ui.Update()
	return nil
}

func (s *PlayerData) Draw(screen *ebiten.Image) {
	screen.Fill(color.RGBA{10, 16, 28, 255})
	s.ui.Draw(screen)
}
