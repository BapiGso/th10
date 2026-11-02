// Package pacing decides which display refreshes run a logic frame.
//
// The original's main loop FUN_00439390 (docs/research/m4_render_re.md §1)
// runs logic on a fixed 1/60 s grid, at most one frame per grid step, and
// drops the steps it missed instead of catching up, so an overloaded frame
// slows the game down rather than making it jump (Throttle). In fullscreen
// it presents at 60 Hz with vsync, so every logic frame is shown for exactly
// one refresh; that even cadence is what makes it feel smooth.
//
// Here Ebiten calls Update once per display refresh (ebiten.SyncWithFPS)
// and the game skips the refreshes Ready refuses, drawing nothing new on
// them, so each logic frame is drawn exactly once. Pacer gets the original's
// even cadence on any display whose refresh rate is within 1% of a whole
// multiple of 60 Hz (60, 59.94, 120, 180, 240 Hz) by locking to every N-th
// refresh, and falls back to the original's grid elsewhere, where
// an uneven cadence (144 Hz: 2, 3, 2, 2, 3, ... refreshes per frame) cannot
// be avoided without interpolating frames.
//
// The game wires it as follows (game/game.go):
//
//	ebiten.SetTPS(ebiten.SyncWithFPS)          // one Update per refresh
//	ebiten.SetScreenClearedEveryFrame(false)   // a skipped refresh keeps the last frame
//	Update: if !pacer.Ready(time.Since(start).Seconds()) { return nil }
//	Draw:   after a logic frame, screen.Clear() and draw; otherwise only
//	        keep the screen marked as modified (see game.keepAlive)
package pacing

import (
	"math"
	"slices"
)

// Step is DAT_00470c90, the grid step in seconds: the double 1/60
// (0x3f91111111111111).
const Step = 1.0 / 60

// Slack is how early, in seconds, a refresh may take the next grid step.
// The original waits for the step itself; a refresh-driven loop cannot
// wait, so without slack a refresh arriving a hair before the step would
// lose the whole refresh, and timing jitter around the grid phase would
// drop frames over and over. Each run still consumes a step, so the grid
// never runs more than one frame per step on average.
const Slack = Step / 4

// Throttle is the timing state FUN_00439390 keeps at this+0x38..+0x48.
type Throttle struct {
	Now  float64 // +0x38, the time of the latest call
	Prev float64 // +0x40, the time of the previous call
	Next float64 // +0x48, the next grid step
}

// Ready reports whether a logic frame runs at time now (seconds, from a
// monotonic clock such as FUN_00439540's). It is FUN_00439390's head with
// the readiness test widened by Slack:
//
//	Now = now
//	if now < Prev { Next = now }           // the clock went back: restart the grid
//	Prev = now
//	if now <= Next-Slack { return false }  // original: now <= Next (Sleep(0))
//	do { Next += Step } while Next < Now   // drop the steps already missed
//	return true
//
// Every comparison and addition is float64, as the original's doubles.
func (t *Throttle) Ready(now float64) bool {
	t.Now = now
	if now < t.Prev {
		t.Next = now
	}
	t.Prev = now
	if now <= t.Next-Slack {
		return false
	}
	t.Next += Step
	for t.Next < t.Now {
		t.Next += Step
	}
	return true
}

// Pacer tuning.
const (
	// Window is how many of the latest refresh intervals the period
	// estimate uses: about 2 s at 60 Hz, 0.5 s at 240 Hz.
	Window = 128
	// MaxLock is the most refreshes per frame Pacer locks to (240 Hz).
	// Above it, Ebiten stops presenting after 4 unmodified frames, which
	// the game's keep-alive draw prevents; the grid paces those displays.
	MaxLock = 4
	// LockTol and UnlockTol bound |N·P - Step| / Step, the relative error of
	// N refresh periods against the step, for locking to every N-th refresh
	// and for staying locked. The gap between them is the hysteresis.
	LockTol   = 0.01
	UnlockTol = 0.02
)

// Pacer is called once per display refresh and decides which refreshes
// run a logic frame. While Lock is 0 it defers to Grid; while Lock is N it
// runs every N-th refresh. The zero value is ready to use.
type Pacer struct {
	Grid  Throttle // the fallback, and FUN_00439390's state
	Lock  int      // refreshes per frame while locked, 0 while on Grid
	P     float64  // the refresh period estimate, seconds (once the window is full)
	Last  float64  // the time of the previous call
	Frame float64  // the time of the last logic frame

	started bool
	iv      [Window]float64 // ring of the latest positive call intervals
	n, i    int             // intervals held (<= Window), next slot to write
}

