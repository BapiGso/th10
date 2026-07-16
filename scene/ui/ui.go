// Package ui 提供基于 ebitenui 的菜单场景共享工具。
package ui

import (
	"image/color"
	"th10/input"
	"th10/render"

	"github.com/ebitenui/ebitenui"
	eimage "github.com/ebitenui/ebitenui/image"
	"github.com/ebitenui/ebitenui/widget"
	"github.com/hajimehoshi/ebiten/v2"
	ebitext "github.com/hajimehoshi/ebiten/v2/text/v2"
)

// Theme 菜单场景通用配色 / 九宫格资源。
type Theme struct {
	TitleFace  ebitext.Face
	ItemFace   ebitext.Face
	HintFace   ebitext.Face
	Background *eimage.NineSlice
	Panel      *eimage.NineSlice
	Button     *widget.ButtonImage
	Slider     *widget.SliderTrackImage
	Handle     *widget.ButtonImage
	Checkbox   *widget.CheckboxImage
	TextColor  color.Color
	DimColor   color.Color
	HintColor  color.Color
	HighColor  color.Color
}

// DefaultTheme 返回默认 TH 风格的菜单主题。
//
// 字体策略：
//   - 标题/条目用系统黑体（中日文齐全），保证菜单中的关卡/Boss 名能显示
//   - 提示行也走系统字体，纯英文也不影响观感
//   - HUD 分数/数字另走 render.LoadHUDFace（Bimini Bold），不在这里
func DefaultTheme() *Theme {
	return &Theme{
		TitleFace:  render.LoadSystemFace(28),
		ItemFace:   render.LoadSystemFace(20),
		HintFace:   render.LoadSystemFace(16),
		Background: eimage.NewNineSliceColor(color.NRGBA{12, 8, 32, 255}),
		Panel:      eimage.NewNineSliceColor(color.NRGBA{0, 0, 0, 170}),
		Button: &widget.ButtonImage{
			Idle:    eimage.NewNineSliceColor(color.NRGBA{40, 30, 80, 200}),
			Hover:   eimage.NewNineSliceColor(color.NRGBA{80, 50, 160, 230}),
			Pressed: eimage.NewNineSliceColor(color.NRGBA{120, 80, 200, 240}),
		},
		Slider: &widget.SliderTrackImage{
			Idle:  eimage.NewNineSliceColor(color.NRGBA{70, 60, 100, 230}),
			Hover: eimage.NewNineSliceColor(color.NRGBA{90, 80, 130, 240}),
		},
		Handle: &widget.ButtonImage{
			Idle:    eimage.NewNineSliceColor(color.NRGBA{200, 170, 110, 255}),
			Hover:   eimage.NewNineSliceColor(color.NRGBA{240, 210, 140, 255}),
			Pressed: eimage.NewNineSliceColor(color.NRGBA{255, 230, 160, 255}),
		},
		Checkbox: &widget.CheckboxImage{
			Unchecked:         eimage.NewNineSliceColor(color.NRGBA{60, 50, 90, 230}),
			UncheckedHovered:  eimage.NewNineSliceColor(color.NRGBA{90, 70, 130, 230}),
			UncheckedDisabled: eimage.NewNineSliceColor(color.NRGBA{60, 50, 90, 150}),
			Checked:           eimage.NewNineSliceColor(color.NRGBA{200, 170, 110, 255}),
			CheckedHovered:    eimage.NewNineSliceColor(color.NRGBA{240, 210, 140, 255}),
			CheckedDisabled:   eimage.NewNineSliceColor(color.NRGBA{160, 130, 80, 180}),
			Greyed:            eimage.NewNineSliceColor(color.NRGBA{120, 110, 110, 200}),
			GreyedHovered:     eimage.NewNineSliceColor(color.NRGBA{140, 130, 130, 200}),
			GreyedDisabled:    eimage.NewNineSliceColor(color.NRGBA{90, 80, 80, 180}),
		},
		TextColor: color.NRGBA{244, 238, 228, 255},
		DimColor:  color.NRGBA{156, 152, 152, 255},
		HintColor: color.NRGBA{228, 222, 214, 240},
		HighColor: color.NRGBA{255, 245, 214, 255},
	}
}

