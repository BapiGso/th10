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
	X, Y       float64
	HP         int
	MaxHP      int
	HitWidth   float64 // 被弹判定宽
	HitHeight  float64 // 被弹判定高
	BodyWidth  float64 // 敌机本体判定宽，ECL setHitbox 使用
	BodyHeight float64 // 敌机本体判定高，ECL setHitbox 使用
	Active     bool
	Type       Type
	ECLFlags   int // 原作 ECL flagSet/flagClear 的保守记录

	// Damaged 实现 TakeDamage：ECL VM 借此按原作（FUN_0040dc80）
	// 扣血、检查阶段中断槽并决定是否击破。返回值同 TakeDamage。
	Damaged func(e *Enemy, dmg int) bool
}

func New(x, y float64, hp int, t Type) *Enemy {
	return &Enemy{
		X: x, Y: y, HP: hp, MaxHP: hp,
		HitWidth: 24, HitHeight: 24,
		BodyWidth: 24, BodyHeight: 24,
		Active: true, Type: t,
	}
}

// Shootable 报告自机子弹/灵击能否命中：ECL 标志 0x1/0x10 使原作整个跳过
// 伤害查询（FUN_0040dc80），子弹直接穿过。
func (e *Enemy) Shootable() bool {
	return e.Active && e.ECLFlags&0x11 == 0
}

// TakeDamage 受到伤害，返回是否击破。伤害由 ECL VM 安装的 Damaged 处理；
// 未安装时不受伤。
func (e *Enemy) TakeDamage(dmg int) bool {
	return e.Active && e.Damaged != nil && e.Damaged(e, dmg)
}

// HitRect 返回受弹判定矩形参数
func (e *Enemy) HitRect() (x, y, w, h float64) {
	return e.X, e.Y, e.HitWidth, e.HitHeight
}
