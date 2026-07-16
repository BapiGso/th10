package item

import (
	"math"
	"th10/collision"
)

// ItemType 道具类型
type ItemType int

const (
	PowerItem    ItemType = iota // 红·火力
	PointItem                    // 蓝·点符
	BigPower                     // 大火力
	LifeFragment                 // 生命碎片
	BombFragment                 // 符卡碎片
	FullPower                    // 最大火力
)

const (
	collectRadius = 24.0  // 拾取判定半径
	magnetRadius  = 80.0  // 集中状态吸附半径
	autoCollectY  = 128.0 // 自动回收线
	fallSpeed     = 2.0
	magnetSpeed   = 8.0
)

// Item 道具实体
type Item struct {
	X, Y   float64
	VX, VY float64
	Type   ItemType
	Active bool
	Age    int
}

// Pool 道具对象池
type Pool struct {
	items []Item
}

func NewPool(capacity int) *Pool {
	return &Pool{items: make([]Item, capacity)}
}

// Spawn 生成道具
func (p *Pool) Spawn(x, y float64, t ItemType) {
	for i := range p.items {
		if !p.items[i].Active {
			p.items[i] = Item{
				X: x, Y: y, VY: -3,
				Type: t, Active: true,
			}
			return
		}
	}
}

// Update 更新所有道具
func (p *Pool) Update(playerX, playerY float64, focused bool, fieldBottom float64) {
	for i := range p.items {
		it := &p.items[i]
		if !it.Active {
			continue
		}
		it.Age++

		// 上抛减速 → 下落
		if it.VY < fallSpeed {
			it.VY += 0.08
		}

		// 集中时吸附 / 回收线自动吸附
		dx := playerX - it.X
		dy := playerY - it.Y
		dist := math.Sqrt(dx*dx + dy*dy)
		if (focused && dist < magnetRadius) || playerY < autoCollectY {
			if dist > 1 {
				it.X += dx / dist * magnetSpeed
				it.Y += dy / dist * magnetSpeed
			}
		} else {
			it.Y += it.VY
		}

		// 出界销毁
		if it.Y > fieldBottom+32 {
			it.Active = false
		}
	}
}

// Collect 检测拾取，返回被拾取的道具类型列表
func (p *Pool) Collect(px, py float64) []ItemType {
	var collected []ItemType
	for i := range p.items {
		it := &p.items[i]
		if !it.Active {
			continue
		}
		if collision.CircleHit(it.X, it.Y, collectRadius, px, py, 8) {
			collected = append(collected, it.Type)
			it.Active = false
		}
	}
	return collected
}

// CollectCallback 零分配版拾取检测，拾取时回调
func (p *Pool) CollectCallback(px, py float64, fn func(ItemType)) {
	for i := range p.items {
		it := &p.items[i]
		if !it.Active {
			continue
		}
		if collision.CircleHit(it.X, it.Y, collectRadius, px, py, 8) {
			fn(it.Type)
			it.Active = false
		}
	}
}

// Each 遍历活跃道具
func (p *Pool) Each(fn func(it *Item)) {
	for i := range p.items {
		if p.items[i].Active {
			fn(&p.items[i])
		}
	}
}
