// Package dialog 关卡内对话：.msg 解析（msg.go）、MSG 解释器（msgvm.go）与对话框绘制。
//
// 对话框绘制是 M5（front.anm 与 face 脚本）之前的过渡实现，按 MSG 的 View 状态画：
//   - 对话框只占 Field 横向区域，高 96 px，贴在 FieldBottom；顶部全透明、底部接近不透明黑的纵向渐变
//   - 双方立绘（op 1/2 显示）站在 Field 底边、压在对话框后面：自机在左，敌方在右并水平镜像；
//     说话方正常亮度，另一方变暗（op 17 两方都暗）
//   - 两行文本，颜色按说话方：自机 0xf8f08f，敌方 0x8088ff（MSG 构造函数 FUN_00415b00）
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
	textPadding      = 24  // 正文距对话框左边的内边距
	lineSpacing      = 28  // 两行文本的行距
	dimLevel         = 0.5 // 非说话方立绘的亮度
)

// 文本颜色（MSG+0x88/+0x8c），阴影统一用深色。
var (
	sideColours = [2]color.NRGBA{{0xf8, 0xf0, 0x8f, 255}, {0x80, 0x88, 0xff, 255}}
	textShadow  = color.NRGBA{0, 0, 0, 200}
)

// Line 一行手写对话（scripts.go 的旧台词表用）
type Line struct {
	Speaker    string // 说话者键（与 speakerFaces 表对齐）
	Expression string // face 表情后缀，例如 "no"/"an"/"sp"。空值视为 "no"
	Text       string // 对话内容
	IsRight    bool   // true=右侧（敌方），false=左侧（自机）
}

var textFace ebitext.Face

// Draw 按 MSG 的 View 绘制立绘、对话框和文本。
func Draw(screen *ebiten.Image, v *View) {
	if textFace == nil {
		textFace = render.LoadSystemFace(18)
	}

	// 1. 立绘：先画听话方，再画说话方。
	sides := [2]int{1 - v.Speaker, v.Speaker}
	for _, side := range sides {
		drawPortrait(screen, v, side)
	}

	// 2. 渐变背景：1px 横条堆叠，alpha 从顶 0 到底 gradientAlphaMax。
	if v.Box {
		for i := 0; i < int(boxHeight); i++ {
			a := uint8(float32(i) / boxHeight * gradientAlphaMax)
			vector.FillRect(screen, boxLeft, boxTop+float32(i), boxWidth, 1,
				color.NRGBA{0, 0, 0, a}, false)
		}
	}

	// 3. 两行文本。
	for i, l := range v.Lines {
		if l.Text == "" {
			continue
		}
		render.DrawTextShadow(screen, l.Text, textFace,
			int(boxLeft)+textPadding, int(boxTop)+22+i*lineSpacing,
			sideColours[l.Side&1], textShadow, 1, 1)
	}
}

// drawPortrait 画一方的立绘（side 0 自机，1 敌方），底边对齐 Field 底边。
func drawPortrait(screen *ebiten.Image, v *View, side int) {
	shown, speaker, id := v.Player, v.PlayerName, v.PlayerFace
	if side == 1 {
		shown, speaker, id = v.Boss, v.BossName, v.BossFace
	}
	if !shown {
		return
	}
	face := getFace(speaker, msgFaceExpression(int(id)))
	if face == nil {
		return
	}
	h := float64(face.Bounds().Dy())
	op := &ebiten.DrawImageOptions{}
	if side == 1 {
		op.GeoM.Scale(-1, 1)
		op.GeoM.Translate(float64(boxRight), float64(boxBottom)-h)
	} else {
		op.GeoM.Translate(float64(boxLeft), float64(boxBottom)-h)
	}
	if v.Dim || side != v.Speaker {
		op.ColorScale.Scale(dimLevel, dimLevel, dimLevel, 1)
	}
	screen.DrawImage(face, op)
}
