package audio

import (
	"bytes"
	"fmt"
	"io"
	"th10/assets"

	"github.com/hajimehoshi/ebiten/v2/audio"
	"github.com/hajimehoshi/ebiten/v2/audio/vorbis"
	"github.com/hajimehoshi/ebiten/v2/audio/wav"
)

// BGMTrack BGM 循环点（基于解码后 PCM 字节偏移，16bit stereo 44100Hz = 4 bytes/sample）
type BGMTrack struct {
	IntroBytes int64 // loop 前奏长度（PCM 字节）
	LoopBytes  int64 // 循环体长度（PCM 字节）
}

// TH10BGM 东方风神录全 18 首 BGM 循环点。路径统一为 wav/bgm_ogg/NN.ogg
var TH10BGM = [18]BGMTrack{
	{IntroBytes: 0x003C3000, LoopBytes: 0x00E2FA00}, // 01 封印されし神々
	{IntroBytes: 0x00297C00, LoopBytes: 0x016E4800}, // 02 人恋し神様
	{IntroBytes: 0x0003B600, LoopBytes: 0x00781000}, // 03 稲田姫様に叱られるから
	{IntroBytes: 0x0009F200, LoopBytes: 0x0146BC00}, // 04 厄神様の通り道
	{IntroBytes: 0x0007CA00, LoopBytes: 0x011CA600}, // 05 運命のダークサイド
	{IntroBytes: 0x00300000, LoopBytes: 0x01EAB600}, // 06 神々が恋した幻想郷
	{IntroBytes: 0x001AA800, LoopBytes: 0x01898500}, // 07 芥川龍之介の河童
	{IntroBytes: 0x001A1000, LoopBytes: 0x02252800}, // 08 フォールオブフォール
	{IntroBytes: 0x00193800, LoopBytes: 0x00E12E00}, // 09 妖怪の山
	{IntroBytes: 0x00082000, LoopBytes: 0x01EC9A00}, // 10 少女が見た日本の原風景
	{IntroBytes: 0x0026A6E8, LoopBytes: 0x01DF1948}, // 11 信仰は儚き人間の為に
	{IntroBytes: 0x00218800, LoopBytes: 0x00B61100}, // 12 御柱の墓場
	{IntroBytes: 0x004D9B00, LoopBytes: 0x01B65000}, // 13 神さびた古戦場
	{IntroBytes: 0x00065100, LoopBytes: 0x021BCB80}, // 14 明日ハレの日、ケの昨日
	{IntroBytes: 0x000D7220, LoopBytes: 0x01D07640}, // 15 ネイティブフェイス
	{IntroBytes: 0x007A2C00, LoopBytes: 0x00EB5200}, // 16 麓の神社
	{IntroBytes: 0x00A70194, LoopBytes: 0x00F04FF4}, // 17 神は恵みの雨を降らす
	{IntroBytes: 0x001CACC8, LoopBytes: 0x0066DCC0}, // 18 プレイヤーズスコア
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
	SEShoot   = "wav/se_gun00.wav"
	SEEnemy   = "wav/se_enep00.wav"
	SEDamage  = "wav/se_damage00.wav"
	SEGraze   = "wav/se_graze.wav"
	SEItem    = "wav/se_item00.wav"
	SEExtend  = "wav/se_extend.wav"
	SEBomb    = "wav/se_ch00.wav"
	SEOk      = "wav/se_ok00.wav"
	SECancel  = "wav/se_cancel00.wav"
	SESelect  = "wav/se_select00.wav"
	SEPause   = "wav/se_pause.wav"
	SEPowerUp = "wav/se_powerup.wav"
	SEDead    = "wav/se_pldead00.wav"
)

// Manager 管理 BGM 和 SE 播放
type Manager struct {
	ctx *audio.Context
	bgm *audio.Player
}

func NewManager(ctx *audio.Context) *Manager {
	return &Manager{ctx: ctx}
}

// PlaySE 播放音效
func (m *Manager) PlaySE(name string) {
	data, err := assets.Assets.ReadFile(name)
	if err != nil {
		return
	}
	stream, err := wav.DecodeWithoutResampling(bytes.NewReader(data))
	if err != nil {
		return
	}
	p, err := m.ctx.NewPlayer(io.NopCloser(stream))
	if err != nil {
		return
	}
	p.Play()
}

// PlayBGM 播放 BGM（OGG Vorbis，带循环）。idx 为 0-based BGM 编号。
func (m *Manager) PlayBGM(idx int) {
	if idx < 0 || idx >= len(TH10BGM) {
		return
	}
	if m.bgm != nil {
		m.bgm.Close()
		m.bgm = nil
	}
	track := &TH10BGM[idx]
	path := fmt.Sprintf("wav/bgm_ogg/%02d.ogg", idx+1)
	data, err := assets.Assets.ReadFile(path)
	if err != nil {
		return
	}
	stream, err := vorbis.DecodeWithoutResampling(bytes.NewReader(data))
	if err != nil {
		return
	}
	loop := audio.NewInfiniteLoopWithIntro(stream, track.IntroBytes, track.LoopBytes)
	p, err := m.ctx.NewPlayer(loop)
	if err != nil {
		return
	}
	m.bgm = p
	p.Play()
}

func (m *Manager) StopBGM() {
	if m.bgm != nil {
		m.bgm.Close()
		m.bgm = nil
	}
}
