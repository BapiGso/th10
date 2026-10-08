package main

import (
	"flag"
	"log"
	"strconv"
	"strings"
	"th10/game"
	"th10/scene/loading"
	selector "th10/scene/select"
	"th10/scene/title"
	"th10/sprite"

	"github.com/hajimehoshi/ebiten/v2"
)

func main() {
	debugStage := flag.String("debug-stage", "", "debug stage: 1-6 or extra")
	debugDiff := flag.String("debug-diff", "normal", "debug difficulty: easy, normal, hard, lunatic, or extra")
	debugPower := flag.Int("debug-power", 0, "debug starting power in hundredths: 0-500")
	flag.Parse()

	ebiten.SetWindowSize(game.ScreenWidth, game.ScreenHeight)
	ebiten.SetWindowTitle("东方风神录 ~ Mountain of Faith")

	g := game.New()
	if stageNum, ok := parseDebugStageArg(*debugStage); ok {
		difficulty := parseDebugDifficultyArg(*debugDiff)
		if stageNum == game.StageExtra {
			difficulty = game.DiffExtra
		}
		if *debugPower < 0 || *debugPower > game.PowerMax {
			log.Fatal("-debug-power must be between 0 and 500")
		}
		sprite.Load()
		g.SetScene(selector.NewDebugStageWithPower(g, title.New(), stageNum, game.CharReimu, game.ShotA, difficulty, *debugPower))
	} else {
		g.SetScene(loading.New())
	}

	if err := ebiten.RunGame(g); err != nil {
		log.Fatal(err)
	}
}

func parseDebugStageArg(raw string) (int, bool) {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "":
		return 0, false
	case "extra", "stage7", "st7", "7":
		return game.StageExtra, true
	}
	n, err := strconv.Atoi(strings.TrimSpace(raw))
	if err == nil && n >= 1 && n <= 6 {
		return n, true
	}
	return 0, false
}

func parseDebugDifficultyArg(raw string) int {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "easy", "0":
		return game.DiffEasy
	case "hard", "2":
		return game.DiffHard
	case "lunatic", "luna", "3":
		return game.DiffLuna
	case "extra", "4":
		return game.DiffExtra
	default:
		return game.DiffNormal
	}
}
