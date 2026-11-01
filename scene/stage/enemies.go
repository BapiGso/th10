package stage

// The stage side of an enemy's damage and defeat: the player's damage query
// (FUN_00428630) and the item manager's spawn (FUN_0041bb00). The enemy
// update itself (FUN_0040dc80) is the ECL VM's.

import (
	"th10/entity/enemy"
	"th10/entity/item"
	"th10/entity/player"
)

// EnemyDamage is the damage enemy e takes this frame: FUN_00428630 (the
// player's shots on its hit box, then the damage areas; nothing on a frame
// the player's state timer did not run), cut to a fifth while the player is
// dying or respawning (states 2 and 0, FUN_0040dc80).
func (c *Context) EnemyDamage(e *enemy.Enemy) int {
	p := c.Player
	if p.StateTimer.Cur == p.StateTimer.Prev {
		return 0
	}
	ex, ey, ew, eh := e.HitRect()
	dmg := p.Bullets.HitEnemy(ex, ey, ew, eh)
	dmg += p.AreaDamage(float32(e.X-gameX), float32(e.Y-gameY))
	if p.State == player.StateRespawn || p.State == player.StateDying {
		dmg /= 5
	}
	return dmg
}

// EnemyHurt credits damage that took HP off an enemy. Not ported: a stand-in
// for the per-hit score and faith of the player's shots.
func (c *Context) EnemyHurt(dmg int) {
	c.State.AddFaith(int64(dmg))
	c.State.AddScore(int64(dmg) * 12)
}

// SpawnItem is FUN_0041bb00 as far as the item port goes: an item of the
// original kind at game position (x, y), x clamped to the field. At full
// power the power items turn into faith items (1, 4 -> 9) and the boss's
// power items into big point items (10, 11 -> 5). The launch angle and speed
// are not ported; the item pool gives every item the same toss.
func (c *Context) SpawnItem(x, y float32, kind int32, angle, speed float32) {
	if c.Items == nil {
		return
	}
	x = min(max(x, -192), 192)
	if c.State.Power/5 > 99 {
		switch kind {
		case 1, 4:
			kind = 9
		case 10, 11:
			kind = 5
		}
	}
	t := item.PointItem
	switch kind {
	case 1, 10:
		t = item.PowerItem
	case 4, 11:
		t = item.BigPower
	case 5:
		t = item.BigPoint
	case 7:
		t = item.LifeFragment
	case 9:
		t = item.FaithItem
	}
	c.Items.Spawn(float64(x)+gameX, float64(y)+gameY, t)
}
