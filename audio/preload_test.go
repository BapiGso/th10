package audio

import (
	"bytes"
	"io"
	"testing"
)

func TestSEPreloadKeepsPCMAndPlaybackCache(t *testing.T) {
	m := NewManager(nil)
	if err := m.preloadSE(44100); err != nil {
		t.Fatal(err)
	}
	if len(m.seCache) != len(seWav) {
		t.Fatalf("preloaded %d files, want %d", len(m.seCache), len(seWav))
	}
	for _, name := range seWav {
		want, err := decodeSEPCM(name, 44100)
		if err != nil {
			t.Fatal(err)
		}
		got := m.seCache[name]
		if !bytes.Equal(got, want) || len(got)%pcmBytesPerFrame != 0 {
			t.Fatalf("%s: preloading changed stereo PCM", name)
		}
	}
	delete(m.seCache, seWav[0])
	next := NewManager(nil)
	if err := next.preloadSE(44100); err != nil || next.seCache[seWav[0]] == nil {
		t.Fatal("manager shared its mutable name map")
	}
}

// Length() uses Ogg's last granule. Reading to EOF also checks the actual
// decoder produces exactly Size/4 sample frames, with no encoder delay/tail.
func TestBGMCompleteDecodedLength(t *testing.T) {
	tracks, err := BGMTracks()
	if err != nil {
		t.Fatal(err)
	}
	for _, tr := range tracks {
		if tr.Intro%pcmBytesPerFrame != 0 || tr.Size%pcmBytesPerFrame != 0 || tr.Intro < 0 || tr.Intro >= tr.Size {
			t.Fatalf("%s: invalid sample-aligned loop", tr.Name)
		}
		stream := decodeBGM(t, tr)
		n, err := io.Copy(io.Discard, stream)
		if err != nil || n != tr.Size {
			t.Errorf("%s: decoded %d bytes, want %d: %v", tr.Name, n, tr.Size, err)
		}
	}
}

func BenchmarkSELoad(b *testing.B) {
	for i := 0; i < b.N; i++ {
		sePCMByRate.Delete(44100)
		m := NewManager(nil)
		if err := m.preloadSE(44100); err != nil {
			b.Fatal(err)
		}
	}
}
