package main

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"math"
	"testing"
	"time"
)

type memoryRegion struct {
	base uint32
	data []byte
}
type fakeMemory struct {
	regions []memoryRegion
	before  func(uint32, []byte) error
	reads   int
}

func (m *fakeMemory) add(addr uint32, n int) []byte {
	b := make([]byte, n)
	m.regions = append(m.regions, memoryRegion{addr, b})
	return b
}

func (m *fakeMemory) ReadMemory(addr uint32, dst []byte) error {
	m.reads++
	if m.before != nil {
		if err := m.before(addr, dst); err != nil {
			return err
		}
	}
	for _, r := range m.regions {
		if addr >= r.base && uint64(addr)+uint64(len(dst)) <= uint64(r.base)+uint64(len(r.data)) {
			copy(dst, r.data[int(addr-r.base):])
			return nil
		}
	}
	return fmt.Errorf("unmapped synthetic read 0x%x (%d)", addr, len(dst))
}

func put32(b []byte, offset int, v uint32)     { binary.LittleEndian.PutUint32(b[offset:], v) }
func putFloat(b []byte, offset int, v float32) { put32(b, offset, math.Float32bits(v)) }

type fakeClock struct{ time time.Time }

func (c *fakeClock) clock() sampleClock {
	return sampleClock{Now: func() time.Time { return c.time }, Sleep: func(d time.Duration) { c.time = c.time.Add(d) }}
}

func syntheticReader() (*reader, *fakeMemory, []byte) {
	m := &fakeMemory{}
	globals := m.add(addrScoreBlock, 0x70)
	put32(globals, addrFrameCounter-addrScoreBlock, 7)
	put32(globals, 4, 123)
	binary.LittleEndian.PutUint16(globals[8:], 100)
	// The adjacent padding proves power must not be read as a dword.
	binary.LittleEndian.PutUint16(globals[10:], 0x7fff)
	put32(globals, 0xc, 5000)
	put32(globals, 0x30, 2)
	ptrs := m.add(addrPointers, 0x178)
	put32(ptrs, addrStagePtr-addrPointers, 0x01000000)
	put32(ptrs, addrPlayerPtr-addrPointers, 0x02000000)
	p := m.add(0x02000000, playerSize)
	putFloat(p, addrPlayerX, -18.55)
	putFloat(p, addrPlayerY, 420)
	put32(p, addrPlayerState, 1)
	loop := m.add(addrDrawCount, addrLoopStart+8-addrDrawCount)
	loop[0] = 1 // post-calc completion pulse
	binary.LittleEndian.PutUint64(loop[addrLoopStart-addrDrawCount:], math.Float64bits(1))
	m.add(addrLoopTime, 8)
	m.add(addrDrawSkip, 1)
	scene := m.add(addrScene, 8)
	put32(scene, 0, 7)
	put32(scene, 4, 7)
	m.add(0x01000000, 0x60)
	return &reader{mem: m}, m, p
}

func testSampleOptions() sampleOptions {
	return sampleOptions{Gap: time.Millisecond, Budget: 5 * time.Millisecond, Poll: time.Millisecond, Attempts: 4}
}

func TestSampleStableSyntheticMemory(t *testing.T) {
	r, _, _ := syntheticReader()
	c, err := r.readControl()
	if err != nil {
		t.Fatal(err)
	}
	clock := &fakeClock{}
	res := r.sampleSettled(context.Background(), c, testSampleOptions(), clock.clock())
	if res.Err != nil || res.Frame.Half != nil || res.Attempts != 2 || res.Frame.Index != 7 ||
		res.Frame.PScore != 1230 || res.Frame.PPower != 500 || res.Frame.PFaith != 50000 {
		t.Fatalf("stable result: %+v frame=%+v", res, res.Frame)
	}
}

