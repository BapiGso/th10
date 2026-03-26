package title

import (
	"bytes"
	"image"
	"image/color"
	_ "image/png"
	"th10/assets"
	"th10/audio"
	"th10/game"
	"th10/input"
	"th10/scene/option"
	selector "th10/scene/select"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

const (
	menuStart = iota
	menuExtraStart
	menuPractice
	menuReplay
	menuPlayerData
	menuMusicRoom
	menuOption
	menuQuit
	menuCount
)

var menuLabels = [menuCount]string{
	"Game Start",
	"Extra Start",
	"Practice Start",
	"Replay",
	"Player Data",
	"Music Room",
	"Option",
	"Quit",
}

// Title 标题场景
type Title struct {
	frame       int
	cursor      int
	extraUnlock bool // Extra 是否解锁
	bgmStarted bool
	bgImg       *ebiten.Image
	logoImg     *ebiten.Image
}

func New() *Title {
	t := &Title{}
	t.bgImg = loadImage("anm/title/title00s.png")
	t.logoImg = loadImage("anm/title/title_logo.png")
	return t
}

func loadImage(path string) *ebiten.Image {
	data, err := assets.Assets.ReadFile(path)
	if err != nil {
		return nil
	}
	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil
	}
	return ebiten.NewImageFromImage(img)
}

func (t *Title) Update(g *game.Game) error {
	t.frame++
	in := &input.Global

	if !t.bgmStarted {
		g.Audio().PlayBGM(audio.BGMTitle)
		t.bgmStarted = true
	}

	if in.JustPressed(input.KeyUp) {
		t.cursor = (t.cursor - 1 + menuCount) % menuCount
		g.Audio().PlaySE(audio.SESelect)
	}
	if in.JustPressed(input.KeyDown) {
		t.cursor = (t.cursor + 1) % menuCount
		g.Audio().PlaySE(audio.SESelect)
	}

	if in.JustPressed(input.KeyOk) {
		g.Audio().PlaySE(audio.SEOk)
		switch t.cursor {
		case menuStart:
			g.GoTo(selector.New(t))
		case menuExtraStart:
			if t.extraUnlock {
				g.GoTo(selector.NewExtra(t))
			}
		case menuPractice:
			// TODO: Practice Start → 选关 + 选人
		case menuReplay:
			// TODO: Replay 列表
		case menuPlayerData:
			// TODO: Player Data 查看
		case menuMusicRoom:
			// TODO: Music Room
		case menuOption:
			g.GoTo(option.New(g, t))
		case menuQuit:
			return ebiten.Termination
		}
	}

	return nil
}

func (t *Title) Draw(screen *ebiten.Image) {
	// 背景
	if t.bgImg != nil {
		screen.DrawImage(t.bgImg, nil)
	} else {
		screen.Fill(color.RGBA{12, 8, 32, 255})
	}

	// Logo 居中偏上
	if t.logoImg != nil {
		op := &ebiten.DrawImageOptions{}
		lw := t.logoImg.Bounds().Dx()
		op.GeoM.Translate(float64(640-lw)/2, 60)
		screen.DrawImage(t.logoImg, op)
	} else {
		ebitenutil.DebugPrintAt(screen, "東方風神録  ~ Mountain of Faith", 150, 80)
		ebitenutil.DebugPrintAt(screen, "Reimplemented with Ebiten", 190, 110)
	}

	// 菜单
	for i := 0; i < menuCount; i++ {
		x := 260
		y := 200 + i*24
		label := "  " + menuLabels[i]

		// Extra 未解锁时变灰
		dimmed := (i == menuExtraStart && !t.extraUnlock)
		_ = dimmed // TODO: 灰色字体

		if i == t.cursor {
			label = "> " + menuLabels[i]
			vector.FillRect(screen, float32(x-4), float32(y-2), 180, 20,
				color.RGBA{60, 40, 120, 180}, false)
		}
		ebitenutil.DebugPrintAt(screen, label, x, y)
	}

	// 底部提示
	ebitenutil.DebugPrintAt(screen, "Arrow Keys: Move   Z: Select   X: Cancel", 140, 440)

	// 闪烁的 Press Z
	if t.frame%60 < 40 {
		ebitenutil.DebugPrintAt(screen, "Press Z to Start", 250, 420)
	}
}
