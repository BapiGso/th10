package stage

import (
	"th10/game"
	"th10/input"
	sceneui "th10/scene/ui"
)

// TryPause is FUN_004223f0 state 0. Stage +0x58 bit 0x10 freezes the
// world, while the menu's own states 1..5 and ANM continue at speed 1.
func (m *MenuController) TryPause(g *game.Game, in *input.State) bool {
	s := m.stage.ctx.State
	if m.Active() || s.RunFlags&game.RunFlagDemo != 0 || s.Frame < 30 {
		return false
	}
	if in.Menu().Pressed&input.MenuPause == 0 && g.SystemFlags&0x10 == 0 {
		return false
	}
	m.OpenPause(g)
	return true
}

// OpenPause is FUN_00422ab0: capture the field, spawn front 0x79, pause
// music and remember the previous speed. The capture uses t5's existing VM.
func (m *MenuController) OpenPause(g *game.Game) {
	m.State, m.Selection = MenuPauseOpening, 0
	m.Timer.Reset()
	m.priorSpeed = m.stage.ctx.Anm.GameSpeed
	m.stage.ctx.Anm.GameSpeed, m.env.GameSpeed = 1, 1
	m.stage.sceneFlags |= 0x10 // 0x422b21
	m.stage.ctx.State.Paused = true
	m.backdrop = m.world.SpawnTail(menuASCIIFile(), 0, 15, m.env)
	m.heading = m.world.SpawnTail(menuFrontFile(), 0x79, 15, m.env)
	m.capturePending = true
	m.pauseSelector = sceneui.Selector{Wrap: true}
	m.interrupt(3)
	m.interrupt(7)             // FUN_0040c4d0
	m.stage.ctx.PlaySEID(0x20) // 0x422bb9: se_pause.wav (sound id 32)
	if g.Audio() != nil {
		g.Audio().PauseBGM()
	}
}

func (m *MenuController) leavePauseHandle(heading bool) {
	if heading {
		m.world.Interrupt(m.heading, 1)
		m.heading = nil
	} else {
		m.world.Interrupt(m.backdrop, 1)
		m.backdrop = nil
	}
}

func (m *MenuController) leavePauseAll() {
	m.leavePauseHandle(false)
	m.leavePauseHandle(true)
	m.world.Interrupt(m.pauseExtra, 1)
	m.pauseExtra = nil
}

// unpause is FUN_00422c30. The caller leaves State at 0 without resetting
// the menu clock; FUN_004223f0's tail now reads the restored game speed.
func (m *MenuController) unpause(g *game.Game) {
	m.stage.sceneFlags &^= 0x10
	m.stage.ctx.State.Paused = false
	m.stage.ctx.Anm.GameSpeed, m.env.GameSpeed = m.priorSpeed, m.priorSpeed
	m.world.Delete(m.pauseExtra)
	m.pauseExtra = nil
	if g.Audio() != nil {
		g.Audio().ResumeBGM()
	}
}

func (m *MenuController) pauseSound(id int) { m.stage.ctx.PlaySEID(id) }

func (m *MenuController) pauseMove(in input.MenuInput, label int) {
	m.pauseSelector.Value, m.pauseSelector.Previous = m.Selection, m.Selection
	if in.Move(input.MenuUp) {
		m.pauseSelector.Move(-1)
	}
	if in.Move(input.MenuDown) {
		m.pauseSelector.Move(1)
	}
	m.Selection = m.pauseSelector.Value
	if m.Selection != m.pauseSelector.Previous {
		m.interrupt(int16(label + m.Selection))
		m.pauseSound(12)
	}
}

func (m *MenuController) leaveChild(script int) {
	m.world.Interrupt(m.world.FindChild(m.heading, script), 6) // FUN_004243f0 on the selected child only
}

