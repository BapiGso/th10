package stage

import (
	"th10/game"
	resultscene "th10/scene/result"
)

// GUIController is the controller portion of DAT_0047770c. Logic-only
// Contexts retain it without allocating GUI resources or running their VMs.
type GUIController struct {
	Flags        uint32 // GUI+0x9eb4, controller bits 0x10/0x20
	ClearTimer   int32  // GUI+0x9ecc
	SceneRequest int32  // DAT_00491fb8, delivered through the existing Game.Scene dispatcher
	ClearMenu    bool   // host delivery of MSG's direct FUN_00423370 call, not an inferred mode bit
}

func (c *GUIController) request(scene int32, systemFlags uint32) {
	if systemFlags&0x1000 != 0 { // FUN_0040ac90 and FUN_00414900, DAT_00491ff4
		scene = 2
	}
	c.SceneRequest = scene
}

// TickClear is the first branch of FUN_00414900 (0x414906..0x414937),
// before it executes MSG. The clear opcode's own frame therefore stays at 0.
func (c *GUIController) TickClear(systemFlags uint32) {
	if c.Flags&0x10 != 0 {
		c.ClearTimer++
		if c.ClearTimer >= 120 {
			c.request(0xe, systemFlags)
		}
	}
}

func (c *GUIController) completed() bool {
	return c.Flags&0x20 != 0 || c.SceneRequest != 0 || c.ClearMenu
}

func (c *Context) systemFlags() uint32 {
	// DAT_00491ff4 is read at the request, not snapshotted at stage start.
	if c.stage != nil && c.stage.liveGame != nil {
		return c.stage.liveGame.SystemFlags
	}
	return c.State.SystemFlags
}

// clearStage is FUN_00415e90 case 0x14. GUI flags, not a Context boolean,
// own completion; the scene layer only delivers the controller's request.
func (c *Context) clearStage() {
	s, ctrl := c.State, c.GUIController
	if ctrl == nil {
		ctrl = &GUIController{}
		c.GUIController = ctrl
	}
	s.RecordStageClear() // 0x416c08 / 0x416c39, also in practice and replay
	if s.RunFlags&game.RunFlagPractice != 0 {
		if s.StageMode == 1 { // FUN_00423370's playback branch
			ctrl.request(4, c.systemFlags())
		} else {
			ctrl.ClearMenu = true
			c.deliverClearMenu()
		}
		return
	}
	if s.Stage != game.StageCount && s.Stage != game.StageExtra {
		ctrl.request(0xb, c.systemFlags()) // FUN_004175e0 advances the stage in the Scene adapter
		return
	}
	ctrl.Flags |= 0x20 // 0x416c69 / 0x416df6: disable score extends before adding bonuses
	s.AwardClearBonus()
	if s.ReplayMode == 1 {
		ctrl.request(4, c.systemFlags()) // 0x416e8a, final replay returns instead of ending/menu
		return
	}
	if s.Stage == game.StageExtra {
		ctrl.ClearMenu = true
		c.deliverClearMenu() // 0x416e55, no 120-frame ending wait for Extra
	} else {
		if c.RequestShake != nil {
			c.RequestShake(5, 120, 0, 0, 0) // FUN_0043c8b0, 0x416d85; t7 owns the sampler/lifetime
		}
		ctrl.Flags |= 0x10
		ctrl.ClearTimer = 0 // 0x416d95..0x416da2
	}
	s.RecordFinalClear() // 0x416dd2, after practice/replay have already exited
	if s.Stage == game.StageExtra && c.stage != nil && c.stage.liveGame != nil {
		_ = c.stage.liveGame.SaveProgress() // persist the counter written after the direct menu open
	}
}

func (c *Context) deliverClearMenu() {
	if s := c.stage; s != nil && s.liveGame != nil && s.ctx.Menu != nil {
		s.openClearMenu(s.liveGame)
	}
}

func (s *Stage) openClearMenu(g *game.Game) {
	if s.ctx.Menu.Active() {
		return
	}
	s.finishReplayMetadata(s.ctx.State.Stage >= game.StageCount)
	if !s.ctx.State.DebugMode {
		g.MarkClear(s.ctx.State)
	}
	s.ctx.Menu.OpenClear(g)
}

func (s *Stage) deliverSceneRequest(g *game.Game) {
	state := s.ctx.State
	s.finishReplayMetadata(state.Stage >= game.StageCount)
	if state.ReplayMode != 1 && !state.DebugMode {
		_ = g.SaveProgress() // FUN_00417c80 saves before releasing/transferring the stage
	}
	switch s.ctx.GUIController.SceneRequest {
	case 0xb, 0xe:
		if s.NextScene != nil {
			g.GoTo(s.NextScene(g, state))
		} else {
			g.GoTo(resultscene.New(state, resultscene.ModeStageClear, nil))
		}
	default: // native 4 (replay exit) or system override 2
		s.goGameOver(g)
	}
}

// updateScoreController is called only at FUN_00418190 -> FUN_00417040,
// after the calc chain. GUI drawing/Numbers alone never grants resources.
func (s *Stage) updateScoreController() {
	c := s.ctx
	if c.GUI == nil {
		return
	}
	c.GUI.Numbers()
	oldCursor := c.State.ReplayState9C
	announce := c.State.UpdateScoreController(c.GUI.shownScore, c.GUIController.Flags&0x20 != 0)
	// FUN_00417040 writes the two continuation digits after its high-score
	// update; refresh their sprites without running the VMs a second time.
	c.GUI.hiScore[0].SetSprite(int(c.State.HiScoreContinues) + 8)
	if c.State.ReplayState9C != oldCursor {
		c.GUI.SetLives(c.State.Life) // FUN_004188a0 -> FUN_00413790, same frame
	}
	if announce {
		c.PlaySEID(0x2c)       // FUN_004188a0, 0x4188ce
		c.GUI.ShowNotice(4, 0) // FUN_00448d00 front script 0x4b, GUI+0x9e18
	}
}
