package collision

import "math"

// CircleHit 圆-圆碰撞检测（东方系列标准判定）
// 判定半径参考（文章第15篇）：自机2，米弹1，小玉2，中玉6，大玉10
func CircleHit(x1, y1, r1, x2, y2, r2 float64) bool {
	dx := x1 - x2
	dy := y1 - y2
	dr := r1 + r2
	return dx*dx+dy*dy < dr*dr
}

// RectHit 矩形碰撞（用于道具拾取等宽松判定）
func RectHit(x1, y1, w1, h1, x2, y2, w2, h2 float64) bool {
	return math.Abs(x1-x2) < (w1+w2)/2 && math.Abs(y1-y2) < (h1+h2)/2
}

// OrthoCircleHit 正交圆判定（文章第12.5篇）
// 东方原作采用的判定方式：先做正交矩形粗判定，再做圆形精判定
// ax,ay 为 A 中心，aw,ah 为 A 的矩形半宽半高，ar 为 A 的圆半径
func OrthoCircleHit(ax, ay, aw, ah, ar, bx, by, bw, bh, br float64) bool {
	dx := math.Abs(ax - bx)
	dy := math.Abs(ay - by)
	// 粗判定：正交矩形
	if dx > aw+bw || dy > ah+bh {
		return false
	}
	// 精判定：圆形
	return CircleHit(ax, ay, ar, bx, by, br)
}

// InBounds 检测坐标是否在游戏区域内（含余量）
func InBounds(x, y, margin, left, top, right, bottom float64) bool {
	return x >= left-margin && x <= right+margin &&
		y >= top-margin && y <= bottom+margin
}
