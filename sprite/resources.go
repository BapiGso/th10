package sprite

import (
	"sync"

	"github.com/hajimehoshi/ebiten/v2"
)

// Global 全局 sprite 资源，Load() 后可安全使用
var Global *Resources

// Resources 全部 sprite 资源
type Resources struct {
	Player  [2]*PlayerSprite // 0=灵梦, 1=魔理沙
	Bullets *BulletSprite
	Enemies *EnemySprite
	Bosses  map[int]*ebiten.Image // stage number (1-7) → boss idle frame
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
			Bullets: LoadBullets("anm/bullet/etama.png"),
			Enemies: LoadEnemies("anm/enemy/enemy.png"),
			Bosses:  LoadBossSprites(),
		}
	})
}
