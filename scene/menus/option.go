package menus

import (
	"fmt"
	"image/color"
	"th10/game"
	"th10/input"
	sceneui "th10/scene/ui"

	"github.com/ebitenui/ebitenui"
	eimage "github.com/ebitenui/ebitenui/image"
	"github.com/ebitenui/ebitenui/widget"
	"github.com/hajimehoshi/ebiten/v2"
)

// Option 设置场景
type Option struct {
	g     *game.Game
	back  game.Scene
	scene *sceneui.Scene

	hint   *widget.Checkbox
	bgm    *widget.Slider
	bgmVal *widget.Text
	se     *widget.Slider
	seVal  *widget.Text

	focusIdx int
}

func NewOption(g *game.Game, back game.Scene) *Option {
	o := &Option{g: g, back: back}
	o.build()
	return o
}

func (o *Option) build() {
	theme := sceneui.DefaultTheme()
	cfg := o.g.Config

	root := sceneui.NewRoot(theme, widget.NewAnchorLayout())

	panel := widget.NewContainer(
		widget.ContainerOpts.BackgroundImage(theme.Panel),
		widget.ContainerOpts.Layout(widget.NewRowLayout(
			widget.RowLayoutOpts.Direction(widget.DirectionVertical),
			widget.RowLayoutOpts.Spacing(14),
			widget.RowLayoutOpts.Padding(widget.NewInsetsSimple(28)),
		)),
		widget.ContainerOpts.WidgetOpts(
			widget.WidgetOpts.LayoutData(widget.AnchorLayoutData{
				HorizontalPosition: widget.AnchorLayoutPositionCenter,
				VerticalPosition:   widget.AnchorLayoutPositionCenter,
				Padding:            widget.NewInsetsSimple(30),
			}),
			widget.WidgetOpts.MinSize(420, 360),
		),
	)
	root.AddChild(panel)

	// 标题
	panel.AddChild(sceneui.Label("Option", theme.TitleFace, theme.HighColor))

	// Hint checkbox
	hintRow := newRow()
	hintRow.AddChild(sceneui.Label("Hint", theme.ItemFace, theme.TextColor))
	o.hint = widget.NewCheckbox(
		widget.CheckboxOpts.Image(theme.Checkbox),
		widget.CheckboxOpts.WidgetOpts(widget.WidgetOpts.MinSize(24, 24)),
		widget.CheckboxOpts.StateChangedHandler(func(args *widget.CheckboxChangedEventArgs) {
			cfg.Hint = args.State == widget.WidgetChecked
		}),
	)
	if cfg.Hint {
		o.hint.SetState(widget.WidgetChecked)
	}
	hintRow.AddChild(o.hint)
	panel.AddChild(hintRow)

	// BGM Slider
	o.bgm, o.bgmVal = o.volumeRow(theme, "BGM Volume", cfg.BGMVolume, func(v float64) {
		cfg.BGMVolume = v
		o.g.Audio().SetBGMVolume(v)
	})
	bgmRow := newRow()
	bgmRow.AddChild(sceneui.Label("BGM Volume", theme.ItemFace, theme.TextColor))
	bgmRow.AddChild(o.bgm)
	bgmRow.AddChild(o.bgmVal)
	panel.AddChild(bgmRow)

	// SE Slider
	o.se, o.seVal = o.volumeRow(theme, "SE Volume", cfg.SEVolume, func(v float64) {
		cfg.SEVolume = v
		o.g.Audio().SetSEVolume(v)
	})
	seRow := newRow()
	seRow.AddChild(sceneui.Label("SE Volume", theme.ItemFace, theme.TextColor))
	seRow.AddChild(o.se)
	seRow.AddChild(o.seVal)
	panel.AddChild(seRow)

	// 按钮
	defaultBtn := sceneui.Button(theme, "Default", func() {
		def := game.DefaultConfig()
		*cfg = *def
		o.g.Audio().SetBGMVolume(cfg.BGMVolume)
		o.g.Audio().SetSEVolume(cfg.SEVolume)
		o.syncFromConfig()
	})
	panel.AddChild(defaultBtn)

	quitBtn := sceneui.Button(theme, "Quit", func() {
		_ = o.g.SaveConfig()
		o.g.GoTo(o.back)
	})
	panel.AddChild(quitBtn)

	// 提示
	panel.AddChild(sceneui.Label(
		"Up/Down: Move   Left/Right: Adjust   Z: Select   X: Back",
		theme.HintFace, theme.HintColor,
	))

	o.scene = &sceneui.Scene{
		UI:    &ebitenui.UI{Container: root},
		Focus: []widget.Focuser{o.hint, o.bgm, o.se, defaultBtn, quitBtn},
		OnCancel: func() {
			_ = o.g.SaveConfig()
			o.g.GoTo(o.back)
		},
	}
	o.scene.SetCursor(0)
}

