package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"th10/replay"
	"th10/trace"
)

type capturePlan struct {
	options probeOptions
	replay  *replay.Replay
	base    string
}

func newCapturePlan(o probeOptions) (*capturePlan, error) {
	p := &capturePlan{options: o}
	if o.Replay != "" {
		raw, err := os.ReadFile(o.Replay)
		if err != nil {
			return nil, err
		}
		p.replay, err = replay.Parse(raw)
		if err != nil {
			return nil, err
		}
		if err := p.replay.ValidatePlayback(); err != nil {
			return nil, err
		}
		if o.Chapter < 0 || o.Chapter >= len(p.replay.Chapters) {
			return nil, fmt.Errorf("chapter %d out of range", o.Chapter)
		}
		p.base = strings.TrimSuffix(filepath.Base(o.Replay), filepath.Ext(o.Replay))
	}
	return p, nil
}

type captureInfo struct {
	header   trace.CaptureHeader
	filename string
}

func (r *reader) captureInfo(c control) (info captureInfo, err error) {
	g, err := r.bytes(addrScoreBlock, 0x70)
	if err != nil {
		return info, err
	}
	info.header = trace.CaptureHeader{Stage: i32(g[0x3c:]), Demo: -1, Character: i32(g[0x28:]), Shot: i32(g[0x2c:]), Difficulty: i32(g[0x34:]),
		HiScore: int64(int32(le32(g))) * 10, HiScoreContinues: i32(g[0x54:])}
	if c.Replay != 0 {
		name, err := r.bytes(addrReplayName, 256)
		if err != nil {
			return info, err
		}
		if n := strings.IndexByte(string(name), 0); n >= 0 {
			info.filename = string(name[:n])
		} else {
			return info, fmt.Errorf("unterminated native replay filename")
		}
	}
	if c.Header != 0 {
		h, err := r.bytes(c.Header, 0x1c4)
		if err != nil {
			return info, err
		}
		info.header.Stage = int(le16(h))
		info.header.Records = int(le32(h[4:])) // native declared limit; local -replay replaces it with actual records
	}
	return info, nil
}

func nativeBase(name string) string { return path.Base(strings.ReplaceAll(name, "\\", "/")) }

func demoNumber(name string) (int, bool) {
	s := strings.ToLower(nativeBase(name))
	if len(s) != 9 || !strings.HasPrefix(s, "demo") || s[5:] != ".rpy" || s[4] < '0' || s[4] > '3' {
		return 0, false
	}
	return int(s[4] - '0'), true
}

// selectInfo uses the active native chapter, not DAT_00474c7c which MSG op 20
// increments before the old chapter's final frame is sampled (0x416ee8).
func (p *capturePlan) selectInfo(c control, info captureInfo) (captureInfo, string, bool) {
	o := p.options
	info.header.Extension, info.header.Limit = 0, o.Frames
	if o.Ext {
		info.header.Extension = trace.ExtendedVersion
	}
	if o.Demos {
		n, ok := demoNumber(info.filename)
		if !ok || c.RunFlags&0x20 == 0 || c.ReplayMode != 1 {
			return info, "", false
		}
		info.header.Demo, info.header.Chapter = n, 0
		return info, fmt.Sprintf("demo%d", n), true
	}
	if p.replay != nil {
		if c.ReplayMode != 1 || !strings.EqualFold(nativeBase(info.filename), filepath.Base(o.Replay)) {
			return info, "", false
		}
		index := -1
		for i := range p.replay.Chapters {
			cfg, err := replay.ConfigFor(p.replay, i)
			if err == nil && cfg.Stage == info.header.Stage {
				index = i
				break
			}
		}
		if index < o.Chapter || index < 0 || !o.All && index != o.Chapter {
			return info, "", false
		}
		cfg, _ := replay.ConfigFor(p.replay, index)
		if cfg.Character != info.header.Character || cfg.ShotType != info.header.Shot || cfg.Difficulty != info.header.Difficulty {
			return info, "", false
		}
		info.header.Chapter, info.header.Records = index, len(p.replay.Chapters[index].Records)
		return info, p.base, true
	}
	return info, "", true
}

type captureSession struct {
	writer                *trace.Writer
	file                  *os.File
	first                 control
	header                trace.CaptureHeader
	half, missing, reread int
	total, slowest        time.Duration
	lastHalf              bool
}

func openCaptureSession(o probeOptions, c control, info captureInfo, base string) (*captureSession, error) {
	out := os.Stdout
	var file *os.File
	if o.All || o.Demos {
		if err := os.MkdirAll(o.OutDir, 0o755); err != nil {
			return nil, err
		}
		name := filepath.Join(o.OutDir, fmt.Sprintf("%s_c%d.txt", base, info.header.Chapter))
		var err error
		file, err = os.OpenFile(name, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o644)
		if err != nil {
			return nil, err
		}
		out = file
	} else if o.Out != "-" {
		var err error
		file, err = os.OpenFile(o.Out, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o644)
		if err != nil {
			return nil, err
		}
		out = file
	}
	s := &captureSession{writer: trace.NewAtomicWriter(out), file: file, first: c, header: info.header}
	if o.Ext || o.All || o.Demos || o.Replay != "" || o.Frames == 0 {
		if err := s.writer.BeginCapture(info.header); err != nil {
			if file != nil {
				err = errors.Join(err, file.Close())
			}
			return nil, err
		}
	}
	return s, nil
}

