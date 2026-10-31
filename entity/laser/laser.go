// Package laser is th10.exe's laser manager (DAT_0047781c), ported function
// by function: the allocator FUN_0041c510, the calc FUN_0041c480/FUN_0041c330,
// the clears FUN_0041c850 (all) and FUN_0041c800 (circle), the lookup
// FUN_0040cf90, and the two laser kinds' vtables: the line laser (line.go,
// ECL 412/428/431/433) and the infinite laser (infinite.go, ECL
// 413/429/432/434).
//
// Everything is float32 in game coordinates (x -192..192, y 0..448), with the
// binary's x87 24-bit rounding, as in entity/bullet: products are wrapped in
// float32() so they cannot fuse.
package laser

import (
	"math"

	"th10/anm"
	"th10/zun"
)

// Max is the manager's capacity: FUN_0041c510 refuses a laser while 0x100
// are live.
const Max = 0x100

// States (+0xc). A line laser starts in StateLine; an infinite laser runs
// StateDelay, StateExpand, StateLine (held at full width), StateShrink.
const (
	StateDead   int32 = 1 // deleted by the next calc
	StateLine   int32 = 2
	StateDelay  int32 = 3
	StateExpand int32 = 4
	StateShrink int32 = 5
)

// Effect and sound ids.
const (
	effectGraze = 434 // bullet.anm, the graze spark
	seGraze     = 28
)

