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
func (p *Player) Options() []Option {
	if p.reimuA == nil || p.State == StateDead || p.reimuA.optionCount <= 0 {
		return nil
	}
	return p.reimuA.options[:p.reimuA.optionCount]
}

func (p *Player) updateReimuAShot(in *input.State) {
	s := p.reimuA
	if p.State == StateDead {
		s.target = nil
		s.optionCount = -1
		p.Bullets.Update()
		return
	}
	focused := in.IsPressed(input.KeyFocus)
	s.updateOptions(p.X, p.Y, p.GS.Power, focused)
	held := in.IsPressed(input.KeyShot)
	if s.clock < 0 && held {
		s.clock = 0
	}
	if s.clock >= 0 {
		for _, shot := range s.file.Shots(p.GS.Power, focused) {
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
	p.Bullets.Update()
}

func (s *reimuAShot) updateOptions(x, y float64, power int, focused bool) {
	offsets := s.file.OptionOffsets(power, focused)
	changed := len(offsets) != s.optionCount
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
