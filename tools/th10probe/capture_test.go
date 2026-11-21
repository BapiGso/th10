package main

import (
	"bytes"
	"context"
	"encoding/binary"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"th10/replay"
	"th10/trace"
)

func captureMemory() (*reader, *fakeMemory) {
	r, m, _ := syntheticReader()
	put32(m.region(addrScoreBlock), addrFrameCounter-addrScoreBlock, 1)
	put32(m.region(addrScoreBlock), 0x3c, 1)
	ptrs := m.region(addrPointers)
	put32(ptrs, addrReplayPtr-addrPointers, 0x09000000)
	rp := m.add(0x09000000, 0x1d4)
	put32(rp, 0x10, 1)
	put32(rp, 0x1d0, 1)
	put32(rp, 0xb0+0x24, 0x0a000000)
	put32(rp, 0xb0+2*0x24, 0x0b000000)
	for i, addr := range []uint32{0x0a000000, 0x0b000000} {
		h := m.add(addr, 0x1c4)
		binary.LittleEndian.PutUint16(h, uint16(i+1))
		put32(h, 4, 2)
	}
	copy(ptrs[addrReplayName-addrPointers:], "replay/th10_05.rpy\x00")
	return r, m
}

func readCaptureFile(t *testing.T, path string, wantStage int) {
	t.Helper()
	f, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = f.Close() }()
	r := trace.NewReader(f)
	for i := 1; i <= 2; i++ {
		frame, err := r.Next()
		if err != nil || frame == nil || frame.Index != i || frame.Half != nil {
			t.Fatalf("%s frame %d: %+v %v", path, i, frame, err)
		}
	}
	if frame, err := r.Next(); frame != nil || err != nil {
		t.Fatalf("%s EOF: frame=%v err=%v", path, frame, err)
	}
	if r.Capture() == nil || r.Capture().Stage != wantStage {
		t.Fatalf("wrong stage metadata in %s", path)
	}
}

func TestCaptureAllChaptersDoesNotStopAtCounterReset(t *testing.T) {
	r, m := captureMemory()
	dir := t.TempDir()
	o := probeOptions{All: true, Replay: filepath.Join(dir, "th10_05.rpy"), Out: "-", OutDir: dir, Poll: 2 * time.Millisecond, Timeout: time.Second, Sample: testSampleOptions()}
	p := &capturePlan{options: o, base: "th10_05", replay: &replay.Replay{Chapters: []replay.Chapter{{Index: 1, Records: make([]replay.Record, 2)}, {Index: 2, Records: make([]replay.Record, 2)}}}}
	clock := &fakeClock{}
	step := clock.clock()
	sleep := step.Sleep
	step.Sleep = func(d time.Duration) {
		sleep(d)
		if d != o.Poll {
			return
		}
		g, rp := m.region(addrScoreBlock), m.region(0x09000000)
		if le32(g[addrFrameCounter-addrScoreBlock:]) == 1 {
			put32(g, addrFrameCounter-addrScoreBlock, 2)
			put32(g, 0x3c, 2)
			req := uint32(11)
			if le32(rp[0x1d0:]) == 2 {
				req = 4
			}
			put32(m.region(addrScene), 4, req)
		} else {
			put32(rp, 0x1d0, 2)
			put32(g, addrFrameCounter-addrScoreBlock, 1)
			put32(m.region(addrScene), 4, 7)
		}
	}
	var log bytes.Buffer
	if err := collectCapture(t.Context(), r, p, &log, step); err != nil {
		t.Fatalf("capture: %v log=%s", err, log.String())
	}
	readCaptureFile(t, filepath.Join(dir, "th10_05_c0.txt"), 1)
	readCaptureFile(t, filepath.Join(dir, "th10_05_c1.txt"), 2)
}

