package stage

import (
	"strings"
	"sync"
	"time"

	"th10/anm"
	"th10/audio"
	"th10/game"
	"th10/gamedata"
	"th10/input"
	"th10/zun"
)

// MenuState is DAT_00477830+4, not the pause flag. States 1..5 belong to
// the separately scoped pause path; FUN_004236f0 handles end-menu states.
type MenuState int32

const (
	MenuDormant    MenuState = 0
	MenuPrompt     MenuState = 6
	MenuRankCheck  MenuState = 7
	MenuChoices    MenuState = 8
	MenuSlots      MenuState = 10
	MenuReplayName MenuState = 11
	MenuRanking    MenuState = 12
	MenuLeaving    MenuState = 13
	MenuFinished   MenuState = 21
)

type MenuAction uint8

const (
	MenuStay     MenuAction = iota
	MenuContinue            // DAT_00491fb8 = 0x0d
	MenuNext                // DAT_00491fb8 = 0x0a
	MenuTitle               // FUN_0040ac90(4)
)

// Exact 0x4746d8 -> 0x46e2f8 table; the last three spaces stand for the
// blank, backspace and finish glyphs (0x81, 0x7f, 0x80).
const menuCharacters = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+-=.,!?@:;[]()_/{}|~^#$%&*   "

var menuFrontFile = sync.OnceValue(func() *anm.File {
	f, err := gamedata.ANM("front.anm")
	if err != nil {
		panic("stage menu: embedded front.anm: " + err.Error())
	}
	return f
})

var menuASCIIFile = sync.OnceValue(func() *anm.File {
	f, err := gamedata.ANM("ascii.anm")
	if err != nil {
		panic("stage menu: embedded ascii.anm: " + err.Error())
	}
	return f
})

// MenuController ports the 0x2c8-byte DAT_00477830 controller. World ANM
// objects draw in layers 15/16; only ranking/name/slot text draws in 0x2e.
type MenuController struct {
	Flags             uint32 // +0: FUN_004220e0 clears the object, then ORs 2
	State             MenuState
	Timer             zun.Timer // +0x10: last/current/accumulator
	Selection         int       // +0x24
	Character         int       // +0xfc
	Name              [8]byte   // +0x2b4
	NameAt            int       // +0x1e0
	Cleared           bool      // +0x1e4
	NoRank            bool      // +0x1e8
	Error             string
	stage             *Stage
	rank              int
	ranking           *[10]game.ScoreEntry
	slots             []game.ReplaySlot
	heading, backdrop *anm.VM // +0x1d4 / +0x1d8
	world             *anm.World
	env               *anm.Env
	priorSpeed        float32 // +0x2c0
	capturePending    bool
}

// NewMenuController is FUN_00422360 -> FUN_004220e0 / FUN_00422150: clear
// state, flag 2, timer at -1, disabled calc/draw until the stage begins.
func NewMenuController(s *Stage) *MenuController {
	w := &anm.World{}
	env := *s.ctx.Anm
	env.GameSpeed, env.Spawn = 1, w.Add
	m := &MenuController{Flags: 2, stage: s, rank: -1, world: w, env: &env}
	m.Timer.Set(-1)
	copy(m.Name[:], "        ")
	return m
}

func (m *MenuController) Active() bool { return m.State != MenuDormant }

// OpenGameOver is FUN_004231d0; OpenClear is FUN_00423370. A replay caller
// takes the title-return branch instead and never opens these menus.
func (m *MenuController) OpenGameOver(g *game.Game) { m.open(g, false) }
func (m *MenuController) OpenClear(g *game.Game)    { m.open(g, true) }

func (m *MenuController) open(g *game.Game, clear bool) {
	if m.stage.ctx.State.StageMode == 1 || m.stage.ctx.State.ReplayMode == 1 {
		return
	}
	m.Cleared, m.State, m.Selection = clear, MenuPrompt, 2
	m.Timer.Set(0)
	m.priorSpeed = m.stage.ctx.Anm.GameSpeed
	m.stage.ctx.Anm.GameSpeed = 1 // 0x42335f / 0x4234ff
	m.backdrop = m.world.SpawnFront(menuASCIIFile(), 0, 15, m.env)
	id := 0x80
	if clear {
		id = 0x81
	}
	m.heading = m.world.SpawnFront(menuFrontFile(), id, 15, m.env)
	m.capturePending = true // FUN_00424480 schedules the 32,16,384,448 capture
	if g.Audio() != nil {
		g.Audio().PlayBGM(audio.BGMResult)
	}
}

