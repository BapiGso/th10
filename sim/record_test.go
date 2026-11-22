package sim

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"th10/replay"
	"th10/scene/stage"
	stagescript "th10/scene/stage/script"
)

// recordFactory keeps the old fallback of stage numbers above 6 to stage 6.
func recordFactory(n int) stage.Script { return stagescript.New(min(max(n, 1), 6)) }

// The opaque player history/option bytes are restored by RestoreReplayChapter
// and passed through CaptureReplayPlayer's reserved hook. Every decoded
// scalar is read back from the running game, not copied into the new header.
func TestRecordTh10_05AllChapters(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", ".tmp", "[th10]", "replay", "th10_05.rpy"))
	if err != nil {
		t.Skipf("original replay unavailable: %v", err)
	}
	source, err := replay.Decode(data)
	if err != nil {
		t.Fatal(err)
	}
	if len(source.Chapters) != 6 {
		t.Fatalf("expected six chapters, got %d", len(source.Chapters))
	}
	recorder := replay.NewRecorder(source.Global)
	for i, ch := range source.Chapters {
		cfg, err := ConfigFor(source, i)
		if err != nil {
			t.Fatal(err)
		}
		runner := New(cfg, recordFactory)
		runner.Recorder = recorder
		if err := runner.Run(ch, io.Discard, 0); err != nil {
			t.Fatal(err)
		}
		got := recorder.Replay().Chapters[i]
		if !reflect.DeepEqual(got.Records, ch.Records) {
			t.Fatalf("chapter %d input words differ", i)
		}
		if !bytes.Equal(got.SlowRates, ch.SlowRates) {
			t.Fatalf("chapter %d slow-rate bytes differ", i)
		}
		t.Logf("chapter %d: %d records / %d samples byte-exact; world %d frames, transition tail %d records",
			i, len(ch.Records), len(ch.SlowRates), runner.sim.state.Frame, len(ch.Records)-runner.sim.state.Frame)
	}
	// th10_05 was saved twice. Reproduce the original writer's count bug
	// rather than importing the doubled +0x04 value into the recorder.
	recorder.PrepareSave()
	recorder.PrepareSave()
	for i, ch := range recorder.Replay().Chapters {
		h, err := replay.ChapterHeader(ch)
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(h, source.Chapters[i].Raw) {
			for at := range h {
				if h[at] != source.Chapters[i].Raw[at] {
					t.Fatalf("chapter %d header first difference at %#x: %#x / %#x", i, at, h[at], source.Chapters[i].Raw[at])
				}
			}
		}
		t.Logf("chapter %d: all 452 header bytes identical (including repeat-save count)", i)
	}
}
