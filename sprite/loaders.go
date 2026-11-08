package sprite

import (
	"sync"

	"github.com/hajimehoshi/ebiten/v2"
)

// Global 全局 sprite 资源，Load() 后可安全使用
var Global *Resources

// Resources 全部 sprite 资源
type Resources struct {
	Bullets *BulletSprite
}

var loadOnce sync.Once

// Load 加载全部 sprite 资源（只执行一次）
func Load() {
	loadOnce.Do(func() {
		Global = &Resources{
			Bullets: LoadBullets("anm/bullet/etama.png"),
		}
	})
}

// ---------- Bullet ----------

// BulletSprite 子弹 sprite 集合
type BulletSprite struct {
	sheet  *Sheet
	Rice   []*ebiten.Image
	Small  []*ebiten.Image
	Middle []*ebiten.Image
	Large  []*ebiten.Image
}

// LoadBullets 加载基础弹幕 spritesheet (etama.png)
// 256x256: Row0=米弹(16x16x16色), Row1=小玉(16x16x16色), y=192起=中玉(32x32x8)
func LoadBullets(path string) *BulletSprite {
	sheet := LoadSheet(path)
	bs := &BulletSprite{sheet: sheet}

	for i := 0; i < 16; i++ {
		bs.Rice = append(bs.Rice, sheet.Grid(16, 16, i, 0))
	}
	for i := 0; i < 16; i++ {
		bs.Small = append(bs.Small, sheet.Grid(16, 16, i, 1))
	}
	for i := 0; i < 8; i++ {
		bs.Middle = append(bs.Middle, sheet.Sub("", i*32, 192, 30, 30))
	}
	// 大玉: 从 etama6.png 加载更合适, 这里先用中玉占位
	bs.Large = bs.Middle
	return bs
}

// Frame 按弹型和颜色索引获取 sprite
func (bs *BulletSprite) Frame(bulletType, colorIdx int) *ebiten.Image {
	var seq []*ebiten.Image
	switch bulletType {
	case 0:
		seq = bs.Rice
	case 1:
		seq = bs.Small
	case 2:
		seq = bs.Middle
	case 3:
		seq = bs.Large
	default:
		seq = bs.Small
	}
	if len(seq) == 0 {
		return nil
	}
	return seq[colorIdx%len(seq)]
}
