package menus

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

type bgmEntry struct {
	Index int
	Label string
}

// Scene Music Room：ebitenui List 驱动。
type MusicRoom struct {
	ui      *ebitenui.UI
	list    *widget.List
	now     *widget.Text
	back    game.Scene
	current int
}

func NewMusicRoom(back game.Scene) *MusicRoom {
	s := &MusicRoom{back: back, current: -1}
	s.build()
	return s
}

func (s *MusicRoom) build() {
	theme := sceneui.DefaultTheme()

	root := sceneui.NewRoot(theme, widget.NewAnchorLayout())

	panel := widget.NewContainer(
		widget.ContainerOpts.BackgroundImage(theme.Panel),
		widget.ContainerOpts.Layout(widget.NewRowLayout(
			widget.RowLayoutOpts.Direction(widget.DirectionVertical),
			widget.RowLayoutOpts.Spacing(10),
			widget.RowLayoutOpts.Padding(widget.NewInsetsSimple(24)),
		)),
		widget.ContainerOpts.WidgetOpts(
			widget.WidgetOpts.LayoutData(widget.AnchorLayoutData{
				HorizontalPosition: widget.AnchorLayoutPositionCenter,
				VerticalPosition:   widget.AnchorLayoutPositionCenter,
			}),
			widget.WidgetOpts.MinSize(520, 420),
		),
	)
	root.AddChild(panel)

	panel.AddChild(sceneui.Label("Music Room", theme.TitleFace, theme.HighColor))

	entries := make([]any, 0, len(audio.BGMNames))
	for i, name := range audio.BGMNames {
		entries = append(entries, bgmEntry{Index: i, Label: fmt.Sprintf("%02d. %s", i+1, name)})
	}

	entryFace := theme.ItemFace
	s.list = widget.NewList(
		widget.ListOpts.ContainerOpts(widget.ContainerOpts.WidgetOpts(
			widget.WidgetOpts.MinSize(480, 320),
		)),
		widget.ListOpts.ScrollContainerImage(&widget.ScrollContainerImage{
			Idle:     theme.Panel,
			Disabled: theme.Panel,
			Mask:     theme.Panel,
		}),
		widget.ListOpts.SliderParams(&widget.SliderParams{
			TrackImage:   theme.Slider,
			HandleImage:  theme.Handle,
			TrackPadding: widget.NewInsetsSimple(2),
		}),
		widget.ListOpts.HideHorizontalSlider(),
		widget.ListOpts.EntryFontFace(&entryFace),
		widget.ListOpts.EntryColor(&widget.ListEntryColor{
			Selected:                   color.NRGBA{R: 255, G: 240, B: 200, A: 255},
			Unselected:                 color.NRGBA{R: 244, G: 238, B: 228, A: 255},
			SelectedBackground:         color.NRGBA{R: 100, G: 70, B: 40, A: 220},
			SelectingBackground:        color.NRGBA{R: 80, G: 60, B: 40, A: 200},
			SelectingFocusedBackground: color.NRGBA{R: 110, G: 80, B: 50, A: 230},
			SelectedFocusedBackground:  color.NRGBA{R: 140, G: 100, B: 60, A: 240},
			FocusedBackground:          color.NRGBA{R: 80, G: 60, B: 40, A: 180},
			DisabledUnselected:         color.NRGBA{R: 120, G: 120, B: 120, A: 255},
			DisabledSelected:           color.NRGBA{R: 120, G: 120, B: 120, A: 255},
			DisabledSelectedBackground: color.NRGBA{R: 60, G: 50, B: 40, A: 200},
		}),
		widget.ListOpts.EntryLabelFunc(func(e interface{}) string {
			return e.(bgmEntry).Label
		}),
		widget.ListOpts.EntryTextPadding(widget.NewInsetsSimple(4)),
		widget.ListOpts.EntryTextPosition(widget.TextPositionStart, widget.TextPositionCenter),
	)
	s.list.SetEntries(entries)
	if len(entries) > 0 {
		s.list.SetSelectedEntry(entries[0])
	}
	panel.AddChild(s.list)

	s.now = widget.NewText(widget.TextOpts.Text(" ", &theme.HintFace, theme.HintColor))
	panel.AddChild(s.now)

	panel.AddChild(sceneui.Label("Up/Down: Move   Z: Play   X: Back", theme.HintFace, theme.HintColor))

	s.ui = &ebitenui.UI{Container: root}
	s.ui.SetFocusedWidget(s.list)
}

func (s *MusicRoom) Update(g *game.Game) error {
	in := &input.Global

	if in.JustPressed(input.KeyUp) {
		s.list.FocusPrevious()
	}
	if in.JustPressed(input.KeyDown) {
		s.list.FocusNext()
	}
	if in.JustPressed(input.KeyOk) {
		s.list.SelectFocused()
		if entry, ok := s.list.SelectedEntry().(bgmEntry); ok {
			g.Audio().PlayBGM(entry.Index)
			s.current = entry.Index
			s.now.Label = fmt.Sprintf("Now Playing: %s", audio.BGMNames[entry.Index])
		}
	}
	if in.JustPressed(input.KeyCancel) {
		g.GoTo(s.back)
	}
	s.ui.Update()
	return nil
}

func (s *MusicRoom) Draw(screen *ebiten.Image) {
	screen.Fill(color.RGBA{16, 10, 20, 255})
	s.ui.Draw(screen)
}
