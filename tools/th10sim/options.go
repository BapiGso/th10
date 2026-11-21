package main

import (
	"flag"
	"fmt"
	"io"
)

type simOptions struct {
	Replay, Out, OutDir, Initial, InitialDir string
	Chapter, Stage, Frames, Demo             int
	Ext, All, Info                           bool
	HiScore                                  int64
	HiContinues                              int
}

func parseSimOptions(args []string, log io.Writer) (simOptions, error) {
	var o simOptions
	fs := flag.NewFlagSet("th10sim", flag.ContinueOnError)
	fs.SetOutput(log)
	fs.StringVar(&o.Replay, "replay", "", "path to a .rpy replay file")
	fs.IntVar(&o.Demo, "demo", -1, "embedded official demo number (0..3)")
	fs.IntVar(&o.Chapter, "chapter", 0, "chapter index within the replay (0-based)")
	fs.IntVar(&o.Stage, "stage", 0, "override stage number (0 = recorded stage)")
	fs.IntVar(&o.Frames, "frames", 3600, "max frames per chapter (0 = full input/world)")
	fs.StringVar(&o.Out, "out", "-", "trace output (- = stdout)")
	fs.StringVar(&o.OutDir, "out-dir", "", "per-chapter output directory, needs -all-chapters")
	fs.BoolVar(&o.All, "all-chapters", false, "run every remaining replay chapter (default -frames 0)")
	fs.BoolVar(&o.Ext, "ext", false, "include native stats, player timers, full items, lasers and spell fields")
	fs.Int64Var(&o.HiScore, "hiscore", -1, "initial displayed high score (-1 = headless default)")
	fs.IntVar(&o.HiContinues, "hiscore-continues", 0, "continuation digit belonging to initial high score")
	fs.StringVar(&o.Initial, "initial", "", "capture trace whose C header supplies the observed entry high score")
	fs.StringVar(&o.InitialDir, "initial-dir", "", "per-chapter capture directory for entry high scores (needs -all-chapters)")
	fs.BoolVar(&o.Info, "info", false, "print replay chapters/record counts and exit without simulating")
	if err := fs.Parse(args); err != nil {
		return o, err
	}
	explicitFrames := false
	fs.Visit(func(f *flag.Flag) {
		if f.Name == "frames" {
			explicitFrames = true
		}
	})
	if !explicitFrames {
		if o.All {
			o.Frames = 0
		} else if o.Demo >= 0 {
			o.Frames = 3001
		}
	}
	if fs.NArg() != 0 || o.Replay == "" && o.Demo < 0 || o.Replay != "" && o.Demo >= 0 || o.Demo > 3 || o.Demo < -1 || o.Chapter < 0 || o.Stage < 0 || o.Stage > 7 || o.Frames < 0 ||
		o.All && (o.OutDir == "" || o.Out != "-" || o.Stage != 0 || o.Initial != "") || !o.All && (o.OutDir != "" || o.InitialDir != "") ||
		o.HiScore < -1 || o.HiScore > 9999999990 || o.HiContinues < 0 || o.HiContinues > 9 || (o.Initial != "" || o.InitialDir != "") && o.HiScore >= 0 {
		return o, fmt.Errorf("invalid replay/demo/chapter/output/initial-state options")
	}
	return o, nil
}