func TestSampleOneMismatchThenRetrySyntheticMemory(t *testing.T) {
	r, m, p := syntheticReader()
	c, err := r.readControl()
	if err != nil {
		t.Fatal(err)
	}
	n := 0
	m.before = func(addr uint32, dst []byte) error {
		if addr == 0x02000000 {
			n++
			if n == 2 {
				putFloat(p, addrPlayerX, 24)
			}
		}
		return nil
	}
	res := r.sampleSettled(context.Background(), c, testSampleOptions(), (&fakeClock{}).clock())
	if res.Err != nil || res.Frame.Half != nil || res.Attempts != 3 || res.Frame.PX != 24 {
		t.Fatalf("reread result: %+v frame=%+v", res, res.Frame)
	}
}

func TestSampleAlwaysUnstableMarkedSyntheticMemory(t *testing.T) {
	r, m, p := syntheticReader()
	c, _ := r.readControl()
	n := 0
	m.before = func(addr uint32, dst []byte) error {
		if addr == 0x02000000 {
			n++
			putFloat(p, addrPlayerX, float32(n))
		}
		return nil
	}
	res := r.sampleSettled(context.Background(), c, testSampleOptions(), (&fakeClock{}).clock())
	if res.Frame.Half == nil || res.Frame.Half.Reason != "unstable" || res.Attempts != 4 || res.Elapsed > 5*time.Millisecond {
		t.Fatalf("unbounded/unmarked unstable result: %+v frame=%+v", res, res.Frame)
	}
}

func TestSampleIdenticalGDIStallStillUnconfirmed(t *testing.T) {
	r, m, _ := syntheticReader()
	// Find the loop region by address rather than depending on fixture order.
	for _, region := range m.regions {
		if region.base == addrDrawCount {
			region.data[0] = 0
		}
	}
	c, _ := r.readControl()
	res := r.sampleSettled(context.Background(), c, testSampleOptions(), (&fakeClock{}).clock())
	if res.Frame.Half == nil || res.Frame.Half.Reason != "calc-unconfirmed" || res.Attempts != 0 || res.Elapsed != 5*time.Millisecond {
		t.Fatalf("a static mid-calc snapshot was accepted: %+v frame=%+v", res, res.Frame)
	}
}

func TestCalcWitnessAddressesAndDrawSkip(t *testing.T) {
	first := control{LoopStart: math.Float64bits(1)}
	if calcWitness(first, first) {
		t.Fatal("static pre-calc memory is not a witness")
	}
	for _, c := range []control{{LoopStart: math.Float64bits(1), DrawCount: 1}, {LoopStart: math.Float64bits(2)}, {LoopStart: math.Float64bits(1), LoopTime: 1}} {
		if !calcWitness(first, c) {
			t.Fatalf("completion witness ignored: %+v", c)
		}
	}
	if calcWitness(first, control{LoopStart: first.LoopStart, DrawCount: 1, DrawSkip: 1}) {
		t.Fatal("previous skip accumulation is not this frame's completion")
	}
}

func TestSampleCrossFrameAndReadFailureMarked(t *testing.T) {
	for _, readFailure := range []bool{false, true} {
		r, m, _ := syntheticReader()
		c, _ := r.readControl()
		m.before = func(addr uint32, dst []byte) error {
			if addr == 0x02000000 {
				if readFailure {
					return io.ErrUnexpectedEOF
				}
				put32(m.regions[0].data, addrFrameCounter-addrScoreBlock, 8)
			}
			return nil
		}
		res := r.sampleSettled(context.Background(), c, testSampleOptions(), (&fakeClock{}).clock())
		if res.Frame.Half == nil {
			t.Fatalf("accepted invalid sample: %+v", res)
		}
		if readFailure && res.Err == nil {
			t.Fatal("lost read error")
		}
		if !readFailure && res.Frame.Half.After != 8 {
			t.Fatalf("lost counter transition: %+v", res.Frame.Half)
		}
	}
}

func TestProbeOptionsRejectWindowCaptureAndUnboundedRetries(t *testing.T) {
	for _, args := range [][]string{{"-shot-frames", "1", "-shot-dir", "shots"}, {"-attempts", "1"}, {"-budget-us", "1"}, {"-frames", "-1"}} {
		if _, err := parseProbeOptions(args, &bytes.Buffer{}); err == nil {
			t.Fatalf("accepted unsafe/invalid options %v", args)
		}
	}
}
