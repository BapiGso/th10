package sprite

import "github.com/hajimehoshi/ebiten/v2"

// PlayerSprite 玩家角色 sprite 数据
type PlayerSprite struct {
	sheet *Sheet
	// 动画帧序列
	Front []*ebiten.Image // 正面 (4帧循环)
	ToLeft  []*ebiten.Image // 转身到左 (过渡帧)
	Left  []*ebiten.Image // 左侧身 (4帧循环)
	ToRight []*ebiten.Image // 转身到右 (过渡帧)
	Right []*ebiten.Image // 右侧身 (4帧循环)
}

const (
	playerCellW = 32
	playerCellH = 48
	playerCols  = 8
)

// LoadPlayer 加载玩家 spritesheet
// pl00.png = 灵梦, pl01.png = 魔理沙
// 布局: 8列x3行, 32x48 单元格
// Row 0: 正面 8 帧 (取前4帧循环)
// Row 1: 左偏航 8 帧 (前4过渡 + 后4循环)
// Row 2: 右偏航 8 帧 (前4过渡 + 后4循环)
func LoadPlayer(path string) *PlayerSprite {
	sheet := LoadSheet(path)
	ps := &PlayerSprite{sheet: sheet}

	allRow0 := sheet.GridFrames(playerCellW, playerCellH, playerCols, 0, 8)
	allRow1 := sheet.GridFrames(playerCellW, playerCellH, playerCols, 1, 8)
	allRow2 := sheet.GridFrames(playerCellW, playerCellH, playerCols, 2, 8)

	// 正面: 前4帧
	ps.Front = allRow0[:4]
	// 左偏: 前4帧过渡, 后4帧循环
	ps.ToLeft = allRow1[:4]
	ps.Left = allRow1[4:8]
	// 右偏: 前4帧过渡, 后4帧循环
	ps.ToRight = allRow2[:4]
	ps.Right = allRow2[4:8]

	return ps
}

// Frame 根据动画状态和帧索引返回对应 sprite
// state: 0=正面, 1=转左, 2=左侧, 3=转右, 4=右侧
func (ps *PlayerSprite) Frame(state, frame int) *ebiten.Image {
	var seq []*ebiten.Image
	switch state {
	case 0:
		seq = ps.Front
	case 1:
		seq = ps.ToLeft
	case 2:
		seq = ps.Left
	case 3:
		seq = ps.ToRight
	case 4:
		seq = ps.Right
	default:
		seq = ps.Front
	}
	if len(seq) == 0 {
		return nil
	}
	return seq[frame%len(seq)]
}
