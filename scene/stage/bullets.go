package stage

// The enemy bullet manager's wiring (entity/bullet, entity/laser): the
// context that owns it, its environment, the per-frame calc and the player
// side of hits and grazes.

import (
	"sync"

	"th10/anm"
	"th10/audio"
	"th10/entity/bullet"
	"th10/entity/effect"
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
	c := &Context{
		State:   state,
		RNG:     rng,
		Player:  player.New(state, game.FieldLeft, game.FieldTop, game.FieldRight, game.FieldBottom),
		Bullets: bullet.NewPool(),
		Lasers:  laser.NewManager(),
		Items:   item.NewPool(256),
		Effects: effect.NewPool(512),
		World:   &anm.World{},
	}
	// The alternate RNG's seed is not traced yet.
	c.Anm = &anm.Env{GameSpeed: 1, RNG: rng, Alt: &zun.RNG{}, Spawn: c.World.Add}
	c.Player.GameSpeed = func() float32 { return c.Anm.GameSpeed }
	c.BulletEnv = &bullet.Env{
		Anm:     c.Anm,
		File:    bulletFile(),
		Effects: c.World,
		Hit:     c.bulletHit,
		Graze:   c.bulletGraze,
		PlaySE:  func(id int32, _ float32, _ bool) { c.PlaySEID(int(id)) },
		// FUN_0041bb00 type 8; the item kind is not ported yet.
		CancelItem: func(x, y float32) {
			c.Items.Spawn(float64(x)+gameX, float64(y)+gameY, item.PointItem)
		},
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

// ClearBullets cancels every bullet and laser round the player, leaving items:
// the bomb's clear until the bombs are ported (M7).
func (c *Context) ClearBullets() {
	p := c.BulletEnv.Player
	c.Bullets.CircleClear(p[0], p[1], 1000, true, true, c.BulletEnv)
	c.Lasers.CircleClear([3]float32{p[0], p[1], 0}, 1000, true, c.LaserEnv)
}

// bulletHit is the player side of FUN_004266b0's hit: none while a dialog is
// up or the player is dying (states 2-4), a cancel without a kill while
// invulnerable.
func (c *Context) bulletHit() bool {
	p := c.Player
	switch {
	case c.msg != nil || p.State == player.StateDead:
		return false
	case p.InvincTimer > 0 || p.State != player.StateNormal:
		return true
	}
	p.Hit()
	c.State.AddFaith(-8000)
	c.Effects.Spawn(p.X, p.Y, effect.TypePlayerDead, 30)
	c.PlaySE(audio.SEDead)
	return true
}

// laserKill is the player side of a laser hit (FUN_004267f0, FUN_00426cf0):
// only a player that can be hit is killed, and only then does the beam cut
// round it.
func (c *Context) laserKill() bool {
	p := c.Player
	if c.msg != nil || p.State != player.StateNormal || p.InvincTimer > 0 {
		return false
	}
	return c.bulletHit()
}

// laserGraze counts a laser graze, as for a bullet: the laser already played
// the sound and spawned the spark.
func (c *Context) laserGraze() { c.bulletGraze(nil) }

// bulletGraze counts a graze. The bullet already played the graze sound and
// spawned its spark; where the binary keeps the count is not traced yet.
func (c *Context) bulletGraze(*bullet.Bullet) {
	if s := c.Player.State; s != player.StateNormal && s != player.StateRespawn {
		return
	}
	c.State.Graze++
	c.State.AddFaith(3)
	c.State.AddScore(100)
}
