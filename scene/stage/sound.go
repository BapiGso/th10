package stage

// Sound 是 Stage 运行时对音频的最小需求。真实实现是 *audio.Manager；无头
// 模拟器（sim 包）把 Context.Audio 留为 nil，逐帧模拟不依赖音频设备。
type Sound interface {
	PlaySE(name string)
	PlayBGM(idx int)
	StopBGM()
}

// PlaySE 播放音效；Audio 为 nil 时静默（无头模拟）。
func (c *Context) PlaySE(name string) {
	if c.Audio != nil {
		c.Audio.PlaySE(name)
	}
}

// PlayBGM 切换 BGM；Audio 为 nil 时静默（无头模拟）。
func (c *Context) PlayBGM(idx int) {
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
