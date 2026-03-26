package option

import (
	"fmt"
	"image/color"
	"th10/game"
	"th10/input"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

const (
	optHint = iota
	optBGMVolume
	optSEVolume
	optKeyConfig
	optDefault
	optQuit
	optCount
)

var optLabels = [optCount]string{
	"Hint",
	"BGM Volume",
	"SE Volume",
	"Key Config",
	"Default",
	"Quit",
}

// Option 设置场景
type Option struct {
	cursor int
	g      *game.Game
	back   game.Scene // 返回的场景（通常是 Title）
}

func New(g *game.Game, back game.Scene) *Option {
	return &Option{g: g, back: back}
}

func (o *Option) Update(g *game.Game) error {
	in := &input.Global
	cfg := g.Config

	if in.JustPressed(input.KeyUp) {
		o.cursor = (o.cursor - 1 + optCount) % optCount
	}
	if in.JustPressed(input.KeyDown) {
		o.cursor = (o.cursor + 1) % optCount
	}

	switch o.cursor {
	case optHint:
		if in.JustPressed(input.KeyOk) || in.JustPressed(input.KeyLeft) || in.JustPressed(input.KeyRight) {
			cfg.Hint = !cfg.Hint
		}

	case optBGMVolume:
		if in.JustPressed(input.KeyLeft) || (in.IsPressed(input.KeyLeft) && in.HoldFrames(input.KeyLeft)%4 == 0) {
			cfg.BGMVolume -= 0.05
			if cfg.BGMVolume < 0 {
				cfg.BGMVolume = 0
			}
		}
		if in.JustPressed(input.KeyRight) || (in.IsPressed(input.KeyRight) && in.HoldFrames(input.KeyRight)%4 == 0) {
			cfg.BGMVolume += 0.05
			if cfg.BGMVolume > 1 {
				cfg.BGMVolume = 1
			}
		}

	case optSEVolume:
		if in.JustPressed(input.KeyLeft) || (in.IsPressed(input.KeyLeft) && in.HoldFrames(input.KeyLeft)%4 == 0) {
			cfg.SEVolume -= 0.05
			if cfg.SEVolume < 0 {
				cfg.SEVolume = 0
			}
		}
		if in.JustPressed(input.KeyRight) || (in.IsPressed(input.KeyRight) && in.HoldFrames(input.KeyRight)%4 == 0) {
			cfg.SEVolume += 0.05
			if cfg.SEVolume > 1 {
				cfg.SEVolume = 1
			}
		}

	case optKeyConfig:
		if in.JustPressed(input.KeyOk) {
			// TODO: 进入按键配置子场景
		}

	case optDefault:
		if in.JustPressed(input.KeyOk) {
			def := game.DefaultConfig()
			*cfg = *def
		}

	case optQuit:
		if in.JustPressed(input.KeyOk) {
			// TODO: 保存配置到文件
			g.GoTo(o.back)
		}
	}

	// X/Esc 也可退出
	if in.JustPressed(input.KeyCancel) {
		g.GoTo(o.back)
	}

	return nil
}

func (o *Option) Draw(screen *ebiten.Image) {
	cfg := o.g.Config
	screen.Fill(color.RGBA{12, 8, 32, 255})

	ebitenutil.DebugPrintAt(screen, "Option", 280, 60)

	for i := 0; i < optCount; i++ {
		x := 180
		y := 140 + i*32
		label := "  " + optLabels[i]

		if i == o.cursor {
			label = "> " + optLabels[i]
			vector.FillRect(screen, float32(x-4), float32(y-2), 280, 22,
				color.RGBA{60, 40, 120, 180}, false)
		}

		// 右侧显示当前值
		var value string
		switch i {
		case optHint:
			if cfg.Hint {
				value = "ON"
			} else {
				value = "OFF"
			}
		case optBGMVolume:
			value = fmt.Sprintf("%s %3d%%", volumeBar(cfg.BGMVolume), int(cfg.BGMVolume*100))
		case optSEVolume:
			value = fmt.Sprintf("%s %3d%%", volumeBar(cfg.SEVolume), int(cfg.SEVolume*100))
		case optKeyConfig:
			value = ">"
		case optDefault:
			value = ""
		case optQuit:
			value = ""
		}

		ebitenutil.DebugPrintAt(screen, label, x, y)
		if value != "" {
			ebitenutil.DebugPrintAt(screen, value, x+160, y)
		}
	}

	ebitenutil.DebugPrintAt(screen, "Left/Right: Adjust   Z: Select   X: Back", 140, 420)
}

// volumeBar 生成一个简单的音量条
func volumeBar(v float64) string {
	bars := int(v * 20)
	s := "["
	for i := 0; i < 20; i++ {
		if i < bars {
			s += "|"
		} else {
			s += " "
		}
	}
	s += "]"
	return s
}
