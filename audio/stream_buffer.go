package audio

import "time"

// SetBGMBufferSize controls decoder pull bursts, not PCM loop points or fade
// timing. Desktop callers keep the zero/default policy; the web host opts in.
func (m *Manager) SetBGMBufferSize(size time.Duration) {
	m.bgmBuffer = max(size, 0)
	if m.bgm != nil {
		m.bgm.SetBufferSize(m.bgmBuffer)
	}
}

func (m *Manager) BGMBufferSize() time.Duration { return m.bgmBuffer }
