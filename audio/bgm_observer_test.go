package audio

import (
	"reflect"
	"testing"
)

func TestBGMUnlockObserverAndMusicRoomPreview(t *testing.T) {
	m := NewManager(nil) // no audio device; native unlock is a queued-request side effect
	var got []int
	m.SetBGMObserver(func(index int) { got = append(got, index) })
	m.PlayBGM(0)
	m.PlayBGM(17)
	m.PlayBGMPreview(13)
	m.PlayBGM(-1)
	m.PlayBGM(18)
	m.StopBGM()
	m.FadeOutBGM(180)
	if !reflect.DeepEqual(got, []int{0, 17}) {
		t.Fatal("BGM preview/invalid index unexpectedly unlocked a track:", got)
	}
	m.SetBGMObserver(nil)
	m.PlayBGM(3)
	if len(got) != 2 {
		t.Fatal("removed observer still called")
	}
}
