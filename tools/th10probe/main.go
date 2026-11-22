// Command th10probe samples a running, hash-checked TH10 v1.00a process.
// It uses only process-memory reads and process/module queries. It never
// writes memory, injects code, suspends threads, sends input or captures a window.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"math"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() { os.Exit(run()) }

type probeOptions struct {
	PID              uint
	Exe, Out         string
	Frames           int
	Poll             time.Duration
	Sample           sampleOptions
	Timeout          time.Duration
	NoWait, SkipHash bool
	Ext              bool
	Replay, OutDir   string
	Chapter          int
	All, Demos       bool
}

func parseProbeOptions(args []string, out io.Writer) (probeOptions, error) {
	var o probeOptions
	fs := flag.NewFlagSet("th10probe", flag.ContinueOnError)
	fs.SetOutput(out)
	var poll, settle, gap, budget int
	fs.UintVar(&o.PID, "pid", 0, "target process id")
	fs.StringVar(&o.Exe, "exe", "", "path to th10.exe (locate process and verify SHA-256)")
	fs.StringVar(&o.Out, "out", "-", "trace output (- = stdout)")
	fs.IntVar(&o.Frames, "frames", 3600, "last stage frame to sample (0 = until the stage ends)")
	fs.IntVar(&poll, "poll-us", 500, "frame-poll interval in microseconds")
	fs.IntVar(&settle, "settle-us", 3000, "minimum settle time after the observed frame tick")
	fs.IntVar(&gap, "stable-us", 500, "gap between two logical snapshots")
	fs.IntVar(&budget, "budget-us", 10000, "total per-frame confirmation/retry budget in microseconds")
	fs.IntVar(&o.Sample.Attempts, "attempts", 4, "maximum full snapshots per sampled frame (at least 2)")
	fs.DurationVar(&o.Timeout, "timeout", 30*time.Minute, "total waiting/capture deadline")
	fs.BoolVar(&o.NoWait, "no-wait-start", false, "start at the current stage frame")
	fs.BoolVar(&o.SkipHash, "skip-hash", false, "explicitly skip executable SHA-256 verification")
	fs.BoolVar(&o.Ext, "ext", false, "include native stats, player timers, all items, lasers and spell fields")
	fs.StringVar(&o.Replay, "replay", "", "local replay file: read-only chapter/identity metadata, never controls the game")
	fs.IntVar(&o.Chapter, "chapter", 0, "first replay chapter to capture (0-based, needs -replay)")
	fs.BoolVar(&o.All, "all-chapters", false, "capture every chapter of -replay without detaching (needs -out-dir)")
	fs.BoolVar(&o.Demos, "demos", false, "observe title demos and collect all four distinct native demo numbers")
	fs.StringVar(&o.OutDir, "out-dir", "", "directory for per-chapter/demo files; existing files are never overwritten")
	if err := fs.Parse(args); err != nil {
		return o, err
	}
	explicitFrames, explicitTimeout := false, false
	fs.Visit(func(f *flag.Flag) {
		if f.Name == "frames" {
			explicitFrames = true
		}
		if f.Name == "timeout" {
			explicitTimeout = true
		}
	})
	if o.All || o.Demos {
		if !explicitFrames {
			o.Frames = 0
		}
		if o.All && !explicitTimeout {
			o.Timeout = 2 * time.Hour
		}
	}
	if fs.NArg() != 0 || o.Frames < 0 || o.Frames > math.MaxInt32 || o.PID > math.MaxUint32 || poll <= 0 || poll > 1000000 || settle < 0 || gap < 0 || gap > 1000000 || budget <= settle || budget > 1000000 || o.Sample.Attempts < 2 || o.Timeout <= 0 {
		return o, fmt.Errorf("invalid frame, timing or retry options")
	}
	if o.Chapter < 0 || o.Chapter > 0 && o.Replay == "" || o.All && (o.Replay == "" || o.OutDir == "") || o.Demos && (o.All || o.Replay != "" || o.OutDir == "") ||
		(o.All || o.Demos) && o.Out != "-" || o.OutDir != "" && !o.All && !o.Demos {
		return o, fmt.Errorf("invalid replay/demo/output combination")
	}
	o.Poll = time.Duration(poll) * time.Microsecond
	o.Sample.Settle = time.Duration(settle) * time.Microsecond
	o.Sample.Gap = time.Duration(gap) * time.Microsecond
	o.Sample.Budget = time.Duration(budget) * time.Microsecond
	o.Sample.Poll = 100 * time.Microsecond
	return o, nil
}

type process interface {
	memoryReader
	io.Closer
	ID() uint32
}

func run() int {
	opts, err := parseProbeOptions(os.Args[1:], os.Stderr)
	if err != nil {
		fmt.Fprintln(os.Stderr, "th10probe:", err)
		return 2
	}
	plan, err := newCapturePlan(opts)
	if err != nil {
		fmt.Fprintln(os.Stderr, "th10probe:", err)
		return 2
	}
	p, err := openProcess(uint32(opts.PID), opts.Exe, opts.SkipHash)
	if err != nil {
		fmt.Fprintln(os.Stderr, "th10probe:", err)
		return 2
	}
	defer func() {
		if err := p.Close(); err != nil {
			fmt.Fprintln(os.Stderr, "th10probe: close process:", err)
		}
	}()
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	rd := &reader{mem: p, ext: opts.Ext}
	fmt.Fprintf(os.Stderr, "th10probe: attached to pid %d (VM_READ/query only)\n", p.ID())
	if err := collectCapture(ctx, rd, plan, os.Stderr, realClock()); err != nil {
		fmt.Fprintln(os.Stderr, "th10probe: INCOMPLETE:", err)
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return 1
		}
		return 2
	}
	return 0
}
