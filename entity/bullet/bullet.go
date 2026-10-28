// Package bullet is th10.exe's enemy bullet manager (DAT_004776f0), ported
// function by function: the emitter FUN_004073e0/FUN_004067d0, the transform
// chain FUN_00406d90 and its handlers, the update FUN_00406240, the manager
// calc FUN_004065c0 and the clears FUN_00408030/FUN_00408100/FUN_00408210.
//
// Everything is float32 in game coordinates (x -192..192, y 0..448), with the
// binary's x87 24-bit rounding: products are wrapped in float32() so they
// cannot fuse. A slot is persistent memory, as in the binary: freeing one
// clears only its state and timers, and spawning one leaves the fields it
// does not write.
package bullet

import (
	"th10/anm"
	"th10/zun"
)

// Slots is the manager's bullet count (2000 slots of 0x7f0 bytes at +0x60).
const Slots = 2000

// Layers is the number of draw lists (bullet layer table 0x474250).
const Layers = 6

// Bullet flags (+0).
const (
	FlagUsed     uint32 = 1
	FlagHittable uint32 = 2
	FlagGrazed   uint32 = 4
	FlagDelete   uint32 = 8
)

// Bullet states (+0x446).
const (
	StateFree       int16 = 0
	StateActive     int16 = 1
	StateSpawning   int16 = 2 // half speed, no hit test, until the script sets I0
	StateCancelling int16 = 3 // half speed until the cancel script ends
)

// Transform types (ex entry +0x10). The ones that stay active set their bit
// in Bullet.ExFlags and run a handler every frame.
const (
	ExSpawnBoost  uint32 = 0x1        // speed boost over the first 17 frames
	ExAccel       uint32 = 0x10       // add a velocity vector each frame
	ExAngular     uint32 = 0x20       // add speed and turn each frame
	ExStepRel     uint32 = 0x40       // slow to a stop, then turn by an angle
	ExStepAim     uint32 = 0x80       // ... then aim at the player plus an angle
	ExStepAbs     uint32 = 0x100      // ... then take an absolute angle
	ExBounce      uint32 = 0x400      // bounce off all four walls
	ExBounceTop   uint32 = 0x800      // bounce off the sides and the top
	ExImmune      uint32 = 0x1000     // cancel-immune for c frames
	ExGrace       uint32 = 0x2000     // offscreen grace of c frames
	ExScript      uint32 = 0x4000     // restart the sprite script
	ExWait        uint32 = 0x8000     // hold the chain for c frames
	ExCancel      uint32 = 0x10000    // cancel the bullet
	ExSound       uint32 = 0x20000    // play sound c
	ExWrapX       uint32 = 0x100000   // wrap round the sides c times
	ExWrapY       uint32 = 0x200000   // wrap round the top and bottom c times
	ExVolley      uint32 = 0x400000   // fire a volley from the bullet
	ExValue       uint32 = 0x1000000  // store c at +0x420
	ExGoto        uint32 = 0x2000000  // jump to entry c
	ExHoming      uint32 = 0x4000000  // turn toward the player
	ExBounceSides uint32 = 0x8000000  // bounce off the sides only
	ExNop         uint32 = 0x80000000 // skip
)

// ExCount is the number of transform entries in a template and a bullet.
const ExCount = 18

// Ex is one transform entry (0x18 bytes).
type Ex struct {
	A, B  float32
	C, D  int32
	Type  uint32
	Async int32 // nonzero lets the chain run past it while handlers are active
}

// Template is an enemy's emitter slot (enemy +0x2c4 + slot*0x210).
type Template struct {
	Type, Colour   int16      // +0, +2
	Pos            [3]float32 // +4
	Angle1, Angle2 float32    // +0x10, +0x14
	Speed1, Speed2 float32    // +0x18, +0x1c
	Ex             [ExCount]Ex
	Count1, Count2 int16  // +0x1f4 (per ring), +0x1f6 (rings)
	Aim            uint16 // +0x1f8, the angle mode 0..8
	Flags          uint32 // +0x1fc
	Sound          int32  // +0x200
	TransformSound int32  // +0x204
	StartEx        int32  // +0x208
}

// Bullet is one slot.
type Bullet struct {
	Flags  uint32 // +0
	Immune int32  // +4, frames the bullet ignores circle clears
	VM     anm.VM // +8

	Pos, Vel     [3]float32 // +0x3b4, +0x3c0
	Speed, Angle float32    // +0x3d8, +0x3e4
	W, H         float32    // +0x3f0, hit size

	T, T2 zun.Timer // +0x3f8 (age), +0x40c

	Value        int32  // +0x420, set by ExValue
	Grace        int32  // +0x434, frames before the offscreen test applies
	CancelScript int32  // +0x438, the effect a cancel spawns (-1 none)
	ExFlags      uint32 // +0x43c, the active transforms
	TplFlags     uint32 // +0x440
	State        int16  // +0x446
	Clear        int32  // +0x450, zeroed by a rect clear
	Unk454       int32  // +0x454, zeroed at spawn

	TransformSound int32 // +0x458
	ExIdx          int32 // +0x45c, the next transform entry
	Layer          int32 // +0x460
	Ex             [ExCount]Ex

	boostT zun.Timer // +0x614

	accel    float32    // +0x65c
	accelAng float32    // +0x660
	accelVec [3]float32 // +0x664
	accelT   zun.Timer  // +0x648
	accelDur int32      // +0x670

	angA, angB float32   // +0x690, +0x694
	angT       zun.Timer // +0x67c
	angDur     int32     // +0x6a4

	stepSpeed, stepAngle float32   // +0x6c4, +0x6c8
	stepT                zun.Timer // +0x6b0
	stepDur, stepMax     int32     // +0x6d8, +0x6dc
	stepN                int32     // +0x6e0

	bounceSpeed     float32 // +0x6f8
	bounceN, bounce int32   // +0x70c, +0x710 (count, limit)

	waitT          zun.Timer // +0x718
	wrapXT, wrapYT zun.Timer // +0x74c, +0x780

	homeA, homeB float32   // +0x7c8, +0x7cc
	homeT        zun.Timer // +0x7b4
	homeDur      int32     // +0x7dc

	Type, Colour int16 // +0x7ea, +0x7ec
}