// Ready reports whether a logic frame runs at time now (seconds, from a
// monotonic clock), one call per display refresh:
//
//  1. On the first call, or when now < Last (the clock went back), start
//     over: Last = Frame = now, Lock = 0, the ring emptied,
//     Grid = Throttle{Now: now, Prev: now, Next: now}; the result is
//     frame(Grid.Ready(now)), which is true.
//  2. d = now - Last; Last = now.
//  3. If d > 0, d goes into the ring, replacing the oldest once it holds
//     Window intervals.
//  4. Once the ring is full, re-estimate. m is the median interval, the
//     (Window/2)-th smallest counting from 0. Each interval d spans
//     round(d/m) refreshes; R is their sum, and the interval is ambiguous
//     when |d/m - round(d/m)| > 1/4. P = (sum of the intervals) / R, the
//     span over the refreshes it covers, which is robust to stalls. Then
//     n = max(1, round(Step/P)), err = |n·P - Step| / Step and amb is the
//     number of ambiguous intervals:
//     - unlocked, n <= MaxLock, err <= LockTol and amb <= Window/8: lock,
//     Lock = n;
//     - locked, and any one of n != Lock, err > UnlockTol or
//     amb > Window/4 holds: unlock, Lock = 0, empty the ring (the next
//     lock waits for Window fresh
//     intervals), and restart the grid from the last frame:
//     Grid = Throttle{Now: now, Prev: now, Next: Frame + Step}.
//  5. Locked: the result is frame(now-Frame >= max((Lock-0.5)·P,
//     Step-Slack)): Lock refreshes since the last frame, rounded, so a
//     refresh missed by a stall still counts; the Step-Slack bound keeps a
//     display change (say 60 to 300 Hz) from running frames faster than
//     the grid would before the lock drops.
//  6. Unlocked: the result is frame(Grid.Ready(now)).
//
// frame(ok) sets Frame = now when ok, and returns ok.
// round is math.Round. With call jitter well below a quarter refresh the
// per-interval counts are exact; with more (amb), Pacer stays on the grid
// rather than lock to a period it cannot measure.
func (p *Pacer) Ready(now float64) bool {
	if !p.started || now < p.Last {
		*p = Pacer{
			Grid:    Throttle{Now: now, Prev: now, Next: now},
			Last:    now,
			Frame:   now,
			started: true,
		}
		return p.frame(p.Grid.Ready(now), now)
	}

	d := now - p.Last
	p.Last = now
	if d > 0 {
		p.iv[p.i] = d
		p.i = (p.i + 1) % Window
		if p.n < Window {
			p.n++
		}
	}

	if p.n == Window {
		period, amb := p.estimate()
		p.P = period
		n := max(1, int(math.Round(Step/period)))
		err := math.Abs(float64(n)*period-Step) / Step
		switch {
		case p.Lock == 0 && n <= MaxLock && err <= LockTol && amb <= Window/8:
			p.Lock = n
		case p.Lock != 0 && (n != p.Lock || err > UnlockTol || amb > Window/4):
			p.Lock = 0
			p.n, p.i = 0, 0
			p.Grid = Throttle{Now: now, Prev: now, Next: p.Frame + Step}
		}
	}

	if p.Lock != 0 {
		return p.frame(now-p.Frame >= max((float64(p.Lock)-0.5)*p.P, Step-Slack), now)
	}
	return p.frame(p.Grid.Ready(now), now)
}

// estimate returns the refresh period of the full ring, the span of the
// intervals over the refreshes they cover, and how many intervals are
// ambiguous (Ready, step 4). The ring itself keeps its order.
func (p *Pacer) estimate() (period float64, amb int) {
	s := p.iv
	slices.Sort(s[:])
	m := s[Window/2]
	var sum float64
	var refreshes int
	for _, d := range p.iv {
		r := math.Round(d / m)
		refreshes += int(r)
		sum += d
		if math.Abs(d/m-r) > 0.25 {
			amb++
		}
	}
	return sum / float64(refreshes), amb
}

// frame records now as the time of a logic frame if ok, and returns ok.
func (p *Pacer) frame(ok bool, now float64) bool {
	if ok {
		p.Frame = now
	}
	return ok
}
