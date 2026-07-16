package loading

import (
	"image/color"
	_ "image/png"
	"th10/game"
	"th10/render"
	"th10/scene/title"
	"th10/sprite"

	"github.com/hajimehoshi/ebiten/v2"
)

// Loading 启动加载场景。背景由 sig.png + sig_r.png 拼成 640x480；
// 底部"少女祈祷中"提示直接用 anm/ascii/loading.png 整图（128x128，
// 内含文字 + 飘落的枫叶），不再做文字渲染或粒子动画。
type Loading struct {
	frame      int
	bgImg      *ebiten.Image // sig.png 左侧 512x480
	bgRImg     *ebiten.Image // sig_r.png 右侧 128x480
	loadingImg *ebiten.Image // anm/ascii/loading.png 整图
}

const (
	totalFrames   = 180 // 3 秒后切到 Title
	loadingMargin = 16  // loading.png 距屏幕右下角的留白
)

func New() *Loading {
	l := &Loading{}
	l.bgImg = render.LoadImage("anm/loading/sig.png")
	l.bgRImg = render.LoadImage("anm/loading/sig_r.png")
	l.loadingImg = render.LoadImage("anm/ascii/loading.png")
	if l.bgImg == nil {
		l.bgImg = ebiten.NewImage(1, 1)
	}
	if l.bgRImg == nil {
		l.bgRImg = ebiten.NewImage(1, 1)
	}
	return l
}

func (l *Loading) Update(g *game.Game) error {
	l.frame++
	if l.frame >= totalFrames {
		sprite.Load()
		g.GoTo(title.New())
	}
	return nil
}

func (l *Loading) Draw(screen *ebiten.Image) {
	screen.Fill(color.RGBA{0, 0, 0, 255})

	// 0~60 帧淡入，120~180 帧淡出。
	alpha := float64(l.frame) / 60
	if alpha > 1 {
		alpha = 1
	}
	if l.frame > 120 {
		alpha = float64(totalFrames-l.frame) / 60
	}
	if alpha < 0 {
		alpha = 0
	}

	// 背景 sig.png(512x480) + sig_r.png(128x480) 拼成 640x480 居中。
	bh := l.bgImg.Bounds().Dy()
	totalW := float64(l.bgImg.Bounds().Dx() + l.bgRImg.Bounds().Dx())
	ox := (float64(game.ScreenWidth) - totalW) / 2
	oy := (float64(game.ScreenHeight) - float64(bh)) / 2

	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(ox, oy)
	op.ColorScale.ScaleAlpha(float32(alpha))
	screen.DrawImage(l.bgImg, op)

	op2 := &ebiten.DrawImageOptions{}
	op2.GeoM.Translate(ox+float64(l.bgImg.Bounds().Dx()), oy)
	op2.ColorScale.ScaleAlpha(float32(alpha))
	screen.DrawImage(l.bgRImg, op2)

	// loading.png 整图贴右下角（图内已含"少女祈祷中"+枫叶）。
	if l.loadingImg != nil {
		lw, lh := l.loadingImg.Bounds().Dx(), l.loadingImg.Bounds().Dy()
		x := game.ScreenWidth - lw - loadingMargin
		y := game.ScreenHeight - lh - loadingMargin
		op3 := &ebiten.DrawImageOptions{}
		op3.GeoM.Translate(float64(x), float64(y))
		op3.ColorScale.ScaleAlpha(float32(alpha))
		screen.DrawImage(l.loadingImg, op3)
	}
}
