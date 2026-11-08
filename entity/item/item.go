// Package item is the item manager DAT_00477818: FUN_0041bb00 spawns,
// FUN_0041afd0 moves and collects, FUN_0041ba30 draws. Positions are game
// coordinates, float32 as the x87 rounds them at 24-bit precision.
package item

import (
	"th10/anm"
	"th10/game"
	"th10/zun"
)

// Slot counts: the dropped items (+0x14), then the cancel items (+0x24eb4),
// laid out back to back with a stride of 0x3f0.
const (
	NormalSlots = 150
	CancelSlots = 2048
)

// Kinds (+0x3e0). 6 is unused.
const (
	KindPower        int32 = 1  // 0.05 power
	KindPoint        int32 = 2  // score by faith, less below the line
	KindBigFaith     int32 = 3  // faith by difficulty
	KindBigPower     int32 = 4  // 1.00 power
	KindFullPoint    int32 = 5  // a point item at full value
	KindLife         int32 = 7  // an extra life
	KindCancel       int32 = 8  // a cancelled bullet's item
	KindFaith        int32 = 9  // a little faith
	KindBossPower    int32 = 10 // the boss's 0.05 power, looks like 1
	KindBossBigPower int32 = 11 // the boss's 1.00 power, looks like 4
)

// States (+0x3dc).
const (
	StateFree    int32 = iota
	StateFall          // tossed, then falling
	StateRise          // a cancel item rising
	StateAuto          // flying to the player at the full homing speed
	StateAttract       // flying to the player at a third of it
	StateDelay         // a cancel item waiting to appear
)

// Player states FUN_0041afd0 tests (DAT_00477834 +0x458).
const (
	playerDying     = 2
	playerDeathbomb = 4
)

// Sound effects by original id.
const (
	seItem   = 0x14
	seLevel  = 0x1d
	seExtend = 0x2c
)

// Item is one slot.
type Item struct {
	VM      anm.VM     // +0x0
	Pos     [3]float32 // +0x3ac
	Vel     [3]float32 // +0x3b8
	T       zun.Timer  // +0x3c8
	State   int32      // +0x3dc
	Kind    int32      // +0x3e0
	Display int32      // +0x3e4: the look (sprites 0x157+, arrows 0x160+)
	Speed   float32    // +0x3e8: the homing speed
	Delay   int32      // +0x3ec: StateDelay's countdown
}

// Manager is the item manager.
type Manager struct {
	Items   [NormalSlots + CancelSlots]Item
	Live    int32 // +0x21ceb4: the items that ran last update
	cursor  int32 // +0x21ceb8: the next cancel slot
	spawned int32 // +0x21cebc: cancel items spawned since the last update
}

// NewManager returns a manager with every slot free.
func NewManager() *Manager { return &Manager{} }

// Player is what FUN_0041afd0 reads of the player (DAT_00477834) and the
// input, as they stand after the player's update.
type Player struct {
	Pos     [2]float32 // +0x3c0
	State   int32      // +0x458
	Speed   float32    // SHT +8: the homing speed
	Collect [4]float32 // +0x4324: left, top, right, bottom
	Focus   [4]float32 // +0x433c: the attract box while focused
	Unfocus [4]float32 // +0x4354: the attract box otherwise
	Focused bool       // DAT_00474e5c bit 2
	Bombing bool       // a bomb is active (DAT_004776ec +0x28)
}

// Per-character sizes by DAT_00474c68 (FUN_004247f0): the homing speed
// written over SHT +8, the attract and the collect sizes.
var (
	homingSpeed = [...]float32{5, 7}     // 0x476fa0
	attractSize = [...]float32{100, 118} // 0x476fa8
	collectSize = [...]float32{40, 48}   // 0x476fb0
)

