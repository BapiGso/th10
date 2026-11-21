package input

import (
	"th10/replay"

	"github.com/hajimehoshi/ebiten/v2"
)

// Key 抽象游戏按键
type Key int

const (
	KeyUp Key = iota
	KeyDown
	KeyLeft
	KeyRight
	KeyShot        // Z
	KeyBomb        // X
	KeyFocus       // Shift
	KeyPause       // Esc
	KeyOk          // Z / Enter
	KeyCancel      // X / Esc
	KeySkip        // Ctrl：对话快进（MSG op 9 允许时）
	KeyEnter       // FUN_0044a5f0: native menu bit 0x1000, separate from Z
	KeyRetry       // R: native menu bit 0x4000
	KeyReturnTitle // Q: native menu bit 0x0200
	KeyStart       // P/Home: native menu bit 0x0800
	KeySpecial     // S: native menu bit 0x0400
	keyCount
)

// 键盘到游戏按键的默认映射
var defaultKeyMap = map[Key][]ebiten.Key{
	KeyUp:          {ebiten.KeyArrowUp, ebiten.KeyNumpad8, ebiten.KeyNumpad7, ebiten.KeyNumpad9},
	KeyDown:        {ebiten.KeyArrowDown, ebiten.KeyNumpad2, ebiten.KeyNumpad1, ebiten.KeyNumpad3},
	KeyLeft:        {ebiten.KeyArrowLeft, ebiten.KeyNumpad4, ebiten.KeyNumpad1, ebiten.KeyNumpad7},
	KeyRight:       {ebiten.KeyArrowRight, ebiten.KeyNumpad6, ebiten.KeyNumpad3, ebiten.KeyNumpad9},
	KeyShot:        {ebiten.KeyZ},
	KeyBomb:        {ebiten.KeyX},
	KeyFocus:       {ebiten.KeyShiftLeft, ebiten.KeyShiftRight},
	KeyPause:       {ebiten.KeyEscape},
	KeyOk:          {ebiten.KeyZ, ebiten.KeyEnter},
	KeyCancel:      {ebiten.KeyX, ebiten.KeyEscape},
	KeySkip:        {ebiten.KeyControlLeft, ebiten.KeyControlRight},
	KeyEnter:       {ebiten.KeyEnter},
	KeyRetry:       {ebiten.KeyR},
	KeyReturnTitle: {ebiten.KeyQ},
	KeyStart:       {ebiten.KeyP, ebiten.KeyHome},
	KeySpecial:     {ebiten.KeyS},
}

// State 每帧输入状态快照，用于 Update 阶段读取
type State struct {
	pressed       [keyCount]bool // 当前帧是否按住
	prev          [keyCount]bool // 上一帧是否按住
	hold          [keyCount]int  // 连续按住帧数
	replayRecord  replay.Record
	fromReplay    bool
	menu          MenuInput // DAT_00474e30: menu input, not the replay's three words
	padBindings   [9]int16
	padConfigured bool
	padButtons    uint32
	practiceLives int
}

// 全局单例
var Global State

// Update 在 Game.Update 最开头调用，采集本帧输入
func (s *State) Update() {
	s.fromReplay = false
	copy(s.prev[:], s.pressed[:])
	for k := Key(0); k < keyCount; k++ {
		s.pressed[k] = false
		for _, ek := range defaultKeyMap[k] {
			if ebiten.IsKeyPressed(ek) {
				s.pressed[k] = true
				break
			}
		}
	}
	s.sampleGamepad()
	s.practiceLives = 0
	for i, key := range [...]ebiten.Key{ebiten.KeyDigit1, ebiten.KeyDigit2, ebiten.KeyDigit3, ebiten.KeyDigit4, ebiten.KeyDigit5, ebiten.KeyDigit6, ebiten.KeyDigit7, ebiten.KeyDigit8, ebiten.KeyDigit9} {
		if ebiten.IsKeyPressed(key) {
			s.practiceLives = i + 1 // FUN_00430ff0, 0x4311a5..0x431315: first digit wins
			break
		}
	}
	raw := menuKeys(s.pressed)
	if ebiten.IsKeyPressed(ebiten.KeyD) {
		raw |= 0x2000
	} // FUN_0044a5f0, 0x44a6a2..0x44a708
	s.menu.Update(raw)
	for k := Key(0); k < keyCount; k++ {
		if s.pressed[k] {
			s.hold[k]++
		} else {
			s.hold[k] = 0
		}
	}
}

// IsPressed 当前帧是否按住
func (s *State) IsPressed(k Key) bool { return s.pressed[k] }

// JustPressed 当前帧刚按下（上一帧没按）
func (s *State) JustPressed(k Key) bool { return s.pressed[k] && !s.prev[k] }

// HoldFrames 连续按住了多少帧
func (s *State) HoldFrames(k Key) int { return s.hold[k] }

// Restore 从快照恢复按键状态，用于 Replay 回放
func (s *State) Restore(bits uint16) {
	s.fromReplay = false
	copy(s.prev[:], s.pressed[:])
	for k := Key(0); k < keyCount; k++ {
		s.pressed[k] = bits&(1<<k) != 0
		if s.pressed[k] {
			s.hold[k]++
		} else {
			s.hold[k] = 0
		}
	}
	s.padButtons, s.practiceLives = 0, 0
	s.menu.Update(menuKeys(s.pressed))
}
