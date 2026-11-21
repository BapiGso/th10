package collision

import "math"

// RectHit 矩形碰撞（用于道具拾取等宽松判定）
func RectHit(x1, y1, w1, h1, x2, y2, w2, h2 float64) bool {
	return math.Abs(x1-x2) < (w1+w2)/2 && math.Abs(y1-y2) < (h1+h2)/2
}
