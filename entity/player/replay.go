package player

import (
	"encoding/binary"

	"th10/game"
)

// ReplayFocus reads player +0x4474 for FUN_0042a450's chapter snapshot.
func (p *Player) ReplayFocus() int32 {
	if p.focus {
		return 1
	}
	return 0
}

// RestoreReplayFocus is the +0x4474 write at 0x42a5e9. This does not skip
// the normal initialization gate in FUN_004250b0 on subsequent frames.
func (p *Player) RestoreReplayFocus(focus int32) { p.focus = focus != 0 }

// CaptureReplayPlayer is FUN_0042a450, 0x42a512-0x42a56a. The header holds
// 33 history pairs, then four column-major arrays of four option pairs:
// targets, current positions, fast offsets and slow offsets/homes.
// Position and focus are written by the stage's existing typed fields.
func (p *Player) CaptureReplayPlayer(header []byte) {
	if len(header) < 0x1c4 {
		return
	}
	put := func(at int, pair [2]int32) {
		binary.LittleEndian.PutUint32(header[at:], uint32(pair[0]))
		binary.LittleEndian.PutUint32(header[at+4:], uint32(pair[1]))
	}
	for i, pair := range p.positionHistory {
		put(0x2c+i*8, pair) // REP MOVSD of 0x42 words, 0x42a51b
	}
	for i, o := range p.options {
		put(0x134+i*8, o.target) // 0x42a533-0x42a53c
		put(0x154+i*8, o.pos)    // 0x42a53f-0x42a547
		put(0x174+i*8, o.fast)   // 0x42a549-0x42a552
		put(0x194+i*8, o.slow)   // 0x42a555-0x42a55e
	}
}

// RestoreReplayPlayer is FUN_0042a450, 0x42a5d2-0x42a687. The stage has
// restored Power, fixed position and focus first. Rebuild options after
// history/focus but before copying the four pair arrays; no initial snap
// may overwrite those restored current positions on the first move.
func (p *Player) RestoreReplayPlayer(header []byte) {
	get := func(at int) [2]int32 {
		return [2]int32{int32(binary.LittleEndian.Uint32(header[at:])), int32(binary.LittleEndian.Uint32(header[at+4:]))}
	}
	if len(header) >= 0x1c4 {
		for i := range p.positionHistory {
			p.positionHistory[i] = get(0x2c + i*8) // 0x42a5d2-0x42a5e0
		}
	}
	p.RefreshOptions() // 0x42a5ef; also valid for a synthetic Chapter without Raw
	if len(header) >= 0x1c4 {
		for i := range p.options {
			o := &p.options[i]
			o.target = get(0x134 + i*8)
			o.pos = get(0x154 + i*8)
			o.fast = get(0x174 + i*8)
			o.slow = get(0x194 + i*8)
			if p.GS.Character == game.CharMarisa && p.GS.ShotType == game.ShotC {
				o.slow = o.target // 0x42a650-0x42a65e: saved absolute lock home
			}
		}
	}
	for i := range p.options {
		p.options[i].snap = false // 0x42a66f-0x42a681
	}
	// FUN_00424d90 (0x42a687): initialize only the state/burst/init clocks,
	// clear the old focus marker, and leave the invincibility clock alone.
	p.State = StateNormal
	p.StateTimer.Reset()
	p.initTimer.Reset()
	if p.shot != nil {
		p.shot.timer.Set(-1)
	}
	p.world.Delete(p.hitbox) // FUN_00424d90, 0x424e78-0x424e8f
	p.hitbox = nil
}
