package menus

import (
	"github.com/hajimehoshi/ebiten/v2"
	"th10/game"
	"th10/input"
	sceneui "th10/scene/ui"
	"th10/supervisor"
)

// Option keeps the original public constructor while replacing the old
// slider/checkbox UI with FUN_0042d920/0042f540's native title pages.
type Option struct {
	Controller *sceneui.TitleController
	back       game.Scene
}

func NewOption(g *game.Game, back game.Scene) *Option {
	o := &Option{back: back}
	c := sceneui.NewTitleController(g, 1)
	o.Controller = c
	c.Selector.Count = 8
	c.Selector.Set(6)
	c.Selector.Push()
	c.SetState(sceneui.TitleOption)
	c.Spawn(0x58)
	return o
}
func (*Option) SceneMode() game.SceneID                   { return game.SceneTitle }
func (*Option) SceneEnter(*game.Game) error               { return nil }
func (o *Option) SceneLeave(_ *game.Game, _ game.SceneID) { o.Controller.Dispose() }
func (o *Option) Update(g *game.Game) error {
	if err := o.Controller.Calc(g, &input.Global); err != nil {
		return err
	}
	if o.Controller.State == sceneui.TitleMain {
		g.GoTo(o.back)
	}
	return nil
}
func (o *Option) Draw(screen *ebiten.Image)            { o.Controller.Draw(screen) }
func (o *Option) DrawPorted(sv *supervisor.Supervisor) { o.Controller.DrawPorted(sv) }
