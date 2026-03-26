package main

import (
	"log"
	"th10/game"
	"th10/scene/loading"

	"github.com/hajimehoshi/ebiten/v2"
)

func main() {
	ebiten.SetWindowSize(game.ScreenWidth, game.ScreenHeight)
	ebiten.SetWindowTitle("东方风神录 ~ Mountain of Faith")

	g := game.New()
	g.SetScene(loading.New())

	if err := ebiten.RunGame(g); err != nil {
		log.Fatal(err)
	}
}
