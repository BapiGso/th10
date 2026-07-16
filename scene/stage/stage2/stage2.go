package stage2

import (
	"image/color"
	"math"
	"th10/audio"
	"th10/render"
	"th10/scene/dialog"
	"th10/scene/stage"
	stageecl "th10/scene/stage/ecl"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

// Script2 第二关脚本 — 鍵山雛。道中与 Boss 弹幕由官方编译字节码
// (assets/ecl/stage02.ecl) 经通用 ECL VM 驱动，与 Stage 1 同源；本文件只负责
// 背景绘制、BGM 切换与对话桥接。
type Script2 struct {
	frame    int
	finished bool
	bgY      float64
	bossBGM  bool
	bg1, bg2 *ebiten.Image // stg2bg.png [256x256], stg2bg2.png [512x512]
	vm       *stageecl.VM
	dlg      *stage.DialogQueue
}

func NewScript() *Script2 {
	s := &Script2{vm: stageecl.NewStageVM("ecl/stage02.ecl")}
	s.loadBg()
	return s
}

func (s *Script2) loadBg() {
	s.bg1 = render.LoadImage("anm/background/stg2bg.png")
	s.bg2 = render.LoadImage("anm/background/stg2bg2.png")
}

func (s *Script2) Init(ctx *stage.Context) {
	ctx.Audio.PlayBGM(audio.BGMStage2)
	// Stage 2 has no mid-boss dialog: the two ECL dialog events are bossPre, post.
	ch := ctx.State.Character
	s.dlg = stage.NewDialogQueue(ctx, dialog.BossPreC("stage2", ch), dialog.PostC("stage2", ch))
	s.vm.SetDialogBridge(s.dlg.Open, ctx.DialogActive)
}

func (s *Script2) Update(ctx *stage.Context) {
	s.frame++
	s.bgY += 2.0

	if s.vm != nil {
		s.vm.Update(ctx)

		// Switch to boss music the first frame a boss is alive.
		if !s.bossBGM && s.vm.BossActive() {
			s.bossBGM = true
			ctx.Audio.PlayBGM(audio.BGMStage2Boss)
		}

		// Stage clears when the ECL main task finishes (boss defeated → main
		// unblocks from deathWait and delete()s). The frame cap is only a
		// safety net against a stuck VM.
		if s.vm.MainDone() {
			s.finished = true
		}
	}
	if s.frame >= 60000 {
		s.finished = true
	}
}

func (s *Script2) BgDraw(field *ebiten.Image) {
	w := float32(field.Bounds().Dx())
	h := float32(field.Bounds().Dy())

	// 暗紫色阶梯参道
	field.Fill(color.RGBA{24, 16, 40, 255})

	if s.bg1 != nil {
		render.TileBackground(field, s.bg1, s.bgY, 1)
	} else {
		offset := float32(math.Mod(s.bgY, 32))
		for y := offset - 32; y < h; y += 32 {
			vector.StrokeLine(field, 0, y, w, y, 0.8, color.RGBA{50, 35, 70, 120}, false)
		}
		for y := offset - 32; y < h; y += 32 {
			vector.FillRect(field, w*0.15, y, w*0.7, 2, color.RGBA{60, 45, 80, 80}, false)
		}
	}

	render.TileBackground(field, s.bg2, s.bgY*0.5, 0.25)
}

func (s *Script2) Finished() bool { return s.finished }

// ShakeOffset 暴露 ECL ins_337 屏幕震动给 Stage.Draw（可选接口）。
func (s *Script2) ShakeOffset() (float64, float64) {
	if s.vm == nil {
		return 0, 0
	}
	return s.vm.ShakeOffset()
}

// BossHUD 暴露 ECL 解析的 Boss 血条/符卡数据给 Stage.Draw（可选接口）。
func (s *Script2) BossHUD() stage.BossHUDInfo {
	if s.vm == nil {
		return stage.BossHUDInfo{}
	}
	return s.vm.BossHUD()
}

var _ stage.Script = (*Script2)(nil)
