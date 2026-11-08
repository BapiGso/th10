package menus

import (
	"bytes"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"th10/anm"
	"th10/game"
	"th10/gamedata"
	"th10/input"
	"th10/supervisor"
	"th10/supervisor/supervisortest"

	"golang.org/x/text/encoding/japanese"
)

func menusGame() *game.Game {
	g := &game.Game{Scores: game.NewMemoryScoreStore(), Replays: game.NewMemoryReplayStore()}
	g.LoadProgress()
	return g
}

func menusInput(t *testing.T) {
	previous := input.Global
	t.Cleanup(func() { input.Global = previous })
	input.Global = input.State{}
}

func settleMenu(t *testing.T, g *game.Game, scene game.Scene, frames int) {
	t.Helper()
	for range frames {
		input.Global.Restore(0)
		if err := scene.Update(g); err != nil {
			t.Fatal(err)
		}
	}
}

func tapMenu(t *testing.T, g *game.Game, scene game.Scene, key input.Key) {
	t.Helper()
	input.Global.Restore(1 << key)
	if err := scene.Update(g); err != nil {
		t.Fatal(err)
	}
	input.Global.Restore(0)
	if err := scene.Update(g); err != nil {
		t.Fatal(err)
	}
}

func TestMusicRoomReadsOriginalCommentsAndBGMOrder(t *testing.T) {
	raw, err := gamedata.ReadFile("misc/musiccmt.txt")
	if err != nil {
		t.Fatal(err)
	}
	original, err := parseMusic(raw, japanese.ShiftJIS)
	if err != nil {
		t.Fatal(err)
	}
	tracks, err := musicFiles()
	if err != nil || len(tracks) != 18 || len(original) != 18 {
		t.Fatalf("track list: %d/%d, %v", len(tracks), len(original), err)
	}
	translated := false
	for i, track := range tracks {
		if track.File != original[i].File || track.BGMIndex != i {
			t.Fatalf("track %d replaced the original filename/order", i)
		}
		for _, text := range track.Comments {
			if text == "" {
				t.Fatal("missing original comment line")
			}
		}
		translated = translated || track.Comments != original[i].Comments
	}
	if !translated {
		t.Fatal("traditional comment overlay was not used")
	}
	if _, err := parseMusic([]byte("@bgm/test.wav\nTITLE\nc0\nc1\n"), japanese.ShiftJIS); err == nil {
		t.Fatal("truncated eight-line comment block accepted")
	}
}

func TestMenuPresentationRetainsOriginalANMTimelines(t *testing.T) {
	original, err := gamedata.ANM("title.anm")
	if err != nil {
		t.Fatal(err)
	}
	presentation, err := menuAnimation("title.anm")
	if err != nil || !reflect.DeepEqual(original.Scripts, presentation.Scripts) {
		t.Fatal("Title ANT changed the subscene animation timelines")
	}
}

func TestMusicRoomUnlockWarningPreviewAndStop(t *testing.T) {
	menusInput(t)
	g := menusGame()
	g.Save.TotalClears, g.Save.ExtraUnlocked = 999, true // these are not music unlock conditions
	g.Save.Native.Stats[0x26+9] = 0xff                   // any nonzero original byte unlocks
	s := NewMusicRoom(nil)
	settleMenu(t, g, s, 20)
	for i := range game.MusicCount {
		if s.TrackUnlocked(i) != (i == 9) {
			t.Fatal("Music Room inferred unlock from a clear/count rather than ST+0x26")
		}
	}
	if err := s.Play(g, 9); err != nil || s.current != 9 || s.warning {
		t.Fatal("unlocked song did not play on the first confirmation")
	}
	if err := s.Play(g, 13); err != nil || !s.warning || s.current != -1 {
		t.Fatal("unknown song did not stop BGM and open the original warning")
	}
	settleMenu(t, g, s, 18)
	if s.commentContent != musicWarning {
		t.Fatal("unseen-song warning was not revealed as eight native lines")
	}
	if err := s.Play(g, 13); err != nil || s.warning || s.current != 13 || s.TrackUnlocked(13) {
		t.Fatal("second confirmation did not permit a read-only audition")
	}
	if !g.Save.Native.MusicUnlocked(0) {
		t.Fatal("original 0x4347e4 title-song write was lost")
	}
	if err := s.Play(g, 14); err != nil || !s.warning || s.current != -1 {
		t.Fatal("a different unknown track incorrectly reused the prior confirmation")
	}
	s.Stop(g)
	if s.current != -1 {
		t.Fatal("explicit stop retained the playing ordinal")
	}
	if err := s.Play(g, -1); err == nil {
		t.Fatal("invalid music selection accepted")
	}
}

