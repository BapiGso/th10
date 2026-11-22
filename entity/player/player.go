package player

import (
	"fmt"
	"math"
	"th10/anm"
	"th10/entity/playerbullet"
	"th10/game"
	"th10/gamedata"
	"th10/input"
	"th10/sht"
	"th10/zun"
)

// State is the player's state (+0x458), numbered as FUN_00425730 switches on
// it.
type State int32

const (
	StateRespawn   State = iota // 0: flying back in from below the playfield
	StateNormal                 // 1
	StateDying                  // 2: from the end of the deathbomb window to the respawn
	State3                      // 3: set by register writes (0x406c2d, 0x41c137); not ported
	StateDeathbomb              // 4: hit, with 8 frames left to bomb
)

// Player 玩家实体
type Player struct {
	VM        anm.VM    // +0x14: embedded player animation, not a world object
	Anm       *anm.File // +0x10: pl00.anm or pl01.anm (FUN_004247f0)
	world     *anm.World
	env       *anm.Env
	bulletAnm *anm.File
	hitbox    *anm.VM // +0x329c: focus hitbox's world handle
	xStep     int32   // +0x44c: previous raw horizontal step

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

	State State
	// StateTimer is +0x474, restarted on each change of state.
	StateTimer zun.Timer
	// Invinc is +0x430c: the frames of invulnerability left.
	Invinc zun.Timer
	// initTimer is +0x488, restarted only at init (FUN_00424d90): focus is
	// forced off while it is under 4.
	initTimer zun.Timer
	// DeathPos is +0x440, where the player died, in game coordinates. The
	// respawn's clears are centred on it.
	DeathPos [2]float32
	// Bomb is the bomb object (DAT_004776ec); nil leaves the player unable
	// to bomb.
	Bomb Bomber
	// NoAttack 由关卡在对话期间置位：射击（FUN_004281d0）和灵击都要求没有
	// MSG，决死灵击不受限。
	NoAttack bool
	// focus is +0x4474, latched by the movement (FUN_004250b0).
	focus bool
	areas [32]damageArea // +0x350c
	// FUN_00426f70/FUN_004250b0: all six layouts share four option slots
	// and 33 fixed-point position samples (+0x436c).
	shotFile        *sht.File
	options         [4]option
	optionCount     int
	optionView      [4]Option
	positionHistory [33][2]int32
	optionEase      int32 // +0x4308, held at 30 except Marisa C's focus lock

	// Hooks into the rest of the stage.
	SetGameSpeed func(float32) // writes DAT_00476f78
	// ClearRound is the respawn's clear round (x, y): FUN_00408100 at r with
	// the immune check, again at r/4 without it, then FUN_0041c800 at r.
	ClearRound func(x, y, r float32)
	// CancelAll cancels every bullet that is not cancel-immune and clears
	// every laser, without items (states 0 and 1).
	CancelAll func()
	// SpawnItem is FUN_0041bb00: an item of kind thrown at angle and speed.
	SpawnItem func(x, y float32, kind int32, angle, speed float32)
	// PowerChanged calls the existing GUI FUN_004054b0 after the bomb/death
	// power write (0x425a10/0x425af7/0x425c7f); nil leaves pure player tests usable.
	PowerChanged func()

	// 游戏区域边界
	FieldLeft, FieldTop, FieldRight, FieldBottom float64

	// 自机子弹池引用
	Bullets *playerbullet.Pool
	shot    *shotState // common SHT runtime of all six shot types
}

