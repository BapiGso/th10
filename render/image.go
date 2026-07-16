// Package render 同时提供图像辅助：从 embed assets 解码 PNG、按瓦片纵向滚动。
//
// 这里集中了原先散落在 scene/stage、scene/title、scene/loading、scene/select、
// scene/dialog 各自重复的 PNG 加载逻辑。
package render

import (
	"bytes"
	"image"
	_ "image/png" // PNG decoder
	"math"
	"sync"
	"th10/assets"

	"github.com/hajimehoshi/ebiten/v2"
)

var imageCache sync.Map // path -> *ebiten.Image （nil 表示已尝试且失败，不重复读盘）

// LoadImage 从 embed assets 加载 PNG 为 *ebiten.Image。
// 失败返回 nil；调用方需要做 nil 检查（一般通过 vector.* 几何 fallback 兜底）。
// 同一路径只解码一次。
func LoadImage(path string) *ebiten.Image {
	if cached, ok := imageCache.Load(path); ok {
		if cached == nil {
			return nil
		}
		return cached.(*ebiten.Image)
	}
	img := decodeImage(path)
	imageCache.Store(path, img)
	return img
}

func decodeImage(path string) *ebiten.Image {
	data, err := assets.Assets.ReadFile(path)
	if err != nil {
		return nil
	}
	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil
	}
	return ebiten.NewImageFromImage(img)
}

// TileBackground 将 img 在 field 上纵向平铺，按 scrollY 滚动。
// img 为 nil 时 no-op；alpha < 1 时以该 alpha 叠加（多层视差用）。
func TileBackground(field *ebiten.Image, img *ebiten.Image, scrollY float64, alpha float32) {
	if img == nil {
		return
	}
	fw := float64(field.Bounds().Dx())
	fh := float64(field.Bounds().Dy())
	tw := float64(img.Bounds().Dx())
	th := float64(img.Bounds().Dy())
	offY := math.Mod(scrollY, th)
	for y := offY - th; y < fh; y += th {
		for x := 0.0; x < fw; x += tw {
			op := &ebiten.DrawImageOptions{}
			op.GeoM.Translate(x, y)
			if alpha < 1 {
				op.ColorScale.ScaleAlpha(alpha)
			}
			field.DrawImage(img, op)
		}
	}
}
