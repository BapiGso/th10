package result

import (
	"fmt"
	"image/color"
	"th10/audio"
	"th10/game"
	"th10/input"
	sceneui "th10/scene/ui"

	"github.com/ebitenui/ebitenui"
	"github.com/ebitenui/ebitenui/widget"
	"github.com/hajimehoshi/ebiten/v2"
)

type Mode int

const (
	ModeGameOver Mode = iota
	ModeStageClear
	ModeExtraClear
)

type Result struct {
	ui         *ebitenui.UI
	state      game.GameState
	mode       Mode
	frame      int
	bgmStarted bool
	back       game.Scene
}

func New(state *game.GameState, mode Mode, back game.Scene) *Result {
	var snap game.GameState
	if state != nil {
		snap = *state
	}
	r := &Result{state: snap, mode: mode, back: back}
	r.build()
	return r
}

func (r *Result) heading() string {
	switch r.mode {
	case ModeGameOver:
		return "Game Over"
	case ModeExtraClear:
		return "Extra Clear"
	default:
		return "Stage Clear"
	}
}

func (r *Result) build() {
	theme := sceneui.DefaultTheme()
	root := sceneui.NewRoot(theme, widget.NewAnchorLayout())

	panel := widget.NewContainer(
		widget.ContainerOpts.BackgroundImage(theme.Panel),
		widget.ContainerOpts.Layout(widget.NewRowLayout(
			widget.RowLayoutOpts.Direction(widget.DirectionVertical),
			widget.RowLayoutOpts.Spacing(6),
			widget.RowLayoutOpts.Padding(widget.NewInsetsSimple(32)),
		)),
		widget.ContainerOpts.WidgetOpts(
			widget.WidgetOpts.LayoutData(widget.AnchorLayoutData{
				HorizontalPosition: widget.AnchorLayoutPositionCenter,
				VerticalPosition:   widget.AnchorLayoutPositionCenter,
			}),
			widget.WidgetOpts.MinSize(540, 400),
		),
	)
	root.AddChild(panel)

	panel.AddChild(sceneui.Label(r.heading(), theme.TitleFace, theme.HighColor))

	grid := widget.NewContainer(
		widget.ContainerOpts.Layout(widget.NewGridLayout(
			widget.GridLayoutOpts.Columns(2),
			widget.GridLayoutOpts.Spacing(30, 4),
		)),
	)
	panel.AddChild(grid)

	row := func(k, v string) {
		grid.AddChild(sceneui.Label(k, theme.ItemFace, theme.TextColor))
		grid.AddChild(sceneui.Label(v, theme.ItemFace, theme.HighColor))
	}
	row("Score", fmt.Sprintf("%012d", r.state.Score))
	row("HiScore", fmt.Sprintf("%012d", r.state.HiScore))
	row("Stage", game.StageName(r.state.Stage))
	row("Difficulty", game.DifficultyName(r.state.Difficulty))
	row("Player", fmt.Sprintf("%s / %s", game.CharacterName(r.state.Character), game.ShotName(r.state.Character, r.state.ShotType)))
	row("Power", fmt.Sprintf("%d.%02d", r.state.Power/100, r.state.Power%100))
	row("Graze", fmt.Sprintf("%d", r.state.Graze))
	row("Point", fmt.Sprintf("%d", r.state.Point))
	row("Faith", fmt.Sprintf("%d", r.state.Faith))

	if r.state.ExtraUnlocked {
		panel.AddChild(sceneui.Label("Extra Stage 已解锁", theme.ItemFace, theme.HighColor))
	}
	panel.AddChild(sceneui.Label("Press Z or X to return to Title", theme.HintFace, theme.HintColor))

	r.ui = &ebitenui.UI{Container: root}
}

func (r *Result) Update(g *game.Game) error {
	r.frame++
	if !r.bgmStarted {
		g.Audio().PlayBGM(audio.BGMResult)
		r.bgmStarted = true
	}
	if r.frame > 15 && (input.Global.JustPressed(input.KeyOk) || input.Global.JustPressed(input.KeyCancel)) {
		g.GoTo(r.back)
	}
	r.ui.Update()
	return nil
}

func (r *Result) Draw(screen *ebiten.Image) {
	screen.Fill(color.RGBA{10, 14, 30, 255})
	r.ui.Draw(screen)
}
