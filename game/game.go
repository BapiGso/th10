package game

import (
	"fmt"
	"math"
	"os"
	thaudio "th10/audio"
	"th10/input"
	"th10/pacing"
	"th10/supervisor"
	"th10/zun"
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
	current        Scene
	next           Scene
	nextSet        bool
	scenes         sceneSwitcher // FUN_004218d0: numbered request/constructor/lifecycle
	TitleEntry     int           // DAT_00491c00: initial/menu/replay/player-data entry
	Selection      MenuSelection
	RunState       *GameState // application globals alias the existing t8 run
	audioContext   *audio.Context
	audioMgr       *thaudio.Manager
	Config         *Config
	Save           *SaveData
	Replays        ReplayStore // nil selects the platform user-data backend
	Scores         ScoreStore
	SpellHistory   *SpellHistory // application-lifetime DAT_0047783c history, shared by all runs
	SystemFlags    uint32        // DAT_00491ff4, controller scene-request override
	alternateRNG   zun.RNG       // DAT_004918a8: application lifetime, never restored from a replay
	initialRNGSeed uint16        // DAT_004918b0's startup seed, before a run/replay overrides it

	pace      pacing.Pacer
	start     time.Time
	fresh     bool    // a logic frame ran since the last Draw
	fps       float64 // logic frames per second, updated once a second
	fpsFrames int     // logic frames since fpsT
	fpsT      float64 // start of the current FPS window, seconds
	showPace  bool    // TH10_PACE=1: show the pacing mode in the FPS readout
}

func New() *Game {
	now := time.Now()
	// FUN_004201b0 (0x4201d0..0x4201e1) seeds both generators from the
	// low 16 bits of startup milliseconds. time also works on js/wasm.
	seed := uint16(now.UnixMilli())
	ctx := audio.NewContext(SampleRate)
	cfg := LoadConfig()
	save := loadSaveDataWithSeed(newScoreStore(), seed)
	audioMgr := thaudio.NewManager(ctx)
	audioMgr.SetBGMVolume(cfg.BGMVolume)
	audioMgr.SetSEVolume(cfg.SEVolume)
	native := cfg.NativeBytes()
	audioMgr.SetNativeVolumes(native[0x20], native[0x21]) // FUN_0042e5a0, same curve after reload
	g := &Game{
		audioContext:   ctx,
		audioMgr:       audioMgr,
		Config:         cfg,
		Save:           save,
		alternateRNG:   zun.RNG{State: seed},
		initialRNGSeed: scoreStartupSeed(seed), // FUN_0042add0 consumes 512 main steps at score initialization
		start:          now,
		showPace:       os.Getenv("TH10_PACE") == "1",
	}
	g.BindMusicProgress(nil) // FUN_00420b10's ST music flags, before title playback
	// Desktop picks display refreshes with the Pacer. The browser instead
	// uses Ebiten's fixed 60 TPS clock without a second pacing gate.
	ebiten.SetTPS(platformTPS())
	ebiten.SetScreenClearedEveryFrame(false)
	return g
}

// AlternateRNG is DAT_004918a8. A zero Game is a deterministic headless
// application; New seeds the interactive application once, not once per run.
func (g *Game) AlternateRNG() *zun.RNG { return &g.alternateRNG }

// BindState attaches application-lifetime state before any STD/ANM starts.
func (g *Game) BindState(state *GameState) {
	g.RunState = state
	g.BindMusicProgress(state)
	state.alternateRNG = g.AlternateRNG()
	state.SystemFlags = g.SystemFlags
	if g.SpellHistory == nil {
		if g.Save != nil && g.Save.SpellHistory != nil {
			g.SpellHistory = g.Save.SpellHistory
		} else {
			g.SpellHistory = DefaultScoreFile(0).history()
		}
	}
	state.SpellHistory = g.SpellHistory
	if g.Save != nil {
		if state.ReplayMode == 1 {
			// MSG 0x14 sets stage flags even in playback. Preserve that state
			// locally while keeping t5's read-only replay persistence policy.
			progress := g.Save.Progress
			state.ScoreProgress = &progress
		} else {
			g.Save.SpellHistory = g.SpellHistory
			state.ScoreProgress = &g.Save.Progress
		}
	}
}

func (g *Game) NewState(character, shotType, difficulty int) *GameState {
	s := NewState(character, shotType, difficulty)
	g.BindState(s)
	s.RNGSeed, s.HasSeed = g.initialRNGSeed, true
	return s
}

func (g *Game) SetScene(s Scene) {
	g.initScenes()
	if life, ok := g.current.(SceneLifecycle); ok {
		life.SceneLeave(g, sceneMode(s))
	}
	g.scenes.previous = g.scenes.current
	g.scenes.current, g.scenes.request = sceneMode(s), sceneMode(s)
	g.next, g.nextSet, g.scenes.pending = nil, false, nil
	g.current = s
	g.scenes.clearPending = true
	if life, ok := s.(SceneLifecycle); ok {
		g.scenes.err = life.SceneEnter(g)
	}
}

func (g *Game) Update() error {
	now := time.Since(g.start).Seconds()
	if !g.logicFrameReady(now) {
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
	if g.Config != nil && g.Config.Native != nil {
		input.Global.SetGamepadBindings(g.Config.Native.JoystickBindings())
	}
	input.Global.Update()

	err := g.CalcScene()
	// FUN_00439390 (0x43943b) flushes sounds after the calc chain, so all
	// same-id x requests contribute to this frame's average pan.
	if g.audioMgr != nil {
		g.audioMgr.Update()
	}
	return err
}

func (g *Game) Draw(screen *ebiten.Image) {
	if !g.fresh {
		keepAlive(screen)
		return
	}
	g.fresh = false
	screen.Clear()
	if g.scenes.clearPending {
		supervisor.Get().ClearColor = 0xff000000 // FUN_004218d0, 0x421914
		g.scenes.clearPending = false
	}
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
	g.initScenes()
	g.scenes.request, g.scenes.pending = sceneMode(s), nil
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

// ReplaySample is FUN_004297d0 at 0x42986b..0x429892: current FPS + 0.5,
// truncate when less than 256, otherwise 0xff. The constants are the
// float32 bit patterns at 0x470b0c and 0x470c34. The first host FPS window
// is not available yet; use the nominal 60 Hz during that window.
func (g *Game) ReplaySample() byte {
	fps := float32(g.fps)
	if fps <= 0 {
		fps = 60
	}
	fps += math.Float32frombits(0x3f000000)
	if fps < math.Float32frombits(0x43800000) {
		return byte(int32(fps))
	}
	return 0xff
}
