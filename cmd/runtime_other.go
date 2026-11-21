//go:build !js

package main

import (
	"th10/game"

	"github.com/hajimehoshi/ebiten/v2"
)

func preparePlatform() error { return nil }

func runPlatformGame(g *game.Game) error { return ebiten.RunGame(g) }
