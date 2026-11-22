package stagescript_test

import (
	"fmt"
	"testing"

	"th10/audio"
	"th10/game"
	"th10/scene/stage"
	stagescript "th10/scene/stage/script"
)

var _ func(int) stage.Script = stagescript.New

type stageSound struct{ bgm []int }

func (s *stageSound) PlayBGM(id int) { s.bgm = append(s.bgm, id) }
func (*stageSound) PlaySEID(int)     {}
func (*stageSound) StopBGM()         {}
func (*stageSound) FadeOutBGM(int)   {}

func TestStageConfigurationFactoriesBindBeforeFirstCalc(t *testing.T) {
	configs := stagescript.Configs
	bgm := []int{audio.BGMStage1, audio.BGMStage2, audio.BGMStage3, audio.BGMStage4, audio.BGMStage5, audio.BGMStage6, audio.BGMExtra}
	boss := []int{audio.BGMStage1Boss, audio.BGMStage2Boss, audio.BGMStage3Boss, audio.BGMStage4Boss, audio.BGMStage5Boss, audio.BGMStage6Boss, audio.BGMExtraBoss}
	for i, config := range configs {
		t.Run(fmt.Sprint(i+1), func(t *testing.T) {
			msg := fmt.Sprintf("stage%d", i+1)
			if i == 6 {
				msg = "extra"
			}
			if config.ECL != fmt.Sprintf("ecl/stage%02d.ecl", i+1) || config.MSG != msg || config.BGM != bgm[i] || config.BossBGM != boss[i] || config.FrameCap != 60000 {
				t.Fatalf("changed stage configuration: %+v", config)
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
