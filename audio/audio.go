// Package audio plays the original sound, all embedded (see package
// gamedata): SE are losslessly packed original WAVs, BGM is thbgm.dat's PCM
// encoded to Ogg Vorbis sample for sample and looped at thbgm.fmt's points.
package audio

import (
	"bytes"
	"fmt"
	"sync"
	"time"

	"th10/datfs"
	"th10/gamedata"

	"github.com/hajimehoshi/ebiten/v2/audio"
	"github.com/hajimehoshi/ebiten/v2/audio/vorbis"
)

const pcmBytesPerFrame = 4 // ebiten streams are 16-bit stereo

// bgmCount is the number of thbgm.fmt records (music-room order).
const bgmCount = 18

// BGM 编号常量（0-based，thbgm.fmt 记录下标）
// 风神录 BGM 顺序: 标题→1面道中→1面Boss→2面道中→2面Boss→...→Extra道中→Extra Boss→ED→Staff→Result
const (
	BGMTitle      = 0  // 01 封印されし神々（标题画面）
	BGMStage1     = 1  // 02 人恋し神様 〜 Romantic Fall（1面道中）
	BGMStage1Boss = 2  // 03 稲田姫様に叱られるから（1面Boss: 秋姉妹）
	BGMStage2     = 3  // 04 厄神様の通り道 〜 Dark Road（2面道中）
	BGMStage2Boss = 4  // 05 運命のダークサイド（2面Boss: 鍵山雛）
	BGMStage3     = 5  // 06 神々が恋した幻想郷（3面道中）
	BGMStage3Boss = 6  // 07 芥川龍之介の河童 〜 Candid Friend（3面Boss: 河城にとり）
	BGMStage4     = 7  // 08 フォールオブフォール 〜 秋めく滝（4面道中）
	BGMStage4Boss = 8  // 09 妖怪の山 〜 Mysterious Mountain（4面Boss: 射命丸文）
	BGMStage5     = 9  // 10 少女が見た日本の原風景（5面道中）
	BGMStage5Boss = 10 // 11 信仰は儚き人間の為に（5面Boss: 東風谷早苗）
	BGMStage6     = 11 // 12 御柱の墓場 〜 Grave of Being（6面道中）
	BGMStage6Boss = 12 // 13 神さびた古戦場 〜 Suwa Foughten Field（6面Boss: 八坂神奈子）
	BGMExtra      = 13 // 14 明日ハレの日、ケの昨日（Extra道中）
	BGMExtraBoss  = 14 // 15 ネイティブフェイス（Extra Boss: 洩矢諏訪子）
	BGMResult     = 17 // 18 プレイヤーズスコア（Result）
)

// Manager 管理 BGM 和 SE 播放
type Manager struct {
	ctx         *audio.Context
	bgm         *audio.Player
	bgmBuffer   time.Duration // optional host source buffer; zero retains Oto's default
	bgmVolume   float64
	fadeLeft    int // FadeOutBGM: frames left, 0 = not fading
	fadeTotal   int
	bgmObserver func(int) // FUN_00420b10/00420b90: music-room unlock ordinal
	seVolume    float64
	seCache     map[string][]byte
	se          seOutput    // one voice per table id; nil without a device
	seQueue     []seRequest // FUN_0043dd10's per-id pan requests for this frame
	// seAux is the unused +6 attribute copied to manager+0x408 (0x43dd8b).
	seAux               [len(seTable)]int16
	nativeVolume        bool
	nativeBGM, nativeSE uint8
	bgmPaused           bool
}

func NewManager(ctx *audio.Context) *Manager {
	m := &Manager{ctx: ctx, bgmVolume: 1, seVolume: 1, seCache: map[string][]byte{}}
	for i := range m.seAux {
		m.seAux[i] = -1 // FUN_004659a0 / FUN_0043e580 initialize the attribute array.
	}
	if ctx != nil {
		_ = m.preloadSE(ctx.SampleRate())
		m.se = newSEOutput(ctx, m.seCache)
	}
	return m
}

// Update flushes FUN_0043dd10's sound queue.
// Call once at the end of the game frame (FUN_00439390, 0x43943b).
func (m *Manager) Update() {
	m.flushSE()
	if m.fadeLeft > 0 && !m.bgmPaused {
		if m.fadeLeft--; m.fadeLeft == 0 {
			m.StopBGM()
		} else if m.bgm != nil {
			gain := m.bgmVolume * float64(m.fadeLeft) / float64(m.fadeTotal)
			if m.nativeVolume {
				gain = nativeGain(m.nativeBGM, int32(m.fadeLeft*5000/m.fadeTotal)-5000, false)
			}
			m.bgm.SetVolume(gain)
		}
	}
}

