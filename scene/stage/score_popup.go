package stage

// DAT_00477840, the score-popup manager, separate from the GUI and from
// the general ASCII manager DAT_004776e0. FUN_0042b4d0 registers calc 0x0f
// (FUN_0042b9a0 -> FUN_0042b6f0) and draw 0x23 (FUN_0042b9b0 -> 42b780).

import (
	"th10/anm"
	"th10/anm/anmdraw"
	"th10/supervisor"
	"th10/zun"
)

const (
	popupSlots = 0x2d3 // 723 slots constructed, calculated and drawn
	popupRing  = 0x2d0 // 720 slots used by FUN_0042b9c0's wrapping cursor
)

// ScorePopup is the 0x40-byte slot at manager+0x3c4. Digits are stored
// least significant first; negative values use the special glyph 10.
type ScorePopup struct {
	Digits [12]uint8  // +0x00
	Pos    [3]float32 // +0x0c, game coordinates
	Color  uint32     // +0x18, ARGB before Draw replaces alpha
	T      zun.Timer  // +0x1c
	Live   bool       // +0x38
	Count  uint8      // +0x39
}

type ScorePopups struct {
	Slots  [popupSlots]ScorePopup
	cursor int
	vm     anm.VM // +0x18, one shared drawing VM
}

// NewScorePopups is FUN_0042b430/0042b4d0: free slots and a reset VM using
// sprite 0xc4 of ascii.anm. The VM is not animated or added to World.
func NewScorePopups() *ScorePopups {
	p := &ScorePopups{}
	p.vm.Reset()
	p.vm.File = asciiFile()
	p.vm.SetSprite(0xc4)
	return p
}

// Spawn is FUN_0042b9c0. A slot is overwritten even if it is still live;
// the cursor wraps before selecting slot 720, leaving the final three unused.
func (p *ScorePopups) Spawn(pos [3]float32, value int32, color uint32) {
	if p.cursor >= popupRing {
		p.cursor = 0
	}
	s := &p.Slots[p.cursor]
	*s = ScorePopup{Pos: pos, Color: color, Live: true}
	if value < 0 {
		s.Digits[0], s.Count = 10, 1
	} else {
		for value != 0 {
			s.Digits[s.Count] = uint8(value % 10)
			s.Count++
			value /= 10
		}
		if s.Count == 0 {
			s.Digits[0], s.Count = 0, 1
		}
	}
	s.T.Reset()
	p.cursor++
}

// Calc is FUN_0042b6f0, at priority 0x0f before the player and pickups.
// Y moves up by 0.5*game speed even on the frame the slot expires (>60).
func (p *ScorePopups) Calc(gs float32) {
	for i := range p.Slots {
		s := &p.Slots[i]
		if !s.Live {
			continue
		}
		s.Pos[1] -= float32(gs * 0.5) // 0x42b707..0x42b716, DAT_00470b0c
		s.T.Tick(gs)
		if s.T.Cur > 60 {
			s.Live = false
		}
	}
}

// popupAlpha is FUN_0042b780 0x42b838..0x42b895: fade by the squared
// distance to the player, not by age. Colour alpha supplied by Spawn is ignored.
func popupAlpha(pos [3]float32, player [2]float32) uint8 {
	dx, dy := player[0]-pos[0], player[1]-pos[1]
	d := int32(float32(float32(dy*dy) + float32(dx*dx)))
	if d > 0x1000 {
		return 0xd0
	}
	if d > 0x400 {
		return uint8((d-0x400)*0x80/0xc00 + 0x50)
	}
	return 0x50
}

// popupSprite is FUN_0042b780 0x42b8b2..0x42b939. Only digits change
// palette at ages 52 and 56; the negative-value glyph stays at 0xce.
func popupSprite(digit uint8, age int32) int {
	base := 0xc4
	if digit != 10 {
		if age >= 56 {
			base = 0xd9
		} else if age >= 52 {
			base = 0xcf
		}
	}
	return base + int(digit)
}

// Draw is FUN_0042b780. Slots stay in array order, and digits are drawn
// most significant first through the shared mode-0 VM (FUN_00443080,
// the same mode-0 path Renderer.Draw dispatches). Pos2 is what the binary
// writes at manager+0x358/0x35c; all other position components remain zero.
func (p *ScorePopups) Draw(r *anmdraw.Renderer, player [2]float32) {
	v := &p.vm
	for i := range p.Slots {
		s := &p.Slots[i]
		if !s.Live {
			continue
		}
		spacing := float32(8)
		if s.T.Cur < 8 {
			// 0x42b7e4..0x42b7fb: age zero gives +Inf spacing. The first
			// digit has -Inf X; adding spacing then produces NaN for the
			// rest (0x42b958..0x42b964). Preserve even this birth-frame bug.
			spacing = 8 / s.T.F
		}
		x := s.Pos[0] - float32(float32(float32(s.Count)*spacing)*0.5)
		v.Pos2 = [3]float32{x + 224, s.Pos[1] + 16, 0}
		c := s.Color
		v.Color = [3]uint8{uint8(c >> 16), uint8(c >> 8), uint8(c)}
		v.Alpha = popupAlpha(s.Pos, player)
		for j := int(s.Count) - 1; j >= 0; j-- {
			v.SetSprite(popupSprite(s.Digits[j], s.T.Cur))
			v.Flags |= 8 // FUN_0042b780 0x42b94c, shared VM's scale cache
			r.Draw(v)
			v.Pos2[0] += spacing
		}
	}
}

// AddScorePopup is the Context-side FUN_0042b9c0 call. It changes no score,
// RNG or gameplay state; item collection calls it with the original values.
func (c *Context) AddScorePopup(pos [3]float32, value int32, color uint32) {
	if c.Popups != nil {
		c.Popups.Spawn(pos, value, color)
	}
}

func (s *Stage) nodeScorePopups(sv *supervisor.Supervisor) {
	if s.ctx.Popups != nil {
		p := s.ctx.Player
		s.ctx.Popups.Draw(sv.ANM, [2]float32{float32(p.X - gameX), float32(p.Y - gameY)})
	}
}
