// Command th10sim runs a stage headlessly from an official replay file and
// writes a per-frame trace in the shared trace format.
//
// Usage:
//
//	th10sim -replay path/to/demoN.rpy [-chapter 0] [-stage 1] [-frames 3600] -out trace.txt
//
// The stage defaults to the chapter index recorded in the replay, and the
// character/shot/difficulty come from the replay header, so the replica runs
// the same configuration the original did.
package main

import (
	"flag"
	"fmt"
	"os"

	"th10/game"
	"th10/replay"
	"th10/scene/stage"
	"th10/scene/stage/stage1"
	"th10/scene/stage/stage2"
	"th10/scene/stage/stage3"
	"th10/scene/stage/stage4"
	"th10/scene/stage/stage5"
	"th10/scene/stage/stage6"
	"th10/sim"
)

func main() {
	os.Exit(run())
}

func run() int {
	var (
		replayPath = flag.String("replay", "", "path to a .rpy replay file")
		chapterIdx = flag.Int("chapter", 0, "chapter index within the replay (0-based)")
		stageNum   = flag.Int("stage", 0, "override stage number (0 = use the replay's)")
		frames     = flag.Int("frames", 3600, "max frames to simulate (0 = all)")
		outPath    = flag.String("out", "-", "output trace path (- = stdout)")
	)
	flag.Parse()
	if *replayPath == "" {
		flag.Usage()
		return 2
	}

	raw, err := os.ReadFile(*replayPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, "th10sim:", err)
		return 2
	}
	rp, err := replay.Parse(raw)
	if err != nil {
		fmt.Fprintln(os.Stderr, "th10sim:", err)
		return 2
	}
	if *chapterIdx < 0 || *chapterIdx >= len(rp.Chapters) {
		fmt.Fprintf(os.Stderr, "th10sim: chapter %d out of range (replay has %d)\n",
			*chapterIdx, len(rp.Chapters))
		return 2
	}
	ch := rp.Chapters[*chapterIdx]

	stage := int(rp.Global.Stage)
	if *stageNum > 0 {
		stage = *stageNum
	}
	if stage < 1 || stage > 6 {
		fmt.Fprintf(os.Stderr, "th10sim: stage %d not simulatable (1..6)\n", stage)
		return 2
	}
	cfg := sim.Config{
		Stage:      stage,
		Character:  int(rp.Global.Character),
		ShotType:   int(rp.Global.ShotType),
		Difficulty: int(rp.Global.Difficulty),
		Power:      startingPower(rp),
		Seed:       ch.Seed,
	}
	if cfg.Difficulty > game.DiffLuna {
		cfg.Difficulty = game.DiffNormal
	}

	out := os.Stdout
	if *outPath != "-" {
		f, err := os.Create(*outPath)
		if err != nil {
			fmt.Fprintln(os.Stderr, "th10sim:", err)
			return 2
		}
		defer f.Close()
		out = f
	}

	runner := sim.New(cfg, scriptFactory)
	fmt.Fprintf(os.Stderr, "th10sim: stage=%d char=%d shot=%d diff=%d chapter=%d records=%d\n",
		cfg.Stage, cfg.Character, cfg.ShotType, cfg.Difficulty, ch.Index, len(ch.Records))
	if err := runner.Run(ch, out, *frames); err != nil {
		fmt.Fprintln(os.Stderr, "th10sim:", err)
		return 2
	}
	return 0
}

// startingPower returns the replay's recorded power when available. The
// original stores it in the chapter header; we do not decode that field yet,
// so runs start from zero as a normal new game does.
func startingPower(_ *replay.Replay) int { return 0 }

func scriptFactory(stageNum int) stage.Script {
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
	default:
		return stage1.NewScript()
	}
}
