package selector

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	_ "image/png"
	"th10/assets"
	"th10/game"
	"th10/input"
	"th10/scene/stage"
	"th10/scene/stage/stage1"
	"th10/scene/stage/stage2"
	"th10/scene/stage/stage3"
	"th10/scene/stage/stage4"
	"th10/scene/stage/stage5"
	"th10/scene/stage/extra"
	"th10/scene/stage/stage6"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

// 选择阶段
const (
	phaseCharacter = iota // 选角色
	phaseShotType         // 选装备
	phaseDifficulty       // 选难度
)

var (
	charNames = [2]string{"博丽灵梦", "雾雨魔理沙"}
	charTitles = [2]string{"乐园的巫女", "普通的魔法使"}

	// 每角色 3 种装备名
	shotNames = [2][3]string{
		{"灵符（诱导弹）", "梦符（前方集中）", "神符（巫女敏符）"},
		{"魔符（魔法导弹）", "恋符（Master Spark）", "星符（星屑光线）"},
	}

	diffNames = [5]string{"Easy", "Normal", "Hard", "Lunatic", "Extra"}
)

// Select 角色/装备/难度选择场景
type Select struct {
	phase      int
	character  int
	shotType   int
	difficulty int
	isExtra    bool       // Extra模式直接跳过难度选择
	back       game.Scene // 返回目标（Title）
	bgImg      *ebiten.Image
	charImgs   [2]*ebiten.Image // Reimu, Marisa portraits
}

func New(back game.Scene) *Select {
	s := &Select{back: back}
	s.loadAssets()
	return s
}

// NewExtra 创建 Extra 模式选人界面（跳过难度选择）
func NewExtra(back game.Scene) *Select {
	s := &Select{isExtra: true, difficulty: game.DiffExtra, back: back}
	s.loadAssets()
	return s
}

func (s *Select) loadAssets() {
	s.bgImg = loadImage("anm/title/select00s.png")
	s.charImgs[0] = loadImage("anm/title/sl_pl00.png")
	s.charImgs[1] = loadImage("anm/title/sl_pl01.png")
}

func loadImage(path string) *ebiten.Image {
	data, err := assets.Assets.ReadFile(path)
	if err != nil {
		return nil
	}
	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil
	}
	return ebiten.NewImageFromImage(img)
}

func (s *Select) Update(g *game.Game) error {
	in := &input.Global

	switch s.phase {
	case phaseCharacter:
		if in.JustPressed(input.KeyLeft) || in.JustPressed(input.KeyRight) {
			s.character = 1 - s.character
		}
		if in.JustPressed(input.KeyOk) {
			s.phase = phaseShotType
		}
		if in.JustPressed(input.KeyCancel) {
			g.GoTo(s.back)
		}

	case phaseShotType:
		if in.JustPressed(input.KeyLeft) {
			s.shotType = (s.shotType + 2) % 3
		}
		if in.JustPressed(input.KeyRight) {
			s.shotType = (s.shotType + 1) % 3
		}
		if in.JustPressed(input.KeyOk) {
			if s.isExtra {
				// Extra 模式直接开始
				s.startGame(g)
			} else {
				s.phase = phaseDifficulty
			}
		}
		if in.JustPressed(input.KeyCancel) {
			s.phase = phaseCharacter
		}

	case phaseDifficulty:
		maxDiff := 3 // Easy~Lunatic
		if in.JustPressed(input.KeyUp) {
			s.difficulty = (s.difficulty - 1 + maxDiff + 1) % (maxDiff + 1)
		}
		if in.JustPressed(input.KeyDown) {
			s.difficulty = (s.difficulty + 1) % (maxDiff + 1)
		}
		if in.JustPressed(input.KeyOk) {
			s.startGame(g)
		}
		if in.JustPressed(input.KeyCancel) {
			s.phase = phaseShotType
		}
	}

	return nil
}

func (s *Select) startGame(g *game.Game) {
	state := game.NewState(s.character, s.shotType, s.difficulty)
	if s.isExtra {
		state.Stage = game.StageExtra
	} else {
		state.Stage = 1
	}
	g.GoTo(buildStage(g, state))
}

func stageScript(stageNum int) stage.Script {
	switch stageNum {
	case 1:
		return stage1.NewScript()
	case 2:
		return stage2.NewScript()
	case 3:
		return stage3.NewScript()
	case 4:
		return stage4.NewScript()
	case 5:
		return stage5.NewScript()
	case 6:
		return stage6.NewScript()
	case game.StageExtra:
		return extra.NewScript()
	default:
		return stage1.NewScript()
	}
}

func buildStage(g *game.Game, state *game.GameState) *stage.Stage {
	s := stage.New(g, state, stageScript(state.Stage))
	s.NextScene = func(g *game.Game, st *game.GameState) game.Scene {
		next := st.NextStage()
		if next == 0 {
			return nil // 通关 → TODO: Ending
		}
		st.Stage = next
		st.Frame = 0
		return buildStage(g, st)
	}
	return s
}

func (s *Select) Draw(screen *ebiten.Image) {
	// 背景
	if s.bgImg != nil {
		screen.DrawImage(s.bgImg, nil)
	} else {
		screen.Fill(color.RGBA{12, 8, 32, 255})
	}

	// 角色立绘（左侧，缩放到约 200px 高度）
	if portrait := s.charImgs[s.character]; portrait != nil {
		op := &ebiten.DrawImageOptions{}
		ph := portrait.Bounds().Dy()
		scale := 200.0 / float64(ph)
		op.GeoM.Scale(scale, scale)
		op.GeoM.Translate(30, float64(480-int(float64(ph)*scale))/2)
		screen.DrawImage(portrait, op)
	}

	switch s.phase {
	case phaseCharacter:
		ebitenutil.DebugPrintAt(screen, "选择角色", 270, 60)
		for i := 0; i < 2; i++ {
			x := 140 + i*200
			y := 160
			c := color.RGBA{40, 30, 80, 200}
			if i == s.character {
				c = color.RGBA{80, 50, 160, 230}
			}
			vector.FillRect(screen, float32(x), float32(y), 150, 120, c, false)
			ebitenutil.DebugPrintAt(screen, charNames[i], x+20, y+30)
			ebitenutil.DebugPrintAt(screen, charTitles[i], x+10, y+60)
		}

	case phaseShotType:
		ebitenutil.DebugPrintAt(screen, fmt.Sprintf("%s - 选择装备", charNames[s.character]), 220, 60)
		for i := 0; i < 3; i++ {
			x := 80 + i*170
			y := 180
			c := color.RGBA{40, 30, 80, 200}
			if i == s.shotType {
				c = color.RGBA{80, 50, 160, 230}
			}
			vector.FillRect(screen, float32(x), float32(y), 150, 80, c, false)
			ebitenutil.DebugPrintAt(screen, fmt.Sprintf("Type %c", 'A'+i), x+50, y+15)
			ebitenutil.DebugPrintAt(screen, shotNames[s.character][i], x+5, y+40)
		}

	case phaseDifficulty:
		ebitenutil.DebugPrintAt(screen, "选择难度", 270, 60)
		for i := 0; i <= 3; i++ {
			x := 240
			y := 150 + i*40
			label := "  " + diffNames[i]
			if i == s.difficulty {
				label = "> " + diffNames[i]
				vector.FillRect(screen, float32(x-4), float32(y-2), 160, 24,
					color.RGBA{60, 40, 120, 180}, false)
			}
			ebitenutil.DebugPrintAt(screen, label, x, y)
		}
	}

	ebitenutil.DebugPrintAt(screen, "Z: Select   X: Cancel", 230, 420)
}
