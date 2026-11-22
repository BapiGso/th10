// Command th10sim drives the existing headless replay runner. Optional capture
// output and diagnostic observers never change the unextended default trace.
package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"th10/game"
	"th10/gamedata"
	"th10/replay"
	"th10/scene/stage"
	stagescript "th10/scene/stage/script"
	"th10/sim"
	"th10/trace"
)

func main() { os.Exit(run()) }

func run() int {
	o, err := parseSimOptions(os.Args[1:], os.Stderr)
	if err != nil {
		_, _ = fmt.Fprintln(os.Stderr, "th10sim:", err)
		return 2
	}
	if err := simulate(o, os.Stderr); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, "th10sim:", err)
		return 2
	}
	return 0
}

func replaySource(o simOptions) (*replay.Replay, string, error) {
	var raw []byte
	var err error
	name := o.Replay
	if o.Demo >= 0 {
		name = fmt.Sprintf("misc/demo%d.rpy", o.Demo)
		raw, err = gamedata.ReadFile(name)
	} else {
		raw, err = os.ReadFile(name)
	}
	if err != nil {
		return nil, "", err
	}
	rp, err := replay.Parse(raw)
	if err != nil {
		return nil, "", err
	}
	return rp, strings.TrimSuffix(filepath.Base(name), filepath.Ext(name)), nil
}

func simulate(o simOptions, log io.Writer) error {
	rp, base, err := replaySource(o)
	if err != nil {
		return err
	}
	if o.Chapter >= len(rp.Chapters) {
		return fmt.Errorf("chapter %d out of range (replay has %d)", o.Chapter, len(rp.Chapters))
	}
	if o.Info {
		for i, ch := range rp.Chapters {
			cfg, err := sim.ConfigFor(rp, i)
			if err != nil {
				return err
			}
			_, _ = fmt.Fprintf(log, "chapter=%d stage=%d char=%d shot=%d diff=%d records=%d duration~%.1fs\n", i, cfg.Stage, cfg.Character, cfg.ShotType, cfg.Difficulty, len(ch.Records), float64(len(ch.Records))/60)
		}
		return nil
	}
	if o.All {
		if err := os.MkdirAll(o.OutDir, 0o755); err != nil {
			return err
		}
	}
	last := o.Chapter + 1
	if o.All {
		last = len(rp.Chapters)
	}
	hi, continues := o.HiScore, o.HiContinues
	for index := o.Chapter; index < last; index++ {
		if o.Stage > 0 {
			rp.Chapters[index].Index = uint16(o.Stage)
		}
		cfg, err := sim.ConfigFor(rp, index)
		if err != nil {
			return err
		}
		initial := o.Initial
		if o.InitialDir != "" {
			initial = filepath.Join(o.InitialDir, fmt.Sprintf("%s_c%d.txt", base, index))
		}
		if initial != "" {
			h, err := readInitial(initial)
			if err != nil {
				return err
			}
			if h.Stage != cfg.Stage || h.Character != cfg.Character || h.Shot != cfg.ShotType || h.Difficulty != cfg.Difficulty {
				return fmt.Errorf("initial capture does not match replay chapter %d", index)
			}
			hi, continues = h.HiScore, h.HiScoreContinues
		}
		path := o.Out
		if o.All {
			path = filepath.Join(o.OutDir, fmt.Sprintf("%s_c%d.txt", base, index))
		}
		hi, continues, err = simulateChapter(o, rp, index, cfg, path, hi, continues, log)
		if err != nil {
			return err
		}
	}
	return nil
}

func readInitial(path string) (trace.CaptureHeader, error) {
	f, err := os.Open(path)
	if err != nil {
		return trace.CaptureHeader{}, err
	}
	defer func() { _ = f.Close() }()
	r := trace.NewReader(f)
	if _, err := r.Next(); err != nil {
		return trace.CaptureHeader{}, err
	}
	if h := r.Capture(); h != nil {
		return *h, nil
	}
	return trace.CaptureHeader{}, fmt.Errorf("%s has no C capture header; use explicit -hiscore for old traces", path)
}

func simulateChapter(o simOptions, rp *replay.Replay, index int, cfg sim.Config, path string, hi int64, continues int, log io.Writer) (endHi int64, endContinues int, result error) {
	out := os.Stdout
	var file *os.File
	if path != "-" {
		var err error
		if o.Ext || o.All {
			file, err = os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o644)
		} else {
			file, err = os.Create(path)
		}
		if err != nil {
			return hi, continues, err
		}
		out = file
	}
	var tw *trace.Writer
	status := "error"
	defer func() {
		if tw != nil {
			result = errors.Join(result, tw.EndCapture(status))
		}
		if file != nil {
			if tw != nil {
				result = errors.Join(result, file.Sync())
			}
			result = errors.Join(result, file.Close())
		}
	}()
	runner := sim.New(cfg, stagescript.New)
	if o.Ext {
		runner.TraceFrame = extendedFrame
	}
	if o.Demo >= 0 || hi >= 0 {
		runner.ConfigureState = func(s *game.GameState) {
			if o.Demo >= 0 {
				s.RunFlags |= game.RunFlagDemo
			}
			if hi >= 0 {
				s.HiScore, s.HiScoreContinues = hi, int32(continues)
			}
		}
	}
	if o.Ext || o.All {
		tw = trace.NewAtomicWriter(out)
		header := trace.CaptureHeader{Chapter: index, Stage: cfg.Stage, Demo: o.Demo, Character: cfg.Character, Shot: cfg.ShotType, Difficulty: cfg.Difficulty, HiScore: max(hi, 0), HiScoreContinues: continues, Records: len(rp.Chapters[index].Records), Limit: o.Frames}
		if o.Ext {
			header.Extension = trace.ExtendedVersion
		}
		if err := tw.BeginCapture(header); err != nil {
			return hi, continues, err
		}
		runner.Writer = tw
	}
	endHi, endContinues = hi, continues
	runner.OnFrame = func(st *stage.Stage) {
		s := stage.ExportContext(st).State
		endHi, endContinues = s.HiScore, int(s.HiScoreContinues)
	}
	ch := rp.Chapters[index]
	_, _ = fmt.Fprintf(log, "th10sim: stage=%d char=%d shot=%d diff=%d chapter=%d records=%d\n", cfg.Stage, cfg.Character, cfg.ShotType, cfg.Difficulty, ch.Index, len(ch.Records))
	if err := runner.Run(ch, out, o.Frames); err != nil {
		return endHi, endContinues, err
	}
	status = "complete"
	return endHi, endContinues, nil
}
