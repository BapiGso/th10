package sim

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"th10/replay"
	"th10/scene/stage"
	"th10/scene/stage/stage1"
	"th10/scene/stage/stage2"
	"th10/scene/stage/stage3"
	"th10/scene/stage/stage4"
	"th10/scene/stage/stage5"
	"th10/scene/stage/stage6"
	"th10/trace"
)

func alternateFactory(n int) stage.Script {
	switch n {
	case 1:
		return stage1.NewScript()
	case 2:
		return stage2.NewScript()
	case 3:
		return stage3.NewScript()
	case 4:
		return stage4.NewScript()
	case 5:
		return stage5.NewScript()
	default:
		return stage6.NewScript()
	}
}

func alternateHeadless(t *testing.T, cfg Config, ch replay.Chapter, seed uint16) *headless {
	t.Helper()
	h, err := newHeadlessWithAlternateSeed(cfg, alternateFactory, seed)
	if err != nil {
		t.Fatal(err)
	}
	h.st.RestoreReplayChapter(ch)
	h.in.RestoreReplay(replay.NewStream(ch).Prime())
	return h
}

// Check the whole logic trace and the game RNG, not merely positions: an
// alternate draw can affect enemy sizes or a bullet's VM lifetime in theory.
// Each chapter and each pair of steps run sequentially; only one frame's
// encoded trace is retained, not hundreds of MB of complete replay traces.
func TestAlternateRNGDoesNotChangeReplayTrace(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("..", ".tmp", "[th10]", "replay", "th10_05.rpy"))
	if err != nil {
		t.Skipf("read-only replay fixture missing: %v", err)
	}
	rp, err := replay.Parse(raw)
	if err != nil {
		t.Fatal(err)
	}
	if len(rp.Chapters) != 6 {
		t.Fatalf("fixture has %d chapters, want 6", len(rp.Chapters))
	}
	var draws uint32
	for chapter, ch := range rp.Chapters {
		t.Run(fmt.Sprintf("c%d", chapter), func(t *testing.T) {
			cfg, err := ConfigFor(rp, chapter)
			if err != nil {
				t.Fatal(err)
			}
			a := alternateHeadless(t, cfg, ch, 0)
			b := alternateHeadless(t, cfg, ch, 0x1234)
			var bufA, bufB bytes.Buffer
			outA, outB := trace.NewWriter(&bufA), trace.NewWriter(&bufB)
			frames := 0
			presentationDiff := false
			for i, rec := range ch.Records {
				a.step(rec)
				b.step(rec)
				if *a.ctx.RNG != *b.ctx.RNG {
					t.Fatalf("frame %d: alternate seed changed main RNG: %+v vs %+v", i+1, a.ctx.RNG, b.ctx.RNG)
				}
				if *a.ctx.Anm.Alt != *b.ctx.Anm.Alt {
					presentationDiff = true
				}
				outA.WriteFrame(a.sample(i + 1))
				outB.WriteFrame(b.sample(i + 1))
				if err := outA.Flush(); err != nil {
					t.Fatal(err)
				}
				if err := outB.Flush(); err != nil {
					t.Fatal(err)
				}
				if !bytes.Equal(bufA.Bytes(), bufB.Bytes()) {
					t.Fatalf("frame %d: alternate seed changed logic trace", i+1)
				}
				bufA.Reset()
				bufB.Reset()
				frames++
				if a.finished || b.finished {
					if a.finished != b.finished {
						t.Fatalf("frame %d: alternate seed changed stage end", i+1)
					}
					break
				}
			}
			draws += a.ctx.Anm.Alt.Count
			if !presentationDiff {
				t.Fatal("different injected seeds never changed presentation RNG")
			}
			t.Logf("%d frames byte-identical; alternate draws %d/%d, main draws %d", frames,
				a.ctx.Anm.Alt.Count/2, b.ctx.Anm.Alt.Count/2, a.ctx.RNG.Count/2)
		})
	}
	if draws == 0 {
		t.Fatal("fixture never exercised the alternate RNG")
	}
}
