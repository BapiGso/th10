// Decode SE eagerly during audio-manager loading, never during actual playback.
package audio

import (
	"bytes"
	"fmt"
	"io"
	"maps"
	"sync"

	"th10/datfs"
	"th10/gamedata"

	"github.com/hajimehoshi/ebiten/v2/audio/wav"
)

type sePCMSet struct {
	once sync.Once
	pcm  map[string][]byte
	err  error
}

var sePCMByRate sync.Map // sample rate -> shared immutable predecoded PCM

// preloadSE mirrors the original loading boundary, not a playback callback.
// Managers share immutable PCM; their name maps remain private. A nil-device
// headless manager need not decode sounds it will never play.
func (m *Manager) preloadSE(sampleRate int) error {
	value, _ := sePCMByRate.LoadOrStore(sampleRate, &sePCMSet{})
	set := value.(*sePCMSet)
	set.once.Do(func() {
		set.pcm = make(map[string][]byte, len(seWav))
		for _, name := range seWav {
			pcm, err := decodeSEPCM(name, sampleRate)
			if err != nil {
				set.err = fmt.Errorf("audio preload %s: %w", name, err)
				continue
			}
			set.pcm[name] = pcm
		}
	})
	m.seCache = maps.Clone(set.pcm)
	return set.err
}

func decodeSEPCM(name string, sampleRate int) ([]byte, error) {
	data, err := gamedata.ReadFile(datfs.EmbedPath(name))
	if err != nil {
		return nil, err
	}
	stream, err := wav.DecodeWithSampleRate(sampleRate, bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	return io.ReadAll(stream)
}
