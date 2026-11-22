package player

import (
	"th10/entity/enemy"
	"th10/entity/playerbullet"
	"th10/input"
	"th10/zun"
)

// shotState is the common SHT burst state of all six shot types
// (FUN_004281d0), held in Player.shot.
type shotState struct {
	timer  zun.Timer // P+0x460: -1 idle, otherwise the original 0..15 burst
	target *enemy.Enemy
}

// SetHomingCandidates bridges the stage's stable enemy order to the original
// sticky player target. Enemy teardown releases it; live targets are not
// replaced by a fresh nearest-enemy search on every shot.
func (p *Player) SetHomingCandidates(enemies []*enemy.Enemy) {
	if p.shot == nil {
		return
	}
	s := p.shot
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

// updateShot is FUN_004281d0 and the shot update after it. Outside
// state 1 the target is dropped and the burst clock holds.
func (p *Player) updateShot(in *input.State) {
	s := p.shot
	held := in.IsPressed(input.KeyShot)
	switch {
	case p.NoAttack:
		s.timer.Set(-1) // FUN_00425730 skips FUN_004281d0 and resets its clock
		s.target = nil
	case p.State != StateNormal:
		s.target = nil
	default:
		p.fireSHT(held)
	}
	p.Bullets.Update()
}

// fireSHT ports FUN_004281d0/FUN_00428160. Slow frames do not repeat the
// same timer phase, and FUN_0044bf40(-15) is not a plain modulo operation.
func (p *Player) fireSHT(held bool) {
	s := p.shot
	if s.timer.Cur < 0 {
		if !held {
			return
		}
		s.timer.Set(0) // 0x4281f9-0x428201
	}
	if s.timer.Cur != s.timer.Prev { // 0x428206-0x428223
		for _, shot := range p.shotFile.Shots(p.GS.Power, p.focus) {
			if int(s.timer.Cur)%shot.Interval != shot.Delay {
				continue
			}
			x, y := p.X, p.Y
			if shot.Source != 0 {
				x, y = p.optionScreen(shot.Source - 1)
			}
			p.Bullets.FireSHT(x, y, shot, s.target)
		}
	}
	if s.timer.Cur >= 15 {
		if held {
			s.timer.Add(-15, p.gameSpeed()) // 0x42823e-0x428243
		} else {
			s.timer.Set(-1) // 0x42824e-0x428252
		}
	} else {
		s.timer.Tick(p.gameSpeed()) // 0x42825d
	}
}
