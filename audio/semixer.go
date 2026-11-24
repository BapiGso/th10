//go:build !js

package audio

import (
	"encoding/binary"
	"sync"
	"time"

	"github.com/hajimehoshi/ebiten/v2/audio"
)

// seMixerBuffer is how far ahead the mixer player reads, so it is also the
// latency it adds to a sound effect. It must cover one device request
// (WASAPI asks for up to its 50 ms buffer, normally a 10 ms period).
const seMixerBuffer = 40 * time.Millisecond

// seVoice is one id's voice: the wav's PCM, the byte offset of the next
// frame (pcm is nil when the voice is silent) and the channel gains.
type seVoice struct {
	pcm         []byte
	pos         int
	left, right float64
}

// seMixer is an endless 16-bit stereo stream summing every voice. Read runs
// on the audio goroutine, play and setMaster on the game's, hence mu.
type seMixer struct {
	mu     sync.Mutex
	voices [len(seTable)]seVoice
	master float64
	acc    []int32 // Read's scratch: the left/right sums of each frame
	carry  [pcmBytesPerFrame]byte
	carryN int // bytes of carry not yet returned (a read that split a frame)
	player *audio.Player
}

func newSEOutput(ctx *audio.Context, _ map[string][]byte) seOutput {
	m := newSEMixer()
	p, err := ctx.NewPlayer(m)
	if err != nil {
		return nil
	}
	p.SetBufferSize(seMixerBuffer)
	p.Play()
	m.player = p
	return m
}

func newSEMixer() *seMixer { return &seMixer{master: 1} }

func (m *seMixer) play(id int, pcm []byte, left, right float64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.voices[id] = seVoice{pcm: pcm, left: left, right: right}
}

func (m *seMixer) setMaster(v float64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.master = v
}

// Read never ends: idle voices give silence. Each voice's sample is scaled
// and truncated to int16 on its own, as the per-play pan stream did before
// (DirectSound attenuates each buffer before mixing); the sum is scaled by
// the master volume and clamped.
func (m *seMixer) Read(p []byte) (int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	n := copy(p, m.carry[pcmBytesPerFrame-m.carryN:])
	m.carryN -= n
	if n == len(p) {
		return n, nil
	}
	frames := (len(p) - n + pcmBytesPerFrame - 1) / pcmBytesPerFrame
	if cap(m.acc) < frames*2 {
		m.acc = make([]int32, frames*2)
	}
	acc := m.acc[:frames*2]
	clear(acc)
	for i := range m.voices {
		v := &m.voices[i]
		if v.pcm == nil {
			continue
		}
		k := min(frames, (len(v.pcm)-v.pos)/pcmBytesPerFrame)
		src := v.pcm[v.pos:]
		for f := range k {
			l := int16(binary.LittleEndian.Uint16(src[f*pcmBytesPerFrame:]))
			r := int16(binary.LittleEndian.Uint16(src[f*pcmBytesPerFrame+2:]))
			acc[2*f] += int32(int16(float64(l) * v.left))
			acc[2*f+1] += int32(int16(float64(r) * v.right))
		}
		v.pos += k * pcmBytesPerFrame
		if len(v.pcm)-v.pos < pcmBytesPerFrame {
			v.pcm = nil
		}
	}
	for f := range frames {
		var frame [pcmBytesPerFrame]byte
		binary.LittleEndian.PutUint16(frame[0:], uint16(clamp16(float64(acc[2*f])*m.master)))
		binary.LittleEndian.PutUint16(frame[2:], uint16(clamp16(float64(acc[2*f+1])*m.master)))
		c := copy(p[n:], frame[:])
		n += c
		if c < pcmBytesPerFrame { // the last frame only partly fits: keep the rest
			m.carry = frame
			m.carryN = pcmBytesPerFrame - c
		}
	}
	return n, nil
}

func clamp16(v float64) int16 {
	if v >= 32767 {
		return 32767
	}
	if v <= -32768 {
		return -32768
	}
	return int16(v)
}
