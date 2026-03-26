package loading

import (
	"bytes"
	"image"
	"image/color"
	_ "image/png"
	"th10/assets"
	"th10/game"
	"th10/scene/title"
	"th10/sprite"
	"math"
	"math/rand"

	"github.com/hajimehoshi/ebiten/v2"
	"golang.org/x/image/font"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/math/fixed"
)

// leaf 枫叶粒子
type leaf struct {
	x, y   float64
	vx, vy float64
	rot    float64 // 旋转角度
	vr     float64 // 旋转速度
	scale  float64
	alpha  float64
}

// Loading 启动加载场景
type Loading struct {
	frame    int
	bgImg    *ebiten.Image // sig.png 左侧 512x480
	bgRImg   *ebiten.Image // sig_r.png 右侧 128x480
	leafImg  *ebiten.Image // leaf.png 枫叶贴图
	leaves   []leaf
	textFace font.Face // "少女祈祷中" 字体
}

const (
	maxLeaves   = 40
	leafSpawnCD = 6 // 每隔几帧生成一片叶子
	textStr     = "少女祈祷中"
	fontSize    = 20
)

func New() *Loading {
	l := &Loading{}
	l.loadAssets()
	l.initLeaves()
	return l
}

func (l *Loading) loadAssets() {
	// 加载背景图 (512x480 + 128x480 拼成 640x480)
	l.bgImg = loadImage("anm/loading/sig.png")
	l.bgRImg = loadImage("anm/loading/sig_r.png")
	// 加载枫叶贴图
	l.leafImg = loadImage("anm/ascii/leaf.png")
	// 加载字体
	l.textFace = loadFont("font/DFYuGaSo-W3.ttc", fontSize)
}

func loadImage(path string) *ebiten.Image {
	data, err := assets.Assets.ReadFile(path)
	if err != nil {
		return ebiten.NewImage(1, 1) // fallback
	}
	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return ebiten.NewImage(1, 1)
	}
	return ebiten.NewImageFromImage(img)
}

func loadFont(path string, size float64) font.Face {
	data, err := assets.Assets.ReadFile(path)
	if err != nil {
		return nil
	}
	// TTC (TrueType Collection) — 用 opentype 解析
	f, err := opentype.ParseCollection(data)
	if err != nil {
		return nil
	}
	face, err := f.Font(0)
	if err != nil {
		return nil
	}
	ff, err := opentype.NewFace(face, &opentype.FaceOptions{
		Size:    size,
		DPI:     72,
		Hinting: font.HintingFull,
	})
	if err != nil {
		return nil
	}
	return ff
}

func (l *Loading) initLeaves() {
	l.leaves = make([]leaf, 0, maxLeaves)
	// 预生成一些叶子，让画面开始就有
	for i := 0; i < maxLeaves/2; i++ {
		l.leaves = append(l.leaves, newLeaf(true))
	}
}

func newLeaf(randomY bool) leaf {
	sw := float64(game.ScreenWidth)
	sh := float64(game.ScreenHeight)
	y := -20.0
	if randomY {
		y = rand.Float64() * sh
	}
	return leaf{
		x:     rand.Float64() * sw,
		y:     y,
		vx:    rand.Float64()*0.6 - 0.3, // 轻微水平飘动
		vy:    0.5 + rand.Float64()*0.8,  // 缓慢下落
		rot:   rand.Float64() * math.Pi * 2,
		vr:    (rand.Float64() - 0.5) * 0.04,
		scale: 0.3 + rand.Float64()*0.4,
		alpha: 0.4 + rand.Float64()*0.4,
	}
}

func (l *Loading) Update(g *game.Game) error {
	l.frame++

	// 生成新枫叶
	if l.frame%leafSpawnCD == 0 && len(l.leaves) < maxLeaves {
		l.leaves = append(l.leaves, newLeaf(false))
	}

	// 更新枫叶
	alive := l.leaves[:0]
	for i := range l.leaves {
		lf := &l.leaves[i]
		lf.x += lf.vx
		lf.y += lf.vy
		lf.rot += lf.vr
		// 超出屏幕底部的叶子移除
		if lf.y < float64(game.ScreenHeight)+40 {
			alive = append(alive, *lf)
		}
	}
	l.leaves = alive

	// 3秒后切换到标题（加载 sprite 资源）
	if l.frame >= 180 {
		sprite.Load()
		g.GoTo(title.New())
	}
	return nil
}

func (l *Loading) Draw(screen *ebiten.Image) {
	screen.Fill(color.RGBA{0, 0, 0, 255})

	// 淡入淡出 alpha
	alpha := float64(l.frame) / 60
	if alpha > 1 {
		alpha = 1
	}
	if l.frame > 120 {
		alpha = float64(180-l.frame) / 60
	}
	if alpha < 0 {
		alpha = 0
	}

	// 1. 绘制背景图 (sig.png 512x480 + sig_r.png 128x480 = 640x480 居中)
	if l.bgImg != nil {
		bh := l.bgImg.Bounds().Dy()
		totalW := float64(l.bgImg.Bounds().Dx())
		if l.bgRImg != nil {
			totalW += float64(l.bgRImg.Bounds().Dx())
		}
		ox := (float64(game.ScreenWidth) - totalW) / 2
		oy := (float64(game.ScreenHeight) - float64(bh)) / 2

		op := &ebiten.DrawImageOptions{}
		op.GeoM.Translate(ox, oy)
		op.ColorScale.ScaleAlpha(float32(alpha))
		screen.DrawImage(l.bgImg, op)

		if l.bgRImg != nil {
			op2 := &ebiten.DrawImageOptions{}
			op2.GeoM.Translate(ox+float64(l.bgImg.Bounds().Dx()), oy)
			op2.ColorScale.ScaleAlpha(float32(alpha))
			screen.DrawImage(l.bgRImg, op2)
		}
	}

	// 2. 绘制飘落枫叶
	if l.leafImg != nil {
		lw, lh := l.leafImg.Bounds().Dx(), l.leafImg.Bounds().Dy()
		hx, hy := float64(lw)/2, float64(lh)/2
		for _, lf := range l.leaves {
			op := &ebiten.DrawImageOptions{}
			op.GeoM.Translate(-hx, -hy)
			op.GeoM.Scale(lf.scale, lf.scale)
			op.GeoM.Rotate(lf.rot)
			op.GeoM.Translate(lf.x, lf.y)
			op.ColorScale.ScaleAlpha(float32(lf.alpha * alpha))
			screen.DrawImage(l.leafImg, op)
		}
	}

	// 3. 绘制 "少女祈祷中" 文字（右下角）
	if l.textFace != nil {
		drawText(screen, textStr, l.textFace, alpha)
	}
}

func drawText(screen *ebiten.Image, s string, face font.Face, alpha float64) {
	// 测量文字宽度
	adv := font.MeasureString(face, s)
	tw := adv.Ceil()
	// 右下角，留一点边距
	x := game.ScreenWidth - tw - 20
	y := game.ScreenHeight - 24

	a := uint8(alpha * 255)
	col := color.RGBA{255, 255, 255, a}

	d := &font.Drawer{
		Dst:  screen,
		Src:  image.NewUniform(col),
		Face: face,
		Dot:  fixed.P(x, y),
	}
	d.DrawString(s)
}
