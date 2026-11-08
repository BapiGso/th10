package player

import (
	"th10/anm"
	"th10/game"
	"th10/sht"
)

// Option is the diagnostic screen-coordinate view, not an animation object.
type Option struct{ X, Y float64 }

// option is O = player+0x32a0+0x98*k (FUN_00426f70/FUN_004250b0).
// Positions and both saved offsets are fixed-point game coordinates.
type option struct {
	state       int32
	target, pos [2]int32 // +0x34/+0x38, +0x3c/+0x40
	fast, slow  [2]int32 // +0x44/+0x48, +0x4c/+0x50
	vm, extra   *anm.VM  // +0x68/+0x6c
	lastFocus   bool     // +0x84
	index       int32    // +0x88
	snap        bool     // +0x8c
}

// Options preserves the existing inspection API. Drawing uses the World
// handles at layer 8 instead (FUN_00448980, draw priority 0x15).
func (p *Player) Options() []Option {
	if p.State == StateDying || p.State == StateRespawn || p.optionCount <= 0 {
		return nil
	}
	for i := range p.optionCount {
		o := p.options[i].pos
		p.optionView[i] = Option{p.originX() + float64(o[0])/100, p.FieldTop + float64(o[1])/100}
	}
	return p.optionView[:p.optionCount]
}

// FUN_00427e90, 0x427f43-0x427f8c: option muzzles use fixed*0.01f,
// not the diagnostic view's decimal conversion.
func (p *Player) optionScreen(i int) (float64, float64) {
	o := p.options[i].pos
	return p.originX() + float64(float32(o[0])*0.01), p.FieldTop + float64(float32(o[1])*0.01)
}

func fixedOffset(o sht.Offset) [2]int32 { return [2]int32{int32(o.X), int32(o.Y)} }

// RefreshOptions is FUN_00426f70. The full-power effects are refreshed
// before the count-equality return (0x426fa9-0x427181).
func (p *Player) RefreshOptions() {
	if p.shotFile == nil {
		return
	}
	if p.anmAttached() {
		for i := range p.options {
			o := &p.options[i]
			if p.GS.Power >= 500 { // raw DAT_00474c48 >= 100 (0x426f94)
				p.world.Delete(o.extra)
				o.extra = p.world.SpawnFront(p.Anm, 0x14+p.GS.ShotType, 15, p.env) // 0x42702f-0x4270db
			} else {
				p.world.Interrupt(o.extra, 1) // 0x4270f7-0x427160; retain the exiting handle
			}
		}
	}
	n := sht.PowerLevel(p.GS.Power)
	if n == p.optionCount {
		return
	}
	fast, slow := p.shotFile.OptionOffsets(p.GS.Power, false), p.shotFile.OptionOffsets(p.GS.Power, true)
	for i := range n {
		o := &p.options[i]
		o.pos = [2]int32{p.IX, p.IY} // 0x4271b1-0x4271bd
		if p.anmAttached() {
			p.world.Delete(o.vm) // 0x427206-0x42723f, including its child chain
		}
		o.vm = nil
		o.index = int32(i)
		switch {
		case p.GS.Character == game.CharMarisa && p.GS.ShotType == game.ShotA:
			// 0x42749b-0x427648: the unfocused positions trail the 33-entry
			// player history; focus freezes their offsets to the player.
			if !p.focus {
				o.fast = fixedOffset(fast[i])
			}
			o.slow = fixedOffset(slow[i])
			o.target = p.positionHistory[8*(i+1)]
			if p.focus && o.state == 0 {
				if i == 0 {
					o.fast = fixedOffset(slow[0])
				} else {
					o.fast = p.options[i-1].fast
				}
			}
			o.pos = p.positionHistory[8*i]
		case p.GS.Character == game.CharMarisa && p.GS.ShotType == game.ShotC:
			// 0x427280-0x42730b: slow is an absolute saved home here,
			// unlike the SHT focus offsets of the other four layouts.
			o.fast = fixedOffset(fast[i])
			if !p.focus || o.state == 0 {
				o.target = [2]int32{p.IX + o.fast[0], p.IY + o.fast[1]}
				o.slow = o.target
			}
		default: // Reimu A/B/C: 0x42764e-0x42782f; Marisa B: 0x427381-0x427496
			o.fast, o.slow = fixedOffset(fast[i]), fixedOffset(slow[i])
			off := o.fast
			if p.focus {
				off = o.slow
			}
			o.target = [2]int32{p.IX + off[0], p.IY + off[1]}
			if p.GS.Character == game.CharReimu {
				o.pos = o.target // 0x42771a-0x427720
			}
		}
		if p.anmAttached() {
			// FUN_00449870/FUN_004489d0: pass 15; script op 68 sets layer 8.
			o.vm = p.world.SpawnTail(p.Anm, 0x11+p.GS.ShotType, 15, p.env)
			if p.GS.Character == game.CharMarisa && p.GS.ShotType == game.ShotC && p.focus {
				p.world.Interrupt(o.vm, 3) // 0x427358-0x42736d
			}
		}
		o.state = 2 // 0x42783a
	}
	for i := n; i < len(p.options); i++ {
		o := &p.options[i]
		o.state = 0
		if p.anmAttached() {
			p.world.Interrupt(o.vm, 1) // 0x427873-0x4278f0
		}
	}
	p.optionCount = n
	p.snapOptions() // 0x42790c-0x427920
}

