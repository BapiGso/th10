package audio

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"th10/datfs"
	"th10/gamedata"

	"github.com/hajimehoshi/ebiten/v2/audio"
	"github.com/hajimehoshi/ebiten/v2/audio/vorbis"
	"github.com/hajimehoshi/ebiten/v2/audio/wav"
)

func TestSEDecode(t *testing.T) {
	for _, name := range []string{SEShoot, SEEnemy, SEDamage, SEGraze, SEItem, SEExtend, SEBomb,
		SEOk, SECancel, SESelect, SEPause, SEPowerUp, SEDead} {
		data, err := gamedata.ReadFile(datfs.EmbedPath(name))
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		s, err := wav.DecodeWithSampleRate(44100, bytes.NewReader(data))
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		if s.Length() <= 0 {
			t.Fatalf("%s: empty", name)
		}
	}
}

// TestBGMLoopSeam checks every embedded track has thbgm.fmt's length and
// that the stream runs past Size straight into Intro.
func TestBGMLoopSeam(t *testing.T) {
	tracks, err := BGMTracks()
	if err != nil {
		t.Fatal(err)
	}
	if len(tracks) != len(BGMNames) {
		t.Fatalf("%d tracks in thbgm.fmt, %d names", len(tracks), len(BGMNames))
	}
	for idx, tr := range tracks {
		s := decodeBGM(t, tr)
		if s.Length() != tr.Size {
			t.Errorf("%s: %d PCM bytes, thbgm.fmt says %d", tr.Name, s.Length(), tr.Size)
		}
		want := make([]byte, 2*pcmBytesPerFrame)
		readAt(t, s, want[:pcmBytesPerFrame], tr.Size-pcmBytesPerFrame)
		readAt(t, s, want[pcmBytesPerFrame:], tr.Intro)

		loop, err := NewBGMStream(idx, tr.SampleRate)
		if err != nil {
			t.Fatal(err)
		}
		got := make([]byte, 2*pcmBytesPerFrame)
		readAt(t, loop, got, tr.Size-pcmBytesPerFrame)
		if !bytes.Equal(got, want) {
			t.Fatalf("%s: seam % x, want % x", tr.Name, got, want)
		}
	}
}

// TestBGMMatchesThbgm checks each embedded track lines up with thbgm.dat
// sample for sample: a shifted encode would loop at the wrong place. Skips
// without a TH10 install.
func TestBGMMatchesThbgm(t *testing.T) {
	dir, err := datfs.Locate()
	if err != nil {
		t.Skip(err)
	}
	f, err := os.Open(filepath.Join(dir, "thbgm.dat"))
	if err != nil {
		t.Skip(err)
	}
	defer f.Close()
	tracks, err := BGMTracks()
	if err != nil {
		t.Fatal(err)
	}
	const window, maxLag = 8192, 64 // frames
	for _, tr := range tracks {
		pos := (tr.Intro + tr.Size) / 2 / pcmBytesPerFrame * pcmBytesPerFrame
		got := make([]byte, window*pcmBytesPerFrame)
		readAt(t, decodeBGM(t, tr), got, pos)
		want := make([]byte, (window+2*maxLag)*pcmBytesPerFrame)
		if _, err := tr.PCM(f).ReadAt(want, pos-maxLag*pcmBytesPerFrame); err != nil {
			t.Fatal(err)
		}
		bestLag, bestErr, energy := 0, math.Inf(1), 0.0
		for lag := -maxLag; lag <= maxLag; lag++ {
			var e float64
			for i := 0; i < window*2; i++ {
				a := float64(int16(binary.LittleEndian.Uint16(got[i*2:])))
				b := float64(int16(binary.LittleEndian.Uint16(want[(i+(maxLag+lag)*2)*2:])))
				e += (a - b) * (a - b)
				if lag == 0 {
					energy += b * b
				}
			}
			if e < bestErr {
				bestLag, bestErr = lag, e
			}
		}
		snr := 10 * math.Log10(energy/bestErr)
		t.Logf("%s: lag %d, SNR %.1f dB", tr.Name, bestLag, snr)
		if bestLag != 0 || snr < 10 {
			t.Errorf("%s: best match at lag %d frames (SNR %.1f dB), want lag 0", tr.Name, bestLag, snr)
		}
	}
}

func decodeBGM(t *testing.T, tr datfs.BGMTrack) *vorbis.Stream {
	t.Helper()
	data, err := gamedata.ReadFile(datfs.BGMPath(tr.Name))
	if err != nil {
		t.Fatal(err)
	}
	s, err := vorbis.DecodeWithoutResampling(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("%s: %v", tr.Name, err)
	}
	return s
}

func readAt(t *testing.T, s io.ReadSeeker, p []byte, off int64) {
	t.Helper()
	if _, err := s.Seek(off, io.SeekStart); err != nil {
		t.Fatal(err)
	}
	if _, err := io.ReadFull(s, p); err != nil {
		t.Fatal(err)
	}
}

// TestPlayBGMLoop 播放指定 BGM 来试听循环点是否正确。
//
// 用法（播放第 1 首，即 index 0）：
//
//	TH10_TEST_AUDIO=1 go test -v -run TestPlayBGMLoop -timeout 0 -args 0
func TestPlayBGMLoop(t *testing.T) {
	if os.Getenv("TH10_TEST_AUDIO") != "1" {
		t.Skip("set TH10_TEST_AUDIO=1 to run the interactive BGM listening test")
	}
	tracks, err := BGMTracks()
	if err != nil {
		t.Fatal(err)
	}
	idx := 0
	for i, a := range os.Args {
		if a == "-args" && i+1 < len(os.Args) {
			if n, err := strconv.Atoi(os.Args[i+1]); err == nil {
				idx = n
			}
			break
		}
	}
	if idx < 0 || idx >= len(tracks) {
		t.Fatalf("index %d 超出范围 [0, %d)", idx, len(tracks))
	}
	tr := tracks[idx]
	fmt.Printf("播放: BGM %02d %s (%s)\n", idx+1, BGMNames[idx], tr.Name)

	bytesPerSec := float64(tr.SampleRate * pcmBytesPerFrame)
	introSec := float64(tr.Intro) / bytesPerSec
	loopSec := float64(tr.Size-tr.Intro) / bytesPerSec
	fmt.Printf("Intro: %.2fs, Loop: %.2fs\n", introSec, loopSec)

	playDuration := time.Duration((introSec+loopSec)*1000)*time.Millisecond + 5*time.Second
	fmt.Printf("将播放 %.0f 秒（完整一遍 + loop 后 5 秒）...\n", playDuration.Seconds())

	ctx := audio.NewContext(tr.SampleRate)
	loop, err := NewBGMStream(idx, tr.SampleRate)
	if err != nil {
		t.Fatal(err)
	}
	p, err := ctx.NewPlayer(loop)
	if err != nil {
		t.Fatal(err)
	}
	p.Play()
	defer p.Close()

	start := time.Now()
	for range time.NewTicker(1 * time.Second).C {
		elapsed := time.Since(start)
		fmt.Printf("\r  已播放 %v / %v", elapsed.Round(time.Second), playDuration.Round(time.Second))
		if elapsed >= playDuration {
			fmt.Println("\n播放完毕。")
			return
		}
	}
}
