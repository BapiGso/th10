package input

import "github.com/hajimehoshi/ebiten/v2"

// Key 抽象游戏按键
type Key int

const (
	KeyUp Key = iota
	KeyDown
	KeyLeft
	KeyRight
	KeyShot   // Z
	KeyBomb   // X
	KeyFocus  // Shift
	KeyPause  // Esc
	KeyOk     // Z / Enter
	KeyCancel // X / Esc
	keyCount
)

// 键盘到游戏按键的默认映射
var defaultKeyMap = map[Key][]ebiten.Key{
	KeyUp:     {ebiten.KeyArrowUp},
	KeyDown:   {ebiten.KeyArrowDown},
	KeyLeft:   {ebiten.KeyArrowLeft},
	KeyRight:  {ebiten.KeyArrowRight},
	KeyShot:   {ebiten.KeyZ},
	KeyBomb:   {ebiten.KeyX},
	KeyFocus:  {ebiten.KeyShiftLeft, ebiten.KeyShiftRight},
	KeyPause:  {ebiten.KeyEscape},
	KeyOk:     {ebiten.KeyZ, ebiten.KeyEnter},
	KeyCancel: {ebiten.KeyX, ebiten.KeyEscape},
}

// State 每帧输入状态快照，用于 Update 阶段读取
type State struct {
	pressed [keyCount]bool // 当前帧是否按住
	prev    [keyCount]bool // 上一帧是否按住
	hold    [keyCount]int  // 连续按住帧数
}

// 全局单例
var Global State

// Update 在 Game.Update 最开头调用，采集本帧输入
func (s *State) Update() {
	copy(s.prev[:], s.pressed[:])
	for k := Key(0); k < keyCount; k++ {
		s.pressed[k] = false
		for _, ek := range defaultKeyMap[k] {
			if ebiten.IsKeyPressed(ek) {
				s.pressed[k] = true
				break
			}
		}
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

// Snapshot 返回当前按键位掩码快照，用于 Replay 记录
func (s *State) Snapshot() uint16 {
	var bits uint16
	for k := Key(0); k < keyCount; k++ {
		if s.pressed[k] {
			bits |= 1 << k
		}
	}
	return bits
}

// Restore 从快照恢复按键状态，用于 Replay 回放
func (s *State) Restore(bits uint16) {
	copy(s.prev[:], s.pressed[:])
	for k := Key(0); k < keyCount; k++ {
		s.pressed[k] = bits&(1<<k) != 0
		if s.pressed[k] {
			s.hold[k]++
		} else {
			s.hold[k] = 0
		}
	}
}
