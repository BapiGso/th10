// Package stagescript supplies the common ECL stage wrapper and every
// stage's configuration. The old safety cap is retained, not newly ported
// behavior.
package stagescript

import (
	"th10/audio"
	"th10/scene/stage"
	stageecl "th10/scene/stage/ecl"
)

// Config contains only resource names, native BGM ordinals and the unchanged
// safety cap.
type Config struct {
	ECL, MSG     string
	BGM, BossBGM int
	FrameCap     int
}

// Configs holds stage n's configuration at index n-1; stage 7 is Extra.
var Configs = [7]Config{
	{ECL: "ecl/stage01.ecl", MSG: "stage1", BGM: audio.BGMStage1, BossBGM: audio.BGMStage1Boss, FrameCap: 60000},
	{ECL: "ecl/stage02.ecl", MSG: "stage2", BGM: audio.BGMStage2, BossBGM: audio.BGMStage2Boss, FrameCap: 60000},
	{ECL: "ecl/stage03.ecl", MSG: "stage3", BGM: audio.BGMStage3, BossBGM: audio.BGMStage3Boss, FrameCap: 60000},
	{ECL: "ecl/stage04.ecl", MSG: "stage4", BGM: audio.BGMStage4, BossBGM: audio.BGMStage4Boss, FrameCap: 60000},
	{ECL: "ecl/stage05.ecl", MSG: "stage5", BGM: audio.BGMStage5, BossBGM: audio.BGMStage5Boss, FrameCap: 60000},
	{ECL: "ecl/stage06.ecl", MSG: "stage6", BGM: audio.BGMStage6, BossBGM: audio.BGMStage6Boss, FrameCap: 60000},
	{ECL: "ecl/stage07.ecl", MSG: "extra", BGM: audio.BGMExtra, BossBGM: audio.BGMExtraBoss, FrameCap: 60000},
}

// New returns stage n's script (7 is Extra); any other n gets stage 1's.
func New(n int) stage.Script {
	if n < 1 || n > len(Configs) {
		n = 1
	}
	return Configs[n-1].NewScript()
}

type Script struct {
	config   Config
	frame    int
	finished bool
	vm       *stageecl.VM
}

func (c Config) NewScript() *Script {
	return &Script{config: c, vm: stageecl.NewStageVM(c.ECL)}
}

func (s *Script) Init(ctx *stage.Context) {
	s.vm.BindContext(ctx) // FUN_00417870: manager exists before the first calc.
	ctx.PlayBGM(s.config.BGM)
	ctx.LoadMSG(s.config.MSG, s.config.BossBGM) // FUN_00415db0/00415b00, ECL 338.
}

func (s *Script) Update(ctx *stage.Context) {
	s.frame++
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

func (s *Script) Finished() bool { return s.finished }

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
