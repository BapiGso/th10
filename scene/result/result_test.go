package result

import (
	"testing"

	"th10/game"
)

// The fallback summary renders from embedded fonts only; it must not need an
// OS font file or a UI toolkit, and it keeps the former rows and hint line.
func TestRenderDrawsTextOnPanel(t *testing.T) {
	st := game.GameState{Score: 123456780, Stage: 3, Power: 250, ExtraUnlocked: true}
	img := New(&st, ModeGameOver, nil).render()
	if img.Rect.Dx() != game.ScreenWidth || img.Rect.Dy() != game.ScreenHeight {
		t.Fatalf("page size %v", img.Rect)
	}
	bright := 0
	for at := 0; at < len(img.Pix); at += 4 {
		if img.Pix[at] > 200 && img.Pix[at+1] > 200 && img.Pix[at+2] > 200 {
			bright++
		}
	}
	if bright < 500 {
		t.Fatalf("only %d text pixels drawn", bright)
	}
}
