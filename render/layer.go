package render

import "github.com/hajimehoshi/ebiten/v2"

// Layer 渲染层级，按文章第3篇的显示次序排列
// 越小越先绘制（越在底层）
type Layer int

const (
	LayerBackground    Layer = iota // 背景
	LayerItem                       // 道具（常规混合）
	LayerPlayerBullet               // 自机子弹（常规混合）
	LayerPlayer                     // 自机
	LayerEnemy                      // 杂兵 + Boss
	LayerBullet                     // 敌弹（常规混合）
	LayerBulletAdditive             // 敌弹（高光/Additive）
	LayerHitbox                     // 判定点（高光）
	LayerEffect                     // 特效（高光）
	LayerForeground                 // 前景（对话框等）
	LayerUI                         // UI / HUD
	layerCount
)

// Drawable 可绘制对象接口
type Drawable interface {
	Draw(screen *ebiten.Image)
	DrawLayer() Layer
}

// Renderer 分层渲染器，减少混合模式切换
type Renderer struct {
	layers [layerCount][]Drawable
}

func New() *Renderer {
	return &Renderer{}
}

// Add 注册一个可绘制对象到对应层
func (r *Renderer) Add(d Drawable) {
	l := d.DrawLayer()
	r.layers[l] = append(r.layers[l], d)
}

// AddToLayer 手动指定层级注册
func (r *Renderer) AddToLayer(l Layer, d Drawable) {
	r.layers[l] = append(r.layers[l], d)
}

// Flush 按层级顺序绘制所有对象，然后清空
func (r *Renderer) Flush(screen *ebiten.Image) {
	for i := Layer(0); i < layerCount; i++ {
		for _, d := range r.layers[i] {
			d.Draw(screen)
		}
		r.layers[i] = r.layers[i][:0]
	}
}

// BlendForLayer 返回该层应使用的混合模式
func BlendForLayer(l Layer) ebiten.Blend {
	switch l {
	case LayerBulletAdditive, LayerHitbox, LayerEffect:
		return ebiten.BlendLighter // Additive（高光）
	default:
		return ebiten.BlendSourceOver // AlphaBlend（常规）
	}
}
