package ending

import (
	"fmt"
	"image/color"
	"th10/audio"
	"th10/game"
	"th10/input"
	"th10/scene/result"
	sceneui "th10/scene/ui"

	"github.com/ebitenui/ebitenui"
	"github.com/ebitenui/ebitenui/widget"
	"github.com/hajimehoshi/ebiten/v2"
)

type Ending struct {
	ui         *ebitenui.UI
	state      game.GameState
	extra      bool
	page       int
	frame      int
	bgmStarted bool
	pages      []string
	back       game.Scene

	bodyText  *widget.Text
	pageLabel *widget.Text
}

func New(state *game.GameState, extra bool, back game.Scene) *Ending {
	var snap game.GameState
	if state != nil {
		snap = *state
	}
	e := &Ending{
		state: snap,
		extra: extra,
		pages: endingPages(&snap, extra),
		back:  back,
	}
	e.build()
	return e
}

func (e *Ending) heading() string {
	if e.extra {
		return "Extra Ending"
	}
	return "Ending"
}

func (e *Ending) build() {
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
			widget.WidgetOpts.MinSize(528, 360),
		),
	)
	root.AddChild(panel)

	panel.AddChild(sceneui.Label(e.heading(), theme.TitleFace, theme.HighColor))
	panel.AddChild(sceneui.Label(
		fmt.Sprintf("%s / %s",
			game.CharacterName(e.state.Character),
			game.ShotName(e.state.Character, e.state.ShotType)),
		theme.ItemFace, theme.TextColor))

	e.bodyText = widget.NewText(widget.TextOpts.Text(e.pages[e.page], &theme.ItemFace, theme.TextColor))
	panel.AddChild(e.bodyText)

	e.pageLabel = widget.NewText(widget.TextOpts.Text(e.pageStatus(), &theme.HintFace, theme.HintColor))
	panel.AddChild(e.pageLabel)

	panel.AddChild(sceneui.Label("Press Z to continue", theme.HintFace, theme.HintColor))

	e.ui = &ebitenui.UI{Container: root}
}

func (e *Ending) pageStatus() string {
	return fmt.Sprintf("Page %d / %d", e.page+1, len(e.pages))
}

func (e *Ending) Update(g *game.Game) error {
	e.frame++
	if !e.bgmStarted {
		g.Audio().PlayBGM(audio.BGMEnding)
		e.bgmStarted = true
	}
	if e.frame > 15 && (input.Global.JustPressed(input.KeyOk) || input.Global.JustPressed(input.KeyCancel)) {
		nextPage := e.page + 1
		e.frame = 0
		if nextPage >= len(e.pages) {
			mode := result.ModeStageClear
			if e.extra {
				mode = result.ModeExtraClear
			}
			g.GoTo(result.New(&e.state, mode, e.back))
			return nil
		}
		e.page = nextPage
		e.bodyText.Label = e.pages[e.page]
		e.pageLabel.Label = e.pageStatus()
	}
	e.ui.Update()
	return nil
}

func (e *Ending) Draw(screen *ebiten.Image) {
	screen.Fill(color.RGBA{18, 16, 28, 255})
	e.ui.Draw(screen)
}

func endingPages(state *game.GameState, extra bool) []string {
	if extra {
		return []string{
			fmt.Sprintf("%s 击败了洩矢諏訪子。", game.CharacterName(state.Character)),
			"守矢神社的骚动暂时平息，妖怪之山也恢复了原本的秩序。",
			"但在幻想乡，新的弹幕骚动永远不会真正结束。",
		}
	}
	return []string{
		fmt.Sprintf("%s 登上了妖怪之山顶，击退了八坂神奈子。", game.CharacterName(state.Character)),
		"守矢神社正式在幻想乡扎根，博丽神社也迎来了新的竞争者。",
		"故事告一段落，但信仰之争才刚刚开始。",
	}
}
