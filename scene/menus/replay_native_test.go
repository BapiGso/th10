package menus

import (
	"bytes"
	"encoding/binary"
	"strings"
	"testing"

	"th10/game"
	"th10/input"
	"th10/replay"
	"th10/supervisor"
	"th10/supervisor/supervisortest"
)

func menuReplayFixture() *replay.Replay {
	r := &replay.Replay{Global: replay.GlobalHeader{
		User: "MENU", Timestamp: 1700000000, Score: 88888, SlowRate: 0.2,
		Character: 1, ShotType: 2, Difficulty: 2, Stage: 8, Raw: make([]byte, 0x64),
	}}
	binary.LittleEndian.PutUint32(r.Global.Raw[0x60:], 5)
	// Deliberately noncontiguous and not stage-sorted. NewPlayback takes
	// the slice index, whereas the menu's native cursor takes stage-1.
	for _, stage := range []uint16{5, 2, 6, 3} {
		r.Chapters = append(r.Chapters, replay.Chapter{Index: stage, Score: int32(stage) * 111111,
			State90: int32(stage) / 2, Seed: 0x1234, Faith: 5000, Lives: 2, Power: 20,
			PlayerY: 40000, Records: []replay.Record{{}, {}, {}}})
	}
	return r
}

func TestNativeReplayMenuDetailsAndChapterMapping(t *testing.T) {
	menusInput(t)
	g := menusGame()
	if err := g.SaveReplay(1, menuReplayFixture()); err != nil {
		t.Fatal(err)
	}
	s := NewReplay(nil)
	settleMenu(t, g, s, 10)
	tapMenu(t, g, s, input.KeyOk)
	if s.state != 4 || s.chapter != 1 {
		t.Fatal("replay list did not enter stage-indexed details at the first available stage")
	}
	d := s.Details()
	for _, word := range []string{"No.01", "MENU", "MarisaC", "Hard", "All", "0.2%"} {
		if !strings.Contains(d.Header, word) {
			t.Fatal("native replay detail header omitted metadata:", d.Header)
		}
	}
	if d.Chapters[1].ChapterIndex != 1 || d.Chapters[1].Score != 333333 || d.Chapters[1].Continues != 1 {
		t.Fatal("stage 2 did not display the next stage's starting score/continue")
	}
	if d.Chapters[4].Score != 666666 || d.Chapters[4].Continues != 3 {
		t.Fatal("stage 5 did not use stage 6's stored header")
	}
	if d.Chapters[5].Score != 88888 || d.Chapters[5].Continues != 5 || d.Chapters[2].Score != 88888 || d.Chapters[2].Continues != 5 {
		t.Fatal("final stage/missing-next-stage did not use the original global final score")
	}
	if d.Chapters[0].Available || !strings.Contains(d.Chapters[0].Text, "---------") || d.Chapters[6].Available {
		t.Fatal("missing stages were not shown as unavailable")
	}
	tapMenu(t, g, s, input.KeyDown)
	if s.chapter != 1 {
		t.Fatal("chapter input escaped the native fifteen-frame gate")
	}
	settleMenu(t, g, s, 15)
	for _, want := range []int{2, 4, 5, 1} {
		tapMenu(t, g, s, input.KeyDown)
		if s.chapter != want {
			t.Fatalf("missing-stage skip selected %d, want %d", s.chapter, want)
		}
	}
	tapMenu(t, g, s, input.KeyCancel)
	if s.state != 2 || s.selection != 0 || s.selected != nil {
		t.Fatal("cancel did not restore the original slot cursor")
	}
}