func (m *MenuController) setState(state MenuState) {
	m.State = state
	m.Timer.Set(0) // FUN_00405410(0)
}

func (m *MenuController) interrupt(label int16) { m.world.Interrupt(m.heading, label) }

// selectChoice is FUN_0040c4d0 (interrupt 3) then FUN_00449470 with
// 7+selection: front.anm children 124/125/126 use labels 7/8/9.
func (m *MenuController) selectChoice() {
	m.interrupt(3)
	m.interrupt(int16(7 + m.Selection))
}

func menuRepeat(in *input.State, key input.Key) bool {
	n := in.HoldFrames(key)
	return in.JustPressed(key) || (n >= 26 && (n-26)%8 == 0) // FUN_0040ac20
}

func wrapMenu(at, n int) int {
	if n == 0 {
		return 0
	}
	return ((at % n) + n) % n // FUN_0044bea0, selector +0xf4 bit 0
}

func (m *MenuController) initName(g *game.Game) {
	copy(m.Name[:], "        ")
	if g.Save != nil {
		copy(m.Name[:], g.Save.LastReplayName)
	}
	m.NameAt = len(strings.TrimRight(string(m.Name[:]), " "))
	m.Character = 0
	if m.NameAt != 0 {
		m.Character = len(menuCharacters) - 1
	}
}

// Calc is FUN_00422a90 -> FUN_004223f0 -> FUN_004236f0, states 6..13.
func (m *MenuController) Calc(g *game.Game, in *input.State) MenuAction {
	defer m.Timer.Tick(m.stage.ctx.Anm.GameSpeed)
	ok, cancel := in.JustPressed(input.KeyOk), in.JustPressed(input.KeyCancel)
	switch m.State {
	case MenuPrompt: // FUN_004236f0 case 6: accept only after nine frames
		if m.Timer.Cur > 9 && ok {
			m.setState(MenuRankCheck)
			m.interrupt(2)
		}
	case MenuRankCheck:
		if m.Timer.Cur > 9 {
			m.stage.ctx.State.HiScore = max(m.stage.ctx.State.HiScore, m.stage.ctx.State.Score) // FUN_00421f60
			if gui := m.stage.ctx.GUI; gui != nil {
				gui.shownScore = scoreStored(m.stage.ctx.State.Score) // FUN_00421f60 writes GUI+0x9e78
			}
			m.rank = -1
			if m.stage.ctx.State.RunFlags&game.RunFlagPractice == 0 { // FUN_004236f0, 0x423792
				m.rank = g.InsertRanking(m.stage.ctx.State, m.Cleared, int32(time.Now().Unix()), 0)
				m.ranking = g.ScoreTable(m.stage.ctx.State)
			} else {
				g.RecordPracticeScore(m.stage.ctx.State)
				_ = g.SaveProgress()
			}
			m.NoRank = m.rank < 0
			if !m.NoRank {
				m.Selection = m.rank
				m.initName(g)
				m.setState(MenuRanking)
				m.world.Hide(m.heading)
			} else {
				m.choices()
			}
		}
	case MenuChoices:
		old := m.Selection
		if menuRepeat(in, input.KeyUp) {
			m.Selection = wrapMenu(m.Selection-1, 3)
		}
		if menuRepeat(in, input.KeyDown) {
			m.Selection = wrapMenu(m.Selection+1, 3)
		}
		if old != m.Selection {
			m.selectChoice()
		}
		if ok {
			m.interrupt(6) // FUN_004243f0
			m.setState(MenuLeaving)
			if m.Selection == 1 {
				if m.stage.ctx.State.Recording == nil {
					m.Error = "This run has no recording"
					m.choices()
					break
				}
				m.slots = g.ListReplays()
				m.Selection = 0
				m.setState(MenuSlots)
				m.world.Hide(m.heading)
			}
		}
		if cancel && m.Selection != 2 {
			m.Selection = 2
			m.selectChoice()
		}
	case MenuSlots:
		if m.Timer.Cur <= 9 {
			break
		}
		if menuRepeat(in, input.KeyUp) {
			m.Selection = wrapMenu(m.Selection-1, game.ReplaySlots)
		}
		if menuRepeat(in, input.KeyDown) {
			m.Selection = wrapMenu(m.Selection+1, game.ReplaySlots)
		}
		if ok {
			m.initName(g)
			m.setState(MenuReplayName)
			r := m.stage.ctx.State.Recording.Replay()
			r.Global.Timestamp = int32(time.Now().Unix()) // FUN_004236f0, case 10
			r.Global.Stage = uint32(m.stage.ctx.State.Stage)
			if m.Cleared && m.stage.ctx.State.RunFlags&game.RunFlagPractice == 0 {
				r.Global.Stage = 8
			}
		}
		if cancel {
			m.choices()
			m.slots = nil
		}
	case MenuReplayName, MenuRanking:
		if m.Timer.Cur <= 9 {
			break
		}
		m.moveName(in)
		if ok && (!m.NoRank || m.State == MenuReplayName) && m.enterName() {
			name := string(m.Name[:])
			if m.State == MenuReplayName {
				r := m.stage.ctx.State.Recording.PrepareSave()
				r.Global.User, r.Global.Score = strings.TrimRight(name, " "), int32(m.stage.ctx.State.Score/10)
				r.Trailer.Raw = nil
				if err := g.SaveReplay(m.Selection+1, r); err != nil {
					m.Error = err.Error()
					break
				}
				m.slots = g.ListReplays()
				m.setState(MenuSlots)
			} else {
				g.ScoreTable(m.stage.ctx.State)[m.rank].Name = name
				m.choices()
			}
			if g.Save == nil {
				g.Save = game.DefaultSaveData()
			}
			g.Save.LastReplayName = name
			if err := g.SaveProgress(); err != nil {
				m.Error = err.Error()
			}
		}
		if ok && m.State == MenuRanking && m.NoRank {
			m.choices()
		}
		if cancel {
			if m.NameAt != 0 {
				m.backspace()
			} else if m.State == MenuReplayName {
				m.setState(MenuSlots)
			}
		}
	case MenuLeaving:
		if m.Timer.Cur > 11 { // FUN_004236f0 case 13
			m.close()
			if m.Selection == 0 {
				if m.Cleared {
					return MenuNext
				}
				return MenuContinue
			}
			return MenuTitle
		}
	}
	return MenuStay
}

