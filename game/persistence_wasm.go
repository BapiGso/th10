//go:build js && wasm

package game

// SaveData 持久化的全局进度数据。WASM demo 不落盘，运行期只保留内存状态。
type SaveData struct {
	HighScore        int64 `json:"high_score"`
	ExtraUnlocked    bool  `json:"extra_unlocked"`
	TotalRuns        int   `json:"total_runs"`
	TotalClears      int   `json:"total_clears"`
	TotalExtraClears int   `json:"total_extra_clears"`
	TotalGameOvers   int   `json:"total_game_overs"`
	LastCharacter    int   `json:"last_character"`
	LastShotType     int   `json:"last_shot_type"`
	LastDifficulty   int   `json:"last_difficulty"`
}

func DefaultSaveData() *SaveData {
	return &SaveData{
		LastDifficulty: DiffNormal,
	}
}

func LoadConfig() *Config {
	return DefaultConfig()
}

func LoadSaveData() *SaveData {
	return DefaultSaveData()
}

func (g *Game) SaveConfig() error {
	return nil
}

func (g *Game) SaveProgress() error {
	return nil
}

func (g *Game) syncHighScore(state *GameState) {
	if g.Save == nil || state == nil {
		return
	}
	if state.HiScore < g.Save.HighScore {
		state.HiScore = g.Save.HighScore
	}
	if state.Score > g.Save.HighScore {
		g.Save.HighScore = state.Score
		state.HiScore = state.Score
	}
}

func (g *Game) BeginRun(state *GameState) {
	if g.Save == nil {
		g.Save = DefaultSaveData()
	}
	if state == nil {
		return
	}
	g.syncHighScore(state)
	state.ExtraUnlocked = g.Save.ExtraUnlocked
	if state.DebugMode {
		state.ExtraUnlocked = true
		return
	}
	g.Save.TotalRuns++
	g.Save.LastCharacter = state.Character
	g.Save.LastShotType = state.ShotType
	g.Save.LastDifficulty = state.Difficulty
}

func (g *Game) MarkGameOver(state *GameState) {
	if g.Save == nil {
		g.Save = DefaultSaveData()
	}
	g.syncHighScore(state)
	if state != nil {
		state.ExtraUnlocked = g.Save.ExtraUnlocked
	}
	g.Save.TotalGameOvers++
}

func (g *Game) MarkClear(state *GameState) {
	if g.Save == nil {
		g.Save = DefaultSaveData()
	}
	g.syncHighScore(state)
	if state != nil && state.Stage == StageExtra {
		g.Save.TotalExtraClears++
	} else {
		g.Save.TotalClears++
		if state != nil && state.Difficulty >= DiffNormal {
			g.Save.ExtraUnlocked = true
		}
	}
	if state != nil {
		state.ExtraUnlocked = g.Save.ExtraUnlocked
	}
}