func (s *captureSession) write(res sampleResult) error {
	if last := s.writer.LastFrame(); last > 0 && res.Frame.Index > last+1 {
		s.missing += res.Frame.Index - last - 1
	}
	s.writer.WriteFrame(res.Frame)
	if res.Frame.Half != nil {
		s.half++
	}
	s.lastHalf = res.Frame.Half != nil
	if res.Attempts > 2 {
		s.reread++
	}
	s.total += res.Elapsed
	s.slowest = max(s.slowest, res.Elapsed)
	return s.writer.Err()
}

func (s *captureSession) close(status string, log io.Writer) error {
	err := s.writer.EndCapture(status)
	if s.file != nil {
		err = errors.Join(err, s.file.Sync(), s.file.Close())
	}
	mean := time.Duration(0)
	if s.writer.Frames() > 0 {
		mean = s.total / time.Duration(s.writer.Frames())
	}
	_, _ = fmt.Fprintf(log, "th10probe: chapter=%d stage=%d demo=%d status=%s frames=%d last=%d half=%d missing=%d reread=%d mean=%s max=%s\n",
		s.header.Chapter, s.header.Stage, s.header.Demo, status, s.writer.Frames(), s.writer.LastFrame(), s.half, s.missing, s.reread, mean, s.slowest)
	return err
}

// collectCapture is driven only by observed memory. The original keeps
// playing independently when this function waits, reports an error or exits.
func collectCapture(ctx context.Context, r *reader, p *capturePlan, log io.Writer, clock sampleClock) (result error) {
	o := p.options
	deadline := clock.Now().Add(o.Timeout)
	var current *captureSession
	var closed *control
	finished := make(map[int]bool)
	status := "error"
	defer func() {
		if current != nil {
			result = errors.Join(result, current.close(status, log))
		}
	}()
	_, _ = fmt.Fprintln(log, "th10probe: waiting for the requested replay/title demos; no input is sent")
	finish := func(s *captureSession) bool {
		index := s.header.Chapter
		if o.Demos {
			index = s.header.Demo
		}
		finished[index] = true
		if o.Demos {
			return len(finished) == 4
		}
		if o.All {
			return len(finished) == len(p.replay.Chapters)-o.Chapter
		}
		return true
	}
	for {
		if ctx.Err() != nil {
			status = "interrupted"
			return ctx.Err()
		}
		if !clock.Now().Before(deadline) {
			status = "timeout"
			return fmt.Errorf("capture timeout: %w", context.DeadlineExceeded)
		}
		c, err := r.readControl()
		if err != nil {
			return err
		}
		if current != nil && (!c.playable() || !current.first.sameStage(c) || c.Frame < int32(current.writer.LastFrame())) {
			// Missing the scene request is not proof that its last frame was
			// captured. If the final counter is still readable, require it.
			end := "incomplete"
			if c.Scene != 7 && int(c.Frame) == current.writer.LastFrame() && !current.lastHalf {
				end = "complete"
			}
			s := current
			current = nil
			if err := s.close(end, log); err != nil {
				return err
			}
			if end != "complete" {
				return fmt.Errorf("stage changed before a confirmed final frame")
			}
			if finish(s) {
				return nil
			}
		}
		if !c.playable() {
			closed = nil
			clock.Sleep(o.Poll)
			continue
		}
		if current == nil {
			if closed != nil && closed.sameStage(c) && c.Frame >= closed.Frame {
				clock.Sleep(o.Poll)
				continue
			}
			if !o.NoWait && c.Frame > 2 {
				clock.Sleep(o.Poll)
				continue
			}
			info, err := r.captureInfo(c)
			if err != nil {
				return err
			}
			info, base, ok := p.selectInfo(c, info)
			if !ok {
				clock.Sleep(o.Poll)
				continue
			}
			index := info.header.Chapter
			if o.Demos {
				index = info.header.Demo
			}
			if finished[index] {
				clock.Sleep(o.Poll)
				continue
			}
			check, err := r.readControl()
			if err != nil {
				return err
			}
			if !c.sameFrame(check) {
				continue
			}
			current, err = openCaptureSession(o, c, info, base)
			if err != nil {
				return err
			}
			_, _ = fmt.Fprintf(log, "th10probe: capture chapter=%d stage=%d demo=%d at frame %d\n", info.header.Chapter, info.header.Stage, info.header.Demo, c.Frame)
		}
		if int(c.Frame) == current.writer.LastFrame() {
			clock.Sleep(o.Poll)
			continue
		}
		if o.Frames > 0 && int(c.Frame) > o.Frames {
			return fmt.Errorf("missed requested final frame %d", o.Frames)
		}
		res := r.sampleSettled(ctx, c, o.Sample, clock)
		if err := current.write(res); err != nil {
			return err
		}
		if res.Err != nil && ctx.Err() == nil {
			_, _ = fmt.Fprintln(log, "th10probe: diagnostic half-frame read:", res.Err)
		}
		if ctx.Err() != nil {
			status = "interrupted"
			return ctx.Err()
		}
		end := c.sameFrame(res.Control) && (res.Control.Request == 4 || res.Control.Request == 11)
		// FUN_00425730 0x425c93 returns failed replays to title. MSG
		// op 20 uses request 11 for the next chapter, 4 for final clear.
		if end || o.Frames > 0 && int(c.Frame) == o.Frames {
			endStatus := "complete"
			if current.lastHalf {
				endStatus = "incomplete"
			}
			s := current
			current = nil
			if err := s.close(endStatus, log); err != nil {
				return err
			}
			if endStatus != "complete" {
				return fmt.Errorf("last sampled frame was not confirmed; replay capture is incomplete")
			}
			if finish(s) {
				return nil
			}
			closed = &c
		}
	}
}
