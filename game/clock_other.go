//go:build !js

package game

import "github.com/hajimehoshi/ebiten/v2"

func platformTPS() int { return ebiten.SyncWithFPS }

func (g *Game) logicFrameReady(now float64) bool { return g.pace.Ready(now) }
