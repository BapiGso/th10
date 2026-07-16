package selector

import (
	"fmt"
	"image/color"
	"th10/game"
	"th10/input"
	"th10/render"
	"th10/scene/ending"
	resultscene "th10/scene/result"
	"th10/scene/stage"
	"th10/scene/stage/extra"
	"th10/scene/stage/stage1"
	"th10/scene/stage/stage2"
	"th10/scene/stage/stage3"
	"th10/scene/stage/stage4"
	"th10/scene/stage/stage5"
	"th10/scene/stage/stage6"
	sceneui "th10/scene/ui"

	"github.com/hajimehoshi/ebiten/v2"
	ebitext "github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

// 选择阶段
const (
	phasePracticeStage = iota // 选练习关卡
	phaseCharacter            // 选角色
	phaseShotType             // 选装备
	phaseDifficulty           // 选难度
)

// 配色
var (
	colWhite     = color.NRGBA{248, 244, 232, 255}
	colGold      = color.NRGBA{255, 230, 164, 255}
	colDim       = color.NRGBA{180, 174, 162, 220}
	colShadow    = color.NRGBA{16, 8, 8, 220}
	colHoverBg   = color.NRGBA{60, 40, 120, 180}
	colCardIdle  = color.NRGBA{40, 30, 80, 200}
	colCardHover = color.NRGBA{80, 50, 160, 230}
)

// Select 角色/装备/难度选择场景
type Select struct {
	phase         int
	character     int
	shotType      int
	difficulty    int
	isExtra       bool // Extra模式直接跳过难度选择
	practice      bool
	practiceStage int
	back          game.Scene // 返回目标（Title）
	bgImg         *ebiten.Image
	charImgs      [2]*ebiten.Image // Reimu, Marisa portraits

	// 列表型 phase 用 MenuList；卡片型 phase（character/shotType）仍手画
	practiceMenu *sceneui.MenuList
	diffMenu     *sceneui.MenuList
}

func New(back game.Scene) *Select {
	s := &Select{back: back, phase: phaseCharacter}
	s.loadAssets()
	return s
}

// NewExtra 创建 Extra 模式选人界面（跳过难度选择）
func NewExtra(back game.Scene) *Select {
	s := &Select{phase: phaseCharacter, isExtra: true, difficulty: game.DiffExtra, back: back}
	s.loadAssets()
	return s
}

func NewPractice(back game.Scene) *Select {
	s := &Select{phase: phasePracticeStage, practice: true, practiceStage: 1, back: back}
	s.loadAssets()
	return s
}

func (s *Select) loadAssets() {
	s.bgImg = render.LoadImage("anm/title/select00s.png")
	s.charImgs[0] = render.LoadImage("anm/title/sl_pl00.png")
	s.charImgs[1] = render.LoadImage("anm/title/sl_pl01.png")
}

// 字体（render.Load*Face 内部 memo，每帧调用零成本）
func titleFace() ebitext.Face { return render.LoadSystemFace(24) }
func bodyFace() ebitext.Face  { return render.LoadSystemFace(18) }
func hintFace() ebitext.Face  { return render.LoadSystemFace(14) }

func (s *Select) ensurePracticeMenu() *sceneui.MenuList {
	if s.practiceMenu != nil {
		return s.practiceMenu
	}
	items := make([]sceneui.MenuItem, game.StageCount)
	for i := 0; i < game.StageCount; i++ {
		items[i] = sceneui.MenuItem{
			Label: fmt.Sprintf("Stage %d - %s", i+1, game.StageName(i+1)),
		}
	}
	s.practiceMenu = &sceneui.MenuList{
		Items:  items,
		Cursor: s.practiceStage - 1,
		X:      174, Y: 132, LineH: 34,
		Style: sceneui.MenuStyle{
			Face:         bodyFace(),
			Idle:         colDim,
			Hover:        colWhite,
			Shadow:       colShadow,
			HoverBgColor: colHoverBg,
			HoverBgPadX:  8,
			HoverBgH:     26,
		},
	}
	return s.practiceMenu
}

func (s *Select) ensureDiffMenu() *sceneui.MenuList {
	if s.diffMenu != nil {
		return s.diffMenu
	}
	items := []sceneui.MenuItem{
		{Label: game.DifficultyName(game.DiffEasy)},
		{Label: game.DifficultyName(game.DiffNormal)},
		{Label: game.DifficultyName(game.DiffHard)},
		{Label: game.DifficultyName(game.DiffLuna)},
	}
	s.diffMenu = &sceneui.MenuList{
		Items:  items,
		Cursor: s.difficulty,
		X:      244, Y: 150, LineH: 40,
		Style: sceneui.MenuStyle{
			Face:         bodyFace(),
			Idle:         colDim,
			Hover:        colWhite,
			Shadow:       colShadow,
			HoverBgColor: colHoverBg,
			HoverBgPadX:  8,
			HoverBgH:     28,
		},
	}
	return s.diffMenu
}

func (s *Select) Update(g *game.Game) error {
	in := &input.Global

	switch s.phase {
	case phasePracticeStage:
		menu := s.ensurePracticeMenu()
		menu.Update()
		s.practiceStage = menu.Cursor + 1
		if in.JustPressed(input.KeyOk) {
			s.phase = phaseCharacter
		}
		if in.JustPressed(input.KeyCancel) {
			g.GoTo(s.back)
		}

	case phaseCharacter:
		if in.JustPressed(input.KeyLeft) || in.JustPressed(input.KeyRight) {
			s.character = 1 - s.character
		}
		if in.JustPressed(input.KeyOk) {
			s.phase = phaseShotType
		}
		if in.JustPressed(input.KeyCancel) {
			if s.practice {
				s.phase = phasePracticeStage
			} else {
				g.GoTo(s.back)
			}
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
				s.startGame(g)
			} else {
				s.phase = phaseDifficulty
			}
		}
		if in.JustPressed(input.KeyCancel) {
			s.phase = phaseCharacter
		}

	case phaseDifficulty:
		menu := s.ensureDiffMenu()
		menu.Update()
		s.difficulty = menu.Cursor
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
	if s.practice {
		state.Stage = s.practiceStage
		state.SingleStage = true
	} else if s.isExtra {
		state.Stage = game.StageExtra
	} else {
		state.Stage = 1
	}
	g.BeginRun(state)
	g.GoTo(buildStage(g, state, s.back))
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

func buildStage(g *game.Game, state *game.GameState, back game.Scene) *stage.Stage {
	s := stage.New(g, state, stageScript(state.Stage))
	s.OnGameOver = func(g *game.Game, st *game.GameState) game.Scene {
		return resultscene.New(st, resultscene.ModeGameOver, back)
	}
	s.NextScene = func(g *game.Game, st *game.GameState) game.Scene {
		if st.SingleStage {
			mode := resultscene.ModeStageClear
			if st.Stage == game.StageExtra {
				mode = resultscene.ModeExtraClear
			}
			if !st.DebugMode {
				g.MarkClear(st)
			}
			return resultscene.New(st, mode, back)
		}
		next := st.NextStage()
		if next == 0 {
			g.MarkClear(st)
			return ending.New(st, st.Stage == game.StageExtra, back)
		}
		st.Stage = next
		st.Frame = 0
		return buildStage(g, st, back)
	}
	return s
}

// NewDebugStage 为 go test 的手动调试入口创建直接进入关卡的场景。
func NewDebugStage(g *game.Game, back game.Scene, stageNum, character, shotType, difficulty int) game.Scene {
	state := game.NewState(character, shotType, difficulty)
	state.Stage = stageNum
	state.SingleStage = true
	state.DebugMode = true
	state.ExtraUnlocked = true
	if stageNum == game.StageExtra {
		state.Difficulty = game.DiffExtra
	}
	g.BeginRun(state)
	return buildStage(g, state, back)
}

func (s *Select) Draw(screen *ebiten.Image) {
	// 背景
	if s.bgImg != nil {
		screen.DrawImage(s.bgImg, nil)
	} else {
		screen.Fill(color.NRGBA{12, 8, 32, 255})
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

	tFace := titleFace()

	switch s.phase {
	case phasePracticeStage:
		render.DrawTextShadow(screen, "选择练习关卡", tFace, 236, 60, colGold, colShadow, 2, 2)
		s.ensurePracticeMenu().Draw(screen)

	case phaseCharacter:
		render.DrawTextShadow(screen, "选择角色", tFace, 258, 60, colGold, colShadow, 2, 2)
		for i := 0; i < 2; i++ {
			x := 140 + i*200
			y := 160
			c := colCardIdle
			if i == s.character {
				c = colCardHover
			}
			vector.FillRect(screen, float32(x), float32(y), 150, 120, c, false)
			render.DrawTextShadow(screen, game.CharacterName(i), tFace, x+10, y+18, colWhite, colShadow, 1, 1)
			render.DrawTextShadow(screen, game.CharacterTitle(i), bodyFace(), x+10, y+60, colDim, colShadow, 1, 1)
		}

	case phaseShotType:
		render.DrawTextShadow(screen,
			fmt.Sprintf("%s - 选择装备", game.CharacterName(s.character)),
			tFace, 200, 60, colGold, colShadow, 2, 2)
		for i := 0; i < 3; i++ {
			x := 80 + i*170
			y := 180
			c := colCardIdle
			if i == s.shotType {
				c = colCardHover
			}
			vector.FillRect(screen, float32(x), float32(y), 150, 90, c, false)
			render.DrawTextShadow(screen, fmt.Sprintf("Type %c", 'A'+i), tFace, x+40, y+8, colWhite, colShadow, 1, 1)
			render.DrawTextShadow(screen, game.ShotDescription(s.character, i), bodyFace(), x+6, y+48, colDim, colShadow, 1, 1)
		}

	case phaseDifficulty:
		render.DrawTextShadow(screen, "选择难度", tFace, 258, 60, colGold, colShadow, 2, 2)
		s.ensureDiffMenu().Draw(screen)
	}

	render.DrawTextShadow(screen, "Z: Select   X: Cancel", hintFace(), 240, 422, colDim, colShadow, 1, 1)
}