// NewPlayer is the view of character char at (x, y). The boxes are the tail
// of FUN_00425730: +0x428 is half the collect size, +0x434 half the attract
// size, and the collect box is half +0x428 round the player.
func NewPlayer(char int, x, y float32, state int32, focused, bombing bool) Player {
	c := float32(collectSize[char] * 0.5)
	a := float32(attractSize[char] * 0.5)
	h := float32(c * 0.5)
	return Player{
		Pos:     [2]float32{x, y},
		State:   state,
		Speed:   homingSpeed[char],
		Collect: [4]float32{x - h, y - h, x + h, y + h},
		Focus:   [4]float32{x - a, y - a, x + a, y + a},
		Unfocus: [4]float32{x - c, y - c, x + c, y + c},
		Focused: focused,
		Bombing: bombing,
	}
}

// Env is what the manager reads and calls outside itself.
type Env struct {
	State   *game.GameState
	Anm     *anm.Env
	File    *anm.File  // bullet.anm (+0x3e0b50)
	Effects *anm.World // free-standing 0x189 effects, FUN_0041bb00/FUN_0041ba50
	Player  Player
	// PlaySE is a centred sound (FUN_0043dc90); PlaySEAt is FUN_0043dd10.
	PlaySE   func(id int32)
	PlaySEAt func(id int32, x float32)
	// Options is FUN_00426f70: the player's options after a power level.
	Options func()
	// Popup is FUN_0042b9c0: game position, integer value, original ARGB.
	Popup func(pos [3]float32, value int32, color uint32)
}

func (e *Env) gameSpeed() float32 {
	if e.Anm == nil {
		return 1
	}
	return e.Anm.GameSpeed
}

func (e *Env) playSE(id int32) {
	if e.PlaySE != nil {
		e.PlaySE(id)
	}
}

func (e *Env) playSEAt(id int32, x float32) {
	if e.PlaySEAt != nil {
		e.PlaySEAt(id, x)
	} else {
		e.playSE(id)
	}
}

func (e *Env) popup(it *Item, value int32, color uint32) {
	if e.Popup != nil {
		e.Popup(it.Pos, value, color)
	}
}

// faithEffect is FUN_00448db0 at 0x41bc42/0x41badd. EDI supplies the item
// position; the caller's stack argument is an output handle, not a vector.
// Script 0x189 selects layer 11 and uses no RNG, despite Spawn's reset
// clearing the initially selected AltRNG flag.
func (e *Env) faithEffect(pos [3]float32) {
	if e.Effects != nil && e.Anm != nil && e.File != nil {
		e.Effects.Spawn(e.File, 0x189, pos, e.Anm)
	}
}

// start is FUN_00404f30: restart the item's script id of bullet.anm with the
// positions zeroed. The script's first frame runs now.
func (it *Item) start(env *Env, id int32) {
	it.VM.Reset()
	it.VM.Offset, it.VM.Pos, it.VM.Pos2 = [3]float32{}, [3]float32{}, [3]float32{}
	if env.Anm == nil {
		it.VM.Start(nil, 0, nil)
		return
	}
	it.VM.Start(env.File, int(id), env.Anm)
}

