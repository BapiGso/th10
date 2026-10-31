package dialog

// The MSG interpreter (FUN_00415e90): one .msg entry plays on its own clock
// while the game keeps running. ECL 338 opens it, op 11 lets ECL 339 go on,
// and op 0 deletes it.

// MSG opcodes, from FUN_00415e90's switch.
const (
	msgEnd           = 0
	msgShowPlayer    = 1 // the player's portrait
	msgShowBoss      = 2 // the boss's portrait (special scripts on stages 6/7)
	msgShowBox       = 3 // the text box
	msgHidePlayer    = 4
	msgHideBoss      = 5
	msgHideAll       = 6
	msgSpeakerPlayer = 7
	msgSpeakerBoss   = 8
	msgSkippable     = 9  // arg & 1: whether Ctrl fast-forwards
	msgWait          = 10 // wait for the shot key, at most arg frames
	msgSignal        = 11 // let ECL 339 go on
	msgPlayerFace    = 12
	msgBossFace      = 13
	msgLine1         = 14
	msgLine2         = 15
	msgText          = 16 // the next line, clearing the box every other one
	msgDimAll        = 17
	msgBossBGM       = 18 // the boss theme and its title
	msgBossName      = 19 // the boss's name plate
	msgStageClear    = 20
	msgFadeBGM       = 21
)

// Instr is one .msg instruction. It runs once the entry's clock reaches
// Time. Arg is the first int argument, Text the decoded line of ops 14-16.
type Instr struct {
	Time uint16
	Op   uint8
	Arg  int32
	Text string
}

// Hooks are the MSG's effects outside the dialog. Nil hooks do nothing.
type Hooks struct {
	SE      func(id int) // op 10's advance sound (FUN_0043dc90)
	BossBGM func()       // op 18 (FUN_00420b10)
	Clear   func()       // op 20
	FadeBGM func()       // op 21 (FUN_00420c30)
}

// TextLine is one line of the text box, coloured by the side that spoke it
// (MSG+0x88: 0xf8f08f for the player, 0x8088ff for the boss).
type TextLine struct {
	Text string
	Side int
}

// View is what the dialog shows. The original draws it with front.anm and
// the face scripts (M5); the overlay in dialog.go stands in.
type View struct {
	Box, Player, Boss, Name bool
	PlayerFace, BossFace    int32
	Speaker                 int  // MSG+0x84: 0 player, 1 boss
	Dim                     bool // op 17 dims both portraits until the next line
	Lines                   [2]TextLine
	PlayerName, BossName    string
}

// MSG is a playing .msg entry: the object FUN_00415db0 builds for ECL 338.
type MSG struct {
	ID int32 // +0x00: the entry ECL 338 opened

	code   []Instr
	pc     int  // +0x5c
	clock  int  // +0x1c: the timeline clock
	wait   int  // +0x30: op 10's timer
	Signal int  // +0x78: op 11 raises it for ECL 339, the next frame drops it
	skip   bool // +0x7c bit 0
	lines  int  // +0x80: lines op 16 has put in the box
	hooks  Hooks

	View View
}

// NewMSG starts entry id's code with clock 0.
func NewMSG(id int32, code []Instr, hooks Hooks) *MSG {
	return &MSG{ID: id, code: code, hooks: hooks}
}

// Pos reports where the MSG is, for traces: its clock, op 10's timer and the
// time and opcode of the instruction it is at.
func (m *MSG) Pos() (clock, wait, at, op int) {
	at, op = -1, -1
	if m.pc < len(m.code) {
		at, op = int(m.code[m.pc].Time), int(m.code[m.pc].Op)
	}
	return m.clock, m.wait, at, op
}

// Update runs the MSG for a frame. advance is the shot key's press edge
// (DAT_00474e62 & 0x1001), skip whether Ctrl is held (DAT_00474e5c & 0x100).
// It reports false once op 0 ends the entry.
func (m *MSG) Update(advance, skip bool) bool {
	if m.Signal > 0 {
		m.Signal--
	}
	// Ctrl jumps the clock to the next instruction.
	if m.skip && skip && m.pc < len(m.code) {
		m.clock = int(m.code[m.pc].Time)
	}
	v := &m.View
	for m.pc < len(m.code) && int(m.code[m.pc].Time) <= m.clock {
		in := &m.code[m.pc]
		switch in.Op {
		case msgEnd:
			return false
		case msgShowPlayer:
			v.Player = true
		case msgShowBoss:
			v.Boss = true
		case msgShowBox:
			v.Box = true
		case msgHidePlayer:
			v.Player = false
		case msgHideBoss:
			v.Boss, v.Name = false, false
		case msgHideAll:
			v.Player, v.Boss, v.Box = false, false, false
		case msgSpeakerPlayer, msgSpeakerBoss:
			v.Speaker = int(in.Op - msgSpeakerPlayer)
			v.Lines = [2]TextLine{}
			v.Dim = false
			m.lines = 0
		case msgSkippable:
			m.skip = in.Arg&1 != 0
		case msgWait:
			if m.wait < 1 {
				m.wait = int(in.Arg)
			}
			m.wait--
			if !advance && m.wait > 0 {
				if !(m.skip && skip) {
					return true // the clock waits too
				}
			} else if m.hooks.SE != nil {
				m.hooks.SE(0)
			}
			m.wait, m.lines = 0, 0
		case msgSignal:
			m.Signal = 1
		case msgPlayerFace:
			v.PlayerFace = in.Arg
		case msgBossFace:
			v.BossFace = in.Arg
		case msgLine1, msgLine2:
			v.Lines[in.Op-msgLine1] = TextLine{in.Text, v.Speaker}
			v.Dim = false
		case msgText:
			if m.lines == 0 {
				v.Lines = [2]TextLine{{in.Text, v.Speaker}, {}}
				m.lines++
			} else {
				v.Lines[1] = TextLine{in.Text, v.Speaker}
				m.lines = 0
			}
			v.Dim = false
		case msgDimAll:
			v.Dim = true
		case msgBossBGM:
			if m.hooks.BossBGM != nil {
				m.hooks.BossBGM()
			}
		case msgBossName:
			v.Name = true
		case msgStageClear:
			if m.hooks.Clear != nil {
				m.hooks.Clear()
			}
		case msgFadeBGM:
			if m.hooks.FadeBGM != nil {
				m.hooks.FadeBGM()
			}
		}
		m.pc++
	}
	if m.pc >= len(m.code) {
		return false // every entry ends in op 0; a cut one ends here
	}
	m.clock++
	return true
}
