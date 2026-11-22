package sim

import (
	"runtime"
	"th10/replay"
	"th10/trace"
	"time"
)

// Profile measures the same headless step as Runner.Run, without formatting
// trace rows or doing output I/O. Loading is outside the per-frame timer.
type Profile struct {
	Stage, Frames, MaxFrame                                  int
	MeanMS, MaxMS                                            float64
	HeapAllocAfterLoad, HeapInuseAfterLoad, SysAfterLoad     uint64
	HeapAllocAtEnd, LinearMemoryAfterLoad, LinearMemoryAtEnd uint64
	GCAfterLoad, GCAtEnd                                     uint32
	RNGState                                                 uint16
	RNGCount                                                 uint32
	AlternateRNGState                                        uint16
	AlternateRNGCount                                        uint32
	Final                                                    *trace.Frame `json:"-"`
}

// Measure does not change Run, its trace format, or any stage logic. It uses
// the existing initializer/restore/input/step path and the same stop boundary.
// memory, when supplied by a WASM host, reports linear-memory bytes.
func (r *Runner) Measure(ch replay.Chapter, maxFrames int, memory func() uint64) (Profile, error) {
	h, err := newHeadless(r.cfg, r.factory, 0, nil)
	if err != nil {
		return Profile{}, err
	}
	r.sim = h
	h.st.RestoreReplayChapter(ch)
	stream := replay.NewStream(ch)
	h.in.RestoreReplay(stream.Prime())
	limit := len(ch.Records)
	if maxFrames > 0 {
		limit = min(limit, maxFrames)
	}
	var stats runtime.MemStats
	runtime.ReadMemStats(&stats)
	profile := Profile{Stage: r.cfg.Stage, HeapAllocAfterLoad: stats.HeapAlloc, HeapInuseAfterLoad: stats.HeapInuse,
		SysAfterLoad: stats.Sys, GCAfterLoad: stats.NumGC}
	if memory != nil {
		profile.LinearMemoryAfterLoad = memory()
	}
	var total, maximum time.Duration
	for i := 0; i < limit; i++ {
		record, rate, _ := stream.Next()
		h.ctx.ReplaySlowSample = rate
		start := time.Now()
		h.step(record)
		elapsed := time.Since(start)
		total += elapsed
		if elapsed > maximum {
			maximum, profile.MaxFrame = elapsed, i+1
		}
		profile.Frames++
		if h.finished {
			break
		}
	}
	if profile.Frames > 0 {
		profile.MeanMS = float64(total) / float64(time.Millisecond) / float64(profile.Frames)
	}
	profile.MaxMS = float64(maximum) / float64(time.Millisecond)
	profile.RNGState, profile.RNGCount = h.ctx.RNG.State, h.ctx.RNG.Count
	profile.AlternateRNGState, profile.AlternateRNGCount = h.state.AlternateRNG().State, h.state.AlternateRNG().Count
	profile.Final = h.sample(profile.Frames)
	runtime.ReadMemStats(&stats)
	profile.HeapAllocAtEnd, profile.GCAtEnd = stats.HeapAlloc, stats.NumGC
	if memory != nil {
		profile.LinearMemoryAtEnd = memory()
	}
	return profile, nil
}
