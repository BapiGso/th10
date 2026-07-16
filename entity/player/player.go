package player

import (
	"math"
	"th10/entity/playerbullet"
	"th10/game"
	"th10/input"
)

// AnimState 行走图动画状态
type AnimState int

const (
	AnimFront  AnimState = iota // 正面
	AnimToSide                  // 转身过渡
	AnimSide                    // 侧身
)

// State 自机状态
type State int

const (
	StateNormal  State = iota
	StateDead          // 被弹后的死亡状态
	StateRespawn       // 复活无敌状态
	StateBomb          // 使用符卡中
)

// Player 玩家实体
type Player struct {
	X, Y         float64 // 位置
	Speed        float64 // 移动速度（非集中）
	FocusSpeed   float64 // 集中移动速度
	HitboxRadius float64 // 判定圈半径
	GrazeRadius  float64 // 擦弹范围

	// 直接引用游戏状态，消除双源同步
	GS *game.GameState

	State       State
	InvincTimer int // 无敌剩余帧数
	DeathTimer  int // 决死判定计时
	BombTimer   int // 符卡持续计时
	ShootTimer  int // 射击间隔计时

	// 行走图动画
	AnimState AnimState
	AnimFrame int  // 当前动画帧索引
	AnimTimer int  // 动画计时器
	FaceRight bool // 面朝方向

	// 游戏区域边界
	FieldLeft, FieldTop, FieldRight, FieldBottom float64

	// 自机子弹池引用
	Bullets *playerbullet.Pool
}

const (
	shootInterval   = 3   // 射击频率：每3帧1发（文章第15篇）
	invincTime      = 300 // 复活无敌时间：300帧=5秒
	deathBombWindow = 16  // 决死时间：16帧（取中间值）
	bombInvincTime  = 60  // Bomb无敌时间
	bombDuration    = 300 // 魔炮持续时间
	frontFrameTime  = 8   // 正面/侧身动画帧间隔
	transFrameTime  = 6   // 转身过渡动画帧间隔
)

func New(gs *game.GameState, fieldL, fieldT, fieldR, fieldB float64) *Player {
	centerX := (fieldL + fieldR) / 2
	return &Player{
		X: centerX, Y: fieldB - 48,
		Speed: 4.5, FocusSpeed: 2,
		HitboxRadius: 2, GrazeRadius: 16,
		GS:        gs,
		State:     StateNormal,
		FieldLeft: fieldL, FieldTop: fieldT,
		FieldRight: fieldR, FieldBottom: fieldB,
		Bullets: playerbullet.NewPool(256),
	}
}

func (p *Player) Update() {
	in := &input.Global

	switch p.State {
	case StateDead:
		p.DeathTimer++
		if p.DeathTimer <= deathBombWindow && in.JustPressed(input.KeyBomb) && p.GS.Bomb > 0 {
			p.GS.Bomb--
			p.State = StateBomb
			p.BombTimer = bombDuration
			p.InvincTimer = bombDuration + bombInvincTime
			return
		}
		if p.DeathTimer > deathBombWindow {
			p.GS.Life--
			p.GS.Power = int(math.Max(float64(p.GS.Power-64), 0))
			p.GS.Bomb = 3
			p.State = StateRespawn
			p.InvincTimer = invincTime
			p.X = (p.FieldLeft + p.FieldRight) / 2
			p.Y = p.FieldBottom - 48
		}
		return

	case StateRespawn:
		p.InvincTimer--
		if p.InvincTimer <= 0 {
			p.State = StateNormal
		}
		p.move(in)
		p.shoot(in)

	case StateBomb:
		p.BombTimer--
		p.InvincTimer--
		if p.BombTimer <= 0 {
			if p.InvincTimer <= 0 {
				p.State = StateNormal
			} else {
				p.State = StateRespawn
			}
		}
		p.move(in)

	case StateNormal:
		if p.InvincTimer > 0 {
			p.InvincTimer--
		}
		if in.JustPressed(input.KeyBomb) && p.GS.Bomb > 0 {
			p.GS.Bomb--
			p.State = StateBomb
			p.BombTimer = bombDuration
			p.InvincTimer = bombDuration + bombInvincTime
			return
		}
		p.move(in)
		p.shoot(in)
	}

	p.updateAnim(in)
	p.Bullets.Update()
}

