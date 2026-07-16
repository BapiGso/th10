package ui

import (
	"image/color"
	"th10/input"
	"th10/render"

	"github.com/hajimehoshi/ebiten/v2"
	ebitext "github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

// MenuItem 菜单项。
type MenuItem struct {
	Label    string
	Disabled bool
}

// MenuStyle 菜单视觉。
//   - Outline 非 nil 时八方向描边（适合复杂背景，如 Title）；
//     Outline 为 nil 时改用 Shadow 偏移阴影（适合纯色背景，如 Select 各 phase）。
//   - HoverBgColor 非 nil 时在选中项后画一个矩形高亮带；HoverBgPadX / HoverBgH 决定其尺寸。
type MenuStyle struct {
	Face         ebitext.Face
	Idle         color.Color
	Hover        color.Color
	Disabled     color.Color
	Outline      color.Color // 描边色（八方向），与 Shadow 二选一
	Shadow       color.Color // 阴影色（+1,+1 偏移），Outline 为 nil 时使用
	HoverBgColor color.Color // 选中项底色（可选）
	HoverBgPadX  int         // 底色矩形相对文字左右各扩 X 像素
	HoverBgH     int         // 底色矩形高度（不设默认按 LineH）
}

// MenuList 极简纵向菜单：上下移动 / Z 触发 / X 取消，自身用 input.Global 驱动。
//   - 每项 y = Y + i*LineH；x = X + i*StepX（StepX > 0 形成"阶梯"效果，参照原版 Title）。
type MenuList struct {
	Items      []MenuItem
	Cursor     int
	X, Y       int
	LineH      int
	StepX      int
	Style      MenuStyle
	OnActivate func(idx int)
	OnCancel   func()
	OnMove     func(idx int) // 上下移动后的回调（一般用来播 SE）
}

// Update 推进光标 / 触发 OnActivate / OnCancel。
//
// 返回 (moved, activated)，调用方需要播 SE 时可据此判断；忽略也无妨。
func (m *MenuList) Update() (moved bool, activated bool) {
	if len(m.Items) == 0 {
		return false, false
	}
	in := &input.Global
	if in.JustPressed(input.KeyUp) {
		m.move(-1)
		moved = true
	}
	if in.JustPressed(input.KeyDown) {
		m.move(1)
		moved = true
	}
	if in.JustPressed(input.KeyOk) {
		if !m.Items[m.Cursor].Disabled && m.OnActivate != nil {
			m.OnActivate(m.Cursor)
		}
		activated = true
	}
	if in.JustPressed(input.KeyCancel) {
		if m.OnCancel != nil {
			m.OnCancel()
		}
	}
	if moved && m.OnMove != nil {
		m.OnMove(m.Cursor)
	}
	return moved, activated
}

func (m *MenuList) move(d int) {
	n := len(m.Items)
	m.Cursor = (m.Cursor + d + n) % n
}

// Draw 把菜单绘制到 screen。
func (m *MenuList) Draw(screen *ebiten.Image) {
	if m.Style.Face == nil {
		return
	}
	for i, item := range m.Items {
		x := m.X + i*m.StepX
		y := m.Y + i*m.LineH

		// 背景高亮（选中项）
		if i == m.Cursor && m.Style.HoverBgColor != nil {
			tw, _ := render.MeasureText(m.Style.Face, item.Label)
			bgH := m.Style.HoverBgH
			if bgH == 0 {
				bgH = m.LineH
			}
			vector.FillRect(screen,
				float32(x-m.Style.HoverBgPadX), float32(y-2),
				float32(tw+m.Style.HoverBgPadX*2), float32(bgH),
				m.Style.HoverBgColor, false)
		}

		col := m.colorFor(i, item.Disabled)
		if m.Style.Outline != nil {
			drawOutlinedText(screen, item.Label, m.Style.Face, x, y, col, m.Style.Outline)
		} else if m.Style.Shadow != nil {
			render.DrawTextShadow(screen, item.Label, m.Style.Face, x, y, col, m.Style.Shadow, 1, 1)
		} else {
			render.DrawText(screen, item.Label, m.Style.Face, x, y, col)
		}
	}
}

func (m *MenuList) colorFor(i int, disabled bool) color.Color {
	if disabled && i != m.Cursor {
		return m.Style.Disabled
	}
	if i == m.Cursor {
		return m.Style.Hover
	}
	return m.Style.Idle
}

// drawOutlinedText 八方向 1px 描边 + 主色填字（用于复杂背景上保证文字清晰）。
func drawOutlinedText(screen *ebiten.Image, s string, face ebitext.Face, x, y int, col, outline color.Color) {
	for dx := -1; dx <= 1; dx++ {
		for dy := -1; dy <= 1; dy++ {
			if dx == 0 && dy == 0 {
				continue
			}
			render.DrawText(screen, s, face, x+dx, y+dy, outline)
		}
	}
	render.DrawText(screen, s, face, x, y, col)
}