func TestMusicRoomNativeRowsRepeatAndCommentTiming(t *testing.T) {
	menusInput(t)
	g := menusGame()
	s := NewMusicRoom(nil)
	for frame := 1; frame <= 9; frame++ {
		if err := s.Update(g); err != nil {
			t.Fatal(err)
		}
		count := 0
		for _, v := range s.labels {
			if v != nil {
				count++
			}
		}
		if count != frame*2 {
			t.Fatalf("entry frame %d created %d labels, want %d", frame, count, frame*2)
		}
	}
	settleMenu(t, g, s, 20)
	for frame := 1; frame <= 34; frame++ {
		input.Global.Restore(1 << input.KeyDown)
		if err := s.Update(g); err != nil {
			t.Fatal(err)
		}
		want := 1
		if frame >= 26 {
			want++
		}
		if frame >= 34 {
			want++
		}
		if s.selection != want {
			t.Fatalf("held frame %d selection %d, want %d", frame, s.selection, want)
		}
	}
	input.Global.Restore(0)
	s.selection, s.first = 17, 8
	s.moveLabels()
	tapMenu(t, g, s, input.KeyDown)
	settleMenu(t, g, s, 4)
	if s.selection != 0 || s.first != 0 {
		t.Fatal("ten-row window failed to wrap to the first track")
	}
	visible := 0
	for i, v := range s.labels {
		if v.Flags&anm.FlagDraw != 0 {
			visible++
			if v.Pos[1] != float32(96+i*20) {
				t.Fatal("Music Room used the wrong original row spacing")
			}
		}
	}
	if visible != 10 || s.labels[0].Pos[0] != 56 {
		t.Fatal("native window/selected eight-pixel indent is wrong")
	}
	if err := s.Play(g, 9); err != nil {
		t.Fatal(err)
	}
	settleMenu(t, g, s, 1)
	if s.commentAt != 1 {
		t.Fatal("comment line zero did not reveal at state time zero")
	}
	settleMenu(t, g, s, 1)
	if s.commentAt != 1 {
		t.Fatal("comment revealed on an odd state frame")
	}
	settleMenu(t, g, s, 1)
	if s.commentAt != 2 {
		t.Fatal("comment did not reveal every two state frames")
	}
}

func TestPlayerDataSelectedNativeStatistics(t *testing.T) {
	menusInput(t)
	g := menusGame()
	for shot := range 6 {
		p := &g.Save.Progress[shot]
		p.Runs, p.Frames = int32(shot*10+1), int32(shot*216000+11*3600+13*60+59)
		for difficulty := range 5 {
			p.Clears[difficulty] = int32(shot*100 + difficulty + 1)
			g.Save.Rankings[shot][difficulty][0].Score = int32(shot*10000 + difficulty*25)
		}
	}
	g.Save.HighScore, g.Save.TotalRuns, g.Save.TotalClears = 999999999, 99999, 88888
	if err := g.SaveProgress(); err != nil {
		t.Fatal(err)
	}
	reloaded := &game.Game{Scores: g.Scores}
	reloaded.LoadProgress()
	s := NewPlayerData(reloaded.Save, nil)
	for shot := range 6 {
		for difficulty := range 5 {
			s.shot, s.difficulty = shot, difficulty
			stats := s.Statistics()
			if stats.Runs != int32(shot*10+1) || stats.Clears != int32(shot*100+difficulty+1) || stats.PlayTime != fmt.Sprintf("%3d:11:13", shot) || stats.Rankings[0].Score != int32(shot*10000+difficulty*25) {
				t.Fatalf("wrong native statistics for shot/difficulty %d/%d: %+v", shot, difficulty, stats)
			}
		}
	}
}