func (p *Player) snapOptions() {
	for i := range p.options {
		p.options[i].snap = true
	}
}

// FUN_004269d0, 0x426b0b-0x426c03: let both scripts exit through label 1,
// rather than deleting their World objects when the logical count is zero.
func (p *Player) clearOptions() {
	for i := range p.options {
		o := &p.options[i]
		o.state = 0
		if p.anmAttached() {
			p.world.Interrupt(o.vm, 1)
			p.world.Interrupt(o.extra, 1)
		}
	}
	p.optionCount = 0
}

// FUN_004250b0, 0x425521-0x4255b3: only unfocused raw movement shifts
// history; entries beyond the last option repeat its terminal sample.
func (p *Player) updatePositionHistory(moved bool) {
	if !p.focus && moved {
		p.advancePositionHistory()
		return
	}
	p.positionHistory[0] = [2]int32{p.IX, p.IY} // 0x4255a1-0x4255b3
}

// advancePositionHistory is the same queue in FUN_004250b0 (0x425532-
// 0x4255b3) and FUN_00425730 state 0 (0x4257b3-0x425833). Respawn advances
// it unconditionally, including focused/stationary frames.
func (p *Player) advancePositionHistory() {
	end := max(0, min(p.optionCount*8, 32))
	for i := 32; i > end; i-- {
		p.positionHistory[i] = p.positionHistory[end]
	}
	for i := end; i > 0; i-- {
		p.positionHistory[i] = p.positionHistory[i-1]
	}
	p.positionHistory[0] = [2]int32{p.IX, p.IY}
}

// FUN_00427960 (Marisa A's O+0x90 callback). The x87 keeps an interpolation
// weight on its stack and adds 0.125 after each x/y pair (0x427a77), so the
// seven weights are 1/8..7/8, not seven copies of 1/8.
func (p *Player) trailOption(o *option) {
	k := int(o.index) * 8
	home := p.positionHistory[k+8]
	if !p.focus {
		o.fast = [2]int32{home[0] - p.IX, home[1] - p.IY} // 0x42799a-0x4279b1
	} else {
		home = [2]int32{p.IX + o.fast[0], p.IY + o.fast[1]}
		p.positionHistory[k+8] = home // 0x4279de-0x4279e7
		weight := float32(0.125)      // DAT_00470b90 = 0x3e000000
		for j := 1; j <= 7; j++ {
			for axis := range 2 {
				base := p.positionHistory[k][axis]
				delta := float32(home[axis] - base)
				p.positionHistory[k+j][axis] = int32(float32(delta*weight) + float32(base))
			}
			weight += 0.125
		}
	}
	o.target = [2]int32{p.IX + o.fast[0], p.IY + o.fast[1]} // 0x427a9e-0x427ab8
	o.lastFocus = p.focus
}

// FUN_00427ae0 (Marisa C's O+0x90 callback), not a draw callback.
func (p *Player) lockOption(o *option) {
	if p.focus {
		if !o.lastFocus && p.anmAttached() {
			p.world.Interrupt(o.vm, 3) // 0x427b1d-0x427b29
		}
		o.target = o.slow // 0x427b2e-0x427b37, saved absolute position
	} else {
		if o.lastFocus && p.anmAttached() {
			p.world.Interrupt(o.vm, 6) // 0x427af5-0x427b03
		}
		o.slow = o.pos // 0x427b08-0x427b11
	}
	o.lastFocus = p.focus
}

// FUN_004250b0, 0x4255b9-0x4256d8: callback, fixed-point easing, then Place
// both handles. A nil ANM attachment leaves all logical positions usable.
func (p *Player) updateOptions(moved bool) {
	if sht.PowerLevel(p.GS.Power) != p.optionCount {
		p.RefreshOptions()
	}
	p.updatePositionHistory(moved)
	for i := range p.options {
		o := &p.options[i]
		if o.state == 0 {
			continue
		}
		off := o.fast
		if p.focus {
			off = o.slow
		}
		o.target = [2]int32{p.IX + off[0], p.IY + off[1]}
		if p.GS.Character == game.CharMarisa {
			switch p.GS.ShotType {
			case game.ShotA:
				p.trailOption(o)
			case game.ShotC:
				p.lockOption(o)
			}
		}
		if o.snap {
			o.snap, o.pos = false, o.target
		} else if p.optionEase >= 30 { // 0x425610-0x425687
			dx := (o.target[0] - o.pos[0]) * p.optionEase / 100
			dy := (o.target[1] - o.pos[1]) * p.optionEase / 100
			if dx == 0 && dy == 0 {
				o.pos = o.target
			} else {
				o.pos[0] += dx
				o.pos[1] += dy
			}
		}
		if p.anmAttached() {
			pos := [3]float32{float32(o.pos[0]) * 0.01, float32(o.pos[1]) * 0.01, 0}
			p.world.Place(o.vm, pos)    // 0x4256b9
			p.world.Place(o.extra, pos) // 0x4256c4
		}
	}
}