// Spawn is FUN_0041bb00: an item of kind at pos, launched at angle and speed.
// A cancel item takes the next cancel slot round, if it is free, and waits
// longer the more have spawned since the last update; any other kind takes
// the first free dropped slot, x held to the playfield. At full power the
// power items come out as faith items, the boss's as full point items. The
// binary also stores the caller's colour at +0x2fc (not ported). Effect
// 0x189 depends on the slot's previous kind, not the new kind (0x41bc23).
func (m *Manager) Spawn(env *Env, pos [3]float32, kind int32, angle, speed float32) {
	if kind == KindCancel {
		m.spawned++
		c := m.cursor
		if it := &m.Items[NormalSlots+c]; it.State == StateFree {
			switch {
			case m.spawned < 0x100:
				it.Delay = c & 3
			case m.spawned < 0x200:
				it.Delay = c&7 + 4
			case m.spawned < 0x400:
				it.Delay = c&15 + 8
			default:
				it.Delay = c&31 + 16
			}
			it.Kind, it.Display, it.State = KindCancel, KindCancel, StateDelay
			it.Pos = pos
			it.Vel[0], it.Vel[1] = zun.Polar(angle, speed)
			it.Vel[2] = 0
			it.T.Reset()
		}
		m.cursor = (c + 1) & (CancelSlots - 1)
		return
	}
	var it *Item
	for i := range NormalSlots {
		if m.Items[i].State == StateFree {
			it = &m.Items[i]
			break
		}
	}
	if it == nil {
		return
	}
	it.State = StateFall
	it.Pos = pos
	if it.Pos[0] < -192 {
		it.Pos[0] = -192
	} else if it.Pos[0] >= 192 {
		it.Pos[0] = 192
	}
	it.Vel[0], it.Vel[1] = zun.Polar(angle, speed)
	it.Vel[2] = 0
	it.T.Reset()
	if env.State != nil && env.State.Power/5 > 99 {
		switch kind {
		case KindPower, KindBigPower:
			kind = KindFaith
		case KindBossPower, KindBossBigPower:
			kind = KindFullPoint
		}
	}
	if it.Kind == KindBigFaith { // FUN_0041bb00, 0x41bc23-0x41bc42, before overwriting +0x3e0
		env.faithEffect(it.Pos)
	}
	it.Kind, it.Display = kind, kind
	script := kind + 0x176
	switch kind {
	case KindBossPower:
		it.Display, script = KindPower, 0x177
	case KindBossBigPower:
		it.Display, script = KindBigPower, 0x17a
	}
	it.start(env, script)
}

// move adds a frame of velocity at game speed gs.
func (it *Item) move(gs float32) {
	vy, vz := float32(gs*it.Vel[1]), float32(gs*it.Vel[2])
	it.Pos[0] = float32(gs*it.Vel[0]) + it.Pos[0]
	it.Pos[1] = vy + it.Pos[1]
	it.Pos[2] = vz + it.Pos[2]
}

// fall is the gravity: 0.03 a frame.
func (it *Item) fall(gs float32) {
	it.Vel[1] = float32(gs*0.03) + it.Vel[1]
}

// home is a frame of flight at the player, the speed gaining 0.2 a frame up
// to 12. The deathbomb window drops the item where it is.
func (it *Item) home(p *Player, gs float32) {
	a := zun.Aim(it.Pos[0], it.Pos[1], p.Pos[0], p.Pos[1])
	it.Vel[0], it.Vel[1] = zun.Polar(a, it.Speed)
	it.move(gs)
	if it.Speed < 12 {
		it.Speed += 0.2
	}
	if p.State == playerDeathbomb {
		it.State = StateFall
		it.Vel[0], it.Vel[1] = 0, 0
	}
}

func inside(box [4]float32, pos [3]float32) bool {
	return box[0] <= pos[0] && box[1] <= pos[1] && pos[0] <= box[2] && pos[1] <= box[3]
}

