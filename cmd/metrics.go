package main

import (
	"sync"
	"time"
)

type runtimeMetrics struct {
	TPS, FPS                 float64
	UpdateMS, DrawMS         float64
	UpdateMeanMS, DrawMeanMS float64
	UpdateMaxMS, DrawMaxMS   float64
	HeapBytes                uint64
}

type frameStats struct {
	mu                     sync.Mutex
	start                  time.Time
	updates, draws         int
	updateTotal, drawTotal time.Duration
	updateMax, drawMax     time.Duration
	updateLast, drawLast   time.Duration
	latest                 runtimeMetrics
}

func (s *frameStats) reset(now time.Time) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.start = now
	s.updates, s.draws = 0, 0
	s.updateTotal, s.drawTotal, s.updateMax, s.drawMax, s.updateLast, s.drawLast = 0, 0, 0, 0, 0, 0
	s.latest = runtimeMetrics{}
}

func (s *frameStats) update(elapsed time.Duration) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.updates++
	s.updateTotal += elapsed
	s.updateMax = max(s.updateMax, elapsed)
	s.updateLast = elapsed
}

func (s *frameStats) draw(elapsed time.Duration) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.draws++
	s.drawTotal += elapsed
	s.drawMax = max(s.drawMax, elapsed)
	s.drawLast = elapsed
}

func milliseconds(duration time.Duration) float64 {
	return float64(duration) / float64(time.Millisecond)
}

func meanMilliseconds(duration time.Duration, count int) float64 {
	if count == 0 {
		return 0
	}
	return milliseconds(duration) / float64(count)
}

func (s *frameStats) publish(now time.Time) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	elapsed := now.Sub(s.start)
	if elapsed < time.Second {
		return false
	}
	s.latest.TPS, s.latest.FPS = float64(s.updates)/elapsed.Seconds(), float64(s.draws)/elapsed.Seconds()
	s.latest.UpdateMeanMS, s.latest.DrawMeanMS = meanMilliseconds(s.updateTotal, s.updates), meanMilliseconds(s.drawTotal, s.draws)
	s.latest.UpdateMaxMS, s.latest.DrawMaxMS = milliseconds(s.updateMax), milliseconds(s.drawMax)
	s.start = now
	s.updates, s.draws = 0, 0
	s.updateTotal, s.drawTotal, s.updateMax, s.drawMax = 0, 0, 0, 0
	return true
}

func (s *frameStats) view() runtimeMetrics {
	s.mu.Lock()
	defer s.mu.Unlock()
	result := s.latest
	result.UpdateMS, result.DrawMS = milliseconds(s.updateLast), milliseconds(s.drawLast)
	return result
}

func (s *frameStats) setHeap(bytes uint64) {
	s.mu.Lock()
	s.latest.HeapBytes = bytes
	s.mu.Unlock()
}
