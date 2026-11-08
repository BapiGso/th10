package stage

import (
	"testing"

	"th10/game"
)

func TestSceneLifecycleRetainsOnlyPostEndingPicture(t *testing.T) {
	s := NewHeadless(game.NewState(0, 0, 1), &recordScript{})
	n := s.ctx.World.Len()
	if n == 0 {
		t.Fatal("fixture must have native world objects")
	}
	s.SceneLeave(&game.Game{}, game.SceneEnding)
	if s.ctx.World.Len() != n {
		t.Fatal("Ending.After's frozen stage picture was destroyed")
	}
	s.SceneLeave(&game.Game{}, game.SceneTitle)
	if s.ctx.World.Len() != 0 || s.ctx.Menu.world.Len() != 0 {
		t.Fatal("scene destructor did not release animation worlds")
	}
}
