package dialog

// The ANM side of FUN_00415e90. All objects are World VMs: their scripts
// choose layers 13/14/15/16, entry/exit motion, shading and text fade. There
// is no screen overlay, expression-filename guessing or manual mirroring.

import (
	"th10/anm"
	"th10/gamedata"
	"th10/render"
)

type PresentationConfig struct {
	Stage, Character int
	World            *anm.World
	Env              *anm.Env
	Front, Text      *anm.File
	Player, Enemy    *anm.File
	Logo             *anm.File
	StageClear       func() // GUI banner only; MSG's existing hook owns logic
}

// Presentation is the six handles at MSG+0x40..0x54. TextContent exposes
// rendered text, independent of the logic-only View's original strings.
type Presentation struct {
	c            PresentationConfig
	Player, Boss *anm.VM
	Box          *anm.VM
	Lines        [2]*anm.VM
	Name         *anm.VM
	TextContent  [2]string
	closed       bool
}

// NewPresentation is the visual constructor of FUN_00415b00: text scripts
// 0/1 and font height 16. Their first frame runs now, but parks invisible.
func NewPresentation(c PresentationConfig) *Presentation {
	p := &Presentation{c: c}
	if !gamedata.Story {
		return p // th10nostory: no text boxes; Apply keeps only non-dialogue effects
	}
	for i := range p.Lines {
		p.Lines[i] = c.World.SpawnTail(c.Text, i, 15, c.Env)
	}
	return p
}

func (p *Presentation) spawn(f *anm.File, script int) *anm.VM {
	return p.c.World.SpawnTail(f, script, 15, p.c.Env) // FUN_00448d00, alternate RNG
}

// Apply runs only when MSG consumes an instruction. Wait frames do not
// repeat an effect or recreate an animation object.
func (p *Presentation) Apply(m *MSG, in Instr) {
	if p.closed {
		return
	}
	w := p.c.World
	if !gamedata.Story {
		// th10nostory keeps the boss-music title and the stage-clear banner,
		// which are not dialogue; portraits, boxes, text and name plates go.
		switch in.Op {
		case msgBossBGM:
			p.spawn(p.c.Logo, 2)
		case msgStageClear:
			if p.c.Stage <= 5 && p.c.StageClear != nil {
				p.c.StageClear()
			}
		}
		return
	}
	switch in.Op {
	case msgShowPlayer:
		p.Player = p.spawn(p.c.Player, 0x1d) // 0x4160cf..0x416126
	case msgShowBoss:
		f, script := p.c.Player, 0x1e
		if p.c.Stage == 6 {
			f, script = p.c.Enemy, 0x20
		} else if p.c.Stage == 7 && m.ID < 1 {
			f, script = p.c.Enemy, 0x25
		}
		p.Boss = p.spawn(f, script)
	case msgShowBox:
		p.Box = p.spawn(p.c.Front, 0x5a) // 0x4160ab
	case msgHidePlayer:
		w.Interrupt(p.Player, 1) // FUN_00409e50, not immediate deletion
		p.Player = nil
	case msgHideBoss:
		w.Interrupt(p.Boss, 1)
		p.Boss = nil
		w.Interrupt(p.Name, 1)
	case msgHideAll:
		w.Interrupt(p.Player, 1)
		w.Interrupt(p.Boss, 1)
		w.Interrupt(p.Box, 1) // 0x416292 keeps the handles
	case msgSpeakerPlayer, msgSpeakerBoss:
		bright, dim := p.Player, p.Boss
		offset := float32(8) // MSG+0x60, constructed at 0x415ccb
		if in.Op == msgSpeakerBoss {
			bright, dim, offset = p.Boss, p.Player, 24
		}
		w.Interrupt(dim, 3)    // FUN_0040c4d0
		w.Interrupt(bright, 2) // FUN_0040c480
		for _, v := range p.Lines {
			w.SetOffset(v, [3]float32{offset, 0, 0}) // 0x4162cc/0x41632a
		}
	case msgPlayerFace:
		top, bottom := 0x34, 0x3c
		if p.c.Character == 1 {
			top, bottom = 0x2d, 0x35
		}
		p.setFace(p.Player, 0x17, p.c.Player, top+int(in.Arg))
		p.setFace(p.Player, 0x1a, p.c.Player, bottom+int(in.Arg))
	case msgBossFace:
		top, bottom, childTop, childBottom := bossFace(p.c.Stage, m.ID)
		p.setFace(p.Boss, childTop, p.c.Enemy, top+int(in.Arg))
		p.setFace(p.Boss, childBottom, p.c.Enemy, bottom+int(in.Arg))
	case msgLine1, msgLine2:
		i := int(in.Op - msgLine1)
		p.writeLine(i, displayText(in), m.View.Speaker)
		w.Interrupt(p.Lines[i], 2) // 0x415f62
	case msgText:
		if m.lines == 1 { // interpreter has just consumed the first op 16
			p.writeLine(0, " ", m.View.Speaker)
			p.writeLine(1, " ", m.View.Speaker)
			p.writeLine(0, displayText(in), m.View.Speaker)
			w.Interrupt(p.Lines[0], 2)
			w.Interrupt(p.Lines[1], 3) // 0x41602a/0x416031
		} else {
			p.writeLine(1, displayText(in), m.View.Speaker)
			w.Interrupt(p.Lines[1], 2)
		}
	case msgDimAll:
		// 0x416096: op 17 dims TEXT VMs (+0x4c/+0x50), not portraits.
		// The old overlay's Dim flag stays test-compatible in View.
		w.Interrupt(p.Lines[0], 3)
		w.Interrupt(p.Lines[1], 3)
	case msgBossBGM:
		p.spawn(p.c.Logo, 2) // 0x416aa1: stage's boss-music title
	case msgBossName:
		if id := bossPlate(p.c.Stage); id >= 0 {
			p.Name = p.spawn(p.c.Enemy, id)
		}
	case msgStageClear:
		if p.c.Stage <= 5 && p.c.StageClear != nil {
			p.c.StageClear()
		}
	case msgAnimatePlayer:
		w.Interrupt(p.Player, 7) // 0x41623c, Arg is ignored
	case msgAnimateBoss:
		w.Interrupt(p.Boss, 7) // 0x41624e
	}
}

