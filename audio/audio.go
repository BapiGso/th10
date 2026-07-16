package audio

import (
	"fmt"
	"io"
	"th10/assets"

	"github.com/hajimehoshi/ebiten/v2/audio"
)

const (
	bgmLoopPointSampleRate   = 44100
	bgmPlaybackSampleRate    = 48000
	bgmChannels              = 2
	pcmBytesPerSample        = 2
	pcmBytesPerFrame         = bgmChannels * pcmBytesPerSample
	maxOpusSamplesPerChannel = 5760
	maxActiveSEPlayers       = 64
)

// BGMTrack BGM 循环点（基于原始 44.1kHz PCM 字节偏移）。
type BGMTrack struct {
	IntroBytes44100   int64 // loop 前奏结束位置（PCM 字节）
	LoopEndBytes44100 int64 // loop 终点位置（PCM 字节）
}

// TH10BGM 东方风神录全 18 首 BGM 循环点。
var TH10BGM = [18]BGMTrack{
	{IntroBytes44100: 0x003C3000, LoopEndBytes44100: 0x00E2FA00}, // 01 封印されし神々
	{IntroBytes44100: 0x00297C00, LoopEndBytes44100: 0x016E4800}, // 02 人恋し神様
	{IntroBytes44100: 0x0003B600, LoopEndBytes44100: 0x00781000}, // 03 稲田姫様に叱られるから
	{IntroBytes44100: 0x0009F200, LoopEndBytes44100: 0x0146BC00}, // 04 厄神様の通り道
	{IntroBytes44100: 0x0007CA00, LoopEndBytes44100: 0x011CA600}, // 05 運命のダークサイド
	{IntroBytes44100: 0x00300000, LoopEndBytes44100: 0x01EAB600}, // 06 神々が恋した幻想郷
	{IntroBytes44100: 0x001AA800, LoopEndBytes44100: 0x01898500}, // 07 芥川龍之介の河童
	{IntroBytes44100: 0x001A1000, LoopEndBytes44100: 0x02252800}, // 08 フォールオブフォール
	{IntroBytes44100: 0x00193800, LoopEndBytes44100: 0x00E12E00}, // 09 妖怪の山
	{IntroBytes44100: 0x00082000, LoopEndBytes44100: 0x01EC9A00}, // 10 少女が見た日本の原風景
	{IntroBytes44100: 0x0026A6E8, LoopEndBytes44100: 0x01DF1948}, // 11 信仰は儚き人間の為に
	{IntroBytes44100: 0x00218800, LoopEndBytes44100: 0x00B61100}, // 12 御柱の墓場
	{IntroBytes44100: 0x004D9B00, LoopEndBytes44100: 0x01B65000}, // 13 神さびた古戦場
	{IntroBytes44100: 0x00065100, LoopEndBytes44100: 0x021BCB80}, // 14 明日ハレの日、ケの昨日
	{IntroBytes44100: 0x000D7220, LoopEndBytes44100: 0x01D07640}, // 15 ネイティブフェイス
	{IntroBytes44100: 0x007A2C00, LoopEndBytes44100: 0x00EB5200}, // 16 麓の神社
	{IntroBytes44100: 0x00A70194, LoopEndBytes44100: 0x00F04FF4}, // 17 神は恵みの雨を降らす
	{IntroBytes44100: 0x001CACC8, LoopEndBytes44100: 0x0066DCC0}, // 18 プレイヤーズスコア
}

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

var bgmOpusFiles = [18]string{
	"01. 封印されし神々.opus",
	"02. 人恋し神様Romantic Fall.opus",
	"03. 稲田姫様に叱られるから.opus",
	"04. 厄神様の通り道 - Dark Road.opus",
	"05. 運命のダークサイド.opus",
	"06. 神々が恋した幻想郷.opus",
	"07. 芥川龍之介の河童 - Candid Friend.opus",
	"08. フォールオブフォール - 秋めく滝.opus",
	"09. 妖怪の山 - Mysterious Mountain.opus",
	"10. 少女が見た日本の原風景.opus",
	"11. 信仰は儚き人間の為に.opus",
	"12. 御柱の墓場 - Grave of Being.opus",
	"13. 神さびた古戦場 - Suwa Foughten Field.opus",
	"14. 明日ハレの日,ケの昨日.opus",
	"15. ネイティブフェイス.opus",
	"16. 麓の神社.opus",
	"17. 神は恵みの雨を降らす - Sylphid Dream.opus",
	"18. プレイヤーズスコア.opus",
}

