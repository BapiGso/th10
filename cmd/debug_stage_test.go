package main

import (
	"flag"
	"fmt"
	"strings"
	"testing"
	"th10/game"
	selector "th10/scene/select"
	"th10/scene/title"
	"th10/sprite"

	"github.com/hajimehoshi/ebiten/v2"
)

var (
	debugStageFlag = flag.String("debug-stage", "", "manual stage debug target: 1-6 or extra")
	debugCharFlag  = flag.String("debug-char", "reimu", "manual debug character: reimu or marisa")
	debugShotFlag  = flag.String("debug-shot", "a", "manual debug shot type: a, b, or c")
	debugDiffFlag  = flag.String("debug-diff", "normal", "manual debug difficulty: easy, normal, hard, lunatic, or extra")
)

// TestDebugStage 是手动调试入口，不会在默认 go test 中启动。
//
// 示例：
//
//	go test ./cmd -run TestDebugStage -timeout 0 -args -debug-stage=4 -debug-char=marisa -debug-shot=b -debug-diff=hard
//	go test ./cmd -run TestDebugStage -timeout 0 -args -debug-stage=extra -debug-char=reimu -debug-shot=c
func TestDebugStage(t *testing.T) {
	if strings.TrimSpace(*debugStageFlag) == "" {
		t.Skip("manual debug entry; pass -args -debug-stage=<1-6|extra>")
	}

	stageNum, err := parseDebugStage(*debugStageFlag)
	if err != nil {
		t.Fatal(err)
	}
	character, err := parseDebugCharacter(*debugCharFlag)
	if err != nil {
		t.Fatal(err)
	}
	shotType, err := parseDebugShot(*debugShotFlag)
	if err != nil {
		t.Fatal(err)
	}
	difficulty, err := parseDebugDifficulty(*debugDiffFlag)
	if err != nil {
		t.Fatal(err)
	}
	if stageNum == game.StageExtra {
		difficulty = game.DiffExtra
	}

	ebiten.SetWindowSize(game.ScreenWidth, game.ScreenHeight)
	ebiten.SetWindowTitle(fmt.Sprintf("th10 debug stage=%s char=%s shot=%s diff=%s", *debugStageFlag, *debugCharFlag, strings.ToUpper(*debugShotFlag), *debugDiffFlag))

	g := game.New()
	sprite.Load()
	g.SetScene(selector.NewDebugStage(g, title.New(), stageNum, character, shotType, difficulty))

	if err := ebiten.RunGame(g); err != nil && err != ebiten.Termination {
		t.Fatal(err)
	}
}

func parseDebugStage(raw string) (int, error) {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "1", "stage1", "st1":
		return 1, nil
	case "2", "stage2", "st2":
		return 2, nil
	case "3", "stage3", "st3":
		return 3, nil
	case "4", "stage4", "st4":
		return 4, nil
	case "5", "stage5", "st5":
		return 5, nil
	case "6", "stage6", "st6":
		return 6, nil
	case "7", "extra", "stage7", "st7":
		return game.StageExtra, nil
	default:
		return 0, fmt.Errorf("invalid -debug-stage=%q", raw)
	}
}

func parseDebugCharacter(raw string) (int, error) {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "reimu", "灵梦", "hakurei":
		return game.CharReimu, nil
	case "marisa", "魔理沙", "kirisame":
		return game.CharMarisa, nil
	default:
		return 0, fmt.Errorf("invalid -debug-char=%q", raw)
	}
}

func parseDebugShot(raw string) (int, error) {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "a", "0":
		return game.ShotA, nil
	case "b", "1":
		return game.ShotB, nil
	case "c", "2":
		return game.ShotC, nil
	default:
		return 0, fmt.Errorf("invalid -debug-shot=%q", raw)
	}
}

func parseDebugDifficulty(raw string) (int, error) {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "easy", "0":
		return game.DiffEasy, nil
	case "normal", "1":
		return game.DiffNormal, nil
	case "hard", "2":
		return game.DiffHard, nil
	case "lunatic", "luna", "3":
		return game.DiffLuna, nil
	case "extra", "4":
		return game.DiffExtra, nil
	default:
		return 0, fmt.Errorf("invalid -debug-diff=%q", raw)
	}
}
