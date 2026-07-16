// Package dialog 关卡内对话覆盖层。
//
// 视觉规格（贴近原作 TH10）：
//   - 对话框只占 Field 横向区域（FieldLeft~FieldRight），不再横跨整个屏幕
//   - 高度 96 px，贴在 FieldBottom；顶部全透明、底部接近不透明黑的纵向渐变
//   - 不画上下边框线；文字（说话者名、正文、">> Z"）统一使用初音蓝
//   - 立绘贴在对话框左/右内沿，比对话框略矮，垂直居中；左侧自机用原向，
//     右侧敌方水平镜像。
//   - 每行可指定 Expression，对应 face 资源中的不同表情后缀（no/an/ct/...）
package dialog

import (
	"image/color"
	"th10/game"
	"th10/render"

	"github.com/hajimehoshi/ebiten/v2"
	ebitext "github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

const (
	boxLeft   = float32(game.FieldLeft)   // 32
	boxRight  = float32(game.FieldRight)  // 416
	boxBottom = float32(game.FieldBottom) // 464
	boxWidth  = boxRight - boxLeft        // 384
	boxHeight = float32(96)               // 由 144 缩到 96，按用户反馈"对话框太高了"
	boxTop    = boxBottom - boxHeight     // 368

	gradientAlphaMax = 220 // 渐变底部的黑色 alpha
	faceDisplayH     = 84  // 立绘缩放后的高度（小于 boxHeight，留 6px 上下边距）
	facePadding      = 6   // 立绘距对话框左右边的内边距
	textPadding      = 12  // 正文距对话框左/右边的内边距
)

// Hatsune blue —— 名字、正文、">> Z" 共用。
var (
	hatsuneBlue       = color.NRGBA{0x39, 0xC5, 0xBB, 255}
	hatsuneBlueShadow = color.NRGBA{8, 24, 28, 230}
)

// Line 一行对话
type Line struct {
	Speaker    string // 说话者键（与 speakerFaces 表对齐）
	Expression string // face 表情后缀，例如 "no"/"an"/"sp"。空值视为 "no"
	Text       string // 对话内容
	IsRight    bool   // true=右侧（敌方），false=左侧（自机）
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

var (
	nameFace ebitext.Face
	textFace ebitext.Face
	hintFace ebitext.Face
)

func ensureFaces() {
	if nameFace == nil {
		nameFace = render.LoadSystemFace(18)
	}
	if textFace == nil {
		textFace = render.LoadSystemFace(18)
	}
	if hintFace == nil {
		hintFace = render.LoadHUDFace(14) // ">> Z" ASCII，走 Bimini
	}
}

// Draw 绘制对话框覆盖层
func (o *Overlay) Draw(screen *ebiten.Image) {
	if o.done || o.current >= len(o.lines) {
		return
	}
	line := &o.lines[o.current]
	ensureFaces()

	// 1. 渐变背景：1px 横条堆叠，alpha 从顶 0 到底 gradientAlphaMax。
	for i := 0; i < int(boxHeight); i++ {
		a := uint8(float32(i) / boxHeight * gradientAlphaMax)
		vector.FillRect(screen, boxLeft, boxTop+float32(i), boxWidth, 1,
			color.NRGBA{0, 0, 0, a}, false)
	}

	// 2. 立绘 —— 贴在对话框左 / 右内沿，垂直居中。
	face := getFace(line.Speaker, line.Expression)
	faceColumn := 0 // 占用对话框该侧多少 px，用于 3. 判断正文起始位置
	if face != nil {
		srcW, srcH := face.Bounds().Dx(), face.Bounds().Dy()
		scale := float64(faceDisplayH) / float64(srcH)
		drawW := float64(srcW) * scale
		faceY := float64(boxTop) + (float64(boxHeight)-faceDisplayH)/2

		op := &ebiten.DrawImageOptions{}
		if line.IsRight {
			// 镜像：先 Scale(-scale, scale) 把图翻转，再把"原点（图的右上角）"
			// 移到对话框右内沿；最终图占 [boxRight-padding-drawW, boxRight-padding]。
			op.GeoM.Scale(-scale, scale)
			op.GeoM.Translate(float64(boxRight)-facePadding, faceY)
		} else {
			op.GeoM.Scale(scale, scale)
			op.GeoM.Translate(float64(boxLeft)+facePadding, faceY)
		}
		op.ColorScale.ScaleAlpha(0.95)
		screen.DrawImage(face, op)
		faceColumn = int(drawW) + facePadding*2
	}

	// 3. 说话者名（与立绘同侧）。
	nameY := int(boxTop) + 8
	if line.IsRight {
		nameW, _ := render.MeasureText(nameFace, line.Speaker)
		render.DrawTextShadow(screen, line.Speaker, nameFace,
			int(boxRight)-faceColumn-nameW-textPadding, nameY,
			hatsuneBlue, hatsuneBlueShadow, 1, 1)
	} else {
		render.DrawTextShadow(screen, line.Speaker, nameFace,
			int(boxLeft)+faceColumn+textPadding, nameY,
			hatsuneBlue, hatsuneBlueShadow, 1, 1)
	}

	// 4. 正文（按 textFace 宽度软换行）。文本始终从对话框中段开始，
	// 左侧立绘时让出左边距、右侧立绘时让出右边距。
	textX := int(boxLeft) + textPadding
	textRight := int(boxRight) - textPadding
	if line.IsRight {
		textRight -= faceColumn
	} else {
		textX += faceColumn
	}
	drawWrapped(screen, line.Text, textFace, textX, int(boxTop)+34,
		textRight-textX, hatsuneBlue, hatsuneBlueShadow)

	// 5. ">> Z" 推进提示（贴右下角）。
	render.DrawTextShadow(screen, ">> Z", hintFace,
		int(boxRight)-44, int(boxBottom)-20,
		hatsuneBlue, hatsuneBlueShadow, 1, 1)
}

func drawWrapped(screen *ebiten.Image, text string, face ebitext.Face, x, y, maxW int, col, shadow color.Color) {
	if face == nil || text == "" {
		return
	}
	line := ""
	lineY := y
	lh := render.LineHeightText(face)
	for _, r := range text {
		line += string(r)
		w, _ := render.MeasureText(face, line)
		if w > maxW {
			last := []rune(line)
			prev := string(last[:len(last)-1])
			cur := string(last[len(last)-1])
			render.DrawTextShadow(screen, prev, face, x, lineY, col, shadow, 1, 1)
			lineY += lh + 2
			line = cur
		}
	}
	if line != "" {
		render.DrawTextShadow(screen, line, face, x, lineY, col, shadow, 1, 1)
	}
}
