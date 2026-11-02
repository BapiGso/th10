package pacing

import (
	"math"
	"math/rand/v2"
	"testing"
)

func TestStepBits(t *testing.T) {
	if got := math.Float64bits(Step); got != 0x3f91111111111111 {
		t.Fatalf("Step bits = %#x, want 0x3f91111111111111", got)
	}
}

// The expectations below are computed at run time, in the order the
// implementation adds, so they match its float64 rounding exactly.

func TestThrottleGrid(t *testing.T) {
	s, sl := float64(Step), float64(Slack)
	var th Throttle
	type call struct {
		now        float64
		ready      bool
		next, prev float64
	}
	cases := []call{
		// 0 > Next-Slack = -Slack: runs, Next = Step.
		{0, true, s, 0},
		// Too early: now <= Next-Slack refuses, the boundary included.
		{s - sl - 1e-9, false, s, s - sl - 1e-9},
		{s - sl, false, s, s - sl},
		// Just inside the slack: runs early and consumes exactly one step.
		{s - sl + 1e-9, true, s + s, s - sl + 1e-9},
		// Running early never runs twice for one step.
		{s + 1e-9, false, s + s, s + 1e-9},
		// Late by 2.5 steps: one frame; the missed steps are dropped and
		// Next is the first step not before now.
		{4.5 * s, true, s + s + s + s + s, 4.5 * s},
	}
	for i, c := range cases {
		got := th.Ready(c.now)
		if got != c.ready || th.Next != c.next || th.Prev != c.prev || th.Now != c.now {
			t.Fatalf("call %d Ready(%v) = %v, state %+v; want %v, Next %v, Prev %v",
				i, c.now, got, th, c.ready, c.next, c.prev)
		}
	}
}

func TestThrottleExactStep(t *testing.T) {
	s := float64(Step)
	// The do-while adds at least once and keeps adding while Next < Now, so
	// landing exactly on a step stops there (Next == Now is not missed).
	th := Throttle{Next: 3 * s}
	if !th.Ready(3 * s) {
		t.Fatal("Ready at the step = false")
	}
	if want := 3*s + s; th.Next != want {
		t.Fatalf("Next = %v, want %v", th.Next, want)
	}
	th = Throttle{Next: s}
	if !th.Ready(s + s) {
		t.Fatal("Ready one step late = false")
	}
	if want := s + s; th.Next != want {
		t.Fatalf("Next = %v, want %v", th.Next, want)
	}
}

func TestThrottleClockBack(t *testing.T) {
	s := float64(Step)
	th := Throttle{Now: 10, Prev: 10, Next: 10 + s}
	// The clock went back to 2: Next restarts at now, so the frame runs and
	// Next is one step on.
	if !th.Ready(2) {
		t.Fatal("Ready after the clock went back = false")
	}
	if th.Next != 2+s || th.Prev != 2 || th.Now != 2 {
		t.Fatalf("state %+v, want Next %v, Prev 2, Now 2", th, 2+s)
	}
}

func TestThrottleOverload(t *testing.T) {
	// Frames taking 2.5 steps each: one frame per call, no catch-up, and
	// Next never falls behind now.
	var th Throttle
	now := 0.0
	for range 20 {
		if !th.Ready(now) {
			t.Fatalf("Ready(%v) = false while overloaded", now)
		}
		if th.Next < now {
			t.Fatalf("Next %v behind now %v", th.Next, now)
		}
		now += 2.5 * Step
	}
}

// display is a run of Pacer calls, one per refresh.
type display struct {
	t      []float64 // call times
	ran    []bool    // whether the call ran a frame
	lock   []int     // Lock after the call
	refIdx []int     // refresh index of the call (differs from the call index after a skip)
}

// drive calls a fresh Pacer once per refresh of a hz display for secs
// seconds, starting at t0, with Gaussian jitter of sd seconds on each call
// (kept monotonic, as a real clock is) and the refreshes for which skip
// returns true missed (no call, as when a frame stalls past a vsync).
func drive(p *Pacer, hz, t0, secs, sd float64, seed uint64, skip func(k int) bool) display {
	r := rand.New(rand.NewPCG(seed, 0x9e3779b97f4a7c15))
	var d display
	last := math.Inf(-1)
	for k := 0; float64(k) < hz*secs; k++ {
		j := r.NormFloat64() * sd
		if skip != nil && skip(k) {
			continue
		}
		now := max(t0+float64(k)/hz+j, last+1e-6)
		last = now
		d.t = append(d.t, now)
		d.ran = append(d.ran, p.Ready(now))
		d.lock = append(d.lock, p.Lock)
		d.refIdx = append(d.refIdx, k)
	}
	return d
}

// frames returns the call indices that ran a frame at or after time from.
func (d display) frames(from float64) []int {
	var f []int
	for i, ok := range d.ran {
		if ok && d.t[i] >= from {
			f = append(f, i)
		}
	}
	return f
}

