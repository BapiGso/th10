package stage

// The enemy bullet manager's wiring (entity/bullet, entity/laser): the
// context that owns it, its environment, the per-frame calc and the player
// side of hits and grazes.

import (
	"sync"

	"th10/anm"
	"th10/audio"
	"th10/entity/bullet"
	"th10/entity/enemy"
	"th10/entity/item"
	"th10/entity/laser"
	"th10/entity/player"
	"th10/game"
	"th10/gamedata"
	"th10/zun"
)

// gameX, gameY is where game coordinate (0, 0) sits in world coordinates.
const (
	gameX = (game.FieldLeft + game.FieldRight) / 2
	gameY = game.FieldTop
)

// bulletFile is bullet.anm, the manager's sprites and sparks (+0x3e0b50).
var bulletFile = sync.OnceValue(func() *anm.File {
	f, err := gamedata.ANM("bullet.anm")
	if err != nil {
		panic("stage: embedded bullet.anm: " + err.Error())
	}
	return f
})

// hitboxSize is the side of the player's hitbox by character (0x476fb8).
var hitboxSize = [...]float32{2, 3.5}

// NewContext builds the runtime context for state: the player, the pools, the
// effect world and the bullet environment. The RNG starts from the replay
// seed when the state has one.
func NewContext(state *game.GameState) *Context {
	rng := &zun.RNG{State: 0x1234}
	if state.HasSeed {
		rng.State = state.RNGSeed
	}
	if r := state.StageRNG(); r != nil && state.StageMode != 1 && state.ReplayMode != 1 {
		rng = r // DAT_004918b0 is not re-seeded by a live scene-0x0b transfer
	}
	c := &Context{
		State:         state,
		RNG:           rng,
		Player:        player.New(state, game.FieldLeft, game.FieldTop, game.FieldRight, game.FieldBottom),
		Bullets:       bullet.NewPool(),
		Lasers:        laser.NewManager(),
		Items:         item.NewManager(),
		World:         &anm.World{},
		GUIController: &GUIController{},
	}
	// FUN_004201b0 seeds DAT_004918a8 once (0x4201e1); stage and replay
	// starts never restore it. GameState references its application's RNG.
	c.Anm = &anm.Env{GameSpeed: 1, RNG: rng, Alt: state.AlternateRNG(), Spawn: c.World.Add}
	c.Player.GameSpeed = func() float32 { return c.Anm.GameSpeed }
	c.Player.SetGameSpeed = func(v float32) { c.Anm.GameSpeed = v }
	c.Player.ClearRound = c.respawnClear
	c.Player.CancelAll = func() {
		c.Bullets.CancelVulnerable(c.BulletEnv)
		c.Lasers.ClearAll(false, c.LaserEnv)
	}
	c.Player.SpawnItem = c.SpawnItem // the death drops
	c.Player.PowerChanged = func() {
		if c.GUI != nil {
			c.GUI.SetPower(powerDigits(c.State.Power)) // FUN_004054b0, original caller uses raw units of 0.05
		}
	}
	c.Bomb = &Bomb{ctx: c}
	c.Player.Bomb = c.Bomb
	c.BulletEnv = &bullet.Env{
		Anm:     c.Anm,
		File:    bulletFile(),
		Effects: c.World,
		Hit:     c.bulletHit,
		Graze:   c.bulletGraze,
		PlaySE: func(id int32, x float32, pan bool) {
			if pan {
				c.PlaySEIDAt(int(id), x) // FUN_0043dd10; false is FUN_0043dc90 at pan 0.
			} else {
				c.PlaySEID(int(id))
			}
		},
		CancelItem: c.cancelItem,
	}
	c.Player.AttachAnm(c.World, c.Anm, c.BulletEnv.File)                           // FUN_004247f0; share bullet.anm
	c.Player.Bullets.PlaySE = c.PlaySEIDAt                                         // FUN_00427e90 -> FUN_0043dd10
	c.Player.Bullets.ManagerPresent = func() bool { return c.ManagerOrder != nil } // DAT_00477704, 0x428328
	c.CaptureReplayPlayer = c.Player.CaptureReplayPlayer                           // FUN_0042a450, 0x42a512-0x42a56a
	c.RestoreReplayPlayer = func(header []byte) {
		c.Player.RestoreReplayPlayer(header)
		if c.GUI != nil {
			c.GUI.SetLives(c.State.Life) // FUN_00424d90 -> FUN_00413790, 0x424e9c
		}
	}
	c.ItemEnv = &item.Env{
		State:    state,
		Anm:      c.Anm,
		File:     c.BulletEnv.File,
		Effects:  c.World, // FUN_0041bb00/FUN_0041ba50: script 0x189 in the same World
		PlaySE:   func(id int32) { c.PlaySEID(int(id)) },
		PlaySEAt: func(id int32, x float32) { c.PlaySEIDAt(int(id), x) },
		Options:  c.Player.RefreshOptions,
	}
	c.LaserEnv = &laser.Env{
		Anm:        c.Anm,
		File:       c.BulletEnv.File,
		Effects:    c.World,
		Kill:       c.laserKill,
		Graze:      c.laserGraze,
		PlaySE:     c.BulletEnv.PlaySE,
		CancelItem: c.BulletEnv.CancelItem,
		Boss: func() ([3]float32, bool) {
			if c.Boss0 == nil {
				return [3]float32{}, false
			}
			return c.Boss0()
		},
	}
	c.SyncPlayer()
	return c
}

