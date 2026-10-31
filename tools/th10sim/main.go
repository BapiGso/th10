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
	"th10/scene/stage/extra"
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

	// The global header's stage is where the run ended (8 = all clear); each
	// chapter records its own stage.
	stage := int(ch.Index)
	if stage < 1 || stage > 7 {
		stage = int(rp.Global.Stage)
	}
	if *stageNum > 0 {
		stage = *stageNum
	}
	if stage < 1 || stage > 7 {
		fmt.Fprintf(os.Stderr, "th10sim: stage %d not simulatable (1..7)\n", stage)
		return 2
	}
	cfg := sim.Config{
		Stage:      stage,
		Character:  int(rp.Global.Character),
		ShotType:   int(rp.Global.ShotType),
		Difficulty: int(rp.Global.Difficulty),
		Power:      int(ch.Power) * 5, // 0.05 units to hundredths
		Lives:      int(ch.Lives),
		Seed:       ch.Seed,
		Rank:       int(ch.Rank),
		PlayerX:    ch.PlayerX,
		PlayerY:    ch.PlayerY,
	}
	// Extra (stage 7) runs at game.DiffExtra, which is above Luna but valid.
	// Any other difficulty above Luna is out of range and runs as Normal.
	if cfg.Difficulty > game.DiffLuna && !(stage == 7 && cfg.Difficulty == game.DiffExtra) {
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
	case 7:
		return extra.NewScript()
	default:
		return stage1.NewScript()
	}
}
