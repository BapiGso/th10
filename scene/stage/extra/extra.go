package extra

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

// ScriptExtra Extra面脚本 — 洩矢諏訪子（中Boss: 八坂神奈子）。道中/中Boss/Boss
// 弹幕由官方编译字节码 (assets/ecl/stage07.ecl) 经通用 ECL VM 驱动；本文件只负责
// 背景、BGM 与对话桥接。
type ScriptExtra struct {
	frame    int
	finished bool
	bgY      float64
	bossBGM  bool
	bg1, bg2 *ebiten.Image // stg7bg.png [256x256], stg7bg2.png [256x256]
	bg3      *ebiten.Image // stg7bg3.png [32x256]
	vm       *stageecl.VM
	dlg      *stage.DialogQueue
}

func NewScript() *ScriptExtra {
	s := &ScriptExtra{vm: stageecl.NewStageVM("ecl/stage07.ecl")}
	s.loadBg()
	return s
}

func (s *ScriptExtra) loadBg() {
	s.bg1 = render.LoadImage("anm/background/stg7bg.png")
	s.bg2 = render.LoadImage("anm/background/stg7bg2.png")
	s.bg3 = render.LoadImage("anm/background/stg7bg3.png")
}

func (s *ScriptExtra) Init(ctx *stage.Context) {
	ctx.PlayBGM(audio.BGMExtra)
	ch := ctx.State.Character
	s.dlg = stage.NewDialogQueue(ctx,
		dialog.MidPreC("extra", ch), dialog.BossPreC("extra", ch), dialog.PostC("extra", ch))
	s.vm.SetDialogBridge(s.dlg.Open, ctx.DialogActive)
}

func (s *ScriptExtra) Update(ctx *stage.Context) {
	s.frame++
	s.bgY += 2.0

	if s.vm != nil {
		s.vm.Update(ctx)
		if !s.bossBGM && s.vm.BossActive() {
			s.bossBGM = true
			ctx.PlayBGM(audio.BGMExtraBoss)
		}
		if s.vm.MainDone() {
			s.finished = true
		}
	}
	if s.frame >= 60000 {
		s.finished = true
	}
}

func (s *ScriptExtra) BgDraw(field *ebiten.Image) {
	w := float32(field.Bounds().Dx())
	h := float32(field.Bounds().Dy())

	// 山の裏 — 暗い土色+ミシャグジ紋様
	field.Fill(color.RGBA{28, 20, 16, 255})

	if s.bg1 != nil {
		render.TileBackground(field, s.bg1, s.bgY, 1)
	} else {
		offset := float32(math.Mod(s.bgY, 60))
		for y := offset - 60; y < h; y += 60 {
			vector.StrokeLine(field, 0, y, w, y, 0.3, color.RGBA{50, 40, 30, 70}, false)
		}
		for i := 0; i < 3; i++ {
			baseX := w * (0.2 + float32(i)*0.3)
			for y := float32(0); y < h; y += 4 {
				ox := float32(math.Sin(float64(y+float32(s.bgY))*0.05+float64(i))) * 15
				vector.FillRect(field, baseX+ox-1, y, 2, 4, color.RGBA{60, 45, 25, 50}, false)
			}
		}
	}

	render.TileBackground(field, s.bg2, s.bgY*0.5, 0.3)
}

func (s *ScriptExtra) Finished() bool { return s.finished }

// ShakeOffset 暴露 ECL ins_337 屏幕震动给 Stage.Draw（可选接口）。
func (s *ScriptExtra) ShakeOffset() (float64, float64) {
	if s.vm == nil {
		return 0, 0
	}
	return s.vm.ShakeOffset()
}

// BossHUD 暴露 ECL 解析的 Boss 血条/符卡数据给 Stage.Draw（可选接口）。
func (s *ScriptExtra) BossHUD() stage.BossHUDInfo {
	if s.vm == nil {
		return stage.BossHUDInfo{}
	}
	return s.vm.BossHUD()
}

var _ stage.Script = (*ScriptExtra)(nil)
