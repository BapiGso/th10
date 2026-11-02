package sprite

import (
	"image"
	"th10/gamedata"

	"github.com/hajimehoshi/ebiten/v2"
)

// Sheet 一张 spritesheet 及其子图缓存
type Sheet struct {
	img    *ebiten.Image
	frames map[string]*ebiten.Image // 命名帧缓存
}

// LoadSheet 按资源路径加载 spritesheet（路径规则见 gamedata.Image）
func LoadSheet(path string) *Sheet {
	img, err := gamedata.Image(path)
	if err != nil {
		return &Sheet{img: ebiten.NewImage(1, 1), frames: make(map[string]*ebiten.Image)}
	}
	return &Sheet{img: ebiten.NewImageFromImage(img), frames: make(map[string]*ebiten.Image)}
}

// Image 返回整张图
func (s *Sheet) Image() *ebiten.Image { return s.img }

// Sub 切出子图区域（带缓存）
func (s *Sheet) Sub(name string, x, y, w, h int) *ebiten.Image {
	if f, ok := s.frames[name]; ok {
		return f
	}
	sub := s.img.SubImage(image.Rect(x, y, x+w, y+h)).(*ebiten.Image)
	s.frames[name] = sub
	return sub
}

// Grid 从规则网格切出一帧（col 列, row 行, 0-based）
func (s *Sheet) Grid(cellW, cellH, col, row int) *ebiten.Image {
	x := col * cellW
	y := row * cellH
	return s.img.SubImage(image.Rect(x, y, x+cellW, y+cellH)).(*ebiten.Image)
}