// Bomber is the bomb object (DAT_004776ec) as the player drives it.
type Bomber interface {
	Active() bool // +0x28: a bomb is running
	Start()       // FUN_00405860
}

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
	shots, err := sht.Parse(data)
	if err != nil {
		panic(fmt.Sprintf("load player SHT %s: %v", path, err))
	}
	if err = shots.ValidateTH10(); err != nil {
		panic(fmt.Sprintf("load player SHT %s: %v", path, err))
	}
	p := &Player{
		Anm: playerFile(gs.Character),
		steps: [4]int32{int32(movement.Fast), int32(movement.Slow),
			int32(movement.FastDiagonal), int32(movement.SlowDiagonal)},
		HitboxRadius: 2, GrazeRadius: 16,
		GS:       gs,
		shotFile: shots, optionCount: -1, optionEase: 30,
		State:     StateNormal, // FUN_00424d90
		FieldLeft: fieldL, FieldTop: fieldT,
		FieldRight: fieldR, FieldBottom: fieldB,
		Bullets: playerbullet.NewPool(128), // FUN_00427e90: P+0x49c, 128 records of 0x5c
		shot:    &shotState{},
	}
	p.StateTimer.Reset()
	p.initTimer.Reset()
	p.Invinc.Set(120)    // FUN_004247f0
	p.shot.timer.Set(-1) // FUN_00424d90, P+0x460
	p.SetFixed(0, 40000) // FUN_004247f0: (0, 400.0)
	for i := range p.positionHistory {
		p.positionHistory[i] = [2]int32{0, 40000} // FUN_004247f0, 0x42492c-0x424955
	}
	p.Bullets.GameSpeed = p.gameSpeed
	p.Bullets.SourcePosition = func(source int) [3]float32 {
		if source == 0 {
			x, y := p.gamePos()
			return [3]float32{x, y, 0}
		}
		o := p.options[source-1].pos // FUN_00427e90/FUN_00428c20: O_(source-1)+0x3c/+0x40
		return [3]float32{float32(o[0]) * 0.01, float32(o[1]) * 0.01, 0}
	}
	p.Bullets.OptionCount = func() int { return max(p.optionCount, 0) }
	p.Bullets.ShotClock = func() int32 { return p.shot.timer.Cur }
	p.Bullets.NoAttack = func() bool { return p.NoAttack }
	p.Bullets.HitArea = p.addArea // FUN_00428630, 0x428835-0x428863 -> FUN_00427b50
	return p
}

func (p *Player) Update() {
	p.UpdateInput(&input.Global)
}

// gameSpeed is DAT_00476f78, 1 when unset.
func (p *Player) gameSpeed() float32 {
	if p.GameSpeed != nil {
		if v := p.GameSpeed(); v != 0 {
			return v
		}
	}
	return 1
}

// UpdateInput advances one tick from an input snapshot, without polling the
// OS: FUN_00425730.
func (p *Player) UpdateInput(in *input.State) {
	switch p.State {
	case StateRespawn:
		if !p.respawn() {
			break
		}
		fallthrough
	case StateNormal:
		if !p.NoAttack && p.canBomb(in) {
			p.Invinc.Set(270)
			p.bomb()
			p.GS.LoseFaith(3000)
		}
		if p.StateTimer.Cur < 30 {
			p.cancelAll()
		}
		p.move(in)
	case StateDeathbomb:
		if p.StateTimer.Cur < 8 {
			if p.canBomb(in) {
				p.StateTimer.Set(60)
				p.Invinc.Set(200)
				p.bomb()
				p.State = StateNormal
			}
			break
		}
		p.die()
		fallthrough
	case StateDying:
		p.dying()
	}
	p.tail(in)
}

// respawn is state 0: the player rises from y 480 to 400 over 60 frames
// while the bullets round where it died are cleared, by a growing circle for
// 30 frames and then all of them. It reports whether the player is back to
// normal, which runs state 1 in the same frame.
func (p *Player) respawn() bool {
	t := p.StateTimer.Cur
	p.IY = 48000 - t*8000/60
	p.syncPos()
	p.snapOptions()            // FUN_00425730 state 0
	p.advancePositionHistory() // 0x4257b3-0x425833, before state-0 clearing/transition.
	if t < 30 {
		r := float32(float32(t)*17.066668) + 64
		if p.ClearRound != nil {
			p.ClearRound(p.DeathPos[0], p.DeathPos[1], r)
		}
	} else {
		p.cancelAll()
	}
	if t <= 59 {
		return false
	}
	p.State = StateNormal
	p.StateTimer.Reset()
	return true
}

func (p *Player) cancelAll() {
	if p.CancelAll != nil {
		p.CancelAll()
	}
}

// canBomb is the bomb test shared by states 1 and 4: no bomb running, at
// least 1.00 power, and the bomb key held (DAT_00474e5c bit 1).
func (p *Player) canBomb(in *input.State) bool {
	return p.Bomb != nil && !p.Bomb.Active() && p.GS.Power >= game.BombPowerCost &&
		in.IsPressed(input.KeyBomb)
}

