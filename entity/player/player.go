package player

import (
	"fmt"
	"math"
	"th10/entity/playerbullet"
	"th10/game"
	"th10/gamedata"
	"th10/input"
	"th10/sht"
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
	X, Y float64 // 屏幕坐标，由 IX/IY 导出（player +0x3c0/+0x3c4 加上场地原点）
	// IX, IY 是以百分之一像素计、相对场地顶边中点的整数位置
	// （player +0x3cc/+0x3d0）。移动只改它们，X/Y 每次按
	// (float)int*0.01f 重算。
	IX, IY int32
	// steps 是每帧步长（百分之一像素）：高速、低速、高速斜向、低速斜向
	// （player +0x3d4/+0x3d8/+0x3dc/+0x3e0）。
	steps        [4]int32
	HitboxRadius float64 // 判定圈半径
	GrazeRadius  float64 // 擦弹范围
	// GameSpeed 返回 DAT_00476f78（ECL 367 写入）；nil 或 0 视为 1。
	GameSpeed func() float32

	// 直接引用游戏状态，消除双源同步
	GS *game.GameState

	State       State
	InvincTimer int // 无敌剩余帧数
	DeathTimer  int // 决死判定计时
	BombTimer   int // 符卡持续计时
	ShootTimer  int // 射击间隔计时
	// NoAttack 由关卡在对话期间置位：射击（FUN_004281d0）和灵击都要求没有
	// MSG，决死灵击不受限。
	NoAttack bool

	// 行走图动画
	AnimState AnimState
	AnimFrame int  // 当前动画帧索引
	AnimTimer int  // 动画计时器
	FaceRight bool // 面朝方向

	// 游戏区域边界
	FieldLeft, FieldTop, FieldRight, FieldBottom float64

	// 自机子弹池引用
	Bullets *playerbullet.Pool
	reimuA  *reimuAShot
}

const (
	shootInterval   = 3   // Legacy shot approximation; SHT firing records not yet executed.
	invincTime      = 300 // Legacy respawn timing; original entry animation is not yet modeled.
	deathBombWindow = 8   // FUN_00425730 state 4: timer 0..7 permits deathbomb
	bombInvincTime  = 60  // Legacy post-bomb approximation.
	bombDuration    = 300 // Placeholder until the original ANM-controlled lifecycle is wired.
	frontFrameTime  = 8   // 正面/侧身动画帧间隔
	transFrameTime  = 6   // 转身过渡动画帧间隔
)

func New(gs *game.GameState, fieldL, fieldT, fieldR, fieldB float64) *Player {
	path := fmt.Sprintf("sht/pl%02d%c.sht", gs.Character, 'a'+gs.ShotType)
	data, err := gamedata.ReadFile(path)
	if err != nil {
		panic(fmt.Sprintf("load player movement %s: %v", path, err))
	}
	movement, err := sht.ParseMovement(data)
	if err != nil {
		panic(fmt.Sprintf("load player movement %s: %v", path, err))
	}
	p := &Player{
		steps: [4]int32{int32(movement.Fast), int32(movement.Slow),
			int32(movement.FastDiagonal), int32(movement.SlowDiagonal)},
		HitboxRadius: 2, GrazeRadius: 16,
		GS:        gs,
		State:     StateNormal,
		FieldLeft: fieldL, FieldTop: fieldT,
		FieldRight: fieldR, FieldBottom: fieldB,
		Bullets: playerbullet.NewPool(256),
	}
	p.SetFixed(0, 40000) // FUN_004247f0: (0, 400.0)
	if gs.Character == game.CharReimu && gs.ShotType == game.ShotA {
		shots, err := sht.Parse(data)
		if err != nil {
			panic(fmt.Sprintf("load Reimu A shots: %v", err))
		}
		if err = shots.ValidateReimuA(); err != nil {
			panic(err)
		}
		p.reimuA = &reimuAShot{file: shots, clock: -1, optionCount: -1}
		p.Bullets = playerbullet.NewPool(128) // original player+0x49c: 128 slots
	}
	return p
}

func (p *Player) Update() {
	p.UpdateInput(&input.Global)
}

