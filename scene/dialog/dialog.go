package dialog

import (
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

const (
	faceDisplayH = 110.0 // target height for face portrait (pixels)
	dialogBoxY   = 340.0 // top of dialog box
)

// Line 一行对话
type Line struct {
	Speaker string // 说话者名字
	Text    string // 对话内容
	IsRight bool   // true=右侧（敌方），false=左侧（自机）
}

// Overlay 对话覆盖层（Stage 子状态，非独立 Scene）
type Overlay struct {
	lines   []Line
	current int
	done    bool
}

func NewOverlay(lines []Line) *Overlay {
	return &Overlay{lines: lines}
}

// Update 返回 true 表示对话仍在进行
func (o *Overlay) Update(justPressedOk bool) bool {
	if o.done {
		return false
	}
	if justPressedOk {
		o.current++
		if o.current >= len(o.lines) {
			o.done = true
			return false
		}
	}
	return true
}

// Done 对话是否结束
func (o *Overlay) Done() bool {
	return o.done
}

// Draw 绘制对话框覆盖层
func (o *Overlay) Draw(screen *ebiten.Image) {
	if o.done || o.current >= len(o.lines) {
		return
	}
	line := &o.lines[o.current]

	sw := float32(screen.Bounds().Dx())
	swF := float64(sw)

	// --- Draw face portrait above / overlapping the dialog box ---
	face := getFace(line.Speaker)
	if face != nil {
		srcH := float64(face.Bounds().Dy())
		scale := faceDisplayH / srcH // uniform scale to target height

		// Face sits above the dialog box, partially overlapping it.
		// Bottom of the face aligns with dialogBoxY + 20 (slight overlap).
		faceY := dialogBoxY - faceDisplayH + 20

		op := &ebiten.DrawImageOptions{}

		if line.IsRight {
			// Right side: flip horizontally so the character faces inward.
			// Scale(-scale, scale) mirrors on X; then translate to right edge.
			op.GeoM.Scale(-scale, scale)
			op.GeoM.Translate(swF-10, faceY) // 10px right margin
		} else {
			// Left side: draw normally.
			op.GeoM.Scale(scale, scale)
			op.GeoM.Translate(10, faceY) // 10px left margin
		}
		screen.DrawImage(face, op)
	}

	// --- Dialog box background ---
	vector.FillRect(screen, 0, dialogBoxY, sw, 140, color.RGBA{0, 0, 0, 180}, false)
	vector.StrokeRect(screen, 2, dialogBoxY+2, sw-4, 136, 1, color.RGBA{128, 128, 200, 200}, false)

	// --- Speaker name ---
	nameX := 20
	if line.IsRight {
		nameX = int(sw) - 120
	}
	ebitenutil.DebugPrintAt(screen, line.Speaker, nameX, int(dialogBoxY)+10)
	ebitenutil.DebugPrintAt(screen, line.Text, 20, int(dialogBoxY)+35)
	ebitenutil.DebugPrintAt(screen, ">> Z", int(sw)-50, int(dialogBoxY)+120)
}
