package game

import (
	"reflect"
	"testing"

	"th10/audio"
)

func TestSceneProgressNativeOffsets(t *testing.T) {
	s := DefaultScoreFile(0x1234)
	before := s.Stats
	s.MarkEndingSeen(0)
	s.MarkEndingSeen(11)
	s.MarkMusicPlayed(0)
	s.MarkMusicPlayed(17)
	for i, b := range before {
		want := b
		switch i {
		case 0x16, 0x21, 0x22, 0x26, 0x37:
			want = 1
		}
		if s.Stats[i] != want {
			t.Fatalf("ST+%#x = %d, want %d", i, s.Stats[i], want)
		}
	}
	before = s.Stats
	for _, i := range []int{-1, 12, 18, 100} {
		s.MarkEndingSeen(i)
		if s.EndingSeen(i) {
			t.Fatalf("invalid ending %d is marked", i)
		}
	}
	for _, i := range []int{-1, 18, 100} {
		s.MarkMusicPlayed(i)
		if s.MusicUnlocked(i) {
			t.Fatalf("invalid music %d is marked", i)
		}
	}
	if !reflect.DeepEqual(before, s.Stats) {
		t.Fatal("out-of-range writes changed ST")
	}
}

func TestSceneProgressPersistsThroughNativeStore(t *testing.T) {
	store := NewMemoryScoreStore()
	g := &Game{Scores: store}
	g.LoadProgress()
	g.Save.Native.MarkEndingSeen(8)
	g.Save.Native.MarkMusicPlayed(15)
	g.Save.Native.MarkMusicPlayed(16)
	if err := g.SaveProgress(); err != nil {
		t.Fatal(err)
	}
	reloaded := &Game{Scores: store}
	reloaded.LoadProgress()
	if !reloaded.Save.Native.EndingSeen(8) || !reloaded.Save.Native.StaffSeen() || !reloaded.Save.Native.MusicUnlocked(15) || !reloaded.Save.Native.MusicUnlocked(16) || reloaded.Save.Native.MusicUnlocked(14) {
		t.Fatal("scene flags did not survive the native CR/ST codec/backend")
	}
}

func TestMusicPlaybackTrackingAndReplayIsolation(t *testing.T) {
	g := &Game{Scores: NewMemoryScoreStore(), audioMgr: audio.NewManager(nil)}
	g.LoadProgress()
	s := NewState(0, 0, DiffNormal)
	s.ReplayMode = 1
	g.BindState(s)
	g.Audio().PlayBGM(audio.BGMExtra)
	if g.Save.Native.MusicUnlocked(audio.BGMExtra) {
		t.Fatal("playback changed shared music progress")
	}
	s.ReplayMode = 0
	g.BindState(s)
	g.Audio().PlayBGM(audio.BGMExtra)
	g.Audio().PlayBGMPreview(audio.BGMExtraBoss)
	if !g.Save.Native.MusicUnlocked(audio.BGMExtra) || g.Save.Native.MusicUnlocked(audio.BGMExtraBoss) {
		t.Fatal("live/preview playback gates reversed")
	}
	g.BindMusicProgress(nil)
	g.Audio().PlayBGM(audio.BGMTitle)
	reloaded := &Game{Scores: g.Scores}
	reloaded.LoadProgress()
	if !reloaded.Save.Native.MusicUnlocked(audio.BGMTitle) || !reloaded.Save.Native.MusicUnlocked(audio.BGMExtra) || reloaded.Save.Native.MusicUnlocked(audio.BGMExtraBoss) {
		t.Fatal("music playback record did not persist through the existing native backend")
	}
}