func TestCaptureDemosByNameNotNextDemoCounter(t *testing.T) {
	r, m := captureMemory()
	put32(m.region(addrScoreBlock), 0x60, 0x20)
	dir := t.TempDir()
	o := probeOptions{Demos: true, Out: "-", OutDir: dir, Poll: 2 * time.Millisecond, Timeout: time.Second, Sample: testSampleOptions()}
	p := &capturePlan{options: o}
	order := []string{"demo/demo1.rpy", "demo/demo2.rpy", "demo/demo3.rpy", "demo/demo0.rpy"}
	at := 0
	title := false
	setName := func() {
		dst := m.region(addrPointers)[addrReplayName-addrPointers:]
		clear(dst[:256])
		copy(dst, order[at])
	}
	setName()
	clock := &fakeClock{}
	step := clock.clock()
	sleep := step.Sleep
	step.Sleep = func(d time.Duration) {
		sleep(d)
		if d != o.Poll {
			return
		}
		g, scene := m.region(addrScoreBlock), m.region(addrScene)
		if title {
			at++
			if at >= len(order) {
				return
			}
			setName()
			put32(m.region(addrPointers), addrStagePtr-addrPointers, 0x01000000)
			put32(scene, 0, 7)
			put32(scene, 4, 7)
			put32(g, addrFrameCounter-addrScoreBlock, 1)
			title = false
		} else if le32(g[addrFrameCounter-addrScoreBlock:]) == 1 {
			put32(g, addrFrameCounter-addrScoreBlock, 2)
			put32(scene, 4, 4)
		} else {
			put32(scene, 0, 4)
			put32(m.region(addrPointers), addrStagePtr-addrPointers, 0)
			title = true
		}
	}
	var log bytes.Buffer
	if err := collectCapture(t.Context(), r, p, &log, step); err != nil {
		t.Fatalf("demo capture: %v log=%s", err, log.String())
	}
	for n := 0; n < 4; n++ {
		readCaptureFile(t, filepath.Join(dir, "demo"+string(rune('0'+n))+"_c0.txt"), 1)
	}
}

func TestCaptureReadErrorFlushesCompleteLastFrame(t *testing.T) {
	r, m, _ := syntheticReader()
	put32(m.region(addrScoreBlock), addrFrameCounter-addrScoreBlock, 1)
	put32(m.region(addrScoreBlock), 0x3c, 1)
	playerReads, guards := 0, 0
	m.before = func(addr uint32, dst []byte) error {
		if addr == 0x02000000 {
			playerReads++
		}
		if addr == addrFrameCounter && playerReads >= 2 {
			guards++
			if guards == 2 {
				return io.ErrUnexpectedEOF
			}
		}
		return nil
	}
	o := probeOptions{Out: filepath.Join(t.TempDir(), "error.txt"), NoWait: true, Poll: 2 * time.Millisecond, Timeout: time.Second, Sample: testSampleOptions()}
	var log bytes.Buffer
	if err := collectCapture(context.Background(), r, &capturePlan{options: o}, &log, (&fakeClock{}).clock()); err == nil {
		t.Fatal("lost process read error")
	}
	data, err := os.ReadFile(o.Out)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(data, []byte("Q 1 last 1 status error\n")) {
		t.Fatalf("last complete frame/footer not flushed: %s", data)
	}
	rd := trace.NewReader(bytes.NewReader(data))
	f, err := rd.Next()
	if err != nil || f == nil || f.Index != 1 {
		t.Fatalf("complete prefix lost: %v %v", f, err)
	}
	if _, err := rd.Next(); err == nil {
		t.Fatal("error capture accepted as complete")
	}
}

func TestCaptureOptionsAndDemoIdentity(t *testing.T) {
	o, err := parseProbeOptions([]string{"-all-chapters", "-replay", "th10_05.rpy", "-out-dir", "output"}, io.Discard)
	if err != nil || o.Frames != 0 || o.Timeout != 2*time.Hour {
		t.Fatalf("whole-run defaults: %+v %v", o, err)
	}
	for _, name := range []string{"demo/demo0.rpy", "demo\\demo1.rpy", "demo2.rpy", "DEMO3.RPY"} {
		if _, ok := demoNumber(name); !ok {
			t.Fatalf("demo name %s", name)
		}
	}
	for _, name := range []string{"demo4.rpy", "th10_05.rpy", "demo1.rpy.tmp"} {
		if _, ok := demoNumber(name); ok {
			t.Fatalf("false demo %s", name)
		}
	}
	for _, args := range [][]string{{"-all-chapters"}, {"-demos"}, {"-demos", "-replay", "r.rpy", "-out-dir", "out"}, {"-out-dir", "out"}} {
		if _, err := parseProbeOptions(args, io.Discard); err == nil {
			t.Fatalf("accepted invalid mode %s", strings.Join(args, " "))
		}
	}
}