func newRow() *widget.Container {
	return widget.NewContainer(
		widget.ContainerOpts.Layout(widget.NewRowLayout(
			widget.RowLayoutOpts.Direction(widget.DirectionHorizontal),
			widget.RowLayoutOpts.Spacing(14),
		)),
	)
}

func (o *Option) volumeRow(theme *sceneui.Theme, _ string, initial float64, onChange func(float64)) (*widget.Slider, *widget.Text) {
	current := int(initial * 100)
	valText := widget.NewText(widget.TextOpts.Text(fmt.Sprintf("%3d%%", current), &theme.HintFace, theme.TextColor))
	slider := widget.NewSlider(
		widget.SliderOpts.Orientation(widget.DirectionHorizontal),
		widget.SliderOpts.MinMax(0, 100),
		widget.SliderOpts.InitialCurrent(current),
		widget.SliderOpts.Images(theme.Slider, theme.Handle),
		widget.SliderOpts.FixedHandleSize(10),
		widget.SliderOpts.TrackOffset(4),
		widget.SliderOpts.PageSizeFunc(func() int { return 5 }),
		widget.SliderOpts.WidgetOpts(widget.WidgetOpts.MinSize(200, 16)),
		widget.SliderOpts.ChangedHandler(func(args *widget.SliderChangedEventArgs) {
			v := float64(args.Current) / 100
			onChange(v)
			valText.Label = fmt.Sprintf("%3d%%", args.Current)
		}),
	)
	return slider, valText
}

func (o *Option) syncFromConfig() {
	cfg := o.g.Config
	if cfg.Hint {
		o.hint.SetState(widget.WidgetChecked)
	} else {
		o.hint.SetState(widget.WidgetUnchecked)
	}
	o.bgm.Current = int(cfg.BGMVolume * 100)
	o.bgmVal.Label = fmt.Sprintf("%3d%%", o.bgm.Current)
	o.se.Current = int(cfg.SEVolume * 100)
	o.seVal.Label = fmt.Sprintf("%3d%%", o.se.Current)
}

func (o *Option) Update(g *game.Game) error {
	in := &input.Global
	cursor := o.scene.Cursor()

	// 对滑条使用左右键调节音量
	if w := o.scene.Focus[cursor]; w == o.bgm || w == o.se {
		s := w.(*widget.Slider)
		step := 5
		repeat := func(dir int) {
			s.Current += dir * step
			if s.Current < 0 {
				s.Current = 0
			}
			if s.Current > 100 {
				s.Current = 100
			}
			// 手动触发 changed handler：重算音量与显示
			v := float64(s.Current) / 100
			if s == o.bgm {
				g.Config.BGMVolume = v
				g.Audio().SetBGMVolume(v)
				o.bgmVal.Label = fmt.Sprintf("%3d%%", s.Current)
			} else {
				g.Config.SEVolume = v
				g.Audio().SetSEVolume(v)
				o.seVal.Label = fmt.Sprintf("%3d%%", s.Current)
			}
		}
		if in.JustPressed(input.KeyLeft) || (in.IsPressed(input.KeyLeft) && in.HoldFrames(input.KeyLeft)%4 == 0) {
			repeat(-1)
		}
		if in.JustPressed(input.KeyRight) || (in.IsPressed(input.KeyRight) && in.HoldFrames(input.KeyRight)%4 == 0) {
			repeat(1)
		}
	}

	o.scene.Update()
	return nil
}

func (o *Option) Draw(screen *ebiten.Image) {
	screen.Fill(color.RGBA{12, 8, 32, 255})
	o.scene.Draw(screen)
}

// 兼容变量：保留旧的背景色常量以防外部引用
var _ = eimage.NewNineSliceColor
