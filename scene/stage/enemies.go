package stage

// The stage side of an enemy's damage and defeat: the player's damage query
// (FUN_00428630) and the item manager (FUN_0041bb00, FUN_0041afd0). The enemy
// update itself (FUN_0040dc80) is the ECL VM's.

import (
	"th10/entity/enemy"
	"th10/entity/item"
	"th10/entity/player"
	"th10/game"
	"th10/zun"
)

// EnemyDamage is the damage enemy e takes this frame: FUN_00428630 (the
// player's shots on its hit box, the bomb, then the damage areas; nothing on
// a frame the player's state timer did not run, and a tenth plus 10, over
// ten, to the stored score for any hit), cut to a fifth while the player is
// dying or respawning (states 2 and 0, FUN_0040dc80).
func (c *Context) EnemyDamage(e *enemy.Enemy) int {
	p := c.Player
	if p.StateTimer.Cur == p.StateTimer.Prev {
		return 0
	}
	ex, ey, ew, eh := e.HitRect()
	x, y := float32(e.X-gameX), float32(e.Y-gameY)
	dmg := p.Bullets.HitEnemy(ex, ey, ew, eh)
	dmg += c.Bomb.Damage(x, y)
	dmg += p.AreaDamage(x, y)
	if dmg != 0 {
		c.State.AddPoints(int64((dmg/10+10)/10) * 10)
	}
	if p.State == player.StateRespawn || p.State == player.StateDying {
		dmg /= 5
	}
	return dmg
}

// SpawnItem is FUN_0041bb00: an item of the original kind at game position
// (x, y), launched at angle and speed.
func (c *Context) SpawnItem(x, y float32, kind int32, angle, speed float32) {
	if c.Items == nil {
		return
	}
	c.Items.Spawn(c.ItemEnv, [3]float32{x, y, 0}, kind, angle, speed)
}

// cancelItem is FUN_0041bb00(8, -1, -pi/2, 0.6): a cancelled bullet's or
// laser's item at game position (x, y).
func (c *Context) cancelItem(x, y float32) {
	c.SpawnItem(x, y, item.KindCancel, -zun.HalfPi, 0.6)
}

// UpdateItems runs the item manager (FUN_0041afd0) for a frame against the
// player as it stands after its update; focused is the focus key held. A
// running bomb collects every item.
func (c *Context) UpdateItems(focused bool) {
	p := c.Player
	char := 0
	if c.State.Character == game.CharMarisa {
		char = 1
	}
	c.ItemEnv.Player = item.NewPlayer(char, float32(p.X-gameX), float32(p.Y-gameY),
		int32(p.State), focused, c.Bomb.Active())
	c.Items.Update(c.ItemEnv)
}
