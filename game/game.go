package game

import (
	thaudio "th10/audio"
	"th10/input"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/audio"
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
}

func New() *Game {
	ctx := audio.NewContext(SampleRate)
	g := &Game{
		audioContext: ctx,
		audioMgr:     thaudio.NewManager(ctx),
		Config:       DefaultConfig(),
	}
	return g
}

func (g *Game) SetScene(s Scene) {
	g.current = s
}

func (g *Game) Update() error {
	// 每帧最先采集输入（文章第10篇：输入采集在数据处理之前）
	input.Global.Update()

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
	if g.current != nil {
		g.current.Draw(screen)
	}
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