func TestPlayerDataSpellHistoryUsesTotalNameAndShotCounts(t *testing.T) {
	menusInput(t)
	g := menusGame()
	history := &game.SpellHistory{}
	id := spellIDs(game.DiffNormal)[0]
	copy(history.Total[id].Name[:], "TOTAL NAME")
	history.Total[id].Attempts = 1
	copy(history.Shots[0][id].Name[:], "WRONG SHOT NAME")
	history.Shots[0][id].Captures, history.Shots[0][id].Attempts = 1234, 54321
	other := spellIDs(game.DiffNormal)[1]
	history.Shots[0][other].Captures, history.Shots[0][other].Attempts = 9, 20
	g.SpellHistory = history // intentionally different from the constructor's stored history
	s := NewPlayerData(g.Save, nil)
	settleMenu(t, g, s, 10)
	tapMenu(t, g, s, input.KeyOk)
	rows := s.SpellRows()
	if len(rows) != 10 || rows[0].Name != "TOTAL NAME" || rows[0].Captures != 1234 || rows[0].Attempts != 54321 || rows[0].Colour != 0xffff80 || !strings.Contains(rows[0].Text, "1234/54321") {
		t.Fatal("Player Data used global counts, shot-specific name, or GUI's 99/99 cap")
	}
	if rows[1].Known || rows[1].Captures != 9 || rows[1].Attempts != 20 || rows[1].Colour != 0x808080 {
		t.Fatal("name visibility did not use Total[id].Attempts independently of shot counts")
	}
	if s.rows[0] == nil || s.rows[0].File == s.core.text {
		t.Fatal("spell list did not use text.anm's native texture writer")
	}
	if len(spellIDs(0)) != 23 || len(spellIDs(1)) != 24 || len(spellIDs(2)) != 25 || len(spellIDs(3)) != 25 || len(spellIDs(4)) != 13 {
		t.Fatal("original PE difficulty table changed")
	}
}

func TestPlayerDataNativeNavigationAndReadOnlyDraw(t *testing.T) {
	menusInput(t)
	g := menusGame()
	s := NewPlayerData(g.Save, nil)
	settleMenu(t, g, s, 10)
	for want := 1; want <= 3; want++ {
		tapMenu(t, g, s, input.KeyOk)
		if s.page != want {
			t.Fatal("Z returned to title instead of advancing the original data page")
		}
	}
	tapMenu(t, g, s, input.KeyDown)
	if s.difficulty != game.DiffHard || s.page != 1 {
		t.Fatal("difficulty change did not retain/reset the history tab")
	}
	tapMenu(t, g, s, input.KeyLeft)
	if s.shot != 5 || s.weapon.ScriptID != 0x9f || s.character.ScriptID != 0x99 {
		t.Fatal("six-shot wrap did not switch native selection graphics")
	}
	before, err := game.EncodeScoreFile(g.Save.Native)
	if err != nil {
		t.Fatal(err)
	}
	dev := &supervisortest.Rec{}
	sv := supervisor.New(dev, nil)
	for range 60 {
		if err := s.Update(g); err != nil {
			t.Fatal(err)
		}
		s.DrawPorted(sv)
		dev.Take()
	}
	after, err := game.EncodeScoreFile(g.Save.Native)
	if err != nil || !bytes.Equal(before, after) {
		t.Fatal("viewing Player Data wrote shared score/history")
	}
	tapMenu(t, g, s, input.KeyCancel)
	settleMenu(t, g, s, 7)
	if !s.closed || s.core.world.Len() != 0 {
		t.Fatal("native data exit did not tear down only this subscene")
	}
}

func TestMusicAndPlayerDataHeadlessDraw(t *testing.T) {
	menusInput(t)
	g := menusGame()
	music, data := NewMusicRoom(nil), NewPlayerData(g.Save, nil)
	dev := &supervisortest.Rec{}
	sv := supervisor.New(dev, nil)
	draws := 0
	for frame := range 160 {
		input.Global.Restore(0)
		if frame == 60 {
			input.Global.Restore(1 << input.KeyOk)
		}
		if err := music.Update(g); err != nil {
			t.Fatal(err)
		}
		music.DrawPorted(sv)
		if err := data.Update(g); err != nil {
			t.Fatal(err)
		}
		data.DrawPorted(sv)
		for _, c := range dev.Take() {
			if strings.HasPrefix(c, "draw ") {
				draws++
			}
		}
	}
	if draws == 0 {
		t.Fatal("native menu ANM/text emitted no headless draws")
	}
	music.Close()
	data.Close()
}

func TestMenuPrivateLoadingEmitterNativeBirth(t *testing.T) {
	menusInput(t)
	g := menusGame()
	s := NewMusicRoom(nil)
	if err := s.Update(g); err != nil {
		t.Fatal(err)
	}
	count := 0
	for l := range anm.Layers {
		s.core.world.Each(l, func(v *anm.VM) {
			if v.File == s.core.ascii && v.ScriptID == 7 {
				count++
			}
		})
	}
	if count != 0 {
		t.Fatal("front-born loader particles were visited/drawn again on their birth pass")
	}
	if err := s.Update(g); err != nil {
		t.Fatal(err)
	}
	count = 0
	for l := range anm.Layers {
		s.core.world.Each(l, func(v *anm.VM) {
			if v.File == s.core.ascii && v.ScriptID == 7 {
				count++
			}
		})
	}
	if count != 16 {
		t.Fatalf("initial ANM-91 burst: %d particles, want 16", count)
	}
}
