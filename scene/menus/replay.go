package menus

import (
	"fmt"
	"image/color"
	"th10/game"
	"th10/input"
	selectscene "th10/scene/select"
	"th10/scene/stage"
	sceneui "th10/scene/ui"
	"time"

	"github.com/ebitenui/ebitenui"
	"github.com/ebitenui/ebitenui/widget"
	"github.com/hajimehoshi/ebiten/v2"
)

type Replay struct {
	ui      *ebitenui.UI
	list    *widget.List
	status  *widget.Text
	back    game.Scene
	entries []*replayMenuEntry
	loaded  bool
	frame   int
}

type replayMenuEntry struct{ slot game.ReplaySlot }

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
			widget.RowLayoutOpts.Spacing(8),
			widget.RowLayoutOpts.Padding(widget.NewInsetsSimple(24)),
		)),
		widget.ContainerOpts.WidgetOpts(
			widget.WidgetOpts.LayoutData(widget.AnchorLayoutData{
				HorizontalPosition: widget.AnchorLayoutPositionCenter,
				VerticalPosition:   widget.AnchorLayoutPositionCenter,
			}),
			widget.WidgetOpts.MinSize(560, 420),
		),
	)
	root.AddChild(panel)

	panel.AddChild(sceneui.Label("Replay", theme.TitleFace, theme.HighColor))
	entryFace := theme.HintFace
	s.list = widget.NewList(
		widget.ListOpts.ContainerOpts(widget.ContainerOpts.WidgetOpts(widget.WidgetOpts.MinSize(512, 280))),
		widget.ListOpts.ScrollContainerImage(&widget.ScrollContainerImage{Idle: theme.Panel, Disabled: theme.Panel, Mask: theme.Panel}),
		widget.ListOpts.SliderParams(&widget.SliderParams{TrackImage: theme.Slider, HandleImage: theme.Handle, TrackPadding: widget.NewInsetsSimple(2)}),
		widget.ListOpts.HideHorizontalSlider(),
		widget.ListOpts.EntryFontFace(&entryFace),
		widget.ListOpts.EntryColor(&widget.ListEntryColor{
			Selected: theme.HighColor, Unselected: theme.TextColor,
			SelectedBackground:         color.NRGBA{100, 70, 40, 220},
			SelectingBackground:        color.NRGBA{80, 60, 40, 200},
			SelectingFocusedBackground: color.NRGBA{110, 80, 50, 230},
			SelectedFocusedBackground:  color.NRGBA{140, 100, 60, 240},
			FocusedBackground:          color.NRGBA{80, 60, 40, 180},
			DisabledUnselected:         theme.DimColor,
			DisabledSelected:           theme.DimColor,
			DisabledSelectedBackground: color.NRGBA{60, 50, 40, 200},
		}),
		widget.ListOpts.EntryLabelFunc(func(e interface{}) string { return replayEntryLabel(e.(*replayMenuEntry).slot) }),
		widget.ListOpts.EntryTextPadding(widget.NewInsetsSimple(4)),
		widget.ListOpts.EntryTextPosition(widget.TextPositionStart, widget.TextPositionCenter),
	)
	entries := make([]any, len(s.entries))
	for i, entry := range s.entries {
		entries[i] = entry
	}
	s.list.SetEntries(entries)
	if len(entries) > 0 {
		s.list.SetSelectedEntry(entries[0])
	}
	panel.AddChild(s.list)
	s.status = sceneui.Label("Select a saved replay. Empty slots are shown too.", theme.HintFace, theme.HintColor)
	panel.AddChild(s.status)
	panel.AddChild(sceneui.Label("Up/Down: Move   Z: Play   X/Esc: Back", theme.HintFace, theme.HintColor))

	s.ui = &ebitenui.UI{Container: root}
	s.ui.SetFocusedWidget(s.list)
}

func (s *Replay) Update(g *game.Game) error {
	if !s.loaded {
		slots := g.ListReplays()
		for _, slot := range slots {
			s.entries = append(s.entries, &replayMenuEntry{slot: slot})
		}
		s.loaded = true
		s.build()
	}
	s.frame++
	in := &input.Global
	if in.JustPressed(input.KeyUp) {
		s.list.FocusPrevious()
	}
	if in.JustPressed(input.KeyDown) {
		s.list.FocusNext()
	}
	if s.frame > 10 && in.JustPressed(input.KeyOk) {
		s.list.SelectFocused()
		if entry, ok := s.list.SelectedEntry().(*replayMenuEntry); ok {
			r, err := g.LoadReplay(entry.slot.Number)
			if err != nil {
				s.status.Label = err.Error()
			} else {
				p, err := stage.NewPlayback(g, r, 0, selectscene.ReplayScript, s)
				if err != nil {
					s.status.Label = err.Error()
				} else {
					g.GoTo(p)
				}
			}
		}
	}
	if in.JustPressed(input.KeyCancel) {
		g.GoTo(s.back)
	}
	s.ui.Update()
	return nil
}

func replayEntryLabel(slot game.ReplaySlot) string {
	if slot.Error != nil {
		return fmt.Sprintf("No.%02d  [invalid replay]", slot.Number)
	}
	if slot.Replay == nil {
		return fmt.Sprintf("No.%02d  -- empty --", slot.Number)
	}
	g := slot.Replay.Global
	chara := []string{"Reimu", "Marisa"}[g.Character]
	diff := []string{"Easy", "Normal", "Hard", "Lunatic", "Extra"}[g.Difficulty]
	return fmt.Sprintf("No.%02d  %-8s %s  %s%c / %s  St%d", slot.Number, g.User,
		time.Unix(int64(g.Timestamp), 0).Format("06/01/02"), chara, 'A'+g.ShotType, diff, g.Stage)
}

func (s *Replay) Draw(screen *ebiten.Image) {
	screen.Fill(color.RGBA{14, 14, 22, 255})
	s.ui.Draw(screen)
}