// Update is FUN_0041afd0, a frame of every item in slot order. A cancel item
// counts down its delay and appears rising. A falling item flies to the
// player above the collection line (y 128) or during a bomb, unless the
// player is dying or in the deathbomb window; otherwise it falls, sideways
// motion stopping once it is going down, at most 2 a frame, and is lost past
// y 472. A rising item flies to the player once it stops rising; one lost
// below costs 4 rank. Unless the player is dying, an item in the collect box
// is collected, and a falling or rising one in the attract box (larger while
// focused) flies at a third of the homing speed. An item reaching full power
// ends the update with FUN_0041ba50.
func (m *Manager) Update(env *Env) {
	gs := env.gameSpeed()
	p := &env.Player
	m.spawned, m.Live = 0, 0
	full := false
	for i := range m.Items {
		it := &m.Items[i]
		switch it.State {
		case StateFree:
			continue
		case StateDelay:
			it.Delay--
			if it.Delay < 0 {
				it.State = StateRise
				it.start(env, it.Kind+0x176)
			}
			continue
		case StateFall:
			if p.State != playerDying && p.State != playerDeathbomb && p.Pos[1] < 128 || p.Bombing {
				it.Speed, it.State = p.Speed, StateAuto
				it.home(p, gs)
				break
			}
			it.move(gs)
			it.fall(gs)
			if it.Vel[1] >= 0 {
				it.Vel[0] = 0
			}
			if it.Vel[1] > 2 {
				it.Vel[1] = 2
			}
			if it.Pos[1] > 472 {
				it.State = StateFree
				continue
			}
		case StateRise:
			it.move(gs)
			it.fall(gs)
			if it.Vel[1] >= 0 {
				it.Speed, it.State = p.Speed, StateAuto
				it.home(p, gs)
				break
			}
			if it.Pos[1] > 472 {
				it.State = StateFree
				env.State.AddRank(-4)
				continue
			}
		case StateAuto, StateAttract:
			it.home(p, gs)
		}
		if p.State != playerDying {
			if inside(p.Collect, it.Pos) {
				if m.collect(env, it) {
					full = true
				}
				it.State = StateFree
				env.playSEAt(seItem, it.Pos[0]) // FUN_0041afd0, 0x41b718..0x41b72d.
				continue
			}
			if it.State != StateAttract && it.State != StateAuto {
				box := p.Unfocus
				if p.Focused {
					box = p.Focus
				}
				if inside(box, it.Pos) {
					it.State, it.Speed = StateAttract, float32(p.Speed*0.33333334)
				}
			}
		}
		if env.Anm != nil {
			it.VM.Run(env.Anm)
		}
		it.T.Tick(gs)
		m.Live++
	}
	if full {
		m.fullPower(env)
	}
}

// collect is the reward for collecting it, reporting whether it brought the
// power to full. Each kind also adds to the faith gauge's hold timer through
// FUN_00412ff0 (AddFaithHold), as FUN_0041afd0 does at the same branch.
// Score popups are presentation-only hooks; the GUI observes Power for
// FUN_004054b0 without changing these collection/resource rules.
func (m *Manager) collect(env *Env, it *Item) (full bool) {
	st := env.State
	faith := int32(st.Faith / 10) // DAT_00474c4c; GameState keeps it x10
	switch it.Kind {
	case KindPower, KindBossPower:
		if st.GainPower(1) {
			m.levelUp(env, it.Pos[0])
			full = st.Power/5 > 99
			st.AddRank(0xc)
			env.popup(it, -1, 0xffffff40) // FUN_0041afd0 0x41b433: power level
		} else {
			env.popup(it, int32(st.Power/10), 0xffff4040) // 0x41b494: stored Power/2
		}
		st.AddFaithHold(60) // FUN_00412ff0(0x3c), both branches (0x41b473, 0x41b4a0)
	case KindBigPower, KindBossBigPower:
		if st.GainPower(20) {
			m.levelUp(env, it.Pos[0])
			st.AddRank(0x18)
			full = st.Power/5 > 99
			env.popup(it, -1, 0xffffff40) // 0x41b69f
		} else {
			env.popup(it, int32(st.Power/10), 0xffff4040) // 0x41b6ea
		}
		st.AddFaithHold(20) // FUN_00412ff0(0x14) (0x41b6cc, 0x41b6f6)
	case KindPoint:
		if env.Player.Pos[1] < 144 {
			st.AddRank(8)
			st.AddPoints(int64(faith * 10))
			st.AddFaithHold(100)                // FUN_00412ff0(100) at LAB_0041b559 (0x41b575)
			env.popup(it, faith*10, 0xffffff00) // 0x41b54f
			break
		}
		// Below the line it is half the value, less (y-144)/304 of
		// faith/2-5000 (0x41b4d5): more than half while faith is under 10000.
		d := float32(float32(env.Player.Pos[1]-144)*(1.0/304)) * float32(float32(float32(faith)*0.5)-5000)
		v := faith*10/2 - int32(d)
		v -= v % 10
		st.AddRank(1)
		st.AddPoints(int64(v))
		st.AddFaithHold(100)         // FUN_00412ff0(100) at LAB_0041b559 (0x41b575)
		env.popup(it, v, 0xffffffff) // 0x41b52b, below the line
	case KindBigFaith:
		n := int64(5000)
		switch st.Difficulty {
		case game.DiffHard:
			n = 8000
		case game.DiffLuna, game.DiffExtra:
			n = 10000
		}
		st.GainFaith(n)
		env.popup(it, int32(n), 0xff00ff00) // 0x41b5b5
		st.AddFaithHold(120)                // FUN_00412ff0(0x78) (0x41b5cc)
	case KindFullPoint:
		st.AddRank(8)
		st.AddPoints(int64(faith * 10))
		st.AddFaithHold(100)                // FUN_00412ff0(100) at LAB_0041b559 (0x41b575)
		env.popup(it, faith*10, 0xffffff00) // 0x41b54f
	case KindLife: // FUN_004188a0
		st.Life++
		if st.Life < 10 {
			env.playSE(seExtend)
		} else {
			st.Life = 9
		}
		st.AddRank(0x100)
	case KindCancel:
		st.GainFaith(10)
		st.AddPoints(10)
		st.AddFaithHold(3) // FUN_00412ff0(3) (0x41b5e9)
	case KindFaith:
		st.GainFaith(100)
		env.popup(it, 100, 0xff00ff00) // 0x41b60c
		st.AddFaithHold(60)            // FUN_00412ff0(0x3c) (0x41b624)
	}
	return full
}

