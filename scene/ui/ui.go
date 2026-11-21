// Package ui 提供基于 ebitenui 的菜单场景共享工具。
package ui

import (
	"image/color"
	"th10/render"

	eimage "github.com/ebitenui/ebitenui/image"
	"github.com/ebitenui/ebitenui/widget"
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

// Label 快速构造一个文字标签。
func Label(text string, face ebitext.Face, col color.Color) *widget.Text {
	return widget.NewText(
		widget.TextOpts.Text(text, &face, col),
	)
}
