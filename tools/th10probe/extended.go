package main

import (
	"fmt"

	"th10/trace"
)

const (
	// DAT_00477818, FUN_0041bb00/FUN_0041afd0.
	itemArrayOff = 0x14
	itemStride   = 0x3f0
	itemLive     = 0x21ceb4
	itemPos      = 0x3ac
	itemClock    = 0x3cc
	itemState    = 0x3dc
	itemKind     = 0x3e0

	laserHead   = 0x18
	laserCount  = 0x438
	laserLineVT = 0x0046da60 // FUN_0041c5b0
	laserInfVT  = 0x0046da10 // FUN_0041c680
)

func (r *reader) sampleExtended(c control, f *trace.Frame, g, p []byte) error {
	if len(p) < playerSize || c.Item == 0 || c.Laser == 0 || c.Spell == 0 || c.Bomb == 0 || c.GUI == 0 {
		return fmt.Errorf("extension managers not ready")
	}
	// FUN_00417040/FUN_00418a00 and the timers in FUN_00425730.
	x := &trace.Extended{
		HiScore: int64(int32(le32(g))) * 10, HiScoreContinues: i32(g[0x54:]),
		FaithHold: i32(g[0x18:]), FaithRate: i32(g[0x10:]), Rank: i32(g[0x58:]),
		PlayerIX: int32(le32(p[0x3cc:])), PlayerIY: int32(le32(p[0x3d0:])),
		StateClock: i32(p[0x478:]), StateClockF: f32(p[0x47c:]), InvincF: f32(p[0x4314:]), Focus: i32(p[0x4474:]),
	}
	f.Ext = x
	shown, err := r.i32(c.GUI + 0x9e78) // FUN_00417040's retained HUD value
	if err != nil {
		return err
	}
	x.ShownScore = int64(shown) * 10
	for slot := 0; slot < 128; slot++ {
		if i32(p[0x49c+slot*0x5c+0x40:]) == 1 {
			f.PBullets++
		}
	}
	bomb, err := r.bytes(c.Bomb+0x14, 0x18) // FUN_00405860/FUN_00405750
	if err != nil {
		return err
	}
	x.BombClock, x.BombClockF = i32(bomb[4:]), f32(bomb[8:])
	x.BombActive = i32(bomb[0x14:]) != 0
	if x.BombActive {
		f.PBombTimer = x.BombClock
	}
	card, err := r.bytes(c.Spell+0x3734, 0x379c-0x3734) // FUN_00409280/FUN_00408d60
	if err != nil {
		return err
	}
	flags := le32(card[0x378c-0x3734:])
	x.Spell = &trace.SpellRow{ID: i32(card[0x3788-0x3734:]), Clock: i32(card[4:]), ClockF: f32(card[8:]),
		Bonus: int64(int32(le32(card[0x3790-0x3734:]))), Initial: i32(card[0x3794-0x3734:]), Duration: i32(card[0x3798-0x3734:]),
		Active: flags&1 != 0, Capture: flags&2 != 0, Survival: flags&8 != 0}
	// The native Live counter immediately follows the complete slot array.
	pool, err := r.block(c.Item+itemArrayOff, trace.MaxItemRows*itemStride+4, &r.items)
	if err != nil {
		return err
	}
	x.ItemCalc = i32(pool[itemLive-itemArrayOff:])
	for slot := 0; slot < trace.MaxItemRows; slot++ {
		b := pool[slot*itemStride:]
		state := i32(b[itemState:])
		if state == 0 {
			continue
		}
		f.Items++
		f.I = append(f.I, trace.ItemRow{Slot: slot, Type: i32(b[itemKind:]), State: state,
			X: f32(b[itemPos:]), Y: f32(b[itemPos+4:]), Clock: i32(b[itemClock:]), ClockF: f32(b[itemClock+4:])})
	}
	manager, err := r.bytes(c.Laser+laserHead, laserCount+4-laserHead)
	if err != nil {
		return err
	}
	x.Lasers = i32(manager[laserCount-laserHead:])
	if x.Lasers < 0 || x.Lasers > trace.MaxLaserRows {
		return fmt.Errorf("laser count out of range: %d", x.Lasers)
	}
	next := le32(manager)
	seen := make(map[uint32]bool)
	for next != 0 {
		if seen[next] || len(seen) >= trace.MaxLaserRows {
			return fmt.Errorf("unstable laser list")
		}
		seen[next] = true
		b, err := r.bytes(next, 0x468) // common fields and template type, never the ANM VMs
		if err != nil {
			return err
		}
		row := trace.LaserRow{ID: i32(b[0x54:]), State: i32(b[0xc:]), Clock: i32(b[0x14:]), ClockF: f32(b[0x18:]),
			X: f32(b[0x24:]), Y: f32(b[0x28:]), Angle: f32(b[0x3c:]), Length: f32(b[0x40:]), Width: f32(b[0x44:])}
		var typeOff int
		switch le32(b) {
		case laserLineVT:
			row.Kind, typeOff = 0, 0x448
		case laserInfVT:
			row.Kind, typeOff = 1, 0x464
		default:
			return fmt.Errorf("unknown laser vtable 0x%08x", le32(b))
		}
		row.Type, row.Color = int(int16(le16(b[typeOff:]))), int(int16(le16(b[typeOff+2:])))
		f.L = append(f.L, row)
		next = le32(b[8:])
	}
	if len(f.L) != x.Lasers {
		return fmt.Errorf("unstable laser count/list: %d/%d", x.Lasers, len(f.L))
	}
	return nil
}
