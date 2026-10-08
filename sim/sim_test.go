package sim

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"th10/game"
	"th10/replay"
	"th10/scene/stage"
	"th10/scene/stage/stage1"
	"th10/trace"
)

func testFactory(stageNum int) stage.Script {
	if stageNum == 1 {
		return stage1.NewScript()
	}
	return stage1.NewScript()
}

func loadDemo(t *testing.T, name string) replay.Chapter {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join("..", ".thtk", "th10_dat", name+".rpy"))
	if err != nil {
		t.Skipf("replay asset missing: %v", err)
	}
	r, err := replay.Parse(raw)
	if err != nil {
		t.Fatalf("parse %s: %v", name, err)
	}
	if len(r.Chapters) == 0 {
		t.Fatalf("%s has no chapters", name)
	}
	return r.Chapters[0]
}

// TestHeadlessRunsReplay drives stage 1 headlessly with the official demo2
// input stream (a stage-3 run, but any replay is a valid stream of button
// presses) and checks the world advances without panics. The stream is short
// of real input, so the run is expected to end early on game over.
func TestHeadlessRunsReplay(t *testing.T) {
	ch := loadDemo(t, "demo2")
	cfg := Config{Stage: 1, Character: game.CharReimu, ShotType: game.ShotA, Difficulty: game.DiffNormal}
	r := New(cfg, testFactory)
	var buf bytes.Buffer
	if err := r.Run(ch, &buf, 3000); err != nil {
		t.Fatalf("Run: %v", err)
	}
	tr := trace.NewReader(&buf)
	frames := 0
	var first, last *trace.Frame
	sawEnemy := false
	for {
		f, err := tr.Next()
		if err != nil {
			t.Fatalf("trace read: %v", err)
		}
		if f == nil {
			break
		}
		if first == nil {
			first = f
		}
		if f.Enemies > 0 {
			sawEnemy = true
		}
		frames++
		last = f
	}
	if frames < 200 {
		t.Fatalf("only %d frames traced, want >=200", frames)
	}
	if last == nil || first == nil {
		t.Fatal("no frames")
	}
	t.Logf("traced %d frames; final: pos=(%.2f,%.2f) life=%d power=%d enemies=%d bullets=%d",
		frames, last.PX, last.PY, last.PLife, last.PPower, last.Enemies, last.Bullets)
	if !sawEnemy {
		t.Error("no enemies spawned during the run")
	}
}

// TestHeadlessDeterministic is the core property the differential harness
// relies on: identical input must produce byte-identical traces.
func TestHeadlessDeterministic(t *testing.T) {
	ch := loadDemo(t, "demo2")
	cfg := Config{Stage: 1, Character: game.CharReimu, ShotType: game.ShotA, Difficulty: game.DiffNormal}

	run := func() string {
		r := New(cfg, testFactory)
		var buf bytes.Buffer
		if err := r.Run(ch, &buf, 900); err != nil {
			t.Fatalf("Run: %v", err)
		}
		return buf.String()
	}
	a := run()
	b := run()
	if a != b {
		// Locate the first differing line for a useful failure message.
		al := bytes.Split([]byte(a), []byte("\n"))
		bl := bytes.Split([]byte(b), []byte("\n"))
		for i := 0; i < len(al) && i < len(bl); i++ {
			if !bytes.Equal(al[i], bl[i]) {
				t.Fatalf("nondeterministic at line %d:\n  a: %s\n  b: %s", i, al[i], bl[i])
			}
		}
		t.Fatal("traces differ in length")
	}
	t.Logf("deterministic: %d bytes of trace identical across runs", len(a))
}

// TestHeadlessInputMovesPlayer verifies the replay input bits actually reach
// the player: a synthetic chapter holding "right" must move the player right.
func TestHeadlessInputMovesPlayer(t *testing.T) {
	ch := replay.Chapter{
		Index: 1,
		Records: make([]replay.Record, 120),
	}
	for i := range ch.Records {
		ch.Records[i] = replay.Record{Keys: replay.KeyRightBit()}
	}
	cfg := Config{Stage: 1, Character: game.CharReimu, ShotType: game.ShotA, Difficulty: game.DiffNormal}
	r := New(cfg, testFactory)
	var buf bytes.Buffer
	if err := r.Run(ch, &buf, 0); err != nil {
		t.Fatalf("Run: %v", err)
	}
	tr := trace.NewReader(&buf)
	var first, last *trace.Frame
	for {
		f, err := tr.Next()
		if err != nil {
			t.Fatal(err)
		}
		if f == nil {
			break
		}
		if first == nil {
			first = f
		}
		last = f
	}
	if first == nil || last == nil {
		t.Fatal("no frames")
	}
	if last.PX <= first.PX {
		t.Errorf("player did not move right: %.2f -> %.2f", first.PX, last.PX)
	}
	t.Logf("player moved %.2f -> %.2f over %d frames", first.PX, last.PX, last.Index)
}
