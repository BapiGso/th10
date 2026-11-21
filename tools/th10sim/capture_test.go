package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"testing"

	"th10/replay"
	"th10/trace"
)

func TestSimDemoOptionsAndAllFourEmbeddedSources(t *testing.T) {
	o, err := parseSimOptions([]string{"-demo", "1"}, io.Discard)
	if err != nil || o.Frames != 3001 {
		t.Fatalf("demo window: %+v %v", o, err)
	}
	o, err = parseSimOptions([]string{"-demo", "1", "-frames", "0"}, io.Discard)
	if err != nil || o.Frames != 0 {
		t.Fatalf("full demo input: %+v %v", o, err)
	}
	for n := 0; n < 4; n++ {
		o = simOptions{Demo: n, Frames: 2, Ext: true, HiScore: 10000000, Out: filepath.Join(t.TempDir(), "demo.txt")}
		if err := simulate(o, io.Discard); err != nil {
			t.Fatalf("demo %d: %v", n, err)
		}
		f, err := os.Open(o.Out)
		if err != nil {
			t.Fatal(err)
		}
		r := trace.NewReader(f)
		for i := 1; i <= 2; i++ {
			frame, err := r.Next()
			if err != nil || frame == nil || frame.Ext == nil || frame.Index != i {
				closeTestFile(t, f)
				t.Fatalf("demo %d frame %d: %v", n, i, err)
			}
		}
		if frame, err := r.Next(); err != nil || frame != nil {
			closeTestFile(t, f)
			t.Fatalf("demo footer: frame=%v err=%v", frame, err)
		}
		if r.Capture() == nil || r.Capture().Demo != n {
			t.Fatal("demo identity not preserved")
		}
		closeTestFile(t, f)
	}
}

func TestSimAllChaptersAndObservedHighScoreInitialization(t *testing.T) {
	source, _, err := replaySource(simOptions{Demo: 0})
	if err != nil {
		t.Fatal(err)
	}
	ch := source.Chapters[0]
	ch.Records = append([]replay.Record(nil), ch.Records[:2]...)
	ch.SlowRates = append([]byte(nil), ch.SlowRates[:1]...)
	ch.Padding = nil
	ch.FrameCount = 2
	first, second := ch, ch
	first.Index, second.Index = 1, 2
	source.Chapters = []replay.Chapter{first, second}
	source.Global.Chapters = 2
	source.PayloadTail = nil
	raw, err := replay.Encode(source)
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	path := filepath.Join(dir, "synthetic.rpy")
	if err := os.WriteFile(path, raw, 0o644); err != nil {
		t.Fatal(err)
	}
	initialDir := filepath.Join(dir, "initial")
	if err := os.Mkdir(initialDir, 0o755); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 2; i++ {
		cfg, err := replay.ConfigFor(source, i)
		if err != nil {
			t.Fatal(err)
		}
		file, err := os.Create(filepath.Join(initialDir, fmt.Sprintf("synthetic_c%d.txt", i)))
		if err != nil {
			t.Fatal(err)
		}
		w := trace.NewAtomicWriter(file)
		if err := w.BeginCapture(trace.CaptureHeader{Chapter: i, Stage: cfg.Stage, Demo: -1, Character: cfg.Character, Shot: cfg.ShotType, Difficulty: cfg.Difficulty, HiScore: int64(111110000 + i), Records: 2}); err != nil {
			t.Fatal(err)
		}
		w.WriteFrame(&trace.Frame{Index: 1})
		if err := w.EndCapture("complete"); err != nil {
			t.Fatal(err)
		}
		closeTestFile(t, file)
	}
	o := simOptions{Replay: path, Demo: -1, All: true, Ext: true, Frames: 0, HiScore: -1, Out: "-", OutDir: filepath.Join(dir, "sim"), InitialDir: initialDir}
	if err := simulate(o, io.Discard); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 2; i++ {
		file, err := os.Open(filepath.Join(o.OutDir, fmt.Sprintf("synthetic_c%d.txt", i)))
		if err != nil {
			t.Fatal(err)
		}
		r := trace.NewReader(file)
		frame, err := r.Next()
		if err != nil || frame == nil {
			t.Fatalf("chapter %d: %v", i, err)
		}
		if r.Capture().HiScore != int64(111110000+i) || frame.Ext.HiScore != int64(111110000+i) {
			t.Fatalf("chapter %d did not use observed high score", i)
		}
		for {
			frame, err := r.Next()
			if err != nil {
				t.Fatal(err)
			}
			if frame == nil {
				break
			}
		}
		closeTestFile(t, file)
	}
}

func TestSimCaptureOptionsRejectConflictingSources(t *testing.T) {
	for _, args := range [][]string{{"-demo", "4"}, {"-demo", "1", "-replay", "x.rpy"}, {"-replay", "x.rpy", "-all-chapters"}, {"-replay", "x.rpy", "-initial", "a.txt", "-hiscore", "10"}} {
		if _, err := parseSimOptions(args, io.Discard); err == nil {
			t.Fatalf("accepted conflicting options %v", args)
		}
	}
}

func closeTestFile(t testing.TB, f *os.File) {
	t.Helper()
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
}