func TestNativeReplayMenuUsesT5PlaybackAndReturns(t *testing.T) {
	menusInput(t)
	g := menusGame()
	r := menuReplayFixture()
	if err := g.SaveReplay(1, r); err != nil {
		t.Fatal(err)
	}
	s := NewReplay(nil)
	settleMenu(t, g, s, 10)
	tapMenu(t, g, s, input.KeyOk)
	settleMenu(t, g, s, 15)
	tapMenu(t, g, s, input.KeyDown) // stage 3 is slice index 3, not index 2
	before, err := game.EncodeScoreFile(g.Save.Native)
	if err != nil {
		t.Fatal(err)
	}
	tapMenu(t, g, s, input.KeyOk)
	settleMenu(t, g, s, 30)
	if s.playing != nil || s.timer.Cur != 32 {
		t.Fatal("replay started before the original 32-frame confirmation")
	}
	settleMenu(t, g, s, 1)
	if s.playing == nil || s.playing.Chapter() != 3 || s.loaded {
		t.Fatal("menu did not call t5 playback with the selected chapter's slice index")
	}
	for range 8 {
		if err := s.playing.UpdateHeadless(g); err != nil {
			t.Fatal(err)
		}
	}
	if !s.playing.Finished() {
		t.Fatal("selected last stored chapter did not finish through t5's EOF path")
	}
	after, err := game.EncodeScoreFile(g.Save.Native)
	if err != nil || !bytes.Equal(before, after) {
		t.Fatal("menu/playback wrote shared score or spell history")
	}
	r.Global.User = "NEW"
	if err := g.SaveReplay(1, r); err != nil {
		t.Fatal(err)
	}
	settleMenu(t, g, s, 10) // same return scene, refresh the metadata from storage
	if s.state != 2 || s.selection != 0 || len(s.entries) != 25 || s.entries[0].slot.Replay.Global.User != "NEW" {
		t.Fatal("playback return did not restore/reload the same replay menu")
	}
}

func TestNativeReplayMenuEmptyCorruptAndHeadlessDraw(t *testing.T) {
	menusInput(t)
	g := menusGame()
	if err := g.SaveReplay(1, menuReplayFixture()); err != nil {
		t.Fatal(err)
	}
	if err := g.Replays.Write(2, []byte("not a replay")); err != nil {
		t.Fatal(err)
	}
	s := NewReplay(nil)
	settleMenu(t, g, s, 10)
	if len(s.Slots()) != 25 || s.entries[1].slot.Error == nil || s.entries[2].slot.Replay != nil {
		t.Fatal("t5 corrupt/empty slot metadata was hidden")
	}
	s.selection = 1
	tapMenu(t, g, s, input.KeyOk)
	if s.state != 2 || s.status == "" || s.playing != nil {
		t.Fatal("corrupt replay entered playback/details")
	}
	s.selection = 2
	tapMenu(t, g, s, input.KeyOk)
	if s.state != 2 || !strings.Contains(s.status, "empty") {
		t.Fatal("empty replay entered playback/details")
	}
	s.selection = 0
	tapMenu(t, g, s, input.KeyOk)
	dev := &supervisortest.Rec{}
	sv := supervisor.New(dev, nil)
	draws := 0
	for range 40 {
		input.Global.Restore(0)
		if err := s.Update(g); err != nil {
			t.Fatal(err)
		}
		s.DrawPorted(sv)
		for _, c := range dev.Take() {
			if strings.HasPrefix(c, "draw ") {
				draws++
			}
		}
	}
	if draws == 0 {
		t.Fatal("replay details did not draw native ANM/small glyphs")
	}
	tapMenu(t, g, s, input.KeyCancel)
	tapMenu(t, g, s, input.KeyCancel)
	settleMenu(t, g, s, 7)
	if !s.closed || s.core.world.Len() != 0 {
		t.Fatal("replay exit leaked the scene's ANM objects")
	}
}

type replayCatalogStub struct {
	r                    *replay.Replay
	listCalls, loadCalls int
}

func (c *replayCatalogStub) ListReplays() []game.ReplaySlot {
	c.listCalls++
	return []game.ReplaySlot{{Number: 26, Replay: &replay.Replay{Global: c.r.Global}}}
}

func (c *replayCatalogStub) LoadReplay(slot int) (*replay.Replay, error) {
	c.loadCalls++
	if slot != 26 {
		panic("catalog slot changed")
	}
	// A catalog/backend uses the same pure t5 codec, not a scene decoder.
	data, err := replay.Encode(c.r)
	if err != nil {
		return nil, err
	}
	return replay.Decode(data)
}

func TestNativeReplayMenuOptionalCatalogUsesExistingCodec(t *testing.T) {
	menusInput(t)
	c := &replayCatalogStub{r: menuReplayFixture()}
	s := NewReplay(nil)
	s.Source = c
	g := menusGame()
	settleMenu(t, g, s, 10)
	tapMenu(t, g, s, input.KeyOk)
	if c.listCalls != 1 || c.loadCalls != 1 || s.Details().Slot != 26 || s.selected == nil {
		t.Fatal("catalog hook bypassed metadata/full-load separation or the original codec")
	}
}
