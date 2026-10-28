// Package audio plays the original sound, all embedded (see package
// gamedata): SE are th10.dat's WAV files as is, BGM is thbgm.dat's PCM
// encoded to Ogg Vorbis sample for sample and looped at thbgm.fmt's points.
package audio

import (
	"bytes"
	"fmt"
	"io"
	"sync"

	"th10/datfs"
	"th10/gamedata"

	"github.com/hajimehoshi/ebiten/v2/audio"
	"github.com/hajimehoshi/ebiten/v2/audio/vorbis"
	"github.com/hajimehoshi/ebiten/v2/audio/wav"
)

const (
	pcmBytesPerFrame   = 4 // ebiten streams are 16-bit stereo
	maxActiveSEPlayers = 64
)

// BGMNames 是音乐室顺序（即 thbgm.fmt 记录顺序）的曲名。
var BGMNames = [18]string{
	"封印されし神々",
	"人恋し神様 〜 Romantic Fall",
	"稲田姫様に叱られるから",
	"厄神様の通り道 〜 Dark Road",
	"運命のダークサイド",
	"神々が恋した幻想郷",
	"芥川龍之介の河童 〜 Candid Friend",
	"フォールオブフォール 〜 秋めく滝",
	"妖怪の山 〜 Mysterious Mountain",
	"少女が見た日本の原風景",
	"信仰は儚き人間の為に",
	"御柱の墓場 〜 Grave of Being",
	"神さびた古戦場 〜 Suwa Foughten Field",
	"明日ハレの日、ケの昨日",
	"ネイティブフェイス",
	"麓の神社",
	"神は恵みの雨を降らす",
	"プレイヤーズスコア",
}

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
	BGMEnding     = 15 // 16 麓の神社（Ending）
	BGMStaff      = 16 // 17 神は恵みの雨を降らす 〜 Sylphid Dream（Staff Roll）
	BGMResult     = 17 // 18 プレイヤーズスコア（Result）
)

// SE 音效名称常量（th10.dat 内的文件名）
const (
	SEShoot   = "se_gun00.wav"
	SEEnemy   = "se_enep00.wav"
	SEDamage  = "se_damage00.wav"
	SEGraze   = "se_graze.wav"
	SEItem    = "se_item00.wav"
	SEExtend  = "se_extend.wav"
	SEBomb    = "se_ch00.wav"
	SEOk      = "se_ok00.wav"
	SECancel  = "se_cancel00.wav"
	SESelect  = "se_select00.wav"
	SEPause   = "se_pause.wav"
	SEPowerUp = "se_powerup.wav"
	SEDead    = "se_pldead00.wav"
)

// Manager 管理 BGM 和 SE 播放
type Manager struct {
	ctx       *audio.Context
	bgm       *audio.Player
	bgmVolume float64
	seVolume  float64
	seCache   map[string][]byte
	sePlayers []*audio.Player
	seQueue   []int // the sound ids PlaySEID started this frame
}

func NewManager(ctx *audio.Context) *Manager {
	return &Manager{ctx: ctx, bgmVolume: 1, seVolume: 1, seCache: map[string][]byte{}}
}

// PlaySE 播放音效
func (m *Manager) PlaySE(name string) {
	m.playSE(name, 1)
}

func (m *Manager) playSE(name string, gain float64) {
	m.cleanupSE()
	if m.seVolume <= 0 {
		return
	}
	pcm, err := m.loadSEPCM(name)
	if err != nil {
		return
	}
	m.trimSEPlayers()
	p := m.ctx.NewPlayerFromBytes(pcm)
	p.SetVolume(m.seVolume * gain)
	p.Play()
	m.sePlayers = append(m.sePlayers, p)
}

func (m *Manager) loadSEPCM(name string) ([]byte, error) {
	if pcm, ok := m.seCache[name]; ok {
		return pcm, nil
	}
	data, err := gamedata.ReadFile(datfs.EmbedPath(name))
	if err != nil {
		return nil, err
	}
	// The SE mix 8/16-bit, mono/stereo and 11.025–44.1 kHz.
	stream, err := wav.DecodeWithSampleRate(m.ctx.SampleRate(), bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	pcm, err := io.ReadAll(stream)
	if err != nil {
		return nil, err
	}
	m.seCache[name] = pcm
	return pcm, nil
}

// Update releases finished short-lived SE players and starts a new frame of
// the PlaySEID queue. Call once per game frame.
func (m *Manager) Update() {
	m.cleanupSE()
	m.seQueue = m.seQueue[:0]
}

func (m *Manager) cleanupSE() {
	n := 0
	for _, p := range m.sePlayers {
		if p == nil {
			continue
		}
		if p.IsPlaying() {
			m.sePlayers[n] = p
			n++
			continue
		}
		_ = p.Close()
	}
	for i := n; i < len(m.sePlayers); i++ {
		m.sePlayers[i] = nil
	}
	m.sePlayers = m.sePlayers[:n]
}

func (m *Manager) trimSEPlayers() {
	if len(m.sePlayers) < maxActiveSEPlayers {
		return
	}
	excess := len(m.sePlayers) - maxActiveSEPlayers + 1
	for _, p := range m.sePlayers[:excess] {
		if p != nil {
			_ = p.Close()
		}
	}
	copy(m.sePlayers, m.sePlayers[excess:])
	for i := len(m.sePlayers) - excess; i < len(m.sePlayers); i++ {
		m.sePlayers[i] = nil
	}
	m.sePlayers = m.sePlayers[:len(m.sePlayers)-excess]
}

// PlayBGM 播放 BGM：[0, Size) 先播一遍，之后循环 [Intro, Size)（thbgm.fmt）。
// idx 为 0-based BGM 编号。
func (m *Manager) PlayBGM(idx int) {
	m.StopBGM()
	loop, err := NewBGMStream(idx, m.ctx.SampleRate())
	if err != nil {
		return
	}
	p, err := m.ctx.NewPlayer(loop)
	if err != nil {
		return
	}
	m.bgm = p
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
		m.bgm.Close()
		m.bgm = nil
	}
}

func (m *Manager) SetBGMVolume(v float64) {
	if v < 0 {
		v = 0
	}
	if v > 1 {
		v = 1
	}
	m.bgmVolume = v
	if m.bgm != nil {
		m.bgm.SetVolume(v)
	}
}

func (m *Manager) SetSEVolume(v float64) {
	if v < 0 {
		v = 0
	}
	if v > 1 {
		v = 1
	}
	m.seVolume = v
	for _, p := range m.sePlayers {
		if p != nil {
			p.SetVolume(v)
		}
	}
}

func scalePCMByteOffset(offset int64, fromRate, toRate int) int64 {
	frames := offset / pcmBytesPerFrame
	scaledFrames := (frames*int64(toRate) + int64(fromRate/2)) / int64(fromRate)
	return scaledFrames * pcmBytesPerFrame
}