// Env is what the lasers read from and do to the rest of the game.
type Env struct {
	// Anm runs the beam and emitter scripts and the sparks.
	Anm *anm.Env
	// File is bullet.anm (manager +0x458).
	File *anm.File
	// Effects is the world list the sparks join.
	Effects *anm.World

	// Player is the player position (DAT_00477834 +0x3c0) and Half its hitbox
	// half-size (+0x41c, +0x420).
	Player [2]float32
	Half   [2]float32

	// Kill is called when the player is inside a beam. It reports whether the
	// player could be hit (not in a dialogue, not dying or respawning, not
	// invulnerable: FUN_004267f0's test) and, if so, kills it (FUN_00426cf0).
	Kill func() bool
	// Graze is called for each graze spark.
	Graze func()
	// PlaySE plays sound id (table 0x4749c8), panned by x when pan is set.
	PlaySE func(id int32, x float32, pan bool)
	// CancelItem spawns the item a cleared segment leaves (FUN_0041bb00
	// type 8).
	CancelItem func(x, y float32)
	// Hold returns the flags at DAT_00477810 +0x58, as for the bullets: bit 0
	// or 2 or bit 10 skips the calc, bit 1 runs it at game speed 0, bit 2
	// skips the draw.
	Hold func() uint32
	// Boss returns boss slot 0's position (enemy +0x1068), which infinite
	// lasers with flag 1 follow.
	Boss func() ([3]float32, bool)
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

func (e *Env) effect(id int32, pos [3]float32) {
	if e.Effects != nil && e.Anm != nil {
		e.Effects.Spawn(e.File, int(id), pos, e.Anm)
	}
}

// cancelItem is FUN_0041bb00(8, -1, -pi/2, 0.6) at pos.
func (e *Env) cancelItem(pos [3]float32) {
	if e.CancelItem != nil {
		e.CancelItem(pos[0], pos[1])
	}
}

// start is FUN_00404f30: restart vm on script id of bullet.anm with the
// positions zeroed. The script's first frame runs now.
func (e *Env) start(vm *anm.VM, id int32) {
	vm.Reset()
	vm.Offset, vm.Pos, vm.Pos2 = [3]float32{}, [3]float32{}, [3]float32{}
	if e.Anm == nil {
		vm.Start(nil, 0, nil)
		return
	}
	vm.Start(e.File, int(id), e.Anm)
}

// run is FUN_0043ee30, one frame of vm.
func (e *Env) run(vm *anm.VM) {
	if e.Anm != nil {
		vm.Run(e.Anm)
	}
}

// outside7a0 is FUN_0041f7a0: whether a box of half-size w*h at pos has
// left the playfield entirely.
func outside7a0(pos [3]float32, w, h float32) bool {
	return !(pos[0]+w > -192 && pos[0]-w < 192 && pos[1]+h > 0 && pos[1]-h < 448)
}

// Common is the part of a laser both kinds share (FUN_0041c030's base).
type Common struct {
	State  int32      // +0xc
	T      zun.Timer  // +0x10, ticked by the calc
	Pos    [3]float32 // +0x24, the tail (where the beam starts)
	Vel    [3]float32 // +0x30
	Angle  float32    // +0x3c
	Length float32    // +0x40
	Width  float32    // +0x44
	Speed  float32    // +0x48
	Dist   float32    // +0x4c, how far the tail has moved; the emitter shows only at 0
	Mark   uint8      // +0x50, set when a cut leaves too little: deleted by the next calc
	ID     int32      // +0x54, the handle ECL looks lasers up by
	SE     int32      // +0x40c, the transform sound

	beam, emitter anm.VM // line +0x600/+0x9ac, infinite +0x61c/+0x9c8
}

// Base returns c, for the Laser interface.
func (c *Common) Base() *Common { return c }

// Beam returns the beam's sprite object.
func (c *Common) Beam() *anm.VM { return &c.beam }

// Emitter returns the emitter's sprite object, or nil once the tail has left
// it (Dist != 0).
func (c *Common) Emitter() *anm.VM {
	if c.Dist != 0 {
		return nil
	}
	return &c.emitter
}

// startVMs is the sprite setup both inits share: the beam on bullet script
// id (drawn additively when add is set) anchored at its bottom centre, and
// the emitter on colour+0x103, additive. Both take interrupt 2 at once.
func (c *Common) startVMs(id, colour int32, add bool, env *Env) {
	env.start(&c.beam, id)
	c.beam.Interrupt(2)
	env.run(&c.beam)
	if add {
		c.beam.Flags = c.beam.Flags&^0x20 | 0x10
	}
	c.beam.Flags = c.beam.Flags&0xfc63ffff | 0x600000
	env.start(&c.emitter, colour+0x103)
	c.emitter.Interrupt(2)
	env.run(&c.emitter)
	c.emitter.Flags = c.emitter.Flags&^0x20 | 0x10
	c.emitter.Flags = c.emitter.Flags&0xfc7fffff | 0x400000
}

// runVMs ends both updates: stretch the beam sprite to width*length (sprite
// W at +0x34, H at +0x30), run it, and run the emitter while Dist is 0.
func (c *Common) runVMs(env *Env) {
	if f := c.beam.File; f != nil && c.beam.Sprite >= 0 {
		if s, ok := f.Sprites[c.beam.Sprite]; ok {
			c.beam.Flags |= 8
			c.beam.Scale[0] = c.Width / s.W
			c.beam.Flags |= 8
			c.beam.Scale[1] = c.Length / s.H
		}
	}
	env.run(&c.beam)
	if c.Dist == 0 {
		env.run(&c.emitter)
	}
}

// hitTest is the beam test both updates share: FUN_004267f0 on the box from
// 0.1 to 0.9 of the length, its half-width hw. A hit cuts the beam round the
// player (vtable +0x18); a graze leaves a spark every fifth frame.
func (c *Common) hitTest(l Laser, m *Manager, hw float32, env *Env) {
	ox, oy := zun.Polar(c.Angle, float32(c.Length*0.1))
	centre := [3]float32{ox + c.Pos[0], oy + c.Pos[1], c.Pos[2]}
	switch hit(env, centre, c.Angle, hw, float32(c.Length*0.8)) {
	case 1:
		player := [3]float32{env.Player[0], env.Player[1], 0}
		l.cut(m, player, [3]float32{32, 32, 0}, false, env)
	case 2:
		if c.T.Cur%5 == 0 {
			env.effect(effectGraze, [3]float32{env.Player[0], env.Player[1], 0})
			env.se(seGraze, c.Pos[0], true)
			if env.Graze != nil {
				env.Graze()
			}
		}
	}
}

// hit is FUN_004267f0: the player's box against the beam box from centre
// along angle, reach long and w wide. 1 for a hit that counts, 2 for a graze
// (1.5 widths either side), 0 otherwise.
func hit(env *Env, centre [3]float32, angle, w, reach float32) int {
	dx := env.Player[0] - centre[0]
	dy := env.Player[1] - centre[1]
	s, co := sincos(-angle)
	u := float32(float32(float64(dx)*co) - float32(s*float64(dy)))
	v := float32(float32(s*float64(dx)) + float32(co*float64(dy)))
	lo := u - env.Half[0]
	vy := v - env.Half[1]
	hi := u + env.Half[0]
	if !(lo <= reach) {
		return 0
	}
	if float32(w*0.5) < vy || hi < 0 || v+env.Half[1] < float32(w*-0.5) {
		if 0 <= hi && vy <= float32(w*1.5) && float32(w*-1.5) <= vy {
			return 2
		}
		return 0
	}
	if env.Kill != nil && env.Kill() {
		return 1
	}
	return 0
}

// sincos is the x87 fsin/fcos pair, kept extended.
func sincos(a float32) (s, c float64) { return math.Sincos(float64(a)) }

// prep is the per-laser part of the draw walk FUN_0041c4e0 (FUN_0041d7c0,
// FUN_0041ea40): the sprites at the tail in screen space, the beam turned
// along the angle.
func (c *Common) prep() {
	c.beam.Pos = [3]float32{c.Pos[0] + 224, c.Pos[1] + 16, c.Pos[2]}
	c.beam.Rot[2] = zun.AddAngle(c.Angle, zun.HalfPi)
	c.beam.Flags |= 4
	if c.Dist == 0 {
		c.emitter.Pos = [3]float32{c.Pos[0] + 224, c.Pos[1] + 16, c.Pos[2]}
	}
}

// Laser is one live laser, a *Line or an *Infinite.
type Laser interface {
	// Base returns the shared fields.
	Base() *Common
	// SetTpl430 is FUN_0040cf60 (ECL 415): write v to laser +0x430, the line
	// template's angle, full length and starting length, or the infinite
	// template's drift velocity.
	SetTpl430(v [3]float32)
	// SetTpl440 is ECL 419: write v to laser +0x440, the line template's width
	// or the infinite template's turn rate.
	SetTpl440(v float32)

	update(m *Manager, env *Env, gs float32) bool // vtable +8
	clear(items bool, env *Env)                   // +0x14
	cut(m *Manager, c, size [3]float32, items bool, env *Env) int
	circle(m *Manager, c [3]float32, r float32, items bool, env *Env) int // +0x1c
}

// Manager is the laser manager.
type Manager struct {
	list   []Laser // list order (+0x18); a deleted node is nil until the calc ends
	count  int     // +0x438
	nextID int32   // +0x43c
	// Centre is the last circle clear's centre (+0x440).
	Centre [3]float32
}

// NewManager returns an empty manager.
func NewManager() *Manager { return &Manager{} }

// Reset drops every laser and restarts the ids.
func (m *Manager) Reset() { *m = Manager{} }

// Count is the number of live lasers.
func (m *Manager) Count() int { return m.count }

// alloc is FUN_0041c510: give l the next id, link it at the tail and
// initialise it. It returns the id, or 0 when the manager is full.
func (m *Manager) alloc(l Laser, init func()) int32 {
	if m.count >= Max {
		return 0
	}
	m.nextID++
	if m.nextID == 0 {
		m.nextID = 1
	}
	l.Base().ID = m.nextID
	m.list = append(m.list, l)
	m.count++
	init()
	return m.nextID
}

// FireLine is FUN_0041c510(0): a line laser from template t.
func (m *Manager) FireLine(t *LineTemplate, env *Env) int32 {
	l := newLine()
	return m.alloc(l, func() { l.init(t, env) })
}

// FireInfinite is FUN_0041c510(1): an infinite laser from template t.
func (m *Manager) FireInfinite(t *InfiniteTemplate, env *Env) int32 {
	l := newInfinite()
	return m.alloc(l, func() { l.init(t, env) })
}

// Find is FUN_0040cf90: the laser with handle id, dead or not, or nil.
func (m *Manager) Find(id int32) Laser {
	for _, l := range m.list {
		if l != nil && l.Base().ID == id {
			return l
		}
	}
	return nil
}

// Calc is FUN_0041c480 / FUN_0041c330: update every laser in list order and
// delete the dead. Each node's successor is read before it runs, so lasers
// fired while the tail runs wait a calc; those fired earlier in the walk run
// in this one. The calc is skipped while Env.Hold has bit 0, 2 or 10 set and
// runs at game speed 0 while it has bit 1 (the sprite scripts still run at
// the normal speed: anm.Env cannot express 0).
//
// The draw prep of FUN_0041c4e0 is done here too, after the walk: it touches
// only the lasers, the draw follows every calc that ran, and nothing writes a
// laser in between.
func (m *Manager) Calc(env *Env) {
	hold := env.hold()
	if (hold>>2|hold)&1 == 0 && hold&0x400 == 0 {
		gs := env.gs()
		if hold&2 != 0 {
			gs = 0
		}
		m.calc(env, gs)
	}
	if hold&4 == 0 {
		for _, l := range m.list {
			if l == nil {
				continue
			}
			if b := l.Base(); b.State != StateDead {
				b.prep()
			}
		}
	}
}

func (m *Manager) calc(env *Env, gs float32) {
	for i := 0; i < len(m.list); i++ {
		l := m.list[i]
		last := i == len(m.list)-1
		b := l.Base()
		dead := false
		if b.Mark != 0 {
			b.Mark++
			dead = b.Mark >= 2
		}
		if !dead {
			if b.State == StateDead || l.update(m, env, gs) {
				dead = true
			} else {
				b.T.Tick(gs)
			}
		}
		if dead {
			m.list[i] = nil
			m.count--
		}
		if last {
			break
		}
	}
	n := 0
	for _, l := range m.list {
		if l != nil {
			m.list[n] = l
			n++
		}
	}
	clear(m.list[n:])
	m.list = m.list[:n]
}

// ClearAll is FUN_0041c850: clear every laser not already dead, leaving a
// spark (and with items an item) on each segment. They are deleted by the
// next calc.
func (m *Manager) ClearAll(items bool, env *Env) {
	for _, l := range m.list {
		if l != nil && l.Base().State != StateDead {
			l.clear(items, env)
		}
	}
}

// CircleClear is FUN_0041c800: cut every laser not already dead where it
// passes within r of c. Pieces left beyond the circle carry on as new line
// lasers; those fired while the tail is cut are not themselves tested. It
// returns the number of segments cut.
func (m *Manager) CircleClear(c [3]float32, r float32, items bool, env *Env) int {
	m.Centre = c
	n := 0
	for i := 0; i < len(m.list); i++ {
		l := m.list[i]
		last := i == len(m.list)-1
		if l != nil && l.Base().State != StateDead {
			n += l.circle(m, c, r, items, env)
		}
		if last {
			break
		}
	}
	return n
}

// Each calls fn with every live laser's sprites in list order (the draw walk
// FUN_0041c4e0); emitter is nil once the tail has left it. Nothing is drawn
// while Env.Hold has bit 2 set.
func (m *Manager) Each(env *Env, fn func(beam, emitter *anm.VM)) {
	if env.hold()&4 != 0 {
		return
	}
	for _, l := range m.list {
		if l == nil {
			continue
		}
		if b := l.Base(); b.State != StateDead {
			fn(b.Beam(), b.Emitter())
		}
	}
}

// walk visits the beam's 12px segments from the tail, the first centred 6px
// along, while the next centre's far edge is within the length (incl: or
// exactly at it). Every walk visits the first segment. z is the segments' z.
func (c *Common) walk(z float32, incl bool, fn func(seg [3]float32)) {
	ux, uy := zun.Polar(c.Angle, 6)
	seg := [3]float32{ux + c.Pos[0], uy + c.Pos[1], z}
	sx, sy := ux+ux, uy+uy
	for k := float32(6); ; {
		fn(seg)
		seg[0] += sx
		seg[1] += sy
		k += 12
		if e := k + 6; !(e < c.Length || incl && e == c.Length) {
			return
		}
	}
}

// mark is the segment pass of the cuts: each segment inside is marked and
// leaves (with items, on screen) an item and then a spark in the colour's
// script. It returns one mark per segment and the number marked.
func (c *Common) mark(colour int32, incl, items bool, env *Env, inside func([3]float32) bool) ([]bool, int) {
	var buf [260]bool // the binary's stack buffer
	marks := buf[:0]
	count := 0
	c.walk(0, incl, func(seg [3]float32) {
		in := inside(seg)
		marks = append(marks, in)
		if !in {
			return
		}
		count++
		if items && !outside7a0(seg, 32, 32) {
			env.cancelItem(seg)
		}
		env.effect(colour*2+0x11, seg)
	})
	return marks, count
}

// inBox is the cuts' test: inside the size box centred on p, edges included.
func inBox(p, size [3]float32) func([3]float32) bool {
	var lo, hi [3]float32
	for k := range 3 {
		lo[k] = p[k] - float32(size[k]*0.5)
		hi[k] = float32(size[k]*0.5) + p[k]
	}
	return func(s [3]float32) bool {
		return lo[0] <= s[0] && s[0] <= hi[0] && lo[1] <= s[1] && s[1] <= hi[1]
	}
}

// inCircle is the circle cuts' test: within r of p.
func inCircle(p [3]float32, r float32) func([3]float32) bool {
	rr := float32(r * r)
	return func(s [3]float32) bool {
		dx, dy := p[0]-s[0], p[1]-s[1]
		return float32(dy*dy)+float32(dx*dx) <= rr
	}
}
