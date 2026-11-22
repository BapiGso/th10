package title

import (
	"fmt"
	"github.com/hajimehoshi/ebiten/v2"
	"th10/game"
	"th10/gamedata"
	"th10/input"
	"th10/replay"
	"th10/scene/menus"
	selector "th10/scene/select"
	"th10/scene/stage"
	stagescript "th10/scene/stage/script"
	sceneui "th10/scene/ui"
	"th10/supervisor"
)

// Title is the game.Scene adapter for DAT_0047784c. Every native menu
// page uses Controller's single world and selector stack.
type Title struct {
	Controller *sceneui.TitleController
}

func New() *Title                      { return &Title{} }
func (*Title) SceneMode() game.SceneID { return game.SceneTitle }
func (t *Title) SceneEnter(g *game.Game) error {
	if t.Controller != nil {
		return nil
	}
	c := sceneui.NewTitleController(g, g.TitleEntry)
	t.Controller = c
	// The replay, player data and music room pages run as their own scenes
	// (t11's public menus), entered through scene request 4.
	for _, page := range []sceneui.TitlePage{sceneui.TitleReplay, sceneui.TitlePlayerData, sceneui.TitleMusicRoom, sceneui.TitlePlayerDataDetail, sceneui.TitlePlayerDataCards} {
		c.Hooks[page] = sceneui.TitlePageHooks{Calc: func(_ *sceneui.TitleController, g *game.Game, _ *input.State) error {
			g.GoToFactory(game.SceneTitle, func(g *game.Game) (game.Scene, error) { return t.pageScene(g, page), nil })
			return nil
		}}
	}
	c.SetRunHooks(func(g *game.Game, state *game.GameState) error { return selector.Launch(g, state, t) }, t.startDemo)
	return nil
}

// pageScene is the scene of a title page that runs as its own scene.
func (t *Title) pageScene(g *game.Game, page sceneui.TitlePage) game.Scene {
	switch page {
	case sceneui.TitleReplay:
		return menus.NewReplay(t)
	case sceneui.TitleMusicRoom:
		return menus.NewMusicRoom(t)
	default:
		return menus.NewPlayerData(g.Save, t)
	}
}
func (t *Title) SceneLeave(g *game.Game, _ game.SceneID) {
	if t.Controller != nil {
		t.Controller.Remember(g)
		t.Controller.Dispose()
		t.Controller = nil
	}
}
func (t *Title) Update(g *game.Game) error {
	if err := t.SceneEnter(g); err != nil {
		return err
	}
	return t.Controller.Calc(g, &input.Global)
}
func (t *Title) Draw(screen *ebiten.Image) {
	if t.Controller != nil {
		t.Controller.Draw(screen)
	}
}
func (t *Title) DrawPorted(sv *supervisor.Supervisor) {
	if t.Controller != nil {
		t.Controller.DrawPorted(sv)
	}
}

// startDemo is FUN_0042cdf0, 0x42ce4a..0x42cf31. The first populated
// chapter and the 1/2/3/0 table order come from the original resources.
func (t *Title) startDemo(g *game.Game, index int) error {
	number := [...]int{1, 2, 3, 0}[index&3]
	data, err := gamedata.ReadFile(fmt.Sprintf("misc/demo%d.rpy", number))
	if err != nil {
		return err
	}
	r, err := replay.Decode(data)
	if err != nil {
		return err
	}
	first := 0
	for first < len(r.Chapters) && len(r.Chapters[first].Records) == 0 {
		first++
	}
	if first == len(r.Chapters) {
		return fmt.Errorf("title: demo%d has no populated chapter", number)
	}
	g.GoToFactory(game.ScenePlayback, func(g *game.Game) (game.Scene, error) {
		return stage.NewDemoPlayback(g, r, first, stagescript.New, t)
	})
	return nil
}