func (p *Player) move(in *input.State) {
	speed := p.Speed
	if in.IsPressed(input.KeyFocus) {
		speed = p.FocusSpeed
	}
	dx, dy := 0.0, 0.0
	if in.IsPressed(input.KeyUp) {
		dy -= 1
	}
	if in.IsPressed(input.KeyDown) {
		dy += 1
	}
	if in.IsPressed(input.KeyLeft) {
		dx -= 1
	}
	if in.IsPressed(input.KeyRight) {
		dx += 1
	}
	if dx != 0 && dy != 0 {
		speed *= math.Sqrt2 / 2
	}
	p.X += dx * speed
	p.Y += dy * speed
	p.X = math.Max(p.FieldLeft+8, math.Min(p.FieldRight-8, p.X))
	p.Y = math.Max(p.FieldTop+16, math.Min(p.FieldBottom-16, p.Y))
}

func (p *Player) shoot(in *input.State) {
	if !in.IsPressed(input.KeyShot) {
		p.ShootTimer = 0
		return
	}
	p.ShootTimer++
	if p.ShootTimer%shootInterval != 0 {
		return
	}

	focused := in.IsPressed(input.KeyFocus)
	switch p.GS.Character {
	case game.CharMarisa:
		switch p.GS.ShotType {
		case game.ShotA:
			p.shootMarisaA(focused)
		case game.ShotB:
			p.shootMarisaB(focused)
		default:
			p.shootMarisaC(focused)
		}
	default:
		switch p.GS.ShotType {
		case game.ShotA:
			p.shootReimuA(focused)
		case game.ShotB:
			p.shootReimuB(focused)
		default:
			p.shootReimuC(focused)
		}
	}
}

func (p *Player) powerTier() int {
	tier := p.GS.Power / 100
	if tier < 0 {
		return 0
	}
	if tier > 5 {
		return 5
	}
	return tier
}

func (p *Player) fireShot(dx, dy, vx, vy float64, damage int) {
	p.Bullets.Fire(p.X+dx, p.Y+dy, vx, vy, damage)
}

func (p *Player) shootReimuA(focused bool) {
	tier := p.powerTier()
	side := 1.8
	if focused {
		side = 0.7
	}
	p.fireShot(-8, -16, 0, -12, 8)
	p.fireShot(8, -16, 0, -12, 8)
	if tier >= 1 {
		p.fireShot(-18, -10, -side, -11, 5)
		p.fireShot(18, -10, side, -11, 5)
	}
	if tier >= 3 {
		p.fireShot(-4, -20, -side*0.6, -13, 7)
		p.fireShot(4, -20, side*0.6, -13, 7)
	}
	if tier >= 5 {
		p.fireShot(-24, -8, -side*1.3, -10, 4)
		p.fireShot(24, -8, side*1.3, -10, 4)
	}
}

func (p *Player) shootReimuB(focused bool) {
	tier := p.powerTier()
	spread := 0.8
	if focused {
		spread = 0.2
	}
	p.fireShot(-6, -16, -spread, -13, 10)
	p.fireShot(6, -16, spread, -13, 10)
	if tier >= 1 {
		p.fireShot(-2, -18, 0, -14, 7)
		p.fireShot(2, -18, 0, -14, 7)
	}
	if tier >= 3 {
		p.fireShot(-12, -14, -spread*0.5, -13, 8)
		p.fireShot(12, -14, spread*0.5, -13, 8)
	}
	if tier >= 5 {
		p.fireShot(-1, -22, 0, -15, 12)
		p.fireShot(1, -22, 0, -15, 12)
	}
}

func (p *Player) shootReimuC(focused bool) {
	tier := p.powerTier()
	spread := 2.0
	if focused {
		spread = 0.9
	}
	p.fireShot(-10, -16, -spread, -11.5, 7)
	p.fireShot(0, -18, 0, -13, 9)
	p.fireShot(10, -16, spread, -11.5, 7)
	if tier >= 2 {
		p.fireShot(-20, -12, -spread*1.2, -10.5, 5)
		p.fireShot(20, -12, spread*1.2, -10.5, 5)
	}
	if tier >= 4 {
		p.fireShot(-5, -20, -spread*0.4, -13, 7)
		p.fireShot(5, -20, spread*0.4, -13, 7)
	}
}

