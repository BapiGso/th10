// Package render —— 字体加载。
//
// 统一约定：
//   - 颜色用 color.NRGBA（straight alpha），避免 color.RGBA 的 pre-multiplied
//     约定与运行时 alpha 相乘后的非法值（典型现象：淡入瞬间字变蓝）。
//   - HUD/分数/英文走 Bimini Bold（嵌入），LoadHUDFace。
//   - 中日文菜单/对话走操作系统黑体，LoadSystemFace（找不到则退回 Bimini，仅
//     渲染英文部分；汉字会显示成 .notdef 方块，但不会让程序崩溃）。
package render

import (
	"bytes"
	"fmt"
	"image/color"
	"os"
	"runtime"
	"sync"
	"th10/assets"

	"github.com/hajimehoshi/ebiten/v2"
	ebitext "github.com/hajimehoshi/ebiten/v2/text/v2"
)

// 嵌入字体路径（仅 Bimini Bold；DFYuGaSo TTC 已从 embed 移除）
const (
	FontHUD = "font/Bimini Bold.ttf"
)

var (
	ttcSourceCache sync.Map // path -> *ebitext.GoTextFaceSource
	textFaceCache  sync.Map // "path@size" -> ebitext.Face

	systemFontPathOnce sync.Once
	systemFontPath     string // 解析一次缓存
)

// LoadTextFace 读取嵌入的 TTC/TTF 并返回 text/v2 Face。
func LoadTextFace(path string, size float64) ebitext.Face {
	key := fmt.Sprintf("embed:%s@%.2f", path, size)
	if cached, ok := textFaceCache.Load(key); ok {
		return cached.(ebitext.Face)
	}
	src := loadEmbedSource(path)
	if src == nil {
		return nil
	}
	face := &ebitext.GoTextFace{Source: src, Size: size}
	if actual, loaded := textFaceCache.LoadOrStore(key, face); loaded {
		return actual.(ebitext.Face)
	}
	return face
}

// LoadHUDFace 返回 Bimini Bold 字体（HUD/分数/数字/英文专用）。
func LoadHUDFace(size float64) ebitext.Face {
	return LoadTextFace(FontHUD, size)
}

// LoadSystemFace 返回操作系统自带的中日文黑体。失败时回退到 HUD 字体。
// 路径解析只做一次；解析后的 Source 也按路径缓存。
func LoadSystemFace(size float64) ebitext.Face {
	path := resolveSystemFontPath()
	if path == "" {
		return LoadHUDFace(size)
	}
	key := fmt.Sprintf("file:%s@%.2f", path, size)
	if cached, ok := textFaceCache.Load(key); ok {
		return cached.(ebitext.Face)
	}
	src := loadFileSource(path)
	if src == nil {
		return LoadHUDFace(size)
	}
	face := &ebitext.GoTextFace{Source: src, Size: size}
	if actual, loaded := textFaceCache.LoadOrStore(key, face); loaded {
		return actual.(ebitext.Face)
	}
	return face
}

func loadEmbedSource(path string) *ebitext.GoTextFaceSource {
	cacheKey := "embed:" + path
	if cached, ok := ttcSourceCache.Load(cacheKey); ok {
		return cached.(*ebitext.GoTextFaceSource)
	}
	data, err := assets.Assets.ReadFile(path)
	if err != nil {
		return nil
	}
	src := parseFontBytes(data)
	if src == nil {
		return nil
	}
	ttcSourceCache.Store(cacheKey, src)
	return src
}

