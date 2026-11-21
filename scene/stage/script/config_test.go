package stagescript_test

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"strings"
	"testing"

	"th10/audio"
	"th10/game"
	"th10/scene/stage"
	"th10/scene/stage/extra"
	stagescript "th10/scene/stage/script"
	"th10/scene/stage/stage1"
	"th10/scene/stage/stage2"
	"th10/scene/stage/stage3"
	"th10/scene/stage/stage4"
	"th10/scene/stage/stage5"
	"th10/scene/stage/stage6"
)

var (
	_ func() stage.Script       = stage1.NewScript
	_ func() stage.Script       = stage1.NewECLScript
	_ func() *stage2.Script2    = stage2.NewScript
	_ func() *stage3.Script3    = stage3.NewScript
	_ func() *stage4.Script4    = stage4.NewScript
	_ func() *stage5.Script5    = stage5.NewScript
	_ func() *stage6.Script6    = stage6.NewScript
	_ func() *extra.ScriptExtra = extra.NewScript
)

func TestStagePackagesContainOnlyConfiguration(t *testing.T) {
	for _, dir := range []string{"stage1", "stage2", "stage3", "stage4", "stage5", "stage6", "extra"} {
		paths, err := filepath.Glob(filepath.Join("..", dir, "*.go"))
		if err != nil || len(paths) == 0 {
			t.Fatalf("%s source files: %v", dir, err)
		}
		for _, path := range paths {
			if strings.HasSuffix(path, "_test.go") {
				continue
			}
			f, err := parser.ParseFile(token.NewFileSet(), path, nil, 0)
			if err != nil {
				t.Fatal(err)
			}
			ast.Inspect(f, func(n ast.Node) bool {
				switch n := n.(type) {
				case *ast.FuncDecl, *ast.FuncLit, *ast.CallExpr:
					t.Errorf("%s contains executable behavior: %T", path, n)
				case *ast.TypeSpec:
					if n.Assign == token.NoPos {
						t.Errorf("%s defines a runtime type instead of a compatibility alias", path)
					}
				}
				return true
			})
		}
	}
}

type stageSound struct{ bgm []int }

func (s *stageSound) PlayBGM(id int) { s.bgm = append(s.bgm, id) }
func (*stageSound) PlaySE(string)    {}
func (*stageSound) PlaySEID(int)     {}
func (*stageSound) StopBGM()         {}
func (*stageSound) FadeOutBGM(int)   {}

func TestStageConfigurationFactoriesBindBeforeFirstCalc(t *testing.T) {
	configs := []stagescript.Config{stage1.Config, stage2.Config, stage3.Config, stage4.Config, stage5.Config, stage6.Config, extra.Config}
	bgm := []int{audio.BGMStage1, audio.BGMStage2, audio.BGMStage3, audio.BGMStage4, audio.BGMStage5, audio.BGMStage6, audio.BGMExtra}
	boss := []int{audio.BGMStage1Boss, audio.BGMStage2Boss, audio.BGMStage3Boss, audio.BGMStage4Boss, audio.BGMStage5Boss, audio.BGMStage6Boss, audio.BGMExtraBoss}
	scroll := []float64{0, 2, 1.8, 2.5, 1.5, 1.2, 2}
	images := [][5]string{
		{"stg1bg.png", "stg1bg2.png", "stg1bg3.png", "stg1bg4.png"},
		{"stg2bg.png", "stg2bg2.png"},
		{"stg3bg.png", "stg3bg2.png", "stg3bg3.png", "stg3bg4.png"},
		{"stg4bg.png", "stg4bg3.png", "stg4bg7.png"},
		{"stg5bg.png", "stg5bg2.png"},
		{"stg6bg.png", "stg6bg2.png", "stg6bg3.png", "stg6bg5.png", "stg6bg6.png"},
		{"stg7bg.png", "stg7bg2.png", "stg7bg3.png"},
	}
	for i, config := range configs {
		t.Run(fmt.Sprint(i+1), func(t *testing.T) {
			msg := fmt.Sprintf("stage%d", i+1)
			if i == 6 {
				msg = "extra"
			}
			if config.ECL != fmt.Sprintf("ecl/stage%02d.ecl", i+1) || config.MSG != msg || config.BGM != bgm[i] || config.BossBGM != boss[i] || config.FrameCap != 60000 || config.BackgroundStage != i+1 || config.BackgroundScroll != scroll[i] {
				t.Fatalf("changed stage configuration: %+v", config)
			}
			for n, name := range images[i] {
				if name != "" {
					name = "anm/background/" + name
				}
				if config.Backgrounds[n] != name {
					t.Fatalf("background %d = %q, want %q", n, config.Backgrounds[n], name)
				}
			}
			s := config.NewScript()
			state := game.NewState(0, 0, 0)
			state.Stage = i + 1
			ctx := stage.NewContext(state)
			sound := &stageSound{}
			ctx.Audio = sound
			s.Init(ctx)
			if ctx.ManagerOrder == nil || ctx.Boss0 == nil || ctx.Spell == nil || ctx.CalcShake == nil || !ctx.Player.Bullets.ManagerPresent() {
				t.Fatal("FUN_00417870 manager hooks are unavailable before the first calc")
			}
			if len(sound.bgm) != 1 || sound.bgm[0] != config.BGM {
				t.Fatalf("initial BGM calls = %v", sound.bgm)
			}
			ctx.OpenMSG(0)
			if ctx.MSG() == nil {
				t.Fatal("configuration did not load its original MSG")
			}
			ctx.World.Clear()
		})
	}
}