func displayText(in Instr) string {
	if in.DisplayText != "" {
		return in.DisplayText
	}
	return in.Text
}

func (p *Presentation) writeLine(i int, text string, side int) {
	colour := uint32(0xf8f08f)
	if side&1 != 0 {
		colour = 0x8088ff
	}
	render.WriteVMText(p.Lines[i], text, render.TextStyle{Size: 16, Color: colour})
	p.TextContent[i] = text
}

// setFace is FUN_004497d0 + FUN_00449670/004496a0: select a specific
// child, then replace its sprite/file (FUN_0043e5a0 also writes +0x308).
func (p *Presentation) setFace(root *anm.VM, child int, f *anm.File, sprite int) {
	if v := p.c.World.FindChild(root, child); v != nil {
		v.File = f
		v.SetSprite(sprite)
	}
}

// bossFace is FUN_00415e90 op 13, 0x416408..0x416a03. Large Kanako
// portraits have different child scripts and bottom-half dimensions.
func bossFace(stage int, entry int32) (top, bottom, childTop, childBottom int) {
	childTop, childBottom = 0x18, 0x1b
	switch stage {
	case 1:
		top, bottom = 0xf, 0x17
	case 2:
		top, bottom = 0x15, 0x1e
	case 3:
		top, bottom = 0x22, 0x2b
	case 4:
		top, bottom = 0x25, 0x2e
	case 5:
		top, bottom = 0x10, 0x18
	case 6:
		top, bottom, childTop, childBottom = 0x30, 0x39, 0x1c, 0x1f
	case 7:
		if entry < 1 {
			top, bottom, childTop, childBottom = 0x44, 0x4d, 0x21, 0x24
		} else {
			top, bottom = 0x2e, 0x37
		}
	}
	return
}

// bossPlate is the op 19 table, 0x416adf..0x416bce.
func bossPlate(stage int) int {
	table := [...]int{-1, 0xd, 0x10, 0x14, 0x15, 0xe, 0x23, 0x1d}
	if stage >= 1 && stage < len(table) {
		return table[stage]
	}
	return -1
}

// Close is FUN_004136c0: delete the six remaining handles and their
// child chains. Hide ops use label 1, allowing their scripted exits.
func (p *Presentation) Close() {
	if p.closed {
		return
	}
	for _, v := range []*anm.VM{p.Player, p.Boss, p.Box, p.Lines[0], p.Lines[1], p.Name} {
		p.c.World.Delete(v)
	}
	p.closed = true
}
