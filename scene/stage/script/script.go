// Package stagescript supplies the common ECL stage wrapper. Reachable legacy
// backgrounds and the old safety cap are retained, not newly ported behavior.
package stagescript

import (
	"th10/scene/stage"
	stageecl "th10/scene/stage/ecl"

	"github.com/hajimehoshi/ebiten/v2"
)

// Config contains only resource names, native BGM ordinals and the unchanged
// legacy presentation/safety settings. Stage packages bind its factory methods.
type Config struct {
	ECL, MSG         string
	BGM, BossBGM     int
	FrameCap         int
	BackgroundStage  int
	BackgroundScroll float64
	Backgrounds      [5]string
}

type Script struct {
	config   Config
	frame    int
	finished bool
	bg       legacyBackground
	vm       *stageecl.VM
}

func (c Config) NewScript() *Script {
	s := &Script{config: c, vm: stageecl.NewStageVM(c.ECL)}
	s.bg.init(c)
	return s
}

// NewScriptInterface preserves stage1's existing func() stage.Script entry.
func (c Config) NewScriptInterface() stage.Script { return c.NewScript() }

func (s *Script) Init(ctx *stage.Context) {
	s.vm.BindContext(ctx) // FUN_00417870: manager exists before the first calc.
	ctx.PlayBGM(s.config.BGM)
	ctx.LoadMSG(s.config.MSG, s.config.BossBGM) // FUN_00415db0/00415b00, ECL 338.
}

func (s *Script) Update(ctx *stage.Context) {
	s.frame++
	s.bg.advance(s.frame)
	if s.vm != nil {
		s.vm.Update(ctx)
		if s.vm.MainDone() {
			s.finished = true
		}
	}
	// Existing MainDone/cap fallbacks only; MSG op 20 remains the clear path.
	if s.frame >= s.config.FrameCap {
		s.finished = true
	}
}

func (s *Script) BgDraw(field *ebiten.Image) { s.bg.draw(field) }
func (s *Script) Finished() bool             { return s.finished }

// ShakeOffset reads FUN_0043c550/0043c5ca's already sampled shift.
func (s *Script) ShakeOffset() (float64, float64) {
	if s.vm == nil {
		return 0, 0
	}
	return s.vm.ShakeOffset()
}

func (s *Script) BossHUD() stage.BossHUDInfo {
	if s.vm == nil {
		return stage.BossHUDInfo{}
	}
	return s.vm.BossHUD()
}

// LateUpdate keeps FUN_00408d60's existing position before GUI/World calc.
func (s *Script) LateUpdate(ctx *stage.Context) {
	if s.vm != nil {
		s.vm.LateUpdate(ctx)
	}
}

var _ stage.Script = (*Script)(nil)
