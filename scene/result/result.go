// Package result is a host fallback summary, reached only when a caller
// supplies no native continuation (Stage.NextScene/OnGameOver or
// Ending.After/ResolveScene). The native flow uses the FUN_00423370
// clear/ranking menus instead, so this screen has no original counterpart.
// It draws with the embedded W3 font through render.RasterText: no OS font
// files and no UI toolkit, which keeps both out of the WASM build.
package result

import (
	"fmt"
	"image"
	"image/color"
	"image/draw"

	"th10/audio"
	"th10/game"
	"th10/input"
	"th10/render"

	"github.com/hajimehoshi/ebiten/v2"
)

type Mode int

const (
	ModeGameOver Mode = iota
	ModeStageClear
	ModeExtraClear
)

type Result struct {
	state      game.GameState
	mode       Mode
	frame      int
	bgmStarted bool
	back       game.Scene
	page       *ebiten.Image
}

func New(state *game.GameState, mode Mode, back game.Scene) *Result {
	var snap game.GameState
	if state != nil {
		snap = *state
	}
	return &Result{state: snap, mode: mode, back: back}
}

func (r *Result) heading() string {
	switch r.mode {
	case ModeGameOver:
		return "Game Over"
	case ModeExtraClear:
		return "Extra Clear"
	default:
		return "Stage Clear"
	}
}

const (
	textColor = 0xf4eee4
	highColor = 0xfff5d6
	hintColor = 0xe4ded6
)

// render lays out the same rows the former widget panel showed.
func (r *Result) render() *image.NRGBA {
	img := image.NewNRGBA(image.Rect(0, 0, game.ScreenWidth, game.ScreenHeight))
	draw.Draw(img, img.Rect, image.NewUniform(color.NRGBA{10, 14, 30, 255}), image.Point{}, draw.Src)
	panel := image.Rect(50, 40, game.ScreenWidth-50, game.ScreenHeight-40)
	draw.Draw(img, panel, image.NewUniform(color.NRGBA{0, 0, 0, 170}), image.Point{}, draw.Over)
	put := func(s string, x, y, size int, c uint32) {
		t := render.RasterText(s, panel.Max.X-x, size+8, render.TextStyle{Size: size, Color: c})
		draw.Draw(img, image.Rect(x, y, x+t.Rect.Dx(), y+t.Rect.Dy()), t, image.Point{}, draw.Over)
	}
	x, y := panel.Min.X+32, panel.Min.Y+32
	put(r.heading(), x, y, 28, highColor)
	y += 48
	row := func(k, v string) {
		put(k, x, y, 20, textColor)
		put(v, x+180, y, 20, highColor)
		y += 28
	}
	row("Score", fmt.Sprintf("%012d", r.state.Score))
	row("HiScore", fmt.Sprintf("%012d", r.state.HiScore))
	row("Stage", game.StageName(r.state.Stage))
	row("Difficulty", game.DifficultyName(r.state.Difficulty))
	row("Player", fmt.Sprintf("%s / %s", game.CharacterName(r.state.Character), game.ShotName(r.state.Character, r.state.ShotType)))
	row("Power", fmt.Sprintf("%d.%02d", r.state.Power/100, r.state.Power%100))
	row("Point", fmt.Sprintf("%d", r.state.Point))
	row("Faith", fmt.Sprintf("%d", r.state.Faith))
	if r.state.ExtraUnlocked {
		y += 6
		put("Extra Stage 已解锁", x, y, 20, highColor)
		y += 28
	}
	put("Press Z or X to return to Title", x, y+6, 16, hintColor)
	return img
}

func (r *Result) Update(g *game.Game) error {
	r.frame++
	if !r.bgmStarted {
		g.Audio().PlayBGM(audio.BGMResult)
		r.bgmStarted = true
	}
	if r.frame > 15 && (input.Global.JustPressed(input.KeyOk) || input.Global.JustPressed(input.KeyCancel)) {
		g.GoTo(r.back)
	}
	return nil
}

func (r *Result) Draw(screen *ebiten.Image) {
	if r.page == nil {
		r.page = ebiten.NewImageFromImage(r.render())
	}
	screen.DrawImage(r.page, nil)
}
