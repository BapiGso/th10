package menus

import (
	"fmt"

	"th10/anm"
	"th10/game"
	"th10/input"
	"th10/render"
	"th10/supervisor"
	"th10/zun"

	"github.com/hajimehoshi/ebiten/v2"
)

// PlayerData is TitleInf state 11, FUN_00431ee0/004329f0.
type PlayerData struct {
	core                     nativeMenu
	save                     *game.SaveData
	history                  *game.SpellHistory
	back                     game.Scene
	shot, difficulty, page   int
	state                    int
	timer                    zun.Timer
	character, weapon, level *anm.VM
	arrows                   [4]*anm.VM
	rows                     [10]*anm.VM
	initialized              bool
	closed                   bool
	Background               func(*supervisor.Supervisor)
}

func NewPlayerData(save *game.SaveData, back game.Scene) *PlayerData {
	s := &PlayerData{save: save, back: back, difficulty: game.DiffNormal}
	s.timer.Reset()
	return s
}

func (s *PlayerData) init(g *game.Game) error {
	if err := s.core.init(g, 0x5e, 0x66); err != nil { // 0x431fc1/0x431fed
		return err
	}
	if g.Save != nil {
		s.save = g.Save
	}
	if s.save == nil {
		s.save = game.DefaultSaveData()
	}
	s.history = g.SpellHistory
	if s.history == nil {
		s.history = s.save.SpellHistory
	}
	s.character = s.core.spawnTitle(0x98 + s.shot/3) // 0x43200f
	s.weapon = s.core.spawnTitle(0x9a + s.shot)      // 0x43201f
	s.level = s.core.spawnTitle(0xa0 + s.difficulty)
	for i := range s.arrows {
		s.arrows[i] = s.core.spawnTitle(0xa8 + i) // up/down/left/right
	}
	for _, id := range []int{0xa5, 0xa6, 0xa7, 0xac} {
		s.core.spawnTitle(id) // runs/time/clears/page labels, 0x432069..0x432090
	}
	s.initialized = true
	s.state = 1
	s.timer.Reset()
	return nil
}

func (s *PlayerData) Update(g *game.Game) error {
	if s.closed {
		return nil
	}
	if !s.initialized {
		if err := s.init(g); err != nil {
			return err
		}
	}
	switch s.state {
	case 1:
		if s.timer.Cur > 6 { // 0x432095, native substate 1
			s.state = 2
			s.timer.Reset()
		}
	case 2:
		oldShot, oldDifficulty := s.shot, s.difficulty
		if input.Global.Menu().Move(input.MenuUp) {
			s.difficulty = wrapSelection(s.difficulty-1, 5)
			s.core.interrupt(s.arrows[0], 2)
		}
		if input.Global.Menu().Move(input.MenuDown) {
			s.difficulty = wrapSelection(s.difficulty+1, 5)
			s.core.interrupt(s.arrows[1], 2)
		}
		if oldDifficulty != s.difficulty {
			s.core.interrupt(s.level, 1)
			s.level = s.core.spawnTitle(0xa0 + s.difficulty)
			if s.page > 0 {
				s.page = 1 // 0x43210b: keep the history tab, reset its page
				s.buildRows()
			}
			menuSE(g, 12)
		}
		if input.Global.Menu().Move(input.MenuLeft) {
			s.shot = wrapSelection(s.shot-1, 6)
			s.core.interrupt(s.arrows[2], 2)
		}
		if input.Global.Menu().Move(input.MenuRight) {
			s.shot = wrapSelection(s.shot+1, 6)
			s.core.interrupt(s.arrows[3], 2)
		}
		if oldShot != s.shot {
			if oldShot/3 != s.shot/3 {
				s.core.interrupt(s.character, 1)
				s.character = s.core.spawnTitle(0x98 + s.shot/3)
			}
			s.core.interrupt(s.weapon, 1)
			s.weapon = s.core.spawnTitle(0x9a + s.shot)
			if s.page > 0 {
				s.buildRows()
			}
			menuSE(g, 12)
		}
		if input.Global.JustPressed(input.KeyOk) {
			s.page = wrapSelection(s.page+1, s.pageCount()) // 0x432269, Z changes page
			s.buildRows()
			menuSE(g, 10)
		}
		if input.Global.JustPressed(input.KeyCancel) {
			for l := range anm.Layers {
				s.core.world.Each(l, func(v *anm.VM) { s.core.interrupt(v, 1) })
			}
			s.state = 3
			s.timer.Reset()
			menuSE(g, 11)
		}
	case 3:
		if s.timer.Cur > 5 { // 0x432619, return to Title's main menu
			s.Close()
			g.GoTo(s.back)
			return nil
		}
	}
	s.timer.Tick(1) // FUN_0042cdf0 calc tail
	s.core.calc()
	return nil
}

func (s *PlayerData) buildRows() {
	for i, v := range s.rows {
		s.core.interrupt(v, 1)
		s.rows[i] = nil
	}
	for i, row := range s.SpellRows() {
		v := s.core.spawnText(0x17 + i) // FUN_00431ee0/00432690: ten text VMs
		render.WriteVMText(v, row.Text, render.TextStyle{Size: 16, Color: row.Colour})
		s.rows[i] = v
	}
}

func (s *PlayerData) Close() { s.closed = true; s.core.close() }

func (s *PlayerData) Draw(screen *ebiten.Image) {
	sv := supervisor.Get()
	s.DrawPorted(sv)
	if screen != nil && sv.Frame != nil {
		screen.DrawImage(sv.Frame, nil)
	}
}

func (s *PlayerData) DrawPorted(sv *supervisor.Supervisor) {
	s.core.draw(sv, s.Background, func() {
		if s.state != 2 {
			return
		}
		stats := s.Statistics()
		if s.page == 0 {
			for i, entry := range stats.Rankings {
				value := uint32(255 - i*16)
				colour := uint32(0xff0000ff) | value<<16 | value<<8 // 0x432a70..0x432a80
				s.core.smallText(sv, rankingLine(i+1, entry), 48, float32(160+i*18), colour)
			}
		}
		s.core.smallText(sv, fmt.Sprintf("    %5d", stats.Runs), 328, 378, 0xffffffff)
		s.core.smallText(sv, stats.PlayTime, 328, 396, 0xffffffff)
		s.core.smallText(sv, fmt.Sprintf("    %5d", stats.Clears), 328, 414, 0xffffffff)
	}) // FUN_004329f0; original extra draw dispatch is 0x42d282
}
