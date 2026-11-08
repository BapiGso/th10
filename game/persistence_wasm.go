//go:build js && wasm

package game

func LoadConfig() *Config {
	return DefaultConfig()
}

func (g *Game) SaveConfig() error {
	return nil
}

// Preserve the existing page-session policy, now with the native score bytes.
var browserScores = NewMemoryScoreStore()

func newScoreStore() ScoreStore { return browserScores }
