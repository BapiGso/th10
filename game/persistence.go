//go:build !js && !wasm

package game

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// 配置/存档路径解析：
//   - 优先使用环境变量 TH10_DATA_DIR（开发期 `TH10_DATA_DIR=./data go run ./cmd/main.go`）
//   - 否则使用 os.UserConfigDir()/th10/，与 cwd 解耦（避免 `go test ./cmd/...` 在 cmd/data/
//     生成多余目录的历史问题）
//   - 最末兜底回到相对路径 "data/"（极端环境如沙箱内 UserConfigDir 报错时仍可工作）
//
// 第一次解析时缓存，避免每次 Save 都重新算。
var resolvedDataDir string

func dataDir() string {
	if resolvedDataDir != "" {
		return resolvedDataDir
	}
	if env := os.Getenv("TH10_DATA_DIR"); env != "" {
		resolvedDataDir = env
		return resolvedDataDir
	}
	if dir, err := os.UserConfigDir(); err == nil {
		resolvedDataDir = filepath.Join(dir, "th10")
		return resolvedDataDir
	}
	resolvedDataDir = "data"
	return resolvedDataDir
}

func configPath() string { return filepath.Join(dataDir(), "config.json") }
func savePath() string   { return filepath.Join(dataDir(), "save.json") }

// SaveData 持久化的全局进度数据。
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
	cfg := DefaultConfig()
	loadJSON(configPath(), cfg)
	return cfg
}

func LoadSaveData() *SaveData {
	save := DefaultSaveData()
	loadJSON(savePath(), save)
	return save
}

func loadJSON(path string, dst any) {
	data, err := os.ReadFile(path)
	if err != nil {
		return
	}
	_ = json.Unmarshal(data, dst)
}

func writeJSON(path string, v any) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

func (g *Game) SaveConfig() error {
	if g.Config == nil {
		return nil
	}
	return writeJSON(configPath(), g.Config)
}

func (g *Game) SaveProgress() error {
	if g.Save == nil {
		return nil
	}
	return writeJSON(savePath(), g.Save)
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
	_ = g.SaveProgress()
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
	_ = g.SaveProgress()
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
	_ = g.SaveProgress()
}
