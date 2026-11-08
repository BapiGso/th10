package audio

import "math"

// NativeVolumeDB ports FUN_0044d4e0 (BGM, power 2) and FUN_0043ddf0's
// 0x43e2da..0x43e31c (SE, power 3). Percent is not a linear PCM gain.
func NativeVolumeDB(percent uint8, base int32, se bool) int32 {
	if percent == 0 {
		return -10000
	}
	v := float32(float32(percent) * math.Float32frombits(0x3c23d70a)) // PE 0x470b00: .01
	v = math.Float32frombits(0x3f800000) - v
	p := float32(v * v)
	if se {
		p = float32(p * v)
	}
	p = math.Float32frombits(0x3f800000) - p
	p = float32(float64(p) * float64(base+5000)) // native FIMUL uses the integer operand directly
	return int32(p) - 5000
}

func nativeGain(percent uint8, base int32, se bool) float64 {
	return math.Pow(10, float64(NativeVolumeDB(percent, base, se))/2000)
}

// SetNativeVolumes is Option's FUN_0042e5a0 -> SetVol command. The cfg
// retains percentages; BGM/SE use their native squared/cubed dB curves.
func (m *Manager) SetNativeVolumes(bgm, se uint8) {
	bgm, se = min(bgm, 100), min(se, 100)
	m.SetBGMVolume(nativeGain(bgm, 0, false))
	m.SetSEVolume(1)
	m.nativeVolume, m.nativeBGM, m.nativeSE = true, bgm, se
}

// PauseBGM/ResumeBGM are FUN_0043e460's Pause (6) / UnPause (7), without
// rewinding the player or destroying the loop stream.
func (m *Manager) PauseBGM() {
	if m.bgm != nil {
		m.bgm.Pause()
	}
	m.bgmPaused = true
}
func (m *Manager) ResumeBGM() {
	if m.bgm != nil {
		m.bgm.Play()
	}
	m.bgmPaused = false
}
