package game

import (
	"fmt"
	"os"
	thaudio "th10/audio"
	"th10/input"
	"th10/pacing"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/audio"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
)

const (
	ScreenWidth  = 640
	ScreenHeight = 480
	SampleRate   = 44100

	// 游戏区域（左侧留给弹幕，右侧留给 HUD）
	FieldLeft   = 32
	FieldTop    = 16
	FieldRight  = 416 // 384px 宽
	FieldBottom = 464 // 448px 高
)

// Scene 所有场景必须实现的接口
type Scene interface {
	Update(g *Game) error
	Draw(screen *ebiten.Image)
}

// Game 主游戏结构，管理场景切换
type Game struct {
	current      Scene
	next         Scene
	nextSet      bool
	audioContext *audio.Context
	audioMgr     *thaudio.Manager
	Config       *Config
	Save         *SaveData

	pace      pacing.Pacer
	start     time.Time
	fresh     bool    // a logic frame ran since the last Draw
	fps       float64 // logic frames per second, updated once a second
	fpsFrames int     // logic frames since fpsT
	fpsT      float64 // start of the current FPS window, seconds
	showPace  bool    // TH10_PACE=1: show the pacing mode in the FPS readout
}

func New() *Game {
	ctx := audio.NewContext(SampleRate)
	cfg := LoadConfig()
	save := LoadSaveData()
	audioMgr := thaudio.NewManager(ctx)
	audioMgr.SetBGMVolume(cfg.BGMVolume)
	audioMgr.SetSEVolume(cfg.SEVolume)
	g := &Game{
		audioContext: ctx,
		audioMgr:     audioMgr,
		Config:       cfg,
		Save:         save,
		start:        time.Now(),
		showPace:     os.Getenv("TH10_PACE") == "1",
	}
	// One Update per display refresh; the Pacer picks the refreshes that run
	// a logic frame, and a skipped refresh keeps the last frame on screen.
	ebiten.SetTPS(ebiten.SyncWithFPS)
	ebiten.SetScreenClearedEveryFrame(false)
	return g
}

func (g *Game) SetScene(s Scene) {
	g.current = s
}

func (g *Game) Update() error {
	now := time.Since(g.start).Seconds()
	if !g.pace.Ready(now) {
		return nil
	}
	g.fresh = true
	g.fpsFrames++
	if now-g.fpsT >= 1 {
		g.fps = float64(g.fpsFrames) / (now - g.fpsT)
		g.fpsFrames = 0
		g.fpsT = now
	}

	// 每帧最先采集输入（文章第10篇：输入采集在数据处理之前）
	input.Global.Update()
	if g.audioMgr != nil {
		g.audioMgr.Update()
	}

	if g.nextSet {
		g.current = g.next
		g.next = nil
		g.nextSet = false
	}
	if g.current == nil {
		return nil
	}
	return g.current.Update(g)
}

func (g *Game) Draw(screen *ebiten.Image) {
	if !g.fresh {
		keepAlive(screen)
		return
	}
	g.fresh = false
	screen.Clear()
	if g.current != nil {
		g.current.Draw(screen)
	}
	// 全局 FPS 显示（右下角）
	fps := fmt.Sprintf("%.0f FPS", g.fps)
	x := ScreenWidth - 70
	if g.showPace {
		mode := " grid"
		if g.pace.P > 0 {
			mode = fmt.Sprintf(" L%d %.2fHz", g.pace.Lock, 1/g.pace.P)
		}
		fps += mode
		x = ScreenWidth - 200
	}
	ebitenutil.DebugPrintAt(screen, fps, x, ScreenHeight-16)
}

// keepAliveDot is the 1x1 image keepAlive draws; it is created on first use.
var keepAliveDot *ebiten.Image

// keepAlive marks screen as modified without changing a pixel: Ebiten
// stops presenting after 4 frames whose screen was not drawn to
// (internal/ui/context.go drawGame), and then paces its loop with 1/60 s
// sleeps instead of vsync, which would break the Pacer's refresh timing.
// A 1x1 draw with BlendDestination keeps the destination as it is.
func keepAlive(screen *ebiten.Image) {
	if keepAliveDot == nil {
		keepAliveDot = ebiten.NewImage(1, 1)
	}
	screen.DrawImage(keepAliveDot, &ebiten.DrawImageOptions{Blend: ebiten.BlendDestination})
}

func (g *Game) Layout(_, _ int) (int, int) {
	return ScreenWidth, ScreenHeight
}

// GoTo 切换到下一个场景（下一帧生效，允许 nil 表示回到空状态）
func (g *Game) GoTo(s Scene) {
	g.next = s
	g.nextSet = true
}

// AudioContext 返回全局音频上下文
func (g *Game) AudioContext() *audio.Context {
	return g.audioContext
}

// Audio 返回音频管理器
func (g *Game) Audio() *thaudio.Manager {
	return g.audioMgr
}
