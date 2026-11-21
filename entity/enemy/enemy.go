package enemy

// Type 敌人类型
type Type int

const (
	TypeFairy   Type = iota // 杂鱼
	TypeMidBoss             // 中Boss
	TypeBoss                // Boss
)

// Enemy 敌人实体
type Enemy struct {
	X, Y         float64
	HP           int
	MaxHP        int
	HitWidth     float64 // 被弹判定宽
	HitHeight    float64 // 被弹判定高
	BodyWidth    float64 // 敌机本体判定宽，ECL setHitbox 使用
	BodyHeight   float64 // 敌机本体判定高，ECL setHitbox 使用
	Active       bool
	Type         Type
	Age          int
	ECLFlags     int // 原作 ECL flagSet/flagClear 的保守记录
	InvulnFrames int // 旧 Boss 脚本的无敌帧（ECL 敌机的无敌计时在 ECL VM 内）

	// Damaged 不为 nil 时接管 TakeDamage：ECL VM 借此按原作（FUN_0040dc80）
	// 扣血、检查阶段中断槽并决定是否击破。返回值同 TakeDamage。
	Damaged func(e *Enemy, dmg int) bool

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
	Name      string                    // 符卡名称（显示用）
	HP        int                       // 此阶段血量
	TimeLimit int                       // 时限（帧），0=无限
	Bonus     int64                     // 基础奖分
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
		BodyWidth: 24, BodyHeight: 24,
		Active: true, Type: t,
	}
}

// Update 更新敌人
func (e *Enemy) Update() {
	if !e.Active {
		return
	}
	e.Age++
	if e.InvulnFrames > 0 {
		e.InvulnFrames--
	}
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

// Shootable 报告自机子弹/灵击能否命中：ECL 标志 0x1/0x10 使原作整个跳过
// 伤害查询（FUN_0040dc80），子弹直接穿过。
func (e *Enemy) Shootable() bool {
	return e.Active && e.ECLFlags&0x11 == 0
}

// TakeDamage 受到伤害，返回是否击破
func (e *Enemy) TakeDamage(dmg int) bool {
	if !e.Active {
		return false
	}
	if e.Damaged != nil {
		return e.Damaged(e, dmg)
	}
	if e.InvulnFrames > 0 || (e.Boss != nil && e.Boss.Invincible) {
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

// HitRect 返回受弹判定矩形参数
func (e *Enemy) HitRect() (x, y, w, h float64) {
	return e.X, e.Y, e.HitWidth, e.HitHeight
}
