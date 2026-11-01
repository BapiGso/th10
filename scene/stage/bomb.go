package stage

// The bomb (DAT_004776ec, calc priority 0x11, between the player and the
// enemies) and the spell card state it reads. The bomb itself is a bullet.anm
// script in the effect world: it lasts as long as that object does, rises with
// its centre and takes the object's y scale as its radius.

import (
	"th10/anm"
	"th10/game"
	"th10/zun"
)

// SpellCard is the spell card part of the enemy manager's helper
// (DAT_004776f4). The ECL VM owns it and points Context.Spell at it; a nil
// card is no card.
type SpellCard struct {
	Active  bool      // +0x378c bit 0: from the declaration to its end
	Capture bool      // +0x378c bit 1: not yet lost to a bomb
	ID      int32     // +0x3788: the card's index, kept after it ends
	Timer   zun.Timer // +0x3734: reset at the declaration, ticked by FUN_00408d60
	Bonus   int64     // +0x3790
}

// Bombed is FUN_00405500: a bomb from a second into the card on loses its
// capture. The timer keeps its value after the card ends.
func (s *SpellCard) Bombed() {
	if s != nil && s.Timer.Cur > 59 {
		s.Bonus = 0
		s.Capture = false
	}
}

func (s *SpellCard) active() bool { return s != nil && s.Active }

// heavy reports the cards a bomb hardly touches: ids 0x5d-0x60 while one is
// up, and 0x6d even after it ends, as FUN_00405860 and FUN_00405ac0 test it.
func (s *SpellCard) heavy() bool {
	if s == nil {
		return false
	}
	return s.Active && s.ID >= 0x5d && s.ID <= 0x60 || s.ID == 0x6d
}

// Bomb is the bomb object (DAT_004776ec).
type Bomb struct {
	ctx   *Context
	state int32      // +0x28: 0 idle, 1 running
	vm    *anm.VM    // +0x2c: the world object
	Pos   [3]float32 // +0x30: its centre, in game coordinates
	// Radius is +0x3c: the clear and damage circle.
	Radius float32
	Timer  zun.Timer // +0x14
	// strong is +0x44: the bomb went off a second or more into a spell card.
	strong bool
}

// Active reports whether a bomb is running.
func (b *Bomb) Active() bool { return b.state != 0 }

// Start is FUN_00405860: a bomb at the player, as it stands before moving
// this frame. A running bomb is left alone.
func (b *Bomb) Start() {
	if b.state != 0 {
		return
	}
	c := b.ctx
	b.state = 1
	b.Timer.Reset()
	b.Radius = 32
	b.Pos = [3]float32{float32(c.Player.X - gameX), float32(c.Player.Y - gameY), 0}
	id := 0x190
	switch {
	case c.Spell.heavy():
		id = 0x1b9
	case c.State.Character == game.CharMarisa:
		id = 0x197
	}
	b.vm = c.World.Spawn(bulletFile(), id, b.Pos, c.Anm)
	c.PlaySEID(0x26)
	// FUN_004054b0 redraws the power gauge.
	c.Spell.Bombed()
	c.State.AddRank(-0x80)
	b.strong = c.Spell.active() && c.Spell.Timer.Cur >= 60
}

// Update is FUN_00405750: once the world has dropped the object the bomb is
// over; until then it rises, carries the object and its chain with it, takes
// the object's y scale as its radius and clears the bullets and lasers inside.
func (b *Bomb) Update() {
	if b.state == 0 {
		return
	}
	c := b.ctx
	if b.state == 1 {
		if !c.World.Contains(b.vm) {
			b.vm, b.state = nil, 0
			return
		}
		rise := float32(1)
		switch {
		case c.Spell.active() && (c.Spell.ID >= 0x5d && c.Spell.ID <= 0x60 || c.Spell.ID == 0x6d):
			rise = 0.5
		case c.State.Character == game.CharMarisa:
			rise = 1.3
		}
		b.Pos[1] -= rise
		b.vm.SetOffset([3]float32{b.Pos[0] + 224, b.Pos[1] + 16, b.Pos[2]}) // FUN_004492f0
		b.Radius = b.vm.Scale[1]
		b.clear()
	}
	b.Timer.Tick(c.Anm.GameSpeed)
}

// clear is FUN_00405ac0: the bullets inside, items only outside a spell card
// and the cancel-immune spared except on the heavy cards, then the lasers.
func (b *Bomb) clear() {
	c := b.ctx
	items := !c.Spell.active()
	c.Bullets.CircleClear(b.Pos[0], b.Pos[1], b.Radius, items, !c.Spell.heavy(), c.BulletEnv)
	c.Lasers.CircleClear(b.Pos, b.Radius, items, c.LaserEnv)
}

// Damage is FUN_004059f0, the bomb's share of an enemy's damage at game
// position (x, y). Inside the circle it deals 5, or 38 while a boss is up; a
// card cuts it to 3 unless the bomb is strong. Outside it reaches a boss
// only, by 20 (16 for Marisa) when no card is up and the bomb not strong, or
// by 26 (23) for a strong bomb during the card.
func (b *Bomb) Damage(x, y float32) int {
	if b.state == 0 {
		return 0
	}
	c := b.ctx
	spell := c.Spell.active()
	marisa := c.State.Character == game.CharMarisa
	boss := false
	if c.Boss0 != nil {
		_, boss = c.Boss0()
	}
	dy, dx := y-b.Pos[1], x-b.Pos[0]
	if float32(float32(dy*dy)+float32(dx*dx)) < float32(b.Radius*b.Radius) {
		switch {
		case spell && b.strong:
			return 38
		case spell:
			return 3
		case boss:
			return 38
		}
		return 5
	}
	switch {
	case !spell && boss && !b.strong:
		if marisa {
			return 16
		}
		return 20
	case spell && b.strong:
		if marisa {
			return 23
		}
		return 26
	}
	return 0
}