// PlayBGM 播放 BGM：[0, Size) 先播一遍，之后循环 [Intro, Size)（thbgm.fmt）。
// idx 为 0-based BGM 编号。
func (m *Manager) PlayBGM(idx int) {
	m.playBGM(idx, true)
}

// PlayBGMPreview is Music Room's FUN_00433ef0 at 0x43479b: same stream,
// but unlike FUN_00420b10, auditioning an unseen track does not unlock it.
func (m *Manager) PlayBGMPreview(idx int) {
	m.playBGM(idx, false)
}

// SetBGMObserver connects original playback requests to score ST+0x26.
// It receives the thbgm.fmt/music-room ordinal, not the WAV name number.
func (m *Manager) SetBGMObserver(observer func(int)) { m.bgmObserver = observer }

func (m *Manager) playBGM(idx int, unlock bool) {
	m.StopBGM()
	m.fadeLeft = 0
	m.bgmPaused = false
	if idx < 0 || idx >= bgmCount {
		return
	}
	// The original writes the unlock byte after queuing the request, even
	// when sound is disabled; device initialization is not the gate.
	if unlock && m.bgmObserver != nil {
		m.bgmObserver(idx)
	}
	if m.ctx == nil {
		return
	}
	loop, err := NewBGMStream(idx, m.ctx.SampleRate())
	if err != nil {
		return
	}
	p, err := m.ctx.NewPlayer(loop)
	if err != nil {
		return
	}
	m.bgm = p
	if m.bgmBuffer > 0 {
		p.SetBufferSize(m.bgmBuffer)
	}
	p.SetVolume(m.bgmVolume)
	p.Play()
}

// NewBGMStream returns track idx as an endless 16-bit stereo stream at
// sampleRate, resampled if it differs from the track's 44.1 kHz.
func NewBGMStream(idx, sampleRate int) (*audio.InfiniteLoop, error) {
	tracks, err := BGMTracks()
	if err != nil {
		return nil, err
	}
	if idx < 0 || idx >= len(tracks) {
		return nil, fmt.Errorf("audio: BGM %d out of range [0, %d)", idx, len(tracks))
	}
	t := tracks[idx]
	if t.Channels != 2 || t.BitsPerSample != 16 {
		return nil, fmt.Errorf("audio: %s is not 16-bit stereo", t.Name)
	}
	data, err := gamedata.ReadFile(datfs.BGMPath(t.Name))
	if err != nil {
		return nil, err
	}
	src, err := vorbis.DecodeWithSampleRate(sampleRate, bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("audio: %s: %w", t.Name, err)
	}
	intro := scalePCMByteOffset(t.Intro, t.SampleRate, sampleRate)
	size := scalePCMByteOffset(t.Size, t.SampleRate, sampleRate)
	return audio.NewInfiniteLoopWithIntro(src, intro, size-intro), nil
}

var (
	bgmOnce   sync.Once
	bgmTracks []datfs.BGMTrack
	bgmErr    error
)

// BGMTracks returns the embedded thbgm.fmt, in BGM index order.
func BGMTracks() ([]datfs.BGMTrack, error) {
	bgmOnce.Do(func() {
		var data []byte
		if data, bgmErr = gamedata.ReadFile(datfs.EmbedPath("thbgm.fmt")); bgmErr == nil {
			bgmTracks, bgmErr = datfs.ParseBGMFmt(data)
		}
	})
	return bgmTracks, bgmErr
}

func (m *Manager) StopBGM() {
	if m.bgm != nil {
		m.bgm.PauseAndStopReading()
		m.bgm = nil
	}
}

// FadeOutBGM fades the BGM out linearly over frames game frames, then stops
// it (MSG op 21, FUN_00420c30).
func (m *Manager) FadeOutBGM(frames int) {
	if m.bgm == nil || frames <= 0 {
		return
	}
	m.fadeLeft, m.fadeTotal = frames, frames
}

func (m *Manager) SetBGMVolume(v float64) {
	m.nativeVolume = false // preserve the existing linear-gain API for non-native callers/tests
	if v < 0 {
		v = 0
	}
	if v > 1 {
		v = 1
	}
	m.bgmVolume = v
	if m.bgm != nil && m.fadeLeft == 0 {
		m.bgm.SetVolume(v)
	}
}

func (m *Manager) SetSEVolume(v float64) {
	m.nativeVolume = false
	if v < 0 {
		v = 0
	}
	if v > 1 {
		v = 1
	}
	m.seVolume = v
	if m.se != nil {
		m.se.setMaster(v)
	}
}

func scalePCMByteOffset(offset int64, fromRate, toRate int) int64 {
	frames := offset / pcmBytesPerFrame
	scaledFrames := (frames*int64(toRate) + int64(fromRate/2)) / int64(fromRate)
	return scaledFrames * pcmBytesPerFrame
}