// bomb starts a bomb (FUN_00405860) for 1.00 power and refreshes the options
// (FUN_00426f70). The power gauge (FUN_004054b0) is the HUD's.
func (p *Player) bomb() {
	p.Bomb.Start()
	p.GS.Power -= game.BombPowerCost
	p.RefreshOptions()
	p.refreshPowerGauge() // FUN_00425730, 0x425a10/0x425c7f, before state-1 faith loss
}

func (p *Player) refreshPowerGauge() {
	if p.PowerChanged != nil {
		p.PowerChanged()
	}
}

// die is FUN_004269d0, the end of the deathbomb window: a life, a third of
// the faith above 5000, the options and 1024 rank are lost, and the player
// is invulnerable for 180 frames.
func (p *Player) die() {
	c := int32(p.GS.Faith / 10) // DAT_00474c4c; GameState keeps it x10
	x := c - 5000
	v := int32(int64(x)*0x55555555>>32) - x
	c += ((v>>1 - v>>31) * 10) / 10
	p.GS.Life--
	p.GS.Faith = int64(max(c, 5000)) * 10
	p.State = StateDying
	p.StateTimer.Reset()
	p.Invinc.Set(180)
	p.clearOptions()
	p.GS.AddRank(-0x400)
}

// dying is state 2. On frame 3 the player loses 3.20 power, scattered as 7
// items fanned upward; from frame 30 it respawns below the playfield, game
// speed back to 1, leaving a damage circle where it died.
func (p *Player) dying() {
	t := p.StateTimer.Cur
	if t == 3 {
		p.GS.Power = max(p.GS.Power-game.DeathPowerLoss, 0)
		p.refreshPowerGauge() // FUN_00425730, 0x425af7: before the seven death drops
		x, y := p.gamePos()
		base := zun.Aim(x, y, 0, y-224) // FUN_00426610
		for i := int32(0); i < 7; i++ {
			a := float32(float32(float32(i)*0.11219974)+base) - 0.39269909
			kind := int32(1)
			if i == 1 || i == 3 || i == 5 {
				kind = 4
			}
			if p.SpawnItem != nil {
				p.SpawnItem(x, y, kind, a, 3)
			}
		}
		p.RefreshOptions()
	}
	if t < 30 || p.GS.Life < 0 { // the stage handles the game over
		return
	}
	p.State = StateRespawn
	if p.SetGameSpeed != nil {
		p.SetGameSpeed(1)
	}
	x, y := p.gamePos()
	p.addArea(x, y, 32, 16, 30, 150)
	p.DeathPos = [2]float32{x, y}
	p.IX, p.IY = 0, 48000
	p.X, p.Y = p.originX(), p.FieldTop+480
	p.Invinc.Set(280)
	p.StateTimer.Set(0)
}

// tail is the part of FUN_00425730 every state runs: the damage areas, the
// invulnerability countdown and blink, the timers, then shooting and the
// shots.
func (p *Player) tail(in *input.State) {
	gs := p.gameSpeed()
	p.updateAreas(gs)
	blink := false
	if p.Invinc.Cur >= 1 {
		p.Invinc.Add(-1, gs)
		blink = p.StateTimer.Cur != p.StateTimer.Prev && p.StateTimer.Cur%3 == 0
	}
	// FUN_00425730, 0x425e46-0x425efb: count down first, then select
	// blue Color2 on the old state clock, then run the embedded VM.
	if p.anmAttached() {
		if blink {
			p.VM.Color2, p.VM.Alpha2 = [3]uint8{0, 0, 255}, 255
			p.VM.Flags |= 0x8000
		} else {
			p.VM.Flags &^= 0x8000
		}
		p.VM.Run(p.env)
	}
	// 0x42618e and 0x4261f2: both clocks advance after the VM.
	p.StateTimer.Tick(gs)
	p.initTimer.Tick(gs)
	p.updateShot(in)
}

// Hittable reports whether a bullet or laser touching the hitbox counts
// (FUN_004266b0, FUN_004267f0): not while dying (states 2-4).
func (p *Player) Hittable() bool {
	return p.State != StateDying && p.State != State3 && p.State != StateDeathbomb
}

