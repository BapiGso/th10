package stage6

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

// Script6 第六关脚本 — 八坂神奈子（中Boss: 諏訪子）。道中/Boss 弹幕由官方编译
// 字节码 (assets/ecl/stage06.ecl) 经通用 ECL VM 驱动；本文件只负责背景、BGM 与对话桥接。
type Script6 struct {
	frame    int
	finished bool
	bgY      float64
	bossBGM  bool
	bg1, bg2 *ebiten.Image // stg6bg.png [512x512], stg6bg2.png [256x256]
	bg3      *ebiten.Image // stg6bg3.png [32x256]
	bg4, bg5 *ebiten.Image // stg6bg5.png [256x256], stg6bg6.png [32x128]
	vm       *stageecl.VM
	dlg      *stage.DialogQueue
}

func NewScript() *Script6 {
	s := &Script6{vm: stageecl.NewStageVM("ecl/stage06.ecl")}
	s.loadBg()
	return s
}

func (s *Script6) loadBg() {
	s.bg1 = render.LoadImage("anm/background/stg6bg.png")
	s.bg2 = render.LoadImage("anm/background/stg6bg2.png")
	s.bg3 = render.LoadImage("anm/background/stg6bg3.png")
	s.bg4 = render.LoadImage("anm/background/stg6bg5.png")
	s.bg5 = render.LoadImage("anm/background/stg6bg6.png")
}

func (s *Script6) Init(ctx *stage.Context) {
	ctx.PlayBGM(audio.BGMStage6)
	// Stage 6 mid-boss (諏訪子) shares dialog: events are bossPre, post.
	ch := ctx.State.Character
	s.dlg = stage.NewDialogQueue(ctx, dialog.BossPreC("stage6", ch), dialog.PostC("stage6", ch))
	s.vm.SetDialogBridge(s.dlg.Open, ctx.DialogActive)
}

func (s *Script6) Update(ctx *stage.Context) {
	s.frame++
	s.bgY += 1.2

	if s.vm != nil {
		s.vm.Update(ctx)
		if !s.bossBGM && s.vm.BossActive() {
			s.bossBGM = true
			ctx.PlayBGM(audio.BGMStage6Boss)
		}
		if s.vm.MainDone() {
			s.finished = true
		}
	}
	if s.frame >= 60000 {
		s.finished = true
	}
}

func (s *Script6) BgDraw(field *ebiten.Image) {
	w := float32(field.Bounds().Dx())
	h := float32(field.Bounds().Dy())

	// 御柱墓场 - 暗灰+紫红
	field.Fill(color.RGBA{20, 12, 24, 255})

	if s.bg1 != nil {
		render.TileBackground(field, s.bg1, s.bgY, 1)
	} else {
		offset := float32(math.Mod(s.bgY, 50))
		for y := offset - 50; y < h; y += 50 {
			vector.StrokeLine(field, 0, y, w, y, 0.3, color.RGBA{45, 30, 50, 80}, false)
		}
		for i := 0; i < 4; i++ {
			x := w*0.15 + float32(i)*w*0.22
			vector.FillRect(field, x-4, 0, 8, h, color.RGBA{80, 50, 30, 50}, false)
		}
	}

	render.TileBackground(field, s.bg2, s.bgY*0.4, 0.25)
}

func (s *Script6) Finished() bool { return s.finished }

// ShakeOffset 暴露 ECL ins_337 屏幕震动给 Stage.Draw（可选接口）。
func (s *Script6) ShakeOffset() (float64, float64) {
	if s.vm == nil {
		return 0, 0
	}
	return s.vm.ShakeOffset()
}

// BossHUD 暴露 ECL 解析的 Boss 血条/符卡数据给 Stage.Draw（可选接口）。
func (s *Script6) BossHUD() stage.BossHUDInfo {
	if s.vm == nil {
		return stage.BossHUDInfo{}
	}
	return s.vm.BossHUD()
}

var _ stage.Script = (*Script6)(nil)
