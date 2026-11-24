package audio

import (
	"fmt"
	"os"
	"testing"
	"time"

	ebaudio "github.com/hajimehoshi/ebiten/v2/audio"
)

// Opt-in stereo listening check, like TestPlayBGMLoop; never plays in CI.
func TestPlaySEPanning(t *testing.T) {
	if os.Getenv("TH10_TEST_AUDIO") != "1" {
		t.Skip("set TH10_TEST_AUDIO=1 for the stereo listening check")
	}
	m := NewManager(ebaudio.NewContext(44100))
	for _, x := range []float32{-192, 0, 192, 0} {
		fmt.Printf("SE 24: game x %g, DirectSound pan %d\n", x, SEPan(x))
		m.PlaySEIDAt(24, x)
		m.Update()
		time.Sleep(time.Second)
	}
}