// NewRoot 构造带背景的根容器。
func NewRoot(theme *Theme, layout widget.Layouter) *widget.Container {
	return widget.NewContainer(
		widget.ContainerOpts.BackgroundImage(theme.Background),
		widget.ContainerOpts.Layout(layout),
	)
}

// NewPanel 构造半透明黑底面板，用于场景中部内容区域。
func NewPanel(theme *Theme, layout widget.Layouter, padding *widget.Insets) *widget.Container {
	return widget.NewContainer(
		widget.ContainerOpts.BackgroundImage(theme.Panel),
		widget.ContainerOpts.Layout(layout),
		widget.ContainerOpts.WidgetOpts(
			widget.WidgetOpts.LayoutData(widget.AnchorLayoutData{
				HorizontalPosition: widget.AnchorLayoutPositionCenter,
				VerticalPosition:   widget.AnchorLayoutPositionCenter,
				StretchHorizontal:  false,
				StretchVertical:    false,
				Padding:            padding,
			}),
		),
	)
}

// Label 快速构造一个文字标签。
func Label(text string, face ebitext.Face, col color.Color) *widget.Text {
	return widget.NewText(
		widget.TextOpts.Text(text, &face, col),
	)
}

// Button 快速构造一个按钮（带统一的内边距与主题）。
func Button(theme *Theme, text string, onClick func()) *widget.Button {
	face := theme.ItemFace
	return widget.NewButton(
		widget.ButtonOpts.Image(theme.Button),
		widget.ButtonOpts.Text(text, &face, &widget.ButtonTextColor{
			Idle:  theme.TextColor,
			Hover: theme.HighColor,
		}),
		widget.ButtonOpts.TextPadding(&widget.Insets{Left: 18, Right: 18, Top: 6, Bottom: 6}),
		widget.ButtonOpts.ClickedHandler(func(args *widget.ButtonClickedEventArgs) {
			if onClick != nil {
				onClick()
			}
		}),
	)
}

// Scene 提供 ebitenui + 键盘导航的最小壳子。
// 使用者在 Build 中构建 root 并把 focusable 组件按顺序加到 Focus 中。
type Scene struct {
	UI       *ebitenui.UI
	Focus    []widget.Focuser // 上下键切换的顺序
	cursor   int
	OnOk     func()
	OnCancel func()
}

// SetCursor 强制设置当前焦点 index。
func (s *Scene) SetCursor(i int) {
	if len(s.Focus) == 0 {
		return
	}
	s.cursor = ((i % len(s.Focus)) + len(s.Focus)) % len(s.Focus)
	s.UI.SetFocusedWidget(s.Focus[s.cursor])
}

// Cursor 当前焦点 index。
func (s *Scene) Cursor() int { return s.cursor }

// Update 按我们自己的 input.Global 驱动 Up/Down 换焦点，Z 触发 Click，X 触发 OnCancel。
func (s *Scene) Update() {
	in := &input.Global
	if len(s.Focus) > 0 {
		if in.JustPressed(input.KeyUp) {
			s.SetCursor(s.cursor - 1)
		}
		if in.JustPressed(input.KeyDown) {
			s.SetCursor(s.cursor + 1)
		}
	}
	if in.JustPressed(input.KeyOk) {
		s.activateFocused()
		if s.OnOk != nil {
			s.OnOk()
		}
	}
	if in.JustPressed(input.KeyCancel) && s.OnCancel != nil {
		s.OnCancel()
	}
	s.UI.Update()
}

func (s *Scene) activateFocused() {
	if w := s.UI.GetFocusedWidget(); w != nil {
		switch v := w.(type) {
		case *widget.Button:
			v.Click()
		}
	}
}

// Draw 透传给 UI。
func (s *Scene) Draw(screen *ebiten.Image) {
	s.UI.Draw(screen)
}
