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

	// -stage replaces the chapter's stage before ConfigFor, which clamps the
	// difficulty by that stage and falls back to the global stage when the
	// chapter's is invalid. ch keeps the recorded index for the log line.
	if *stageNum > 0 {
		if *stageNum > 7 {
			fmt.Fprintf(os.Stderr, "th10sim: stage %d not simulatable (1..7)\n", *stageNum)
			return 2
		}
		rp.Chapters[*chapterIdx].Index = uint16(*stageNum)
	}
	cfg, err := sim.ConfigFor(rp, *chapterIdx)
	if err != nil {
		fmt.Fprintln(os.Stderr, "th10sim:", err)
		return 2
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
