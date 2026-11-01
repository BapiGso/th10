package stage

import (
	"image"
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/vector"

	"th10/anm"
	"th10/game"
	"th10/scene/dialog"
	"th10/sprite"
	"th10/supervisor"
)

// legacy is where a legacy node draws: the back buffer (sv.Legacy()) clipped
// to the viewport of the camera in use (sv.Cur.Viewport), in back-buffer
// coordinates, and the offset an ANM quad would get now (sv.ANM.Shift).
func legacy(sv *supervisor.Supervisor) (dst *ebiten.Image, dx, dy float64) {
	v := sv.Cur.Viewport
	r := image.Rect(int(v.X), int(v.Y), int(v.X+v.W), int(v.Y+v.H))
	dst = sv.Legacy().SubImage(r).(*ebiten.Image)
	return dst, float64(sv.ANM.Shift[0]), float64(sv.ANM.Shift[1])
}

// layers returns a node that draws World layers ls in order, each object
// through sprite.DrawVM onto legacy's target.
func layers(ls ...int) func(*Stage, *supervisor.Supervisor) {
	return func(s *Stage, sv *supervisor.Supervisor) {
		dst, dx, dy := legacy(sv)
		for _, l := range ls {
			s.ctx.World.Each(l, func(vm *anm.VM) {
				sprite.DrawVM(dst, vm, dx, dy)
			})
		}
	}
}

// nodeBackground is node 0x07, STD's background. BgDraw draws the field's
// offscreen image, which is copied to the field's place on screen shifted by
// Field.Shift: STD copies that shift into its camera, so the background
// shakes. The copy is clipped to the field rectangle, not to the camera
// viewport.
func (s *Stage) nodeBackground(sv *supervisor.Supervisor) {
	frame := sv.Legacy()
	s.field.Clear()
	s.script.BgDraw(s.field)
	dst := frame.SubImage(image.Rect(game.FieldLeft, game.FieldTop, game.FieldRight, game.FieldBottom)).(*ebiten.Image)
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(game.FieldLeft+float64(sv.Field.Shift[0]), game.FieldTop+float64(sv.Field.Shift[1]))
	dst.DrawImage(s.field, op)
}

// nodeEnemies is node 0x14.
func (s *Stage) nodeEnemies(sv *supervisor.Supervisor) {
	s.drawEnemies(legacy(sv))
}

// nodePlayer is node 0x16: the player's bullets, options, the player and the
// effects, in that order. The order is a guess until R6-F traces the call
// sites.
func (s *Stage) nodePlayer(sv *supervisor.Supervisor) {
	dst, dx, dy := legacy(sv)
	s.drawPlayerBullets(dst, dx, dy)
	s.drawPlayerOptions(dst, dx, dy)
	s.drawPlayer(dst, dx, dy)
	s.drawEffects(dst, dx, dy)
}

// nodeItems is node 0x19.
func (s *Stage) nodeItems(sv *supervisor.Supervisor) {
	s.drawItems(legacy(sv))
}

// nodeLasers is node 0x1b.
func (s *Stage) nodeLasers(sv *supervisor.Supervisor) {
	s.drawLasers(legacy(sv))
}

// nodeBullets is node 0x1d.
func (s *Stage) nodeBullets(sv *supervisor.Supervisor) {
	s.drawBullets(legacy(sv))
}

// nodeGUI is node 0x2b. Cur is Screen here, so dst is the whole back buffer
// and the offset is zero. The boss HUD is drawn on the field's image in
// field coordinates and copied to the field's place, unshaken.
func (s *Stage) nodeGUI(sv *supervisor.Supervisor) {
	dst, _, _ := legacy(sv)
	s.field.Clear()
	s.drawBossHUD(s.field)
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(game.FieldLeft, game.FieldTop)
	dst.DrawImage(s.field, op)

	// Dialogue (portraits and the box are inside the field), the front frame, then the HUD text.
	if s.ctx.msg != nil {
		dialog.Draw(dst, &s.ctx.msg.View)
	}
	if s.frontFrame != nil {
		dst.DrawImage(s.frontFrame, nil)
	}
	s.hud.draw(dst)
}

// nodePause is node 0x2e, the pause mask. Its place in the chain is a guess
// until FUN_00422150 is traced.
func (s *Stage) nodePause(sv *supervisor.Supervisor) {
	if !s.ctx.State.Paused {
		return
	}
	dst, _, _ := legacy(sv)
	vector.FillRect(dst, 0, 0, game.ScreenWidth, game.ScreenHeight,
		color.RGBA{0, 0, 0, 128}, false)
	ebitenutil.DebugPrintAt(dst, "PAUSED  (Esc to resume)", game.ScreenWidth/2-80, game.ScreenHeight/2)
}
