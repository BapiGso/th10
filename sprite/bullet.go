package sprite

import "github.com/hajimehoshi/ebiten/v2"

// BulletSprite 子弹 sprite 集合
type BulletSprite struct {
	sheet *Sheet
	// 按弹型和颜色索引的帧
	Rice   []*ebiten.Image // 米弹 (多颜色)
	Small  []*ebiten.Image // 小玉 (多颜色)
	Middle []*ebiten.Image // 中玉 (多颜色)
	Large  []*ebiten.Image // 大玉 (多颜色)
}

// LoadBullets 加载基础弹幕 spritesheet (etama.png)
// etama.png 256x256 布局:
//   Row 0-7: 16x16 小弹区 (16列), 前几行是米弹/针弹
//   Row 8-9: 16x16 小玉区
//   底部: 30x30 中大弹区
func LoadBullets(path string) *BulletSprite {
	sheet := LoadSheet(path)
	bs := &BulletSprite{sheet: sheet}

	// 米弹: 第0行, 16x16, 16色
	for i := 0; i < 16; i++ {
		bs.Rice = append(bs.Rice, sheet.Grid(16, 16, i, 0))
	}

	// 小玉: 第1行, 16x16, 16色
	for i := 0; i < 16; i++ {
		bs.Small = append(bs.Small, sheet.Grid(16, 16, i, 1))
	}

	// 中玉: 第6行起, 约 30x30 区域 (在 y=192 处, 256/30 约 8 列)
	// 实际按 32x32 对齐读取更安全
	for i := 0; i < 8; i++ {
		bs.Middle = append(bs.Middle, sheet.Sub(
			"", i*32, 192, 30, 30,
		))
	}

	// 大玉: 从 etama6.png 加载更合适, 这里先用中玉占位
	bs.Large = bs.Middle

	return bs
}

// Frame 按弹型和颜色索引获取 sprite
func (bs *BulletSprite) Frame(bulletType, colorIdx int) *ebiten.Image {
	var seq []*ebiten.Image
	switch bulletType {
	case 0: // Rice
		seq = bs.Rice
	case 1: // Small
		seq = bs.Small
	case 2: // Middle
		seq = bs.Middle
	case 3: // Large
		seq = bs.Large
	default:
		seq = bs.Small
	}
	if len(seq) == 0 {
		return nil
	}
	return seq[colorIdx%len(seq)]
}