// calcPause is FUN_00422c80. Every deadline reads the pre-tail timer.
// 10: choices; 20/30: confirmation opens/enables; 20: answer; 12: exit.
func (m *MenuController) calcPause(g *game.Game, in input.MenuInput) MenuAction {
	switch m.State {
	case MenuPauseOpening:
		if m.Timer.Cur >= 10 {
			m.State = MenuPauseChoices // 0x422cae: no timer reset here
			m.pauseSelector.Count = 3
			if m.stage.ctx.State.StageMode != 0 {
				m.pauseSelector.Count = 2
			}
			m.Selection, m.pauseSelector.Value = 0, 0
			m.interrupt(7)
		}
	case MenuPauseChoices:
		m.pauseMove(in, 7)
		if in.Pressed&input.MenuConfirm != 0 {
			m.pauseSound(10)
			switch m.Selection {
			case 0:
				m.leavePauseAll()
				m.setState(MenuPauseLeaving)
			case 1:
				m.leaveChild(0x74)
				if m.stage.ctx.State.StageMode != 0 {
					m.setState(MenuPauseLeaving)
				} else {
					m.setState(MenuPauseConfirm)
				}
			case 2:
				m.leaveChild(0x75)
				m.setState(MenuPauseConfirm)
			}
		}
		if in.Pressed&input.MenuRetry != 0 {
			m.pauseSound(10)
			m.leaveChild(0x75)
			m.setState(MenuPauseLeaving)
			m.pauseSelector.Set(2)
			m.Selection = m.pauseSelector.Value
			m.leavePauseHandle(false) // 0x422e90
		}
		if in.Pressed&input.MenuReturnTitle != 0 {
			m.pauseSound(10)
			m.leaveChild(0x74)
			m.setState(MenuPauseLeaving)
			m.pauseSelector.Set(1)
			m.Selection = m.pauseSelector.Value
		}
		if in.Pressed&input.MenuPause != 0 {
			m.Selection = 0
			m.leavePauseAll()
			m.setState(MenuPauseLeaving)
		}
	case MenuPauseLeaving:
		if m.Timer.Cur >= 12 {
			m.State = MenuDormant // 0x42314f: do not reset the timer
			switch m.Selection {
			case 0:
				m.unpause(g)
			case 1:
				m.leavePauseAll()
				return MenuTitle
			case 2:
				m.leavePauseHandle(true)
				return MenuRetry
			}
		}
	case MenuPauseConfirm:
		if m.Timer.Cur < 20 {
			return MenuStay
		}
		if m.Timer.Cur == 20 {
			m.pauseSelector.Value = m.Selection
			m.pauseSelector.Push()
			m.pauseSelector.Count = 2
			m.pauseSelector.Set(1)
			m.Selection = 1
			m.interrupt(14) // 0x422f59
		}
		if m.Timer.Cur < 30 {
			return MenuStay
		}
		if m.Timer.Cur == 30 {
			m.interrupt(int16(15 + m.Selection))
		}
		m.pauseMove(in, 15)
		if in.Pressed&input.MenuConfirm != 0 {
			m.pauseSound(10)
			m.leaveChild(0x77 + m.Selection)
			m.setState(MenuPauseAnswer)
		}
		if in.Pressed&input.MenuPause != 0 {
			m.Selection = 0
			m.leavePauseAll()
			m.setState(MenuPauseLeaving)
		}
	case MenuPauseAnswer:
		if m.Timer.Cur >= 20 {
			yes := m.Selection == 0
			m.pauseSelector.Pop()
			m.Selection = m.pauseSelector.Value
			if yes {
				m.leavePauseHandle(false)
				m.leavePauseHandle(true)
				m.setState(MenuPauseLeaving)
			} else {
				m.interrupt(int16(7 + m.Selection))
				m.setState(MenuPauseChoices)
			}
		}
	}
	return MenuStay
}

// UpdateMenu is the pause/end-menu entry used by live play and playback.
// false means no menu blocked this frame (including a just-resolved Resume).
func (s *Stage) UpdateMenu(g *game.Game, in *input.State) (MenuAction, bool) {
	m := s.ctx.Menu
	if !m.Active() && !m.TryPause(g, in) {
		if m.world.Len() != 0 {
			m.world.Run(m.env)
		}
		return MenuStay, false
	}
	action := m.Calc(g, in)
	m.world.Run(m.env)
	return action, m.State != MenuDormant || action != MenuStay
}