// SyncPlayer copies the player's position and hitbox into the bullet and
// laser environments, in game coordinates.
func (c *Context) SyncPlayer() {
	p := c.Player
	x, y := float32(p.X-gameX), float32(p.Y-gameY)
	h := hitboxSize[0] * 0.5
	if c.State.Character == game.CharMarisa {
		h = hitboxSize[1] * 0.5
	}
	c.BulletEnv.Player = [2]float32{x, y}
	c.BulletEnv.Box = [4]float32{x - h, y - h, x + h, y + h}
	c.LaserEnv.Player = [2]float32{x, y}
	c.LaserEnv.Half = [2]float32{h, h}
}

// UpdateBullets runs the laser manager (FUN_0041c480), then the bullet
// manager (FUN_004065c0), for a frame: the calc chain's order.
func (c *Context) UpdateBullets() {
	c.SyncPlayer()
	c.Lasers.Calc(c.LaserEnv)
	c.Bullets.Calc(c.BulletEnv)
}

// UpdateWorld runs the effect objects for a frame.
func (c *Context) UpdateWorld() {
	c.World.Run(c.Anm)
}

// respawnClear is the respawn's clear round (x, y), radius r (FUN_00425730
// state 0): FUN_00408100 at r sparing the cancel-immune bullets, again at r/4
// sparing none, then FUN_0041c800 at r, all without items.
func (c *Context) respawnClear(x, y, r float32) {
	c.Bullets.CircleClear(x, y, r, false, true, c.BulletEnv)
	c.Bullets.CircleClear(x, y, float32(r*0.25), false, false, c.BulletEnv)
	c.Lasers.CircleClear([3]float32{x, y, 0}, r, false, c.LaserEnv)
}

// EnemyContact is FUN_004266b0 at FUN_0040dc80's 0x40e2df call site:
// the enemy body (+0xb8/+0xbc), not its shootable hurtbox (+0xb0/+0xb4),
// against the player's post-movement rectangle. The ignored return value
// neither cancels the enemy nor produces bullet graze effects.
func (c *Context) EnemyContact(e *enemy.Enemy) {
	if c.Player == nil || c.BulletEnv == nil {
		return
	}
	x, y := float32(e.X-gameX), float32(e.Y-gameY)
	hw, hh := float32(e.BodyWidth)*0.5, float32(e.BodyHeight)*0.5
	box := c.BulletEnv.Box
	// FUN_004266b0, 0x4266b3-0x42671c: round each bound to float32;
	// touching edges count, as with the original bullet hit test.
	if x+hw < box[0] || y+hh < box[1] || box[2] < x-hw || box[3] < y-hh {
		return
	}
	c.bulletHit() // FUN_004266b0, 0x42671e-0x426761 -> FUN_00426cf0
}

// bulletHit is the player side of FUN_004266b0's hit: none while a dialog is
// up or the player is dying (states 2-4), a cancel without a kill while
// invulnerable.
func (c *Context) bulletHit() bool {
	p := c.Player
	if c.msg != nil || !p.Hittable() {
		return false
	}
	if p.Invinc.Cur < 1 {
		c.hitPlayer()
	}
	return true
}

// hitPlayer opens the deathbomb window (FUN_00426cf0).
func (c *Context) hitPlayer() {
	p := c.Player
	p.Hit()
	// FUN_00426cf0 has the same timer > 59 failure writes as FUN_00405500.
	c.Spell.Bombed()
	c.PlaySE(audio.SEDead)
}

// laserKill is the player side of a laser hit (FUN_004267f0, FUN_00426cf0):
// only a player that can be hit is killed, and only then does the beam cut
// round it.
func (c *Context) laserKill() bool {
	p := c.Player
	if c.msg != nil || !p.Hittable() || p.Invinc.Cur >= 1 {
		return false
	}
	c.hitPlayer()
	return true
}

// laserGraze is the laser's graze hook; see bulletGraze.
func (c *Context) laserGraze() { c.bulletGraze(nil) }

// bulletGraze is the graze hook. The original's graze branch (FUN_00406240 when
// FUN_004266b0 returns 2, bullet flag 4; FUN_0041d3d0 for lasers) only sets the
// flag, spawns the graze spark and plays the sound, and the bullet and laser
// packages do all three before calling this. It changes no score, faith or
// graze count, and TH10 keeps no graze count, so there is nothing to do here.
func (c *Context) bulletGraze(*bullet.Bullet) {}
