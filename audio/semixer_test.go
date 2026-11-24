//go:build !js

package audio

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"testing"
)

func pcmSamples(samples ...int16) []byte {
	pcm := make([]byte, len(samples)*2)
	for i, sample := range samples {
		binary.LittleEndian.PutUint16(pcm[i*2:], uint16(sample))
	}
	return pcm
}

// readMixer reads n bytes from m in reads of size bytes.
func readMixer(t *testing.T, m *seMixer, n, size int) []byte {
	t.Helper()
	var got bytes.Buffer
	buf := make([]byte, size)
	for got.Len() < n {
		k, err := m.Read(buf[:min(size, n-got.Len())])
		if err != nil || k == 0 {
			t.Fatalf("Read = %d, %v", k, err)
		}
		got.Write(buf[:k])
	}
	return got.Bytes()
}

// A voice scales and truncates each sample on its own, with the table gain
// times the DirectSound pan law, then falls silent; odd read sizes must not
// swap channels or split a sample.
func TestMixerVoiceSamples(t *testing.T) {
	input := pcmSamples(10000, -10000, -32768, 32767, 1, -1)
	for _, tc := range []struct {
		gain float64
		pan  int32
		want []byte
	}{
		{1, 0, input},
		{1, 1000, pcmSamples(3162, -10000, -10362, 32767, 0, -1)},
		{0.5, -1000, pcmSamples(5000, -1581, -16384, 5180, 0, 0)},
		{0.1, 0, pcmSamples(1000, -1000, -3276, 3276, 0, 0)},
	} {
		for size := 1; size <= 17; size++ {
			t.Run(fmt.Sprintf("%g/%d/read%d", tc.gain, tc.pan, size), func(t *testing.T) {
				original := append([]byte(nil), input...)
				m := newSEMixer()
				l, r := sePanGains(tc.pan)
				m.play(3, input, tc.gain*l, tc.gain*r)
				got := readMixer(t, m, len(input)+8, size)
				want := append(append([]byte(nil), tc.want...), make([]byte, 8)...)
				if !bytes.Equal(got, want) {
					t.Fatalf("samples = % x, want % x", got, want)
				}
				if !bytes.Equal(input, original) {
					t.Fatal("mixer modified the shared PCM cache")
				}
			})
		}
	}
}

// Playing an id again restarts its voice from the first frame (FUN_0043ddf0
// stops and rewinds the id's buffer); other ids keep sounding and add up,
// clamped to 16 bits, and the master volume scales the sum.
func TestMixerVoicesRestartSumAndMaster(t *testing.T) {
	a := pcmSamples(1000, 2000, 3000, 4000, 5000, 6000)
	b := pcmSamples(30000, -30000, 30000, -30000, 30000, -30000)
	m := newSEMixer()
	m.play(1, a, 1, 1)
	if got := readMixer(t, m, 4, 4); !bytes.Equal(got, pcmSamples(1000, 2000)) {
		t.Fatalf("first frame = % x", got)
	}
	m.play(1, a, 1, 1) // rewind
	m.play(2, b, 1, 1)
	if got, want := readMixer(t, m, 12, 12), pcmSamples(31000, -28000, 32767, -26000, 32767, -24000); !bytes.Equal(got, want) {
		t.Fatalf("restart+sum = % x, want % x", got, want)
	}
	m.play(1, a, 1, 1)
	m.setMaster(0.5)
	if got, want := readMixer(t, m, 4, 4), pcmSamples(500, 1000); !bytes.Equal(got, want) {
		t.Fatalf("master 0.5 = % x, want % x", got, want)
	}
}

// The stream never ends: with no voice it is silence of the full length.
func TestMixerIdleIsSilence(t *testing.T) {
	m := newSEMixer()
	buf := bytes.Repeat([]byte{0xff}, 64)
	if n, err := m.Read(buf); n != len(buf) || err != nil || !bytes.Equal(buf, make([]byte, 64)) {
		t.Fatalf("idle Read = %d, %v, % x", n, err, buf)
	}
	if n, err := m.Read(buf[:3]); n != 3 || err != nil {
		t.Fatalf("partial idle Read = %d, %v", n, err)
	}
}

// Steady-state reads and plays must not allocate: that was the browser stutter.
func TestMixerNoAllocs(t *testing.T) {
	m := newSEMixer()
	pcm := make([]byte, 4096*pcmBytesPerFrame)
	buf := make([]byte, 1024)
	m.Read(buf)
	if n := testing.AllocsPerRun(100, func() {
		m.play(5, pcm, 0.5, 0.25)
		m.Read(buf)
	}); n != 0 {
		t.Fatalf("%v allocations per play+read", n)
	}
}
