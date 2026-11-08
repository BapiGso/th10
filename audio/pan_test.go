package audio

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"math"
	"testing"
)

func pcmSamples(samples ...int16) []byte {
	pcm := make([]byte, len(samples)*2)
	for i, sample := range samples {
		binary.LittleEndian.PutUint16(pcm[i*2:], uint16(sample))
	}
	return pcm
}

func TestSEPan(t *testing.T) {
	for _, tc := range []struct {
		x    float32
		want int32
	}{
		{0, 0}, {192, 1000}, {-192, -1000}, {96, 500}, {-96, -500},
		{1, 5}, {-1, -5}, {0.1, 0}, {-0.1, 0}, {0.2, 1}, {-0.2, -1},
		{191.99998, 999}, {-191.99998, -999},
	} {
		if got := SEPan(tc.x); got != tc.want {
			t.Errorf("SEPan(%g) = %d, want %d", tc.x, got, tc.want)
		}
	}
}

func TestSEPanGains(t *testing.T) {
	for _, tc := range []struct {
		pan         int32
		left, right float64
	}{{0, 1, 1}, {1000, 0.31622776601683794, 1}, {-1000, 1, 0.31622776601683794},
		{2000, 0.1, 1}, {-2000, 1, 0.1}} {
		l, r := sePanGains(tc.pan)
		if math.Abs(l-tc.left) > 1e-15 || math.Abs(r-tc.right) > 1e-15 {
			t.Errorf("pan %d gains = %.17g/%.17g, want %.17g/%.17g", tc.pan, l, r, tc.left, tc.right)
		}
	}
}

func TestPanStreamSamples(t *testing.T) {
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
				s := newPanStream(bytes.NewReader(input), tc.gain, tc.pan)
				var got bytes.Buffer
				buf := make([]byte, size)
				for {
					n, err := s.Read(buf)
					got.Write(buf[:n])
					if err == io.EOF {
						break
					}
					if err != nil {
						t.Fatal(err)
					}
				}
				if !bytes.Equal(got.Bytes(), tc.want) {
					t.Fatalf("samples = % x, want % x", got.Bytes(), tc.want)
				}
				if !bytes.Equal(input, original) {
					t.Fatal("pan wrapper modified the shared PCM cache")
				}
			})
		}
	}
}

type shortPCMReader struct{ *bytes.Reader }

func (r shortPCMReader) Read(p []byte) (int, error) {
	return r.Reader.Read(p[:min(7, len(p))])
}

func TestPanStreamShortSourceReads(t *testing.T) {
	input := bytes.Repeat(pcmSamples(10000, -10000), 1500)
	s := newPanStream(shortPCMReader{bytes.NewReader(input)}, 1, 1000)
	got, err := io.ReadAll(s)
	if err != nil {
		t.Fatal(err)
	}
	if want := bytes.Repeat(pcmSamples(3162, -10000), 1500); !bytes.Equal(got, want) {
		t.Fatal("short source reads changed channels or samples across the internal buffer")
	}
}

func TestPanStreamSeek(t *testing.T) {
	input := pcmSamples(10000, -10000, -32768, 32767)
	want := pcmSamples(3162, -10000, -10362, 32767)
	s := newPanStream(bytes.NewReader(input), 1, 1000)
	first := make([]byte, 3)
	if _, err := io.ReadFull(s, first); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		offset int64
		whence int
		pos    int64
	}{{-1, io.SeekCurrent, 2}, {1, io.SeekStart, 1}, {-3, io.SeekEnd, 5}, {0, io.SeekStart, 0}} {
		if pos, err := s.Seek(tc.offset, tc.whence); err != nil || pos != tc.pos {
			t.Fatalf("seek %d/%d = %d, %v; want %d", tc.offset, tc.whence, pos, err, tc.pos)
		}
		got, err := io.ReadAll(s)
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(got, want[tc.pos:]) {
			t.Fatalf("seek %d output = % x, want % x", tc.pos, got, want[tc.pos:])
		}
	}
	if _, err := s.Seek(-1, io.SeekStart); err == nil {
		t.Fatal("negative seek succeeded")
	}
	if _, err := s.Seek(0, 99); err == nil {
		t.Fatal("unknown seek whence succeeded")
	}
}

func TestPanStreamIncompleteFrame(t *testing.T) {
	s := newPanStream(bytes.NewReader([]byte{1, 2, 3}), 1, 0)
	if _, err := io.ReadAll(s); err != io.ErrUnexpectedEOF {
		t.Fatalf("truncated PCM error = %v, want unexpected EOF", err)
	}
}

func TestSEQueuePanAndAux(t *testing.T) {
	m := NewManager(nil)
	if m.seAux[2] != -1 {
		t.Fatal("aux attribute was not initialized to -1")
	}
	m.PlaySEIDAt(2, 192)
	m.PlaySEIDAt(2, -96)
	m.PlaySEID(2)
	if len(m.seQueue) != 1 || m.seQueue[0].count != 3 || m.seQueue[0].pan() != 166 {
		t.Fatalf("queue = %+v, want one id with three requests and pan trunc(500/3)=166", m.seQueue)
	}
	if m.seAux[2] != 5 {
		t.Fatalf("aux = %d, want table +6 value 5", m.seAux[2])
	}
	m.Update()
	if len(m.seQueue) != 0 || m.seAux[2] != 5 {
		t.Fatal("frame flush did not clear requests or incorrectly cleared the unused attribute")
	}
	m.PlaySEIDAt(2, -0.2)
	m.PlaySEID(2)
	if m.seQueue[0].pan() != 0 {
		t.Fatalf("negative average = %d, want trunc(-1/2)=0", m.seQueue[0].pan())
	}
}

func TestSEQueueLimits(t *testing.T) {
	m := NewManager(nil)
	m.PlaySEID(-1)
	m.PlaySEID(len(seTable))
	for id := 0; id <= seQueueMax; id++ {
		m.PlaySEID(id)
	}
	if len(m.seQueue) != seQueueMax || m.seAux[seQueueMax] != -1 {
		t.Fatalf("queue limit = %d, aux for dropped id = %d", len(m.seQueue), m.seAux[seQueueMax])
	}
	for range sePanMax - 1 {
		m.PlaySEIDAt(0, 192)
	}
	m.PlaySEIDAt(0, -192) // request 129 must not change the first 128 pans.
	q := m.seQueue[0]
	if q.count != sePanMax || q.sum != 127000 || q.pan() != 992 {
		t.Fatalf("128-pan limit = %+v, average %d", q, q.pan())
	}
}