// Env is what the bullets read from and do to the rest of the game.
type Env struct {
	// Anm runs the bullet scripts and their effects: the game speed, the two
	// RNGs (spawn modes 6-8 draw from Anm.RNG) and Spawn for op-88 children.
	Anm *anm.Env
	// File is bullet.anm (manager +0x3e0b50).
	File *anm.File
	// Effects is the world list cancel, graze and clear sparks join.
	Effects *anm.World

	// Player is the player position (DAT_00477834 +0x3c0) and Box its hitbox
	// left, top, right, bottom (+0x404, +0x408, +0x410, +0x414).
	Player [2]float32
	Box    [4]float32

	// Hit is called when a hittable bullet overlaps the box. It reports whether
	// the hit counts (FUN_004266b0 returns 1), killing the player unless it is
	// invulnerable; a counted hit cancels the bullet.
	Hit func() bool
	// Graze is called when a bullet first grazes the player.
	Graze func(b *Bullet)
	// PlaySE plays sound id (table 0x4749c8), panned by x when pan is set.
	PlaySE func(id int32, x float32, pan bool)
	// CancelItem spawns the item a cleared bullet leaves (FUN_0041bb00 type 8).
	CancelItem func(x, y float32)
	// Hold returns the flags at DAT_00477810 +0x58 (0 when there is no such
	// object): bit 0 or 2 skips the bullet calc, bits 1 and 10 together hold
	// every bullet in place, bit 2 skips the draw.
	Hold func() uint32
}

func (e *Env) hold() uint32 {
	if e.Hold == nil {
		return 0
	}
	return e.Hold()
}

func (e *Env) gs() float32 {
	if e.Anm == nil || e.Anm.GameSpeed == 0 {
		return 1
	}
	return e.Anm.GameSpeed
}

func (e *Env) se(id int32, x float32, pan bool) {
	if id >= 0 && e.PlaySE != nil {
		e.PlaySE(id, x, pan)
	}
}

func (e *Env) effect(id int32, pos [3]float32) *anm.VM {
	if e.Effects == nil {
		return nil
	}
	return e.Effects.Spawn(e.File, int(id), pos, e.Anm)
}

// Pool is the manager.
type Pool struct {
	Slots  [Slots]Bullet
	cursor int // +0x10, where the next slot search starts

	// Rect is the rect clear's box (+0x44 centre, +0x50 size). Its writer is
	// not found yet; it defaults to the playfield.
	Rect struct{ X, Y, W, H float32 }

	layers [Layers][]*Bullet
	// Count is the number of bullets the last Calc kept.
	Count int
}

// NewPool returns an empty manager.
func NewPool() *Pool {
	p := &Pool{}
	p.Rect.Y, p.Rect.W, p.Rect.H = 224, 384, 448
	return p
}

// Live reports whether b's slot is in use.
func (b *Bullet) Live() bool { return b.State != StateFree }

// free is FUN_00405be0.
func (b *Bullet) free() {
	b.State = StateFree
	b.T.Reset()
	b.T2.Reset()
}

// sprite returns the size of b's current sprite, if it has one (the VM's
// sprite pointer at bullet +0x39c; W at sprite +0x34, H at +0x30).
func (b *Bullet) sprite() (w, h float32, ok bool) {
	if b.VM.Sprite < 0 || b.VM.File == nil {
		return 0, 0, false
	}
	s, ok := b.VM.File.Sprites[b.VM.Sprite]
	if !ok {
		return 0, 0, false
	}
	return s.W, s.H, true
}

// Each calls fn for every live slot in slot order.
func (p *Pool) Each(fn func(*Bullet)) {
	for i := range p.Slots {
		if p.Slots[i].State != StateFree {
			fn(&p.Slots[i])
		}
	}
}

// Live counts the live slots.
func (p *Pool) Live() int {
	n := 0
	for i := range p.Slots {
		if p.Slots[i].State != StateFree {
			n++
		}
	}
	return n
}

// EachLayer calls fn for the bullets the last Calc put in draw layer l, in
// the order the binary draws them (FUN_004066e0).
func (p *Pool) EachLayer(l int, fn func(*Bullet)) {
	if l < 0 || l >= Layers {
		return
	}
	for _, b := range p.layers[l] {
		fn(b)
	}
}

// Reset frees every slot and forgets the draw lists, as when a stage starts.
func (p *Pool) Reset() {
	p.Slots = [Slots]Bullet{}
	p.cursor = 0
	for i := range p.layers {
		p.layers[i] = p.layers[i][:0]
	}
	p.Count = 0
}
