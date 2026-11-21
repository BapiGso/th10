package sim

import (
	"bytes"
	"testing"
	"th10/game"
	"th10/replay"
	"th10/trace"
)

func TestMeasuredRunMatchesNormalTraceAndRNG(t *testing.T) {
	chapter := replay.Chapter{Index: 1, Seed: 1234, Faith: 5000, Lives: 2, Records: make([]replay.Record, 600)}
	for i := range chapter.Records {
		chapter.Records[i].Keys = replay.KeyRight
	}
	cfg := Config{Stage: 1, Character: game.CharReimu, ShotType: game.ShotC, Difficulty: game.DiffEasy, Seed: chapter.Seed}
	normal := New(cfg, testFactory)
	var full bytes.Buffer
	if err := normal.Run(chapter, &full, 300); err != nil {
		t.Fatal(err)
	}
	measured := New(cfg, testFactory)
	profile, err := measured.Measure(chapter, 300, func() uint64 { return 1234 })
	if err != nil {
		t.Fatal(err)
	}
	if profile.Frames != 300 || profile.MaxFrame < 1 || profile.MaxFrame > 300 || profile.MeanMS < 0 || profile.MaxMS < profile.MeanMS || profile.LinearMemoryAfterLoad != 1234 || profile.LinearMemoryAtEnd != 1234 {
		t.Fatalf("invalid profile: %+v", profile)
	}
	if profile.RNGState != normal.sim.ctx.RNG.State || profile.RNGCount != normal.sim.ctx.RNG.Count {
		t.Fatal("profiling changed main RNG")
	}
	encode := func(frame *trace.Frame) []byte {
		var buffer bytes.Buffer
		writer := trace.NewWriter(&buffer)
		writer.WriteFrame(frame)
		if err := writer.Flush(); err != nil {
			t.Fatal(err)
		}
		return buffer.Bytes()
	}
	if !bytes.Equal(encode(profile.Final), encode(normal.sim.sample(profile.Frames))) {
		t.Fatal("profiling changed the final trace frame")
	}
}