// Hit is FUN_00426cf0: the deathbomb window opens. The caller has checked
// Hittable and the invulnerability.
func (p *Player) Hit() {
	p.State = StateDeathbomb
	// FUN_00426cf0, 0x426d09-0x426e05: tail-insert one burst followed
	// by 32 sparks, before resetting the state and invulnerability clocks.
	if p.anmAttached() {
		x, y := p.gamePos()
		pos := [3]float32{x + 224, y + 16, 0}
		v := p.world.SpawnTail(p.bulletAnm, 0x162, 0, p.env)
		p.world.SetOffset(v, pos)
		for range 32 {
			v = p.world.SpawnTail(p.bulletAnm, 0x163, 0, p.env)
			p.world.SetOffset(v, pos)
		}
	}
	p.StateTimer.Reset()
	p.Invinc.Set(6)
	if p.anmAttached() {
		p.VM.Start(p.Anm, 0, p.env) // 0x426ecb-0x426ed0
	}
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

// gamePos is the float position +0x3c0/+0x3c4, relative to the playfield's
// top-centre.
func (p *Player) gamePos() (float32, float32) {
	return float32(p.X - p.originX()), float32(p.Y - p.FieldTop)
}

// syncPos derives the float position: (float)int * 0.01f (DAT_00470b00),
// rounded to float32 as the x87 does at 24-bit precision.
func (p *Player) syncPos() {
	p.X = p.originX() + float64(float32(p.IX)*float32(0.01))
	p.Y = p.FieldTop + float64(float32(p.IY)*float32(0.01))
}

// move ports the stepping in FUN_004250b0: an 8-way direction picks the
// cardinal or diagonal step, the focus key the slow pair, and each axis moves
// by ftol(step * game speed), truncated, before the fixed-point clamp. The
// options follow.
func (p *Player) move(in *input.State) {
	// Focus is latched here and held off for the first 4 frames after init.
	p.focus = in.IsPressed(input.KeyFocus) && p.initTimer.Cur >= 4
	// FUN_004250b0, 0x4251a0-0x4251d0: Marisa C holds its options in
	// place until 30 unfocused movement ticks have elapsed.
	if p.initTimer.Cur < 4 {
		p.optionEase = 30
	} else if p.GS.Character == game.CharMarisa && p.GS.ShotType == game.ShotC {
		if p.focus {
			p.optionEase = 0
		} else if p.optionEase < 30 {
			p.optionEase++
		}
	}
	// FUN_004250b0, 0x4251e0-0x425245 and 0x4252d0-0x4252f8.
	if p.anmAttached() {
		if !p.focus {
			p.world.Interrupt(p.hitbox, 1)
			p.hitbox = nil
		} else if p.hitbox == nil {
			p.hitbox = p.world.SpawnTail(p.bulletAnm, 0x160, 9, p.env)
		}
	}
	up, down := in.IsPressed(input.KeyUp), in.IsPressed(input.KeyDown)
	left, right := in.IsPressed(input.KeyLeft), in.IsPressed(input.KeyRight)
	straight, diag := p.steps[0], p.steps[2]
	if p.focus {
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
	// 0x425385-0x4253f8: switch on the raw step's sign, even when
	// game speed truncates the actual movement to zero or a clamp stops it.
	id := -1
	switch {
	case vx < 0 && p.xStep >= 0:
		id = 1
	case vx > 0 && p.xStep <= 0:
		id = 3
	case vx == 0 && p.xStep < 0:
		id = 2
	case vx == 0 && p.xStep > 0:
		id = 4
	}
	if id >= 0 && p.anmAttached() {
		p.VM.Start(p.Anm, id, p.env)
	}
	p.xStep = vx
	gs := p.gameSpeed()
	p.IX = min(max(p.IX+int32(float32(vx)*gs), -fixedMaxX), fixedMaxX)
	p.IY = min(max(p.IY+int32(float32(vy)*gs), fixedMinY), fixedMaxY)
	p.syncPos()
	// 0x4254c8-0x42551b: follow before the option loop, using a raw
	// screen offset (FUN_004492f0), not Place's extra (224, 16).
	if p.anmAttached() && p.world.Contains(p.hitbox) {
		x, y := p.gamePos()
		p.world.SetOffset(p.hitbox, [3]float32{x + 224, y + 16, 0})
	} else {
		p.hitbox = nil
	}
	p.updateOptions(vx != 0 || vy != 0)
}

// AddPower 增加火力
func (p *Player) AddPower(n int) {
	p.GS.Power = max(0, min(p.GS.Power+n, p.GS.MaxPower))
}
