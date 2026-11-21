package main

import (
	"fmt"
	"os"
	"th10/replay"
	selector "th10/scene/select"
	"th10/sim"
)

func measureLogic(path string, chapter, frames, warmup int) (*sim.Profile, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	r, err := replay.Decode(data)
	if err != nil {
		return nil, err
	}
	if chapter < 0 || chapter >= len(r.Chapters) {
		return nil, fmt.Errorf("chapter %d outside replay", chapter)
	}
	cfg, err := sim.ConfigFor(r, chapter)
	if err != nil {
		return nil, err
	}
	for range warmup {
		runner := sim.New(cfg, selector.ReplayScript)
		if _, err := runner.Measure(r.Chapters[chapter], frames, nil); err != nil {
			return nil, err
		}
	}
	runner := sim.New(cfg, selector.ReplayScript)
	profile, err := runner.Measure(r.Chapters[chapter], frames, wasmMemoryBytes)
	return &profile, err
}
