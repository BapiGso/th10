package stage3

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

// Script3 第三关脚本 — 河城にとり。道中/中Boss/Boss 弹幕由官方编译字节码
// (assets/ecl/stage03.ecl) 经通用 ECL VM 驱动；本文件只负责背景、BGM 与对话桥接。
type Script3 struct {
	frame    int
	finished bool
	bgY      float64
	bossBGM  bool
	bg1, bg2 *ebiten.Image // stg3bg.png [256x256], stg3bg2.png [128x128]
	bg3, bg4 *ebiten.Image // stg3bg3.png [512x512], stg3bg4.png [256x512]
	vm       *stageecl.VM
	dlg      *stage.DialogQueue
}

func NewScript() *Script3 {
	s := &Script3{vm: stageecl.NewStageVM("ecl/stage03.ecl")}
	s.loadBg()
	return s
}

func (s *Script3) loadBg() {
	s.bg1 = render.LoadImage("anm/background/stg3bg.png")
	s.bg2 = render.LoadImage("anm/background/stg3bg2.png")
	s.bg3 = render.LoadImage("anm/background/stg3bg3.png")
	s.bg4 = render.LoadImage("anm/background/stg3bg4.png")
}

func (s *Script3) Init(ctx *stage.Context) {
	ctx.PlayBGM(audio.BGMStage3)
	// Stage 3 has a mid-boss: dialog events are midPre, bossPre, post.
	ch := ctx.State.Character
	s.dlg = stage.NewDialogQueue(ctx,
		dialog.MidPreC("stage3", ch), dialog.BossPreC("stage3", ch), dialog.PostC("stage3", ch))
	s.vm.SetDialogBridge(s.dlg.Open, ctx.DialogActive)
}

func (s *Script3) Update(ctx *stage.Context) {
	s.frame++
	s.bgY += 1.8

	if s.vm != nil {
		s.vm.Update(ctx)
		if !s.bossBGM && s.vm.BossActive() {
			s.bossBGM = true
			ctx.PlayBGM(audio.BGMStage3Boss)
		}
		if s.vm.MainDone() {
			s.finished = true
		}
	}
	if s.frame >= 60000 {
		s.finished = true
	}
}

func (s *Script3) BgDraw(field *ebiten.Image) {
	w := float32(field.Bounds().Dx())
	h := float32(field.Bounds().Dy())

	// 蓝绿色山间溪流
	field.Fill(color.RGBA{12, 32, 28, 255})

	if s.bg1 != nil {
		render.TileBackground(field, s.bg1, s.bgY, 1)
	} else {
		offset := float32(math.Mod(s.bgY, 48))
		for y := offset - 48; y < h; y += 48 {
			vector.StrokeLine(field, 0, y, w, y, 0.5, color.RGBA{30, 60, 55, 100}, false)
		}
		ox := float32(math.Sin(float64(s.bgY)*0.02) * 20)
		vector.StrokeLine(field, w*0.4+ox, 0, w*0.4+ox, h, 2, color.RGBA{40, 80, 120, 80}, false)
		vector.StrokeLine(field, w*0.6-ox, 0, w*0.6-ox, h, 2, color.RGBA{40, 80, 120, 80}, false)
	}

	render.TileBackground(field, s.bg2, s.bgY*0.7, 0.3)
}

func (s *Script3) Finished() bool { return s.finished }

// ShakeOffset 暴露 ECL ins_337 屏幕震动给 Stage.Draw（可选接口）。
func (s *Script3) ShakeOffset() (float64, float64) {
	if s.vm == nil {
		return 0, 0
	}
	return s.vm.ShakeOffset()
}

// BossHUD 暴露 ECL 解析的 Boss 血条/符卡数据给 Stage.Draw（可选接口）。
func (s *Script3) BossHUD() stage.BossHUDInfo {
	if s.vm == nil {
		return stage.BossHUDInfo{}
	}
	return s.vm.BossHUD()
}

var _ stage.Script = (*Script3)(nil)
