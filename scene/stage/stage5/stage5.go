package stage5

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

// Script5 第五关脚本 — 東風谷早苗。道中/中Boss/Boss 弹幕由官方编译字节码
// (assets/ecl/stage05.ecl) 经通用 ECL VM 驱动；本文件只负责背景、BGM 与对话桥接。
type Script5 struct {
	frame    int
	finished bool
	bgY      float64
	bossBGM  bool
	bg1, bg2 *ebiten.Image // stg5bg.png [256x256], stg5bg2.png [256x256]
	vm       *stageecl.VM
	dlg      *stage.DialogQueue
}

func NewScript() *Script5 {
	s := &Script5{vm: stageecl.NewStageVM("ecl/stage05.ecl")}
	s.loadBg()
	return s
}

func (s *Script5) loadBg() {
	s.bg1 = render.LoadImage("anm/background/stg5bg.png")
	s.bg2 = render.LoadImage("anm/background/stg5bg2.png")
}

func (s *Script5) Init(ctx *stage.Context) {
	ctx.Audio.PlayBGM(audio.BGMStage5)
	ch := ctx.State.Character
	s.dlg = stage.NewDialogQueue(ctx, dialog.BossPreC("stage5", ch), dialog.PostC("stage5", ch))
	s.vm.SetDialogBridge(s.dlg.Open, ctx.DialogActive)
}

func (s *Script5) Update(ctx *stage.Context) {
	s.frame++
	s.bgY += 1.5

	if s.vm != nil {
		s.vm.Update(ctx)
		if !s.bossBGM && s.vm.BossActive() {
			s.bossBGM = true
			ctx.Audio.PlayBGM(audio.BGMStage5Boss)
		}
		if s.vm.MainDone() {
			s.finished = true
		}
	}
	if s.frame >= 60000 {
		s.finished = true
	}
}

func (s *Script5) BgDraw(field *ebiten.Image) {
	w := float32(field.Bounds().Dx())
	h := float32(field.Bounds().Dy())

	// 神社境内，深蓝+紫
	field.Fill(color.RGBA{16, 16, 40, 255})

	if s.bg1 != nil {
		render.TileBackground(field, s.bg1, s.bgY, 1)
	} else {
		offset := float32(math.Mod(s.bgY, 40))
		for y := offset - 40; y < h; y += 40 {
			vector.StrokeLine(field, 0, y, w, y, 0.4, color.RGBA{40, 35, 70, 90}, false)
		}
		vector.FillRect(field, w*0.2-3, 0, 6, h, color.RGBA{120, 30, 20, 60}, false)
		vector.FillRect(field, w*0.8-3, 0, 6, h, color.RGBA{120, 30, 20, 60}, false)
	}

	render.TileBackground(field, s.bg2, s.bgY*0.5, 0.3)
}

func (s *Script5) Finished() bool { return s.finished }

// ShakeOffset 暴露 ECL ins_337 屏幕震动给 Stage.Draw（可选接口）。
func (s *Script5) ShakeOffset() (float64, float64) {
	if s.vm == nil {
		return 0, 0
	}
	return s.vm.ShakeOffset()
}

// BossHUD 暴露 ECL 解析的 Boss 血条/符卡数据给 Stage.Draw（可选接口）。
func (s *Script5) BossHUD() stage.BossHUDInfo {
	if s.vm == nil {
		return stage.BossHUDInfo{}
	}
	return s.vm.BossHUD()
}

var _ stage.Script = (*Script5)(nil)
