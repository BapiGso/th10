package sprite

import (
	"sync"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/yohamta/ganim8/v2"
)

// Global 全局 sprite 资源，Load() 后可安全使用
var Global *Resources

// Resources 全部 sprite 资源
type Resources struct {
	Player   [2]*PlayerSprite // 0=灵梦, 1=魔理沙
	ReimuAnm *AnmSheet        // original pl00.anm shot/option scripts
	Bullets  *BulletSprite
}

var loadOnce sync.Once

// Load 加载全部 sprite 资源（只执行一次）
func Load() {
	loadOnce.Do(func() {
		Global = &Resources{
			Player: [2]*PlayerSprite{
				LoadPlayer("anm/player/pl00/pl00.png"),
				LoadPlayer("anm/player/pl01/pl01.png"),
			},
			ReimuAnm: LoadAnmSheet("anm/pl00.anm"),
			Bullets:  LoadBullets("anm/bullet/etama.png"),
		}
	})
}

// ---------- Player ----------

// PlayerSprite 玩家角色 sprite 数据（基于 ganim8）
type PlayerSprite struct {
	sheet *Sheet
	img   *ebiten.Image

	Front   *ganim8.Animation // 正面循环
	ToLeft  *ganim8.Animation // 向左转身
	Left    *ganim8.Animation // 左侧身循环
	ToRight *ganim8.Animation // 向右转身
	Right   *ganim8.Animation // 右侧身循环
}

const (
	playerCellW = 32
	playerCellH = 48
	playerCols  = 8
)

// LoadPlayer 加载玩家 spritesheet (pl00=灵梦, pl01=魔理沙)
// 布局: 8列x3行, 32x48 单元格。Row0=正面, Row1=左, Row2=右。
func LoadPlayer(path string) *PlayerSprite {
	sheet := LoadSheet(path)
	img := sheet.img
	bounds := img.Bounds()
	gridW := bounds.Dx()
	if gridW < playerCellW {
		gridW = playerCellW * playerCols
	}
	gridH := bounds.Dy()
	if gridH < playerCellH {
		gridH = playerCellH * 3
	}

	g := ganim8.NewGrid(playerCellW, playerCellH, gridW, gridH)
	frontD := 100 * time.Millisecond
	transD := 70 * time.Millisecond

	return &PlayerSprite{
		sheet:   sheet,
		img:     img,
		Front:   ganim8.New(img, g.Frames("1-4", 1), frontD),
		ToLeft:  ganim8.New(img, g.Frames("1-4", 2), transD, ganim8.PauseAtEnd),
		Left:    ganim8.New(img, g.Frames("5-8", 2), frontD),
		ToRight: ganim8.New(img, g.Frames("1-4", 3), transD, ganim8.PauseAtEnd),
		Right:   ganim8.New(img, g.Frames("5-8", 3), frontD),
	}
}

// AnimKind 玩家动画段。
type AnimKind int

const (
	AnimFront AnimKind = iota
	AnimToLeft
	AnimLeft
	AnimToRight
	AnimRight
)

// Anim 取对应方向/姿态的 ganim8 动画指针。
func (ps *PlayerSprite) Anim(kind AnimKind) *ganim8.Animation {
	switch kind {
	case AnimToLeft:
		return ps.ToLeft
	case AnimLeft:
		return ps.Left
	case AnimToRight:
		return ps.ToRight
	case AnimRight:
		return ps.Right
	default:
		return ps.Front
	}
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
