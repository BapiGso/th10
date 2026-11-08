package stage

import (
	"image"
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/vector"

	"th10/game"
	"th10/supervisor"
)

// nodeBackground is node 0x07, STD's background. With s.bg (a stage in
// stdConvertible) it is stddraw's Node7 (FUN_00402850). Otherwise BgDraw
// draws the field's offscreen image, which is copied to the field's place on
// screen shifted by Field.Shift: STD copies that shift into its camera, so
// the background shakes. The copy is clipped to the field rectangle, not to
// the camera viewport.
func (s *Stage) nodeBackground(sv *supervisor.Supervisor) {
	if s.bg != nil {
		s.bg.Node7(sv)
		return
	}
	frame := sv.Legacy()
	s.field.Clear()
	s.script.BgDraw(s.field)
	dst := frame.SubImage(image.Rect(game.FieldLeft, game.FieldTop, game.FieldRight, game.FieldBottom)).(*ebiten.Image)
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(game.FieldLeft+float64(sv.Field.Shift[0]), game.FieldTop+float64(sv.Field.Shift[1]))
	dst.DrawImage(s.field, op)
}

// drawLegacyPause preserves the host pause display outside the original
// chain. Node 0x2e is DAT_00477830's end menu, not this untraced pause flag.
func (s *Stage) drawLegacyPause(dst *ebiten.Image) {
	if !s.ctx.State.Paused {
		return
	}
	vector.FillRect(dst, 0, 0, game.ScreenWidth, game.ScreenHeight,
		color.RGBA{0, 0, 0, 128}, false)
	ebitenutil.DebugPrintAt(dst, "PAUSED  (Esc to resume)", game.ScreenWidth/2-80, game.ScreenHeight/2)
}
