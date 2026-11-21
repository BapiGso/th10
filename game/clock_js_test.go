//go:build js

package game

import (
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
)

func TestBrowserFixedTPSHasNoSecondPacer(t *testing.T) {
	ebiten.SetTPS(platformTPS())
	if ebiten.TPS() != 60 {
		t.Fatal("browser logic is not configured for 60 TPS")
	}
	g := &Game{}
	for _, now := range []float64{0, 0, .001, .001, .002, 1} {
		if !g.logicFrameReady(now) {
			t.Fatal("browser fixed ticks were filtered again by desktop pacing")
		}
	}
}