func (p *Player) shootMarisaA(focused bool) {
	tier := p.powerTier()
	spread := 0.5
	if !focused {
		spread = 1.0
	}
	p.fireShot(-7, -14, -spread, -11, 12)
	p.fireShot(7, -14, spread, -11, 12)
	if tier >= 1 {
		p.fireShot(-15, -10, -spread*1.4, -10, 8)
		p.fireShot(15, -10, spread*1.4, -10, 8)
	}
	if tier >= 3 {
		p.fireShot(-2, -18, -spread*0.4, -12, 11)
		p.fireShot(2, -18, spread*0.4, -12, 11)
	}
	if tier >= 5 {
		p.fireShot(-22, -8, -spread*1.7, -9.5, 6)
		p.fireShot(22, -8, spread*1.7, -9.5, 6)
	}
}

func (p *Player) shootMarisaB(focused bool) {
	tier := p.powerTier()
	spread := 0.35
	if !focused {
		spread = 0.8
	}
	p.fireShot(-10, -14, -spread, -14.5, 8)
	p.fireShot(-3, -16, -spread*0.3, -15, 8)
	p.fireShot(3, -16, spread*0.3, -15, 8)
	p.fireShot(10, -14, spread, -14.5, 8)
	if tier >= 2 {
		p.fireShot(-16, -10, -spread*0.8, -13.5, 7)
		p.fireShot(16, -10, spread*0.8, -13.5, 7)
	}
	if tier >= 4 {
		p.fireShot(0, -20, 0, -16, 14)
	}
}

func (p *Player) shootMarisaC(focused bool) {
	tier := p.powerTier()
	spread := 2.4
	if focused {
		spread = 1.1
	}
	p.fireShot(-6, -15, -spread*0.7, -12, 8)
	p.fireShot(0, -18, 0, -13.5, 10)
	p.fireShot(6, -15, spread*0.7, -12, 8)
	if tier >= 1 {
		p.fireShot(-18, -11, -spread, -10.5, 6)
		p.fireShot(18, -11, spread, -10.5, 6)
	}
	if tier >= 3 {
		p.fireShot(-10, -20, -spread*0.35, -13, 8)
		p.fireShot(10, -20, spread*0.35, -13, 8)
	}
	if tier >= 5 {
		p.fireShot(-26, -6, -spread*1.1, -9.5, 5)
		p.fireShot(26, -6, spread*1.1, -9.5, 5)
	}
}

func (p *Player) updateAnim(in *input.State) {
	movingLeft := in.IsPressed(input.KeyLeft)
	movingRight := in.IsPressed(input.KeyRight)

	var target AnimState
	switch {
	case movingLeft || movingRight:
		p.FaceRight = movingRight
		target = AnimSide
	default:
		target = AnimFront
	}

	if target == AnimSide && p.AnimState == AnimFront {
		p.AnimState = AnimToSide
		p.AnimFrame = 0
		p.AnimTimer = 0
	} else if target == AnimFront && p.AnimState == AnimSide {
		p.AnimState = AnimToSide
		p.AnimFrame = 0
		p.AnimTimer = 0
	}

	frameTime := frontFrameTime
	if p.AnimState == AnimToSide {
		frameTime = transFrameTime
	}
	p.AnimTimer++
	if p.AnimTimer >= frameTime {
		p.AnimTimer = 0
		p.AnimFrame++
		if p.AnimState == AnimToSide && p.AnimFrame >= 3 {
			p.AnimState = target
			p.AnimFrame = 0
		}
	}
}

// Hit 自机被弹
func (p *Player) Hit() {
	if p.InvincTimer > 0 || p.State != StateNormal {
		return
	}
	p.State = StateDead
	p.DeathTimer = 0
}

// IsInvincible 是否处于无敌状态
func (p *Player) IsInvincible() bool {
	return p.InvincTimer > 0 || p.State == StateDead || p.State == StateBomb
}

// AddPower 增加火力
func (p *Player) AddPower(n int) {
	p.GS.Power = int(math.Min(float64(p.GS.Power+n), 500))
}

// BombDuration 返回符卡持续帧数，供 UI（如 Bomb 闪光）计算进度。
func BombDuration() int { return bombDuration }