// UpdateInput advances one tick from an input snapshot, without polling the OS.
func (p *Player) UpdateInput(in *input.State) {
	if p.reimuA != nil {
		defer p.updateReimuAShot(in)
	}
	switch p.State {
	case StateDead:
		p.DeathTimer++
		if p.DeathTimer <= deathBombWindow && in.IsPressed(input.KeyBomb) && p.tryBomb() {
			return
		}
		if p.DeathTimer > deathBombWindow {
			p.GS.Life--
			// Original power is in 0.05 units: 0x40 lost = 3.20 displayed.
			p.GS.Power = max(p.GS.Power-game.DeathPowerLoss, 0)
			p.State = StateRespawn
			p.InvincTimer = invincTime
			p.X = (p.FieldLeft + p.FieldRight) / 2
			p.Y = p.FieldBottom - 48
		}
		return

	case StateRespawn:
		if !p.NoAttack && in.IsPressed(input.KeyBomb) && p.tryBomb() {
			return
		}
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
		if !p.NoAttack && in.IsPressed(input.KeyBomb) && p.tryBomb() {
			return
		}
		p.move(in)
		p.shoot(in)
	}

	p.updateAnim(in)
	if p.reimuA == nil {
		p.Bullets.Update()
	}
}

func (p *Player) tryBomb() bool {
	if p.State == StateBomb || p.GS.Power < game.BombPowerCost {
		return false
	}
	p.GS.Power -= game.BombPowerCost
	p.State = StateBomb
	p.DeathTimer = 0
	p.BombTimer = bombDuration
	p.InvincTimer = bombDuration + bombInvincTime
	return true
}

// Bounds of the fixed-point position (FUN_004250b0): X ±184, Y 32..432.
const (
	fixedMaxX = 0x47e0
	fixedMinY = 0xc80
	fixedMaxY = 0xa8c0
)

// SetFixed places the player at a fixed-point position (hundredths of a
// pixel from the playfield's top-centre), as FUN_00428e10 does when a replay
// chapter starts.
func (p *Player) SetFixed(ix, iy int32) {
	p.IX, p.IY = ix, iy
	p.syncPos()
}

// SetPos places the player at screen coordinates, rounded to the
// fixed-point grid.
func (p *Player) SetPos(x, y float64) {
	p.SetFixed(int32(math.Round((x-p.originX())*100)), int32(math.Round((y-p.FieldTop)*100)))
}

func (p *Player) originX() float64 { return (p.FieldLeft + p.FieldRight) / 2 }

// syncPos derives the float position: (float)int * 0.01f (DAT_00470b00),
// rounded to float32 as the x87 does at 24-bit precision.
func (p *Player) syncPos() {
	p.X = p.originX() + float64(float32(p.IX)*float32(0.01))
	p.Y = p.FieldTop + float64(float32(p.IY)*float32(0.01))
}

// move ports the stepping in FUN_004250b0: an 8-way direction picks the
// cardinal or diagonal step, the focus key the slow pair, and each axis moves
// by ftol(step * game speed), truncated, before the fixed-point clamp.
func (p *Player) move(in *input.State) {
	focused := in.IsPressed(input.KeyFocus)
	up, down := in.IsPressed(input.KeyUp), in.IsPressed(input.KeyDown)
	left, right := in.IsPressed(input.KeyLeft), in.IsPressed(input.KeyRight)
	straight, diag := p.steps[0], p.steps[2]
	if focused {
		straight, diag = p.steps[1], p.steps[3]
	}
	var vx, vy int32
	switch {
	case up && left: // 5
		vx, vy = -diag, -diag
	case down && left: // 7
		vx, vy = -diag, diag
	case up && right: // 6
		vx, vy = diag, -diag
	case down && right: // 8
		vx, vy = diag, diag
	case down: // 2
		vy = straight
	case up: // 1
		vy = -straight
	case left: // 3
		vx = -straight
	case right: // 4
		vx = straight
	}
	gs := float32(1)
	if p.GameSpeed != nil {
		if v := p.GameSpeed(); v != 0 {
			gs = v
		}
	}
	p.IX = min(max(p.IX+int32(float32(vx)*gs), -fixedMaxX), fixedMaxX)
	p.IY = min(max(p.IY+int32(float32(vy)*gs), fixedMinY), fixedMaxY)
	p.syncPos()
}

func (p *Player) shoot(in *input.State) {
	if p.reimuA != nil {
		return
	}
	if p.NoAttack || !in.IsPressed(input.KeyShot) {
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
	p.GS.Power = max(0, min(p.GS.Power+n, p.GS.MaxPower))
}

// BombDuration 返回符卡持续帧数，供 UI（如 Bomb 闪光）计算进度。
func BombDuration() int { return bombDuration }
