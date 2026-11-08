package stage_test

import (
	"io"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"th10/game"
	"th10/replay"
	"th10/scene/stage"
	"th10/scene/stage/stage1"
	"th10/sim"

	"github.com/hajimehoshi/ebiten/v2"
)

type playbackScript struct{ clearAt int }

func (*playbackScript) Init(*stage.Context) {}
func (s *playbackScript) Update(c *stage.Context) {
	if s.clearAt > 0 && c.State.Frame == s.clearAt {
		c.OpenMSG(999)
	}
}
func (*playbackScript) BgDraw(*ebiten.Image) {}
func (*playbackScript) Finished() bool       { return false }

func playbackFixture() *replay.Replay {
	r := &replay.Replay{Global: replay.GlobalHeader{Stage: 2, Difficulty: 1}}
	for i := 1; i <= 2; i++ {
		ch := replay.Chapter{Index: uint16(i), Seed: uint16(i * 100), Faith: 5000, Lives: 2,
			PlayerY: 40000, Records: make([]replay.Record, 120)}
		for j := range ch.Records {
			ch.Records[j].Keys = replay.KeyRight
		}
		r.Chapters = append(r.Chapters, ch)
	}
	r.Chapters[1].Score, r.Chapters[1].Rank, r.Chapters[1].Lives = 12345, 256, 1
	r.Chapters[1].PlayerX, r.Chapters[1].PlayerY = -1855, 42000
	return r
}

func TestPlaybackStoredReplayRestoresEachChapterWithoutProgress(t *testing.T) {
	g := &game.Game{Save: game.DefaultSaveData(), Replays: game.NewMemoryReplayStore()}
	g.Save.TotalRuns, g.Save.TotalClears, g.Save.HighScore = 3, 2, 999999
	wantSave := *g.Save
	if err := g.SaveReplay(1, playbackFixture()); err != nil {
		t.Fatal(err)
	}
	source, err := g.LoadReplay(1)
	if err != nil {
		t.Fatal(err)
	}
	p, err := stage.NewPlayback(g, source, 0, func(int) stage.Script { return &playbackScript{} }, nil)
	if err != nil {
		t.Fatal(err)
	}
	for range 120 {
		if err := p.UpdateHeadless(g); err != nil {
			t.Fatal(err)
		}
	}
	if p.Chapter() != 1 {
		t.Fatal("second chapter did not start")
	}
	c := stage.ExportContext(p.Stage())
	if c.State.Score != 123450 || c.State.Rank != 256 || c.State.Life != 1 || c.Player.IX != -1855 || c.Player.IY != 42000 || c.State.Frame != 0 {
		t.Fatal("chapter-start state differs")
	}
	if c.State.ReplayMode != 1 || c.State.Recording != nil {
		t.Fatal("playback started a recorder")
	}
	for range 120 {
		if err := p.UpdateHeadless(g); err != nil {
			t.Fatal(err)
		}
	}
	if !p.Finished() || c.Player.IX <= -1855 {
		t.Fatal("recorded input did not move the player or playback did not finish")
	}
	if !reflect.DeepEqual(*g.Save, wantSave) {
		t.Fatal("playback changed persistent progress")
	}
}

type playbackState struct {
	frame, life, power, rank, bullets, enemies int
	x, y                                       int32
	score, faith                               int64
	seed                                       uint16
}

func playbackSnapshot(s *stage.Stage) playbackState {
	c := stage.ExportContext(s)
	return playbackState{frame: c.State.Frame, life: c.State.Life, power: c.State.Power, rank: c.State.Rank,
		bullets: c.Bullets.Live(), enemies: len(c.Enemies), x: c.Player.IX, y: c.Player.IY,
		score: c.State.Score, faith: c.State.Faith, seed: c.RNG.State}
}

func TestInteractivePlaybackMatchesSim(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "..", ".tmp", "[th10]", "replay", "th10_05.rpy"))
	if err != nil {
		t.Skipf("original replay unavailable: %v", err)
	}
	r, err := replay.Decode(data)
	if err != nil {
		t.Fatal(err)
	}
	factory := func(int) stage.Script { return stage1.NewScript() }
	cfg, err := sim.ConfigFor(r, 0)
	if err != nil {
		t.Fatal(err)
	}
	var expected []playbackState
	runner := sim.New(cfg, factory)
	runner.OnFrame = func(s *stage.Stage) { expected = append(expected, playbackSnapshot(s)) }
	if err := runner.Run(r.Chapters[0], io.Discard, 1500); err != nil {
		t.Fatal(err)
	}
	g := &game.Game{Save: game.DefaultSaveData()}
	p, err := stage.NewPlayback(g, r, 0, factory, nil)
	if err != nil {
		t.Fatal(err)
	}
	var got []playbackState
	p.OnFrame = func(s *stage.Stage) { got = append(got, playbackSnapshot(s)) }
	for range 1500 {
		if err := p.UpdateHeadless(g); err != nil {
			t.Fatal(err)
		}
	}
	if !reflect.DeepEqual(got, expected) {
		for i := range min(len(got), len(expected)) {
			if got[i] != expected[i] {
				t.Fatalf("frame %d differs: %+v / %+v", i+1, got[i], expected[i])
			}
		}
		t.Fatalf("frame counts %d / %d", len(got), len(expected))
	}
}

func TestPlaybackRejectsMissingChapterAndFactory(t *testing.T) {
	g := &game.Game{}
	for _, chapter := range []int{-1, 2} {
		if _, err := stage.NewPlayback(g, playbackFixture(), chapter, func(int) stage.Script { return &playbackScript{} }, nil); err == nil {
			t.Fatal("invalid chapter accepted")
		}
	}
	if _, err := stage.NewPlayback(g, playbackFixture(), 0, nil, nil); err == nil {
		t.Fatal("missing factory accepted")
	}
}