// levelUp is a power level gained: the options (FUN_00426f70) and its sound.
func (m *Manager) levelUp(env *Env, x float32) {
	if env.Options != nil {
		env.Options()
	}
	env.playSEAt(seLevel, x) // FUN_0041afd0, 0x41b446/0x41b68c: the item's x.
}

// fullPower is FUN_0041ba50: every dropped power item still out (kinds 1 and
// 4; the boss's are spared) turns into a faith item tossed up from where it
// is, then spawns one script 0x189 at that item's position.
func (m *Manager) fullPower(env *Env) {
	for i := range NormalSlots {
		it := &m.Items[i]
		if it.State != StateFree && (it.Kind == KindPower || it.Kind == KindBigPower) {
			pos := it.Pos
			it.State = StateFree
			m.Spawn(env, pos, KindFaith, -zun.HalfPi, 2.2)
			env.faithEffect(pos) // FUN_0041ba50, 0x41bad2-0x41badd, after FUN_0041bb00
		}
	}
}

// Draw is FUN_0041b8e0, FUN_0041ba30's body: every item not free in slot
// order, its sprite placed on the playfield and drawn by draw. An item above
// y -8 shows as its arrow at the top edge, fading out as it falls back to the
// edge. Its look and alpha are set every frame: the test against the sprite
// already set is one off and never holds.
func (m *Manager) Draw(draw func(vm *anm.VM)) {
	for i := range m.Items {
		it := &m.Items[i]
		if it.State == StateFree {
			continue
		}
		vm := &it.VM
		vm.Pos = [3]float32{it.Pos[0] + 224, it.Pos[1] + 16, it.Pos[2]}
		if !(vm.Pos[1] < 8) {
			if vm.Sprite != int(it.Display)+0x158 {
				vm.SetSprite(int(it.Display) + 0x157)
				vm.Alpha = 0xff
			}
		} else {
			d := vm.Pos[1] - 8
			vm.Pos[1] = 24
			if d < 32 {
				vm.Alpha = uint8(int32(float32(d*0.03125) * 255))
			} else {
				vm.Alpha = 0xff
			}
			if vm.Sprite != int(it.Display)+0x161 {
				vm.SetSprite(int(it.Display) + 0x160)
			}
		}
		draw(vm)
	}
}

// Each calls fn for every item not free, in slot order.
func (m *Manager) Each(fn func(it *Item)) {
	for i := range m.Items {
		if m.Items[i].State != StateFree {
			fn(&m.Items[i])
		}
	}
}

// ActiveCount is the number of items not free.
func (m *Manager) ActiveCount() int {
	n := 0
	for i := range m.Items {
		if m.Items[i].State != StateFree {
			n++
		}
	}
	return n
}
