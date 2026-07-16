package dialog

import (
	"sync"
	"th10/render"

	"github.com/hajimehoshi/ebiten/v2"
)

// 风神录的面像图命名规则：face<编号><表情后缀>_u.png（_u=上半身）。
// 表情常见值：no=普通, an=怒, ct=切れ, n2=别样, sp=惊, sw=笑, pr=自信, dp=失落, lo=笑(大), hp=喜.
// 表中 key=表情后缀；找不到时 dialog.getFace 会回退到 "no"。
type expressionMap = map[string]string

func plPath(prefix, expr string) string {
	return "anm/face/" + prefix + "/face_" + prefix + expr + "_u.png"
}

func enemyPath(folder, num, expr string) string {
	if expr == "ct" {
		return "anm/face/" + folder + "/face" + num + expr + ".png"
	}
	return "anm/face/" + folder + "/face" + num + expr + "_u.png"
}

// speakerFaces 给每个说话者列出 (表情→资源路径) 的字典。
//   - 玩家两机的别名都映射到同一组路径
//   - 各 stage Boss / 中 Boss 共用 enemyN 文件夹的资源
//   - 各表情真不存在的 Boss（比如 enemy6 缺 an/sp/sw）由 getFace 自动回退
var speakerFaces = map[string]expressionMap{
	// Player —— 灵梦
	"灵梦":   plExpr("pl00"),
	"博麗霊夢": plExpr("pl00"),
	// Player —— 魔理沙
	"魔理沙":   plExpr("pl01"),
	"霧雨魔理沙": plExpr("pl01"),
	// Stage 1 —— 秋姉妹（共用 enemy1 文件夹）
	"秋静葉": enemyExpr("enemy1", "01"),
	"秋穣子": enemyExpr("enemy1", "01"),
	// Stage 2
	"鍵山雛": enemyExpr("enemy2", "02"),
	// Stage 3
	"河城にとり": enemyExpr("enemy3", "03"),
	// Stage 4
	"射命丸文": enemyExpr("enemy4", "04"),
	"犬走椛":  enemyExpr("enemy4", "04"),
	// Stage 5
	"東風谷早苗": enemyExpr("enemy5", "05"),
	// Stage 6
	"八坂神奈子": enemyExpr("enemy6", "06"),
	// Extra
	"洩矢諏訪子": enemyExpr("enemy7", "07"),
}

// plExpr —— 玩家两机所有可用表情清单（按 assets/anm/face/pl00、pl01 实际有的文件）。
func plExpr(prefix string) expressionMap {
	exprs := []string{"no", "an", "dp", "hp", "n2", "pr", "sp", "sw"}
	m := make(expressionMap, len(exprs))
	for _, e := range exprs {
		m[e] = plPath(prefix, e)
	}
	return m
}

// enemyExpr —— Boss 表情清单。某些 Boss 没有的表情会缺 key，getFace 自动回退。
// 这里给一份"通用最大集"，加载时 LoadImage 找不到文件就返回 nil，再回退一次。
func enemyExpr(folder, num string) expressionMap {
	exprs := []string{"no", "an", "ct", "lo", "n2", "sp", "sw", "pr", "dp", "hp"}
	m := make(expressionMap, len(exprs))
	for _, e := range exprs {
		m[e] = enemyPath(folder, num, e)
	}
	return m
}

var (
	faceImageCache sync.Map // path -> *ebiten.Image
)

// expressionFallbacks defines semantic fallbacks for .msg face ids. The official
// files use ids 0-8, but not every character ships every suffix; prefer a close
// neighbor over dropping straight to "no".
var expressionFallbacks = map[string][]string{
	"an": {"an", "pr", "n2", "no"},
	"dp": {"dp", "lo", "n2", "no"},
	"hp": {"hp", "sw", "pr", "no"},
	"n2": {"n2", "dp", "lo", "no"},
	"pr": {"pr", "sw", "lo", "no"},
	"sp": {"sp", "n2", "dp", "no"},
	"sw": {"sw", "pr", "hp", "no"},
}

// getFace 返回 (speaker, expression) 对应的立绘。表情缺失或图片缺失时按相近表情回退，
// "no" 也找不到则返回 nil（绘制层会自然跳过）。
func getFace(speaker, expr string) *ebiten.Image {
	exprs, ok := speakerFaces[speaker]
	if !ok {
		return nil
	}
	if expr == "" {
		expr = "no"
	}
	chain := expressionFallbacks[expr]
	if len(chain) == 0 {
		chain = []string{expr, "no"}
	}
	seen := map[string]bool{}
	for _, e := range chain {
		if seen[e] {
			continue
		}
		seen[e] = true
		if img := loadFaceByPath(exprs[e]); img != nil {
			return img
		}
	}
	return nil
}

func loadFaceByPath(path string) *ebiten.Image {
	if path == "" {
		return nil
	}
	if v, ok := faceImageCache.Load(path); ok {
		if img, _ := v.(*ebiten.Image); img != nil {
			return img
		}
		return nil
	}
	// render.LoadImage 内部已 memoize；这里再加一层是为了把 nil 结果也缓存住，
	// 避免每帧都对缺失资源做一次 ReadFile。
	img := render.LoadImage(path)
	faceImageCache.Store(path, img)
	return img
}
