package audio

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/hajimehoshi/ebiten/v2/audio"
	"github.com/hajimehoshi/ebiten/v2/audio/vorbis"
)

// TestPlayBGMLoop 播放指定 BGM 来试听循环点是否正确。
//
// 用法（播放第 1 首，即 index 0）：
//
//	go test -v -run TestPlayBGMLoop -timeout 0 -args 0
func TestPlayBGMLoop(t *testing.T) {
	idx := 0
	for i, a := range os.Args {
		if a == "-args" && i+1 < len(os.Args) {
			if n, err := strconv.Atoi(os.Args[i+1]); err == nil {
				idx = n
			}
			break
		}
	}
	if idx < 0 || idx >= len(TH10BGM) {
		t.Fatalf("index %d 超出范围 [0, %d)", idx, len(TH10BGM))
	}

	track := TH10BGM[idx]
	filePath := fmt.Sprintf("../assets/wav/bgm_ogg/%02d.ogg", idx+1)
	data, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("读取文件失败: %v", err)
	}

	fmt.Printf("播放: BGM %02d\n", idx+1)
	fmt.Printf("IntroBytes: %d (0x%X)\n", track.IntroBytes, track.IntroBytes)
	fmt.Printf("LoopBytes:  %d (0x%X)\n", track.LoopBytes, track.LoopBytes)

	bytesPerSec := int64(44100 * 4)
	introSec := float64(track.IntroBytes) / float64(bytesPerSec)
	loopSec := float64(track.LoopBytes) / float64(bytesPerSec)
	totalSec := introSec + loopSec
	fmt.Printf("Intro: %.2fs, Loop: %.2fs, Total: %.2fs\n", introSec, loopSec, totalSec)

	playDuration := time.Duration(totalSec*1000)*time.Millisecond + 5*time.Second
	fmt.Printf("将播放 %.0f 秒（完整一遍 + loop 后 5 秒）...\n", playDuration.Seconds())

	ctx := audio.NewContext(44100)
	stream, err := vorbis.DecodeWithoutResampling(bytes.NewReader(data))
	if err != nil {
		t.Fatal(err)
	}
	loop := audio.NewInfiniteLoopWithIntro(stream, track.IntroBytes, track.LoopBytes)
	p, err := ctx.NewPlayer(io.NopCloser(loop))
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
