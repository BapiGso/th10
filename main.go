package main

import (
	"embed"
	"log"

	"github.com/hajimehoshi/ebiten/v2"
)

//go:embed assets
var assets embed.FS

func main() {
	ebiten.SetWindowSize(ScreenWidth, ScreenHeight)
	ebiten.SetWindowTitle("东方风神录　～ Mountain of Faith. ver alpha")
	if err := ebiten.RunGame(&Game{}); err != nil {
		log.Fatal(err)
	}
}
