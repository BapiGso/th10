package sprite

import (
	"fmt"

	"github.com/hajimehoshi/ebiten/v2"
)

// LoadBossSprites 加载各关 Boss 贴图，返回 stage number → idle frame (64x64)
// 1=Stage1, 2=Stage2, ..., 6=Stage6, 7=Extra
func LoadBossSprites() map[int]*ebiten.Image {
	bosses := make(map[int]*ebiten.Image, 7)

	for stageNum := 1; stageNum <= 7; stageNum++ {
		var path string
		if stageNum <= 6 {
			path = fmt.Sprintf("anm/stgenm/stg%denm.png", stageNum)
		} else {
			path = "anm/stgenm/stg7enm.png"
		}

		sheet := LoadSheet(path)
		if sheet.img.Bounds().Dx() <= 1 {
			// 加载失败（LoadSheet 返回 1x1 占位图），跳过
			continue
		}

		// 取左上角 64x64 作为 boss idle frame
		frame := sheet.Grid(64, 64, 0, 0)
		bosses[stageNum] = frame
	}

	return bosses
}
