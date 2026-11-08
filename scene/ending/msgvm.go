package ending

import (
	"fmt"
	"path"

	"th10/anm"
	"th10/audio"
	"th10/game"
	"th10/render"
	"th10/zun"
)

const (
	msgSkip    = 1 // MSG +0x74
	msgStaff   = 2
	msgLoading = 4
)

type resourceLoad struct {
	name string
	slot int
}

type msgVM struct {
	owner        *Ending
	name         string
	instructions []instruction
	pc           int
	clock, wait  zun.Timer
	elapsed      zun.Timer
	flags        uint32
	lines        [5]*anm.VM
	content      [5]string
	line         int
	color        uint32
	files        [4]*anm.File
	images       [16]*anm.VM
}

func (m *msgVM) reset(name string, instructions []instruction, staff bool) {
	owner := m.owner
	*m = msgVM{owner: owner, name: name, instructions: instructions, color: 0xffffff, flags: msgSkip}
	m.clock.Reset()
	m.wait.Reset()
	m.elapsed.Reset()
	if staff {
		m.flags = msgStaff // FUN_0040bd80 opcode 12 zeroes the old skip bit
	}
}

// step is FUN_0040bd80. A blocked wait/load returns before the script
// clock's tick; FUN_0040bd20 still ticks the independent elapsed timer.
func (m *msgVM) step(g *game.Game, advance, skip bool) (bool, error) {
	if m.flags&msgLoading != 0 {
		return false, nil
	}
	if m.pc >= len(m.instructions) {
		return true, nil
	}
	if !m.owner.firstEnding && m.flags&msgStaff == 0 && m.flags&msgSkip != 0 && skip {
		m.clock.Set(int32(m.instructions[m.pc].time)) // 0x40bdb7..0x40bdc1
	}
	for m.pc < len(m.instructions) && int32(m.instructions[m.pc].time) <= m.clock.Cur {
		in := m.instructions[m.pc]
		switch in.op {
		case 0: // 0x40c2c7
			return true, nil
		case 3: // 0x40bdf7..0x40bed4
			m.write(in.displayText)
		case 4: // 0x40bee0
			for _, v := range m.lines {
				m.owner.world.Interrupt(v, 3)
			}
		case 5, 6: // 0x40befa/0x40bf9a
			if m.wait.Cur < 1 {
				m.wait.Set(in.arg(0))
			}
			m.wait.Add(-1, 1)
			if in.op == 5 && in.arg(0) < 0 {
				m.wait.Set(999) // 0x40bf24: reset on every execution, never EOF by timeout
			}
			if !advance && m.wait.Cur > 0 {
				if m.owner.firstEnding || !skip || m.wait.Cur%6 != 0 {
					return false, nil
				}
			} else if a := g.Audio(); a != nil {
				a.PlaySEID(0) // timeout/press, not the silent Ctrl path
			}
			m.wait.Set(0)
			if in.op == 6 {
				m.line = 0 // 0x40c015, also clears global ending-skip request
			}
		case 7: // 0x40c2d4: async resource load, resume at the following instruction
			if len(in.args) < 5 || m.pc+1 >= len(m.instructions) {
				return false, fmt.Errorf("ending: invalid ANM load")
			}
			// FUN_0040c3c0 reads +4 from the advanced PC, not the old
			// opcode-7 payload. Shipped files put an op-8 slot 0 here.
			slot := int(m.instructions[m.pc+1].arg(0))
			if slot < 0 || slot >= len(m.files) {
				return false, fmt.Errorf("ending: ANM slot %d", slot)
			}
			m.owner.pending = &resourceLoad{name: cstring(in.args[4:]), slot: slot}
			m.owner.showLoading([3]float32{480, 480, 0})
			m.flags |= msgLoading
			m.pc++
			return false, nil
		case 8, 15, 16, 17: // 0x40c023/0x40c077/0x40c0c4/0x40c111
			if in.op == 8 || m.owner.state.Difficulty == int(in.op)-14 {
				if err := m.spawnImage(in); err != nil {
					return false, err
				}
			}
		case 9: // 0x40c15e
			m.color = uint32(in.arg(0))
		case 10: // 0x40c169, file path is plaintext
			if err := m.playBGM(g, cstring(in.args)); err != nil {
				return false, err
			}
		case 11: // 0x40c1a5 passes float32 0x40400000 (3 seconds)
			if a := g.Audio(); a != nil {
				a.FadeOutBGM(180) // FUN_0043e500 at 0x43e50b multiplies seconds by 60
			}
			m.flags &^= msgSkip
		case 12: // 0x40c1b8..0x40c26d, FUN_0040b480
			for _, v := range m.lines {
				m.owner.world.Delete(v)
			}
			name := cstring(in.args)
			instructions, err := loadMSG(name)
			if err != nil {
				return false, err
			}
			m.reset(name, instructions, true)
			continue
		case 13, 14: // 0x40c23b/0x40c249, hidden EBX=0x31 at 0x40c255
			kind := int32(0)
			if in.op == 14 {
				kind = 5
			}
			m.owner.fades = append(m.owner.fades, newScreenFade(kind, in.arg(0)))
		}
		m.pc++
	}
	m.clock.Tick(1) // 0x40c27c..0x40c33a, not reached on a wait/load
	return false, nil
}

func (m *msgVM) write(text string) {
	w := &m.owner.world
	if m.line == 0 {
		for i, v := range m.lines {
			render.WriteVMText(v, " ", render.TextStyle{Size: 16, Color: 0xffffff})
			m.content[i] = " "
			w.Interrupt(v, 3) // FUN_0040c4d0, 0x40be3d
		}
	}
	render.WriteVMText(m.lines[m.line], text, render.TextStyle{Size: 16, Color: m.color})
	m.content[m.line] = text
	w.Interrupt(m.lines[m.line], 2) // FUN_0040c480, 0x40be7f/0x40bebd
	m.line++
	if m.line >= len(m.lines) {
		m.line = 0
	}
}

func (m *msgVM) spawnImage(in instruction) error {
	slot, file := int(in.arg(0)), int(in.arg(1))
	if len(in.args) < 12 || slot < 0 || slot >= len(m.images) || file < 0 || file >= len(m.files) || m.files[file] == nil {
		return fmt.Errorf("ending: invalid image/file slot %d/%d", slot, file)
	}
	w := &m.owner.world
	w.Delete(m.images[slot]) // FUN_004492a0, not a restart of the old image
	m.images[slot] = w.SpawnTail(m.files[file], int(in.arg(2)), 15, &m.owner.env)
	return nil
}

func (m *msgVM) playBGM(g *game.Game, name string) error {
	tracks, err := audio.BGMTracks()
	if err != nil {
		return err
	}
	for i, track := range tracks {
		if track.Name == path.Base(name) {
			if a := g.Audio(); a != nil {
				a.PlayBGM(i) // FUN_00420a90/00420b10
			}
			if g.Save != nil && g.Save.Native != nil && m.owner.state.ReplayMode != 1 {
				g.Save.Native.MarkMusicPlayed(i)
				return g.SaveProgress()
			}
			return nil
		}
	}
	return fmt.Errorf("ending: unknown BGM %q", name)
}
