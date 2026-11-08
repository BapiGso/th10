package game

func (g *Game) syncHighScore(state *GameState) {
	if g.Save == nil || state == nil {
		return
	}
	state.HiScore = max(state.HiScore, state.Score) // FUN_00421f60, completion/menu commit
	g.Save.HighScore = max(g.Save.HighScore, state.HiScore)
}

func (g *Game) BeginRun(state *GameState) {
	if state == nil || state.ReplayMode == 1 {
		return
	}
	if g.Save == nil {
		g.Save = DefaultSaveData()
	}
	g.BindState(state)
	if state.StartStage == 0 {
		state.StartStage = state.Stage
	}
	if state.FreshRun {
		state.ResetRank(state.RunFlags&RunFlagContinued != 0) // FUN_00417870, 0x4179cc
		top := g.ScoreTable(state)[0]                         // FUN_00417870, 0x4178f8..0x417916
		state.HiScore, state.HiScoreContinues = int64(top.Score)*10, int32(int8(top.Continues))
		state.RunFlags &^= RunFlagHighScore // 0x417992
	}
	g.syncHighScore(state)
	state.ExtraUnlocked = g.Save.ExtraUnlocked
	if state.DebugMode {
		state.ExtraUnlocked = true
		return
	}
	if state.StageMode == 0 && state.FreshRun {
		if p := state.shotProgress(); p != nil && p.Runs < 99999 {
			p.Runs++ // FUN_00417870, 0x4179a5..0x4179c4
		}
	}
	g.Save.TotalRuns++
	g.Save.LastCharacter, g.Save.LastShotType, g.Save.LastDifficulty = state.Character, state.ShotType, state.Difficulty
	_ = g.SaveProgress()
}

func (g *Game) MarkGameOver(state *GameState) {
	if state != nil && state.ReplayMode == 1 {
		return
	}
	if g.Save == nil {
		g.Save = DefaultSaveData()
	}
	g.syncHighScore(state)
	if state != nil {
		state.ExtraUnlocked = g.Save.ExtraUnlocked
	}
	g.Save.TotalGameOvers++
	_ = g.SaveProgress()
}

func (g *Game) MarkClear(state *GameState) {
	if state != nil && state.ReplayMode == 1 {
		return
	}
	if g.Save == nil {
		g.Save = DefaultSaveData()
	}
	g.syncHighScore(state)
	if state != nil && state.RunFlags&RunFlagPractice == 0 {
		switch state.Stage {
		case StageExtra:
			g.Save.TotalExtraClears++
		case StageCount:
			g.Save.TotalClears++
			if state.Difficulty >= DiffNormal {
				g.Save.ExtraUnlocked = true
			}
		}
	}
	if state != nil {
		state.ExtraUnlocked = g.Save.ExtraUnlocked
	}
	_ = g.SaveProgress()
}
