package stage

import "th10/game"

// Sound 是 Stage 运行时对音频的最小需求。真实实现是 *audio.Manager；无头
// 模拟器（sim 包）把 Context.Audio 留为 nil，逐帧模拟不依赖音频设备。
type Sound interface {
	PlaySEID(id int)
	PlayBGM(idx int)
	StopBGM()
	FadeOutBGM(frames int)
}

// PlaySEID 按原作音效表（0x4749c8）的编号播放音效；Audio 为 nil 时静默。
func (c *Context) PlaySEID(id int) {
	if c.Audio != nil {
		c.Audio.PlaySEID(id)
	}
}

// PlaySEIDAt is FUN_0043dd10: x is in game coordinates. The optional
// interface keeps existing headless sound recorders compatible.
func (c *Context) PlaySEIDAt(id int, x float32) {
	if c.Audio == nil {
		return
	}
	if p, ok := c.Audio.(interface{ PlaySEIDAt(int, float32) }); ok {
		p.PlaySEIDAt(id, x)
	} else {
		c.Audio.PlaySEID(id)
	}
}

// PlayBGM 切换 BGM；Audio 为 nil 时静默（无头模拟）。
func (c *Context) PlayBGM(idx int) {
	if c.State != nil && c.State.RunFlags&game.RunFlagDemo != 0 {
		return
	} // FUN_00417870: title-demo keeps the title track
	if c.Audio != nil {
		c.Audio.PlayBGM(idx)
	}
}

// StopBGM 停止 BGM；Audio 为 nil 时静默（无头模拟）。
func (c *Context) StopBGM() {
	if c.Audio != nil {
		c.Audio.StopBGM()
	}
}

// FadeBGM 用 frames 帧淡出 BGM；Audio 为 nil 时静默（无头模拟）。
func (c *Context) FadeBGM(frames int) {
	if c.Audio != nil {
		c.Audio.FadeOutBGM(frames)
	}
}
