package main

import (
	"errors"
	"fmt"
	"image/color"
	"sync"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
)

// startGate is a host gate, not a scene. No original calc/draw runs while
// audio is locked or an import is being committed before play.
type startGate struct {
	mu                              sync.Mutex
	initialized, requested, started bool
	importing                       bool
}

func (s *startGate) initialize() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	first := !s.initialized
	s.initialized = true
	return first
}

func (s *startGate) request() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.initialized {
		return errors.New("runtime is still initializing")
	}
	if s.importing {
		return errors.New("wait for the import transaction to finish")
	}
	s.requested = true
	return nil
}

func (s *startGate) tryStart(audioReady bool) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.started || !s.requested || !audioReady || s.importing {
		return false
	}
	s.started = true
	return true
}

func (s *startGate) isStarted() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.started
}

func (s *startGate) pendingStart() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.requested && !s.started && !s.importing
}

func (s *startGate) beginImport() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.started || s.requested {
		return errors.New("imports are only allowed before clicking Start; reload the page first")
	}
	if s.importing {
		return errors.New("another import is in progress")
	}
	s.importing = true
	return nil
}

func (s *startGate) endImport() {
	s.mu.Lock()
	s.importing = false
	s.mu.Unlock()
}

type gatedGame struct {
	inner                                 ebiten.Game
	gate                                  startGate
	audioReady                            func() bool
	onReady                               func()
	onStart                               func()
	onMetrics                             func(runtimeMetrics)
	resumeClock                           func()
	waitForDraw, clockArmed, clockResumed bool
	debug                                 bool
	stats                                 frameStats
}

func (g *gatedGame) Update() error {
	// Ebiten's before-Update audio hook has already installed Oto's real
	// mouseup/keyup/touchend listeners by this first wrapper Update.
	if g.gate.initialize() && g.onReady != nil {
		g.onReady()
	}
	if g.waitForDraw && !g.clockResumed {
		return nil
	}
	if g.gate.tryStart(g.audioReady()) {
		g.stats.reset(time.Now())
		if g.onStart != nil {
			g.onStart()
		}
	}
	if !g.gate.isStarted() {
		return nil
	}
	start := time.Now()
	err := g.inner.Update()
	g.stats.update(time.Since(start))
	return err
}

func (g *gatedGame) Draw(screen *ebiten.Image) {
	if !g.gate.isStarted() {
		// Observe at least one zero-TPS Draw before resuming. The browser
		// clock cannot bank download/hidden-tab time into the first game frame.
		if g.waitForDraw && !g.clockArmed {
			g.clockArmed = true
		} else if g.waitForDraw && !g.clockResumed && g.gate.pendingStart() && g.audioReady() {
			g.clockResumed = true
			g.resumeClock()
		}
		if screen != nil {
			screen.Fill(color.Black)
		}
		return
	}
	start := time.Now()
	g.inner.Draw(screen)
	g.stats.draw(time.Since(start))
	if g.stats.publish(time.Now()) && g.debugEnabled() && g.onMetrics != nil {
		g.onMetrics(g.stats.view())
	}
	if g.debugEnabled() && screen != nil {
		m := g.stats.view()
		text := fmt.Sprintf("WASM | %.1f TPS / %.1f FPS | 60 logic ticks/s\nUpdate %.3f ms | mean %.3f | max %.3f\nDraw   %.3f ms | mean %.3f | max %.3f\nGo heap %.1f MiB | F3: hide | F: fullscreen", m.TPS, m.FPS, m.UpdateMS, m.UpdateMeanMS, m.UpdateMaxMS, m.DrawMS, m.DrawMeanMS, m.DrawMaxMS, float64(m.HeapBytes)/(1<<20))
		ebitenutil.DebugPrintAt(screen, text, 8, 8)
	}
}

func (g *gatedGame) toggleDebug() bool {
	g.gate.mu.Lock()
	defer g.gate.mu.Unlock()
	g.debug = !g.debug
	return g.debug
}

func (g *gatedGame) debugEnabled() bool {
	g.gate.mu.Lock()
	defer g.gate.mu.Unlock()
	return g.debug
}

func (g *gatedGame) Layout(width, height int) (int, int) { return g.inner.Layout(width, height) }
