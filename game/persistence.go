//go:build !js && !wasm

package game

import (
	"encoding/json"
	"fmt"
	"io"
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

func LoadConfig() *Config {
	cfg := DefaultConfig()
	loadJSON(configPath(), cfg)
	if data, err := os.ReadFile(filepath.Join(dataDir(), "th10.cfg")); err == nil {
		if native, err := DecodeNativeConfig(data); err == nil {
			cfg.Native = native
			cfg.BGMVolume, cfg.SEVolume = float64(native.Raw[0x20])/100, float64(native.Raw[0x21])/100
		}
	}
	return cfg
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
	if err := writeJSON(configPath(), g.Config); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dataDir(), "th10.cfg"), g.Config.NativeBytes(), 0o644)
}

type fileScoreStore struct{ dir string }

func newScoreStore() ScoreStore { return &fileScoreStore{dir: dataDir()} }

func readScoreUserFile(path string) ([]byte, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer func() { _ = f.Close() }()
	data, err := io.ReadAll(io.LimitReader(f, maxScoreBytes+1))
	if err != nil {
		return nil, err
	}
	if len(data) > maxScoreBytes {
		return nil, fmt.Errorf("game: score exceeds %d bytes", maxScoreBytes)
	}
	return data, nil
}

func (s *fileScoreStore) Read() ([]byte, error) {
	return readScoreUserFile(filepath.Join(s.dir, "score.dat"))
}

func (s *fileScoreStore) LegacyJSON() ([]byte, error) {
	return readScoreUserFile(filepath.Join(s.dir, "save.json"))
}

// As with replay writes, a failed save never truncates an existing file.
func (s *fileScoreStore) Write(data []byte) error {
	if len(data) > maxScoreBytes {
		return fmt.Errorf("game: score exceeds %d bytes", maxScoreBytes)
	}
	if err := os.MkdirAll(s.dir, 0o755); err != nil {
		return err
	}
	f, err := os.CreateTemp(s.dir, ".th10-score-*.tmp")
	if err != nil {
		return err
	}
	tmp, saved := f.Name(), false
	defer func() {
		_ = f.Close()
		if !saved {
			_ = os.Remove(tmp)
		}
	}()
	if _, err := f.Write(data); err != nil {
		return err
	}
	if err := f.Sync(); err != nil {
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	if err := os.Rename(tmp, filepath.Join(s.dir, "score.dat")); err != nil {
		return err
	}
	saved = true
	return nil
}
