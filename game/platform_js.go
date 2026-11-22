//go:build js

package game

import (
	"sync"
	"syscall/js"
	"th10/browserstore"
	"th10/pacing"
	"time"
)

func platformTPS() int { return 60 }

func (*Game) logicFrameReady(float64) bool { return true }

// StartBrowserClock discards host time spent downloading/waiting for a gesture.
// It does not reset a scene, input stream, score, or either original RNG.
func (g *Game) StartBrowserClock() {
	g.start = time.Now()
	g.pace = pacing.Pacer{}
	g.fresh, g.fps, g.fpsFrames, g.fpsT = false, 0, 0, 0
}

var (
	browserInit sync.Once
	browserData browserstore.Store
	browserErr  error
)

// InitBrowserStorage runs before the game is constructed, so a denied database
// never silently becomes a session-only save. It may wait; call from a goroutine.
func InitBrowserStorage() error {
	browserInit.Do(func() { browserData, browserErr = browserstore.Open() })
	return browserErr
}

type browserAccess struct{}

func (browserAccess) Read(key string) ([]byte, error) {
	if err := InitBrowserStorage(); err != nil {
		return nil, err
	}
	return browserData.Read(key)
}

func (browserAccess) Write(key string, data []byte) error {
	if err := InitBrowserStorage(); err != nil {
		return err
	}
	return browserData.Write(key, data)
}

func LoadConfig() *Config {
	cfg, err := loadBinaryConfig(browserAccess{})
	if err != nil {
		js.Global().Get("console").Call("error", "[TH10 storage] cfg load:", err.Error())
	}
	return cfg
}

func (g *Game) SaveConfig() error {
	return saveBinaryConfig(browserAccess{}, g.Config)
}

func newScoreStore() ScoreStore   { return binaryScoreStore{store: browserAccess{}} }
func newReplayStore() ReplayStore { return binaryReplayStore{store: browserAccess{}} }
