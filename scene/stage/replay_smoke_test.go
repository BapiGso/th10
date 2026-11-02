package stage_test

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"th10/replay"
	"th10/scene/stage"
	"th10/scene/stage/extra"
	"th10/scene/stage/stage1"
	"th10/scene/stage/stage2"
	"th10/scene/stage/stage3"
	"th10/scene/stage/stage4"
	"th10/scene/stage/stage5"
	"th10/scene/stage/stage6"
	"th10/sim"
	"th10/supervisor"
	"th10/supervisor/supervisortest"
)

// TestReplayDrawSmoke drives every chapter of the user's replays through the
// headless stage and draws every frame with the ported nodes (DrawPorted) on
// a recording device, so a panic in the ported drawing (anmdraw render modes,
// a visible object with no sprite, a ring overflow, ...) shows up here rather
// than in the game. It checks that drawing happens, not what it looks like.
//
// The replays are the .rpy files in env TH10_REPLAYS, else in
// .tmp/[th10]/replay under the repository root; the test skips if there are
// none. Each chapter runs for at most 3000 frames. TH10_SMOKE=full runs every
// frame of every chapter instead, and -short caps each chapter at 300 frames.
func TestReplayDrawSmoke(t *testing.T) {
	// The stages read TH10_LEGACY_BG when they are built; keep the STD drawer on.
	t.Setenv("TH10_LEGACY_BG", "")
	dir := os.Getenv("TH10_REPLAYS")
	if dir == "" {
		dir = filepath.Join("..", "..", ".tmp", "[th10]", "replay")
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Skipf("no replay directory: %v", err)
	}
	var files []string
	for _, e := range entries {
		if !e.IsDir() && strings.EqualFold(filepath.Ext(e.Name()), ".rpy") {
			files = append(files, filepath.Join(dir, e.Name()))
		}
	}
	if len(files) == 0 {
		t.Skipf("no .rpy files in %s", dir)
	}
	limit := 3000
	switch {
	case os.Getenv("TH10_SMOKE") == "full":
		limit = 0
	case testing.Short():
		limit = 300
	}
	strips := 0
	for _, path := range files {
		name := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
		raw, err := os.ReadFile(path)
		if err != nil {
			t.Errorf("%s: %v", name, err)
			continue
		}
		rp, err := replay.Parse(raw)
		if err != nil {
			t.Errorf("%s: parse: %v", name, err)
			continue
		}
		for i := range rp.Chapters {
			t.Run(fmt.Sprintf("%s/ch%d", name, i), func(t *testing.T) {
				strips += drawChapter(t, rp, i, limit)
			})
		}
	}
	t.Logf("total strips across all chapters: %d", strips)
}

// drawChapter runs chapter i of rp headlessly, draws every frame with the
// ported nodes and logs the draw counts. It returns the number of strips.
func drawChapter(t *testing.T, rp *replay.Replay, i, limit int) int {
	cfg, err := sim.ConfigFor(rp, i)
	if err != nil {
		t.Skipf("config: %v", err)
	}
	d := &supervisortest.Rec{}
	sv := supervisor.New(d, nil)
	r := sim.New(cfg, scriptFactory)
	frames, drawTL, strips, draw3D := 0, 0, 0, 0
	r.OnFrame = func(st *stage.Stage) {
		if !stage.HasSTDDrawer(st) {
			t.Fatalf("stage %d has no STD drawer", cfg.Stage)
		}
		stage.DrawPorted(st, sv)
		frames++
		for _, c := range d.Take() {
			switch {
			case strings.HasPrefix(c, "draw 5 "):
				strips++
				drawTL++
			case strings.HasPrefix(c, "draw "):
				drawTL++
			case strings.HasPrefix(c, "draw3d "):
				draw3D++
			}
		}
	}
	if err := r.Run(rp.Chapters[i], io.Discard, limit); err != nil {
		t.Fatalf("run: %v", err)
	}
	t.Logf("stage %d: %d frames, %d DrawTL (%d strips), %d Draw3D",
		cfg.Stage, frames, drawTL, strips, draw3D)
	if frames > 0 && drawTL+draw3D == 0 {
		t.Errorf("%d frames drew nothing", frames)
	}
	return strips
}

// scriptFactory builds the stage script for a stage number, as tools/th10sim
// does.
func scriptFactory(stageNum int) stage.Script {
	switch stageNum {
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
	case 6:
		return stage6.NewScript()
	case 7:
		return extra.NewScript()
	default:
		return stage1.NewScript()
	}
}
