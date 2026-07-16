package audio

import (
	"fmt"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/hajimehoshi/ebiten/v2/audio"
)

// TestPlayBGMLoop 播放指定 BGM 来试听循环点是否正确。
//
// 用法（播放第 1 首，即 index 0）：
//
//	go test -v -run TestPlayBGMLoop -timeout 0 -args 0
func TestPlayBGMLoop(t *testing.T) {
	return
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
	filePath := fmt.Sprintf("../assets/wav/bgm/%s", bgmOpusFiles[idx])
	data, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("读取文件失败: %v", err)
	}
	pcm, totalBytes, err := newOggOpusStream(data)
	if err != nil {
		t.Fatalf("解码 Opus 失败: %v", err)
	}

	fmt.Printf("播放: BGM %02d\n", idx+1)
	introBytes := track.IntroBytes(bgmPlaybackSampleRate)
	loopBytes := track.LoopLengthBytes(bgmPlaybackSampleRate, totalBytes)
	fmt.Printf("IntroBytes: %d (0x%X)\n", introBytes, introBytes)
	fmt.Printf("LoopBytes:  %d (0x%X)\n", loopBytes, loopBytes)

	bytesPerSec := int64(bgmPlaybackSampleRate * pcmBytesPerFrame)
	introSec := float64(introBytes) / float64(bytesPerSec)
	loopSec := float64(loopBytes) / float64(bytesPerSec)
	totalSec := introSec + loopSec
	fmt.Printf("Intro: %.2fs, Loop: %.2fs, Total: %.2fs\n", introSec, loopSec, totalSec)

	playDuration := time.Duration(totalSec*1000)*time.Millisecond + 5*time.Second
	fmt.Printf("将播放 %.0f 秒（完整一遍 + loop 后 5 秒）...\n", playDuration.Seconds())

	ctx := audio.NewContext(bgmPlaybackSampleRate)
	loop := audio.NewInfiniteLoopWithIntro(pcm, introBytes, loopBytes)
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
