package stage4

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

// Script4 第四关脚本 — 射命丸文。道中/中Boss/Boss 弹幕由官方编译字节码
// (assets/ecl/stage04.ecl) 经通用 ECL VM 驱动；本文件只负责背景、BGM 与对话桥接。
type Script4 struct {
	frame    int
	finished bool
	bgY      float64
	bossBGM  bool
	bg1      *ebiten.Image // stg4bg.png [512x512]
	bg2      *ebiten.Image // stg4bg3.png [512x256]
	bg3      *ebiten.Image // stg4bg7.png [384x448]
	vm       *stageecl.VM
	dlg      *stage.DialogQueue
}

func NewScript() *Script4 {
	s := &Script4{vm: stageecl.NewStageVM("ecl/stage04.ecl")}
	s.loadBg()
	return s
}

func (s *Script4) loadBg() {
	s.bg1 = render.LoadImage("anm/background/stg4bg.png")
	s.bg2 = render.LoadImage("anm/background/stg4bg3.png")
	s.bg3 = render.LoadImage("anm/background/stg4bg7.png")
}

func (s *Script4) Init(ctx *stage.Context) {
	ctx.Audio.PlayBGM(audio.BGMStage4)
	// Stage 4 mid-boss shares the boss (文): dialog events are bossPre, post.
	ch := ctx.State.Character
	s.dlg = stage.NewDialogQueue(ctx, dialog.BossPreC("stage4", ch), dialog.PostC("stage4", ch))
	s.vm.SetDialogBridge(s.dlg.Open, ctx.DialogActive)
}

func (s *Script4) Update(ctx *stage.Context) {
	s.frame++
	s.bgY += 2.5

	if s.vm != nil {
		s.vm.Update(ctx)
		if !s.bossBGM && s.vm.BossActive() {
			s.bossBGM = true
			ctx.Audio.PlayBGM(audio.BGMStage4Boss)
		}
		if s.vm.MainDone() {
			s.finished = true
		}
	}
	if s.frame >= 60000 {
		s.finished = true
	}
}

func (s *Script4) BgDraw(field *ebiten.Image) {
	w := float32(field.Bounds().Dx())
	h := float32(field.Bounds().Dy())

	// 红叶瀑布
	field.Fill(color.RGBA{32, 20, 16, 255})

	if s.bg1 != nil {
		render.TileBackground(field, s.bg1, s.bgY, 1)
	} else {
		offset := float32(math.Mod(s.bgY, 36))
		for y := offset - 36; y < h; y += 36 {
			vector.StrokeLine(field, 0, y, w, y, 0.5, color.RGBA{60, 40, 30, 100}, false)
		}
		vector.FillRect(field, w*0.4, 0, w*0.2, h, color.RGBA{30, 50, 80, 40}, false)
		wy := float32(math.Mod(float64(s.bgY)*3, float64(h)))
		for y := float32(0); y < h; y += 8 {
			a := uint8(40 + 20*math.Sin(float64(y+wy)*0.1))
			vector.StrokeLine(field, w*0.45, y, w*0.55, y+4, 1, color.RGBA{60, 90, 140, a}, false)
		}
	}

	render.TileBackground(field, s.bg2, s.bgY*0.6, 0.25)
}

func (s *Script4) Finished() bool { return s.finished }

// ShakeOffset 暴露 ECL ins_337 屏幕震动给 Stage.Draw（可选接口）。
func (s *Script4) ShakeOffset() (float64, float64) {
	if s.vm == nil {
		return 0, 0
	}
	return s.vm.ShakeOffset()
}

// BossHUD 暴露 ECL 解析的 Boss 血条/符卡数据给 Stage.Draw（可选接口）。
func (s *Script4) BossHUD() stage.BossHUDInfo {
	if s.vm == nil {
		return stage.BossHUDInfo{}
	}
	return s.vm.BossHUD()
}

var _ stage.Script = (*Script4)(nil)