// BGM 编号常量（0-based 索引）
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

// SE 音效名称常量
const (
	SEShoot   = "wav/se/se_gun00.opus"
	SEEnemy   = "wav/se/se_enep00.opus"
	SEDamage  = "wav/se/se_damage00.opus"
	SEGraze   = "wav/se/se_graze.opus"
	SEItem    = "wav/se/se_item00.opus"
	SEExtend  = "wav/se/se_extend.opus"
	SEBomb    = "wav/se/se_ch00.opus"
	SEOk      = "wav/se/se_ok00.opus"
	SECancel  = "wav/se/se_cancel00.opus"
	SESelect  = "wav/se/se_select00.opus"
	SEPause   = "wav/se/se_pause.opus"
	SEPowerUp = "wav/se/se_powerup.opus"
	SEDead    = "wav/se/se_pldead00.opus"
)

// Manager 管理 BGM 和 SE 播放
type Manager struct {
	ctx       *audio.Context
	bgm       *audio.Player
	bgmVolume float64
	seVolume  float64
	seCache   map[string][]byte
	sePlayers []*audio.Player
}

func NewManager(ctx *audio.Context) *Manager {
	return &Manager{ctx: ctx, bgmVolume: 1, seVolume: 1, seCache: map[string][]byte{}}
}

// PlaySE 播放音效
func (m *Manager) PlaySE(name string) {
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
	p.SetVolume(m.seVolume)
	p.Play()
	m.sePlayers = append(m.sePlayers, p)
}

func (m *Manager) loadSEPCM(name string) ([]byte, error) {
	if pcm, ok := m.seCache[name]; ok {
		return pcm, nil
	}
	data, err := assets.Assets.ReadFile(name)
	if err != nil {
		return nil, err
	}
	stream, _, err := newOggOpusStream(data)
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

// Update releases finished short-lived SE players. Call once per game frame.
func (m *Manager) Update() {
	m.cleanupSE()
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

// PlayBGM 播放 BGM（Ogg/Opus，解码为 PCM 后带循环）。idx 为 0-based BGM 编号。
func (m *Manager) PlayBGM(idx int) {
	if idx < 0 || idx >= len(TH10BGM) {
		return
	}
	if m.bgm != nil {
		m.bgm.Close()
		m.bgm = nil
	}
	track := &TH10BGM[idx]
	path := fmt.Sprintf("wav/bgm/%s", bgmOpusFiles[idx])
	data, err := assets.Assets.ReadFile(path)
	if err != nil {
		return
	}
	stream, totalBytes, err := newOggOpusStream(data)
	if err != nil {
		return
	}
	introBytes := track.IntroBytes(bgmPlaybackSampleRate)
	loopBytes := track.LoopLengthBytes(bgmPlaybackSampleRate, totalBytes)
	loop := audio.NewInfiniteLoopWithIntro(stream, introBytes, loopBytes)
	p, err := m.ctx.NewPlayer(loop)
	if err != nil {
		return
	}
	m.bgm = p
	p.SetVolume(m.bgmVolume)
	p.Play()
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

func (t BGMTrack) IntroBytes(sampleRate int) int64 {
	return scalePCMByteOffset(t.IntroBytes44100, bgmLoopPointSampleRate, sampleRate)
}

func (t BGMTrack) LoopLengthBytes(sampleRate int, totalPCMBytes int64) int64 {
	introBytes := t.IntroBytes(sampleRate)
	loopEndBytes := scalePCMByteOffset(t.LoopEndBytes44100, bgmLoopPointSampleRate, sampleRate)
	if loopEndBytes > totalPCMBytes {
		loopEndBytes = totalPCMBytes
	}
	if loopEndBytes <= introBytes {
		if totalPCMBytes > introBytes {
			return totalPCMBytes - introBytes
		}
		return totalPCMBytes
	}
	return loopEndBytes - introBytes
}

func scalePCMByteOffset(offset int64, fromRate, toRate int) int64 {
	frames := offset / pcmBytesPerFrame
	scaledFrames := (frames*int64(toRate) + int64(fromRate/2)) / int64(fromRate)
	return scaledFrames * pcmBytesPerFrame
}