// perSecond counts the frames in each whole second [t0+k, t0+k+1).
func (d display) perSecond(t0 float64) []int {
	var n []int
	for i, ok := range d.ran {
		s := int(math.Floor(d.t[i] - t0))
		if s < 0 { // jitter on the first calls
			continue
		}
		for len(n) <= s {
			n = append(n, 0)
		}
		if ok {
			n[s]++
		}
	}
	return n
}

func TestPacerFirstCall(t *testing.T) {
	var p Pacer
	if !p.Ready(5) {
		t.Fatal("first Ready = false")
	}
	if p.Lock != 0 || p.Frame != 5 || p.Last != 5 || p.Grid.Next != 5+float64(Step) {
		t.Fatalf("state after the first call %+v", p)
	}
}

func TestPacerLocks(t *testing.T) {
	// Displays within 1% of a whole multiple of 60 Hz, with 0.3 ms of
	// jitter: once the window has filled (3 s is enough at 60 Hz), Pacer
	// stays locked and runs exactly every N-th refresh.
	for _, c := range []struct {
		hz float64
		n  int
	}{{60, 1}, {59.94, 1}, {60.05, 1}, {120, 2}, {119.88, 2}, {180, 3}, {240, 4}} {
		for seed := range uint64(3) {
			var p Pacer
			d := drive(&p, c.hz, 5, 30, 0.0003, seed, nil)
			for i, tm := range d.t {
				if tm >= 8 && d.lock[i] != c.n {
					t.Fatalf("%v Hz seed %d: Lock = %d at %.3f s, want %d", c.hz, seed, d.lock[i], tm-5, c.n)
				}
			}
			f := d.frames(8)
			for k := 1; k < len(f); k++ {
				if g := f[k] - f[k-1]; g != c.n {
					t.Fatalf("%v Hz seed %d: frames %d refreshes apart at %.3f s, want %d",
						c.hz, seed, g, d.t[f[k]]-5, c.n)
				}
			}
		}
	}
}

func TestPacerGrid(t *testing.T) {
	// Displays away from a multiple of 60 Hz stay on the grid: frames
	// floor(hz/60) or ceil(hz/60) refreshes apart, min(hz, 60) a second.
	for _, hz := range []float64{144, 165, 75, 100, 90, 50} {
		var p Pacer
		d := drive(&p, hz, 5, 20, 0.0002, 1, nil)
		lo, hi := int(math.Floor(hz/60)), int(math.Ceil(hz/60))
		if lo == 0 {
			lo = 1
		}
		for i := range d.lock {
			if d.lock[i] != 0 {
				t.Fatalf("%v Hz: Lock = %d at %.3f s", hz, d.lock[i], d.t[i]-5)
			}
		}
		f := d.frames(5)
		for k := 1; k < len(f); k++ {
			if g := f[k] - f[k-1]; g < lo || g > hi {
				t.Fatalf("%v Hz: frames %d refreshes apart at %.3f s, want %d..%d",
					hz, g, d.t[f[k]]-5, lo, hi)
			}
		}
		want := int(math.Min(hz, 60))
		for s, n := range d.perSecond(5)[1:19] {
			if n < want-1 || n > want+1 {
				t.Fatalf("%v Hz: %d frames in second %d, want %d±1", hz, n, s+1, want)
			}
		}
	}
}

func TestPacerStall(t *testing.T) {
	// 120 Hz with every 97th refresh missed: still locked, and a missed
	// refresh counts, so frames stay 2 refreshes apart, 3 across a miss
	// that falls between two frames.
	var p Pacer
	d := drive(&p, 120, 5, 30, 0.0002, 1, func(k int) bool { return k%97 == 50 })
	for i, tm := range d.t {
		if tm >= 8 && d.lock[i] != 2 {
			t.Fatalf("Lock = %d at %.3f s, want 2", d.lock[i], tm-5)
		}
	}
	f := d.frames(8)
	for k := 1; k < len(f); k++ {
		if g := d.refIdx[f[k]] - d.refIdx[f[k-1]]; g != 2 && g != 3 {
			t.Fatalf("frames %d refreshes apart at %.3f s, want 2 or 3", g, d.t[f[k]]-5)
		}
	}
}

func TestPacerHeavyJitterStaysOnGrid(t *testing.T) {
	// 240 Hz with 1 ms of jitter, a quarter of the refresh period: the
	// period cannot be measured, so Pacer never locks, and the grid still
	// runs 60 frames a second.
	var p Pacer
	d := drive(&p, 240, 5, 20, 0.001, 1, nil)
	for i := range d.lock {
		if d.lock[i] != 0 {
			t.Fatalf("Lock = %d at %.3f s", d.lock[i], d.t[i]-5)
		}
	}
	for s, n := range d.perSecond(5)[1:19] {
		if n < 59 || n > 61 {
			t.Fatalf("%d frames in second %d, want 60±1", n, s+1)
		}
	}
}

