package player

import (
	"math"
	"th10/entity/enemy"
	"th10/entity/playerbullet"
	"th10/input"
	"th10/sht"
)

type Option struct{ X, Y float64 }

type reimuAShot struct {
	file        *sht.File
	clock       int // -1 idle; original 0..15 burst clock
	optionCount int
	options     [4]Option
	target      *enemy.Enemy
	// snap is +0x332c and its siblings: the next move puts the options
	// straight on their targets instead of easing them there.
	snap bool
}

// SetHomingCandidates bridges the stage's stable enemy order to the original
// sticky player target. Enemy teardown releases it; live targets are not
// replaced by a fresh nearest-enemy search on every shot.
func (p *Player) SetHomingCandidates(enemies []*enemy.Enemy) {
	if p.reimuA == nil {
		return
	}
	s := p.reimuA
	if s.target != nil && s.target.Active {
		return
	}
	s.target = nil
	for _, e := range enemies {
		if playerbullet.HomingTargetable(e) {
			s.target = e
			break
		}
	}
}

// Options exposes only initialized option state. Draw must not advance it.
// The options only follow the player in state 1, so they are hidden from
// the death to the end of the respawn.
func (p *Player) Options() []Option {
	if p.reimuA == nil || p.State == StateDying || p.State == StateRespawn || p.reimuA.optionCount <= 0 {
		return nil
	}
	return p.reimuA.options[:p.reimuA.optionCount]
}

// updateReimuAShot is FUN_004281d0 and the shot update after it. Outside
// state 1 the target is dropped and the burst clock holds.
func (p *Player) updateReimuAShot(in *input.State) {
	s := p.reimuA
	held := in.IsPressed(input.KeyShot)
	switch {
	case p.NoAttack:
		s.clock, s.target = -1, nil // FUN_004281d0 is skipped and its timer reset
	case p.State != StateNormal:
		s.target = nil
	default:
		p.fireReimuA(held)
	}
	p.Bullets.Update()
}

func (p *Player) fireReimuA(held bool) {
	s := p.reimuA
	if s.clock < 0 && held {
		s.clock = 0
	}
	if s.clock >= 0 {
		for _, shot := range s.file.Shots(p.GS.Power, p.focus) {
			if s.clock%shot.Interval != shot.Delay {
				continue
			}
			x, y := p.X, p.Y
			if shot.Source != 0 {
				option := s.options[shot.Source-1]
				x, y = option.X, option.Y
			}
			p.Bullets.FireSHT(x, y, shot, s.target)
		}
		if s.clock >= 15 {
			if held {
				s.clock -= 15
			} else {
				s.clock = -1
			}
		} else {
			s.clock++
		}
	}
}

// RefreshOptions is FUN_00426f70 for Reimu A: when the option count (one per
// 1.00 power, at most 4) changes, the options are put on their offsets from
// the player and snap there on the next move.
func (p *Player) RefreshOptions() {
	s := p.reimuA
	if s == nil {
		return
	}
	offsets := s.file.OptionOffsets(p.GS.Power, p.focus)
	if len(offsets) == s.optionCount {
		return
	}
	x, y := int(math.Round(p.X*100)), int(math.Round(p.Y*100))
	for i, offset := range offsets {
		s.options[i] = Option{float64(x+offset.X) / 100, float64(y+offset.Y) / 100}
	}
	s.optionCount = len(offsets)
	s.snap = true
}

// updateOptions is the option loop at the end of FUN_004250b0, run by state
// 1 after the move: each option eases 30% of the way to its offset from the
// player, or snaps there. A count change the items made without
// RefreshOptions snaps too.
func (s *reimuAShot) updateOptions(x, y float64, power int, focused bool) {
	offsets := s.file.OptionOffsets(power, focused)
	changed := s.snap || len(offsets) != s.optionCount
	s.snap = false
	for i, offset := range offsets {
		tx, ty := int(math.Round(x*100))+offset.X, int(math.Round(y*100))+offset.Y
		if !changed {
			cx, cy := int(math.Round(s.options[i].X*100)), int(math.Round(s.options[i].Y*100))
			dx, dy := (tx-cx)*30/100, (ty-cy)*30/100
			// FUN_004250b0 snaps only when both fixed-point deltas truncate to 0.
			if dx != 0 || dy != 0 {
				tx, ty = cx+dx, cy+dy
			}
		}
		s.options[i] = Option{float64(tx) / 100, float64(ty) / 100}
	}
	s.optionCount = len(offsets)
}
