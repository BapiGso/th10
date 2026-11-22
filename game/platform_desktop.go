//go:build !js

package game

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"th10/browserstore"

	"github.com/hajimehoshi/ebiten/v2"
)

func platformTPS() int { return ebiten.SyncWithFPS }

func (g *Game) logicFrameReady(now float64) bool { return g.pace.Ready(now) }

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

// dirStore is the desktop user-data directory as a browserstore.Store: a
// key is a slash-separated path under the directory, so the browser's keys
// name the same files (score.dat, th10.cfg, replay/th10_01.rpy, save.json).
type dirStore string

func userData() browserstore.Store { return dirStore(dataDir()) }

func (d dirStore) path(key string) string { return filepath.Join(string(d), filepath.FromSlash(key)) }

func (d dirStore) Read(key string) ([]byte, error) {
	f, err := os.Open(d.path(key))
	if err != nil {
		return nil, err
	}
	defer func() { _ = f.Close() }()
	data, err := io.ReadAll(io.LimitReader(f, browserstore.MaxBlobBytes+1))
	if err != nil {
		return nil, err
	}
	if len(data) > browserstore.MaxBlobBytes {
		return nil, fmt.Errorf("game: %s exceeds %d bytes", key, browserstore.MaxBlobBytes)
	}
	return data, nil
}

// Write creates a temporary file in the same directory, syncs it and renames
// it over the key's file, so a failed save never truncates an existing file.
func (d dirStore) Write(key string, data []byte) error {
	path := d.path(key)
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	f, err := os.CreateTemp(dir, ".th10-*.tmp")
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
	if err := os.Rename(tmp, path); err != nil {
		return err
	}
	saved = true
	return nil
}

func newScoreStore() ScoreStore   { return binaryScoreStore{store: userData()} }
func newReplayStore() ReplayStore { return binaryReplayStore{store: userData()} }

func configPath() string { return filepath.Join(dataDir(), "config.json") }

// LoadConfig reads the pre-native config.json, then th10.cfg over it.
func LoadConfig() *Config {
	cfg := DefaultConfig()
	loadJSON(configPath(), cfg)
	_ = applyNativeConfig(cfg, userData())
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
	return saveBinaryConfig(userData(), g.Config)
}
