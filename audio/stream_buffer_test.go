package audio

import (
	"testing"
	"time"
)

func TestBGMBufferIsOptInAndDoesNotPlay(t *testing.T) {
	m := NewManager(nil)
	if m.BGMBufferSize() != 0 {
		t.Fatal("desktop default buffer changed")
	}
	m.SetBGMBufferSize(200 * time.Millisecond)
	if m.BGMBufferSize() != 200*time.Millisecond || m.bgm != nil {
		t.Fatal("buffer setting played or reset a stream")
	}
	m.SetBGMBufferSize(-time.Second)
	if m.BGMBufferSize() != 0 {
		t.Fatal("negative buffer did not restore the default policy")
	}
}