func loadFileSource(path string) *ebitext.GoTextFaceSource {
	cacheKey := "file:" + path
	if cached, ok := ttcSourceCache.Load(cacheKey); ok {
		return cached.(*ebitext.GoTextFaceSource)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	src := parseFontBytes(data)
	if src == nil {
		return nil
	}
	ttcSourceCache.Store(cacheKey, src)
	return src
}

func parseFontBytes(data []byte) *ebitext.GoTextFaceSource {
	// TTC 优先：取第一个 face；失败则按单 face 解析。
	sources, err := ebitext.NewGoTextFaceSourcesFromCollection(bytes.NewReader(data))
	if err == nil && len(sources) > 0 {
		return sources[0]
	}
	single, err := ebitext.NewGoTextFaceSource(bytes.NewReader(data))
	if err != nil {
		return nil
	}
	return single
}

// resolveSystemFontPath 在常见位置寻找一份 CJK 字体。命中即返回；找不到返回空串。
//
// 顺序（按平台依次尝试）：
//
//	Windows: msyh.ttc(微软雅黑) -> msyh.ttf -> simhei.ttf -> simsun.ttc
//	macOS:   PingFang.ttc -> STHeiti Medium.ttc -> Hiragino Sans GB.ttc
//	Linux:   Noto Sans CJK SC -> WenQuanYi Micro Hei -> Source Han Sans SC
func resolveSystemFontPath() string {
	systemFontPathOnce.Do(func() {
		for _, p := range candidateSystemFontPaths() {
			if _, err := os.Stat(p); err == nil {
				systemFontPath = p
				return
			}
		}
	})
	return systemFontPath
}

func candidateSystemFontPaths() []string {
	switch runtime.GOOS {
	case "windows":
		root := os.Getenv("WINDIR")
		if root == "" {
			root = `C:\Windows`
		}
		return []string{
			root + `\Fonts\msyh.ttc`,
			root + `\Fonts\msyh.ttf`,
			root + `\Fonts\msyhbd.ttc`,
			root + `\Fonts\simhei.ttf`,
			root + `\Fonts\simsun.ttc`,
		}
	case "darwin":
		return []string{
			"/System/Library/Fonts/PingFang.ttc",
			"/System/Library/Fonts/STHeiti Medium.ttc",
			"/System/Library/Fonts/Hiragino Sans GB.ttc",
			"/Library/Fonts/Songti.ttc",
		}
	default:
		// Linux / *BSD：枚举常见位置，没装的话最后回退到 Bimini。
		return []string{
			"/usr/share/fonts/opentype/noto/NotoSansCJK-Regular.ttc",
			"/usr/share/fonts/noto-cjk/NotoSansCJK-Regular.ttc",
			"/usr/share/fonts/google-noto-cjk/NotoSansCJK-Regular.ttc",
			"/usr/share/fonts/truetype/wqy/wqy-microhei.ttc",
			"/usr/share/fonts/wenquanyi/wqy-microhei/wqy-microhei.ttc",
			"/usr/share/fonts/adobe-source-han-sans/SourceHanSansSC-Regular.otf",
		}
	}
}

// MeasureText 返回文本像素宽高。
func MeasureText(face ebitext.Face, s string) (int, int) {
	if face == nil || s == "" {
		return 0, 0
	}
	w, h := ebitext.Measure(s, face, 0)
	return int(w + 0.5), int(h + 0.5)
}

// LineHeightText 返回行高（含 line gap），用于多行排版。
func LineHeightText(face ebitext.Face) int {
	if face == nil {
		return 0
	}
	m := face.Metrics()
	return int(m.HAscent + m.HDescent + m.HLineGap + 0.5)
}

// DrawText 在 (x, y) 处绘制文本，y 是文字顶部坐标（与 CSS 一致）。
func DrawText(screen *ebiten.Image, s string, face ebitext.Face, x, y int, col color.Color) {
	if face == nil || s == "" {
		return
	}
	op := &ebitext.DrawOptions{}
	op.GeoM.Translate(float64(x), float64(y))
	op.ColorScale.ScaleWithColor(col)
	ebitext.Draw(screen, s, face, op)
}

// DrawTextShadow 文本 + 阴影偏移。col/shadow 推荐 color.NRGBA。
func DrawTextShadow(screen *ebiten.Image, s string, face ebitext.Face, x, y int, col, shadow color.Color, shadowDX, shadowDY int) {
	DrawText(screen, s, face, x+shadowDX, y+shadowDY, shadow)
	DrawText(screen, s, face, x, y, col)
}
