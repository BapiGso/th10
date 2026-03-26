package sprite

import "github.com/hajimehoshi/ebiten/v2"

// EnemySprite 敌人 sprite 集合
type EnemySprite struct {
	sheet *Sheet
	// 杂兵动画: [种类][帧]
	Fairy [][]*ebiten.Image
	// 大型敌人 (64x64)
	Big []*ebiten.Image
}

// LoadEnemies 加载通用杂兵 spritesheet (enemy.png)
// 512x512, 主区 32x32 网格 (12列), 底部/右侧 64x64
func LoadEnemies(path string) *EnemySprite {
	sheet := LoadSheet(path)
	es := &EnemySprite{sheet: sheet}

	// 杂兵: 前 8 行, 每行 12 帧 = 一种杂兵的完整动画
	// 正面4帧 + 左偏4帧 + 右偏4帧
	for row := 0; row < 8; row++ {
		frames := sheet.GridFrames(32, 32, 12, row, 12)
		es.Fairy = append(es.Fairy, frames)
	}

	// 大型: 底部 64x64 区域 (y=256起, 8列x4行)
	for row := 0; row < 4; row++ {
		for col := 0; col < 4; col++ {
			x := col * 64
			y := 256 + row*64
			es.Big = append(es.Big, sheet.Sub("", x, y, 64, 64))
		}
	}

	return es
}

// FairyFrame 获取杂兵帧 (kind=种类0-7, frame=帧索引)
func (es *EnemySprite) FairyFrame(kind, frame int) *ebiten.Image {
	if kind < 0 || kind >= len(es.Fairy) {
		return nil
	}
	seq := es.Fairy[kind]
	if len(seq) == 0 {
		return nil
	}
	return seq[frame%len(seq)]
}