func (m *MenuController) choices() {
	m.Selection = 2 // original initial choice after a non-ranking result
	m.setState(MenuChoices)
	m.world.Show(m.heading)
	m.selectChoice()
}

func (m *MenuController) moveName(in *input.State) {
	if menuRepeat(in, input.KeyUp) {
		m.Character = wrapMenu(m.Character-13, len(menuCharacters))
	}
	if menuRepeat(in, input.KeyDown) {
		m.Character = wrapMenu(m.Character+13, len(menuCharacters))
	}
	if menuRepeat(in, input.KeyLeft) {
		delta := -1
		if m.Character%13 == 0 {
			delta = 12
		}
		m.Character = wrapMenu(m.Character+delta, len(menuCharacters))
	}
	if menuRepeat(in, input.KeyRight) {
		delta := 1
		if m.Character%13 == 12 {
			delta = -12
		}
		m.Character = wrapMenu(m.Character+delta, len(menuCharacters))
	}
}

func (m *MenuController) enterName() bool {
	n := len(menuCharacters)
	if m.Character == n-1 {
		return true
	}
	if m.Character == n-2 {
		m.backspace()
		return false
	}
	c := byte(' ')
	if m.Character < n-3 {
		c = menuCharacters[m.Character]
	}
	m.Name[min(m.NameAt, 7)] = c
	if m.NameAt < 8 {
		m.NameAt++
	}
	if m.NameAt == 8 {
		m.Character = n - 1
	}
	return false
}

func (m *MenuController) backspace() {
	if m.NameAt > 0 {
		m.NameAt--
		m.Name[m.NameAt] = ' '
	}
}

// close is FUN_00423510: destroy both handles and restore DAT_00476f78.
func (m *MenuController) close() {
	m.world.Delete(m.heading)
	m.world.Delete(m.backdrop)
	m.stage.ctx.Anm.GameSpeed = m.priorSpeed
	m.setState(MenuFinished)
}