func TestPacerDisplayChange(t *testing.T) {
	// The window moves to another display at t = 10 s. Every whole second
	// keeps 60±3 frames, no two frames are closer than Step-Slack, and
	// Pacer ends on the new display's mode.
	for _, c := range []struct {
		from, to float64
		lock     int
	}{{60, 144, 0}, {144, 60, 1}, {120, 60, 1}, {240, 60, 1}, {60, 240, 4}, {60, 120, 2}, {60, 300, 0}} {
		var p Pacer
		a := drive(&p, c.from, 5, 5, 0, 1, nil)
		b := drive(&p, c.to, 10, 10, 0, 1, nil)
		d := display{
			t:   append(a.t, b.t...),
			ran: append(a.ran, b.ran...),
		}
		if p.Lock != c.lock {
			t.Errorf("%v -> %v Hz: Lock = %d, want %d", c.from, c.to, p.Lock, c.lock)
		}
		last := math.Inf(-1)
		for i, ok := range d.ran {
			if !ok {
				continue
			}
			if d.t[i]-last < float64(Step)-float64(Slack)-1e-9 {
				t.Fatalf("%v -> %v Hz: frames %.2f ms apart at %.3f s",
					c.from, c.to, (d.t[i]-last)*1000, d.t[i]-5)
			}
			last = d.t[i]
		}
		for s, n := range d.perSecond(5)[1:14] {
			if n < 57 || n > 63 {
				t.Fatalf("%v -> %v Hz: %d frames in second %d", c.from, c.to, n, s+1)
			}
		}
	}
}

func TestPacerOverload(t *testing.T) {
	// Frames taking 2.5 steps each (24 calls a second): every call runs a
	// frame, nothing is caught up, and Pacer does not lock.
	var p Pacer
	for k := range 200 {
		if !p.Ready(5 + float64(k)*2.5*Step) {
			t.Fatalf("call %d refused while overloaded", k)
		}
		if p.Lock != 0 {
			t.Fatalf("call %d: Lock = %d", k, p.Lock)
		}
	}
}

func TestPacerVsyncOff(t *testing.T) {
	// A loop not held by vsync, calling every 0.1 ms: 60 frames a second,
	// never closer than Step-Slack, no lock.
	var p Pacer
	n, last := 0, math.Inf(-1)
	for k := range 20000 {
		now := 5 + float64(k)*0.0001
		if p.Ready(now) {
			if now-last < float64(Step)-float64(Slack)-1e-9 {
				t.Fatalf("frames %.2f ms apart at %.4f s", (now-last)*1000, now-5)
			}
			n, last = n+1, now
		}
		if p.Lock != 0 {
			t.Fatalf("Lock = %d at %.4f s", p.Lock, now-5)
		}
	}
	if n < 119 || n > 121 {
		t.Fatalf("%d frames in 2 s, want 120±1", n)
	}
}

func TestPacerClockBack(t *testing.T) {
	// Locked at 120 Hz, then the clock jumps back 8 s: that call starts
	// over and runs a frame, Lock drops to 0, and Pacer locks again once
	// the window has refilled.
	var p Pacer
	drive(&p, 120, 10, 5, 0, 1, nil)
	if p.Lock != 2 {
		t.Fatalf("Lock = %d before the jump, want 2", p.Lock)
	}
	if !p.Ready(7) {
		t.Fatal("Ready after the clock went back = false")
	}
	if p.Lock != 0 || p.Frame != 7 || p.Last != 7 || p.Grid.Next != 7+float64(Step) {
		t.Fatalf("state after the jump %+v", p)
	}
	drive(&p, 120, 7+1.0/120, 3, 0, 1, nil)
	if p.Lock != 2 {
		t.Fatalf("Lock = %d 3 s after the jump, want 2", p.Lock)
	}
}

func TestPacerUnlockKeepsGrid(t *testing.T) {
	// Unlocking restarts the grid one step after the last frame.
	var p Pacer
	drive(&p, 240, 5, 3, 0, 1, nil)
	if p.Lock != 4 {
		t.Fatalf("Lock = %d, want 4", p.Lock)
	}
	frame := p.Frame
	// 144 Hz from here: the lock drops within the window.
	now := p.Last
	for k := 0; p.Lock != 0; k++ {
		if k > Window*2 {
			t.Fatal("still locked after two windows at 144 Hz")
		}
		now += 1.0 / 144
		ok := p.Ready(now)
		if p.Lock == 0 {
			if want := frame + float64(Step); !ok && p.Grid.Next != want {
				t.Fatalf("Grid.Next = %v after unlocking, want Frame+Step = %v", p.Grid.Next, want)
			}
			break
		}
		if ok {
			frame = now
		}
	}
}
