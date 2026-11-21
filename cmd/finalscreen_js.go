//go:build js

package main

import "github.com/hajimehoshi/ebiten/v2"

// Preserve Ebiten's aspect/centering transform, but never use a smoothing filter.
func (*gatedGame) DrawFinalScreen(screen ebiten.FinalScreen, offscreen *ebiten.Image, geoM ebiten.GeoM) {
	screen.DrawImage(offscreen, &ebiten.DrawImageOptions{GeoM: geoM, Filter: ebiten.FilterNearest})
}
