package main

import (
	"fmt"
	"io"
	"runtime"
	"time"

	thaudio "th10/audio"
	"th10/game"

	"github.com/hajimehoshi/ebiten/v2/audio"
)

type pullMeasurement struct {
	PCMBytes int
	AudioMS  float64
	InitMS   float64
	MeanMS   float64
	MaxMS    float64
}

type audioMeasurement struct {
	Track, Reads                   int
	Name                           string
	Pulls                          []pullMeasurement
	SeekNearEndMS, BoundaryReadMS  float64
	GoHeapBytes, LinearMemoryBytes uint64
}

func durationMS(duration time.Duration) float64 { return float64(duration) / float64(time.Millisecond) }

func measureAudio(blocks int) ([]audioMeasurement, error) {
	tracks, err := thaudio.BGMTracks()
	if err != nil {
		return nil, err
	}
	results := make([]audioMeasurement, 0, len(tracks))
	// NewBGMStream's public contract is 16-bit stereo at the requested rate.
	// Oto's float32 source buffer pulls half as many bytes from that decoder.
	windows := []int{game.SampleRate * 4 / 2, game.SampleRate * 4 / 5, game.SampleRate * 4 / 10, 2048 * 4}
	for index, track := range tracks {
		row := audioMeasurement{Track: index, Reads: blocks, Name: track.Name}
		var stream *audio.InfiniteLoop
		for _, size := range windows {
			start := time.Now()
			stream, err = thaudio.NewBGMStream(index, game.SampleRate)
			if err != nil {
				return nil, err
			}
			pull := pullMeasurement{PCMBytes: size, AudioMS: float64(size) * 1000 / float64(game.SampleRate*4), InitMS: durationMS(time.Since(start))}
			buffer := make([]byte, size)
			var total, maximum time.Duration
			for range blocks {
				start := time.Now()
				if _, err := io.ReadFull(stream, buffer); err != nil {
					return nil, fmt.Errorf("track %d pull: %w", index, err)
				}
				elapsed := time.Since(start)
				total += elapsed
				maximum = max(maximum, elapsed)
			}
			pull.MeanMS, pull.MaxMS = durationMS(total)/float64(blocks), durationMS(maximum)
			row.Pulls = append(row.Pulls, pull)
		}
		start := time.Now()
		if _, err := stream.Seek(max(track.Size-4096, 0), io.SeekStart); err != nil {
			return nil, err
		}
		row.SeekNearEndMS = durationMS(time.Since(start))
		buffer := make([]byte, 8192)
		start = time.Now()
		if _, err := io.ReadFull(stream, buffer); err != nil {
			return nil, err
		}
		row.BoundaryReadMS = durationMS(time.Since(start))
		var memory runtime.MemStats
		runtime.ReadMemStats(&memory)
		row.GoHeapBytes, row.LinearMemoryBytes = memory.HeapAlloc, wasmMemoryBytes()
		results = append(results, row)
	}
	return results, nil
}
