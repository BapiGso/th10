package enemy

import (
	"th10/collision"
	"th10/render"

	"github.com/hajimehoshi/ebiten/v2"
)

// Type 敌人类型
type Type int

const (
	TypeFairy   Type = iota // 杂鱼
	TypeMidBoss             // 中Boss
	TypeBoss                // Boss
)

// Enemy 敌人实体
type Enemy struct {
	X, Y      float64
	HP        int
	MaxHP     int
	HitWidth  float64 // 被弹判定宽
	HitHeight float64 // 被弹判定高
	Active    bool
	Type      Type
	Age       int
	DropPower int // 击破掉落P点数
	DropPoint int // 击破掉落得点数
	SpriteID  int // Boss 贴图 ID（1-7 对应关卡号，0=无贴图）

	// Boss 专用（杂兵不使用）
	Boss *BossData

	// 移动控制
	path    []PathNode
	pathIdx int
}

// PathNode 路径节点
type PathNode struct {
	X, Y   float64 // 目标位置
	Frames int     // 移动帧数
}

// SpellCard 符卡定义
type SpellCard struct {
	Name      string  // 符卡名称（显示用）
	HP        int     // 此阶段血量
	TimeLimit int     // 时限（帧），0=无限
	Bonus     int64   // 基础奖分
	Update    func(e *Enemy, frame int) // 弹幕逻辑（每帧调用）
}

// BossData Boss 专用数据
type BossData struct {
	Phases       []SpellCard // 阶段列表（非符 + 符卡交替）
	PhaseIdx     int         // 当前阶段索引
	PhaseFrame   int         // 当前阶段已过帧数
	PhaseHP      int         // 当前阶段剩余 HP
	Invincible   bool        // 阶段切换期间无敌
	DialogBefore int         // 在第几个阶段前触发对话（-1=不触发）
	DialogAfter  int         // 在第几个阶段后触发对话（-1=不触发）
}

func New(x, y float64, hp int, t Type) *Enemy {
	return &Enemy{
		X: x, Y: y, HP: hp, MaxHP: hp,
		HitWidth: 24, HitHeight: 24,
		Active: true, Type: t,
		DropPower: 3, DropPoint: 1,
	}
}

// NewBoss 创建 Boss 敌人
func NewBoss(x, y float64, phases []SpellCard) *Enemy {
	totalHP := 0
	for _, p := range phases {
		totalHP += p.HP
	}
	e := &Enemy{
		X: x, Y: y, HP: totalHP, MaxHP: totalHP,
		HitWidth: 48, HitHeight: 48,
		Active: true, Type: TypeBoss,
		DropPower: 30, DropPoint: 20,
		Boss: &BossData{
			Phases:       phases,
			DialogBefore: -1,
			DialogAfter:  -1,
		},
	}
	if len(phases) > 0 {
		e.Boss.PhaseHP = phases[0].HP
		e.HP = phases[0].HP
	}
	return e
}

// SetPath 设置移动路径
func (e *Enemy) SetPath(nodes []PathNode) {
	e.path = nodes
	e.pathIdx = 0
}

// Update 更新敌人
func (e *Enemy) Update() {
	if !e.Active {
		return
	}
	e.Age++
	e.movePath()

	// Boss 阶段逻辑
	if e.Boss != nil {
		e.updateBoss()
	}
}

func (e *Enemy) movePath() {
	if e.pathIdx >= len(e.path) {
		return
	}
	node := &e.path[e.pathIdx]
	if node.Frames <= 0 {
		e.pathIdx++
		return
	}
	e.X += (node.X - e.X) / float64(node.Frames)
	e.Y += (node.Y - e.Y) / float64(node.Frames)
	node.Frames--
	if node.Frames <= 0 {
		e.pathIdx++
	}
}

func (e *Enemy) updateBoss() {
	bd := e.Boss
	if bd.PhaseIdx >= len(bd.Phases) {
		return
	}
	phase := &bd.Phases[bd.PhaseIdx]
	bd.PhaseFrame++

	// 时限到 → 强制切阶段
	if phase.TimeLimit > 0 && bd.PhaseFrame >= phase.TimeLimit {
		e.advancePhase()
		return
	}

	// 执行弹幕脚本
	if !bd.Invincible && phase.Update != nil {
		phase.Update(e, bd.PhaseFrame)
	}
}

// TakeDamage 受到伤害，返回是否击破
func (e *Enemy) TakeDamage(dmg int) bool {
	if !e.Active {
		return false
	}
	if e.Boss != nil && e.Boss.Invincible {
		return false
	}

	e.HP -= dmg

	// Boss: HP 耗尽切阶段
	if e.Boss != nil && e.HP <= 0 {
		e.advancePhase()
		return false // Boss 不会因单阶段 HP=0 而击破
	}

	if e.HP <= 0 {
		e.Active = false
		return true
	}
	return false
}

func (e *Enemy) advancePhase() {
	bd := e.Boss
	bd.PhaseIdx++
	bd.PhaseFrame = 0

	if bd.PhaseIdx >= len(bd.Phases) {
		// 全部阶段结束，Boss 击破
		e.Active = false
		return
	}

	phase := &bd.Phases[bd.PhaseIdx]
	bd.PhaseHP = phase.HP
	e.HP = phase.HP
	// 阶段切换短暂无敌（由 Stage 脚本控制清弹等）
	bd.Invincible = true
}

// EndInvincible Stage 脚本在清弹/对话后调用，解除 Boss 无敌
func (e *Enemy) EndInvincible() {
	if e.Boss != nil {
		e.Boss.Invincible = false
	}
}

// CurrentPhaseName 返回当前阶段名称
func (e *Enemy) CurrentPhaseName() string {
	if e.Boss == nil || e.Boss.PhaseIdx >= len(e.Boss.Phases) {
		return ""
	}
	return e.Boss.Phases[e.Boss.PhaseIdx].Name
}

// PhaseProgress 当前阶段 HP 比例 (0~1)
func (e *Enemy) PhaseProgress() float64 {
	if e.Boss == nil || e.Boss.PhaseHP == 0 {
		return 0
	}
	return float64(e.HP) / float64(e.Boss.PhaseHP)
}

// RemainingPhases 剩余阶段数
func (e *Enemy) RemainingPhases() int {
	if e.Boss == nil {
		return 0
	}
	return len(e.Boss.Phases) - e.Boss.PhaseIdx
}

// HitRect 返回受弹判定矩形参数
func (e *Enemy) HitRect() (x, y, w, h float64) {
	return e.X, e.Y, e.HitWidth, e.HitHeight
}

// InBounds 检查是否在游戏区域内
func (e *Enemy) InBounds(left, top, right, bottom float64) bool {
	return collision.InBounds(e.X, e.Y, 64, left, top, right, bottom)
}

// DrawLayer 渲染层
func (e *Enemy) DrawLayer() render.Layer { return render.LayerEnemy }

// Draw 绘制敌人
func (e *Enemy) Draw(screen *ebiten.Image) {
	if !e.Active {
		return
	}
	// TODO: 根据 Type 绘制对应贴图
}
