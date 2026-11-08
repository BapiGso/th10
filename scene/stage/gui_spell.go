package stage

// The GUI-facing part of the spell HUD. The binary stores these thirteen
// VMs in DAT_004776f4 (+0x778 and +0x24d8), not in DAT_0047770c. Their
// draw is FUN_00409270 -> FUN_004091c0 at node 0x25. The spell controller
// supplies the reward, clock and history; the GUI does not compute them.

import (
	"th10/anm"
	"th10/anm/anmdraw"
	"th10/supervisor"
)

// SpellHUD owns only presentation. Captures/Attempts are supplied by the
// spell/save controller, not stored or incremented here.
type SpellHUD struct {
	Bonus   [8]anm.VM // DAT_004776f4+0x778, front scripts 58..65
	History [5]anm.VM // +0x24d8, scripts 66..70: two counts and the slash
	Active  bool
	dim     bool // DAT_004776f4+0x378c bit 2
}

func (h *SpellHUD) start(front *anm.File, env *anm.Env) {
	h.Active = true // declaration does not clear helper +0x378c bit 2
	for i := range h.Bonus {
		h.Bonus[i].Start(front, i+0x3a, env) // FUN_00409280 0x4093d0
	}
	for i := range h.History {
		h.History[i].Start(front, i+0x42, env) // 0x409400
	}
}

// update is the display tail of FUN_00408d60 (0x408e5b..0x409171), not
// its bonus formula or timer. The ECL controller sends overlap interrupts
// before this step; only the standalone GUI adapter checks overlap here.
func (h *SpellHUD) update(g *GUI, bonus int64, capture bool, captures, attempts int32, frames int32) {
	if !h.Active {
		return
	}
	if g.ctx.Spell == nil && frames >= 120 && g.ctx.Player != nil {
		y := float32(g.ctx.Player.Y - gameY)
		label := int16(0)
		if !h.dim && y < 96 {
			h.dim, label = true, 3
		} else if h.dim && y > 128 {
			h.dim, label = false, 2
		}
		if label != 0 {
			g.ctx.World.Interrupt(g.spellName, label)
			g.InterruptSpellPresentation(label)
		}
	}
	env := g.ctx.Anm
	if capture {
		n, place := min(max(bonus, 0), 99999999), int64(10000000)
		for i := range h.Bonus {
			h.Bonus[i].SetSprite(int(n/place) + 30)
			n %= place
			place /= 10
			h.Bonus[i].Run(env)
		}
	} else {
		// FUN_00408d60 0x408fbc: only +0xed0 (VM 2) steps after failure.
		// The other seven draw bits were cleared at the failure call site.
		h.Bonus[2].SetSprite(0x29)
		h.Bonus[2].Run(env)
	}
	captures = min(max(captures, 0), 99)
	attempts = min(max(attempts, 0), 99)
	h.History[0].SetSprite(int(captures/10) + 30)
	h.History[1].SetSprite(int(captures%10) + 30)
	h.History[3].SetSprite(int(attempts/10) + 30)
	h.History[4].SetSprite(int(attempts%10) + 30)
	for i := range h.History {
		h.History[i].Run(env)
	}
}

// loseCapture is the seven flag-bit-1 writes shared by FUN_00405500,
// FUN_00426cf0 and FUN_004127a0. +0xed0's flags (+0x122c) are untouched.
func (h *SpellHUD) loseCapture() {
	if h == nil {
		return
	}
	for i := range h.Bonus {
		if i != 2 {
			h.Bonus[i].Flags &^= anm.FlagDraw
		}
	}
}

// InterruptSpellPresentation is FUN_00408d60's direct +0x304 writes,
// after the three World placards have received FUN_00449470's label.
func (g *GUI) InterruptSpellPresentation(label int16) {
	g.spellHUD.dim = label == 3
	for i := range g.spellHUD.Bonus {
		g.spellHUD.Bonus[i].Interrupt(label)
	}
	for i := range g.spellHUD.History {
		g.spellHUD.History[i].Interrupt(label)
	}
}

// Draw is FUN_004091c0: eight bonus VMs, then five history VMs. The
// manager's active bit gates the whole walk (0x4091c0), not capture.
func (h *SpellHUD) Draw(r *anmdraw.Renderer) {
	if !h.Active {
		return
	}
	for i := range h.Bonus {
		r.Draw(&h.Bonus[i])
	}
	for i := range h.History {
		r.Draw(&h.History[i])
	}
}

// BeginSpellPresentation starts the thirteen helper VMs and the name VM.
// The caller supplies the original Shift-JIS bytes to FUN_00447ae0.
func (g *GUI) BeginSpellPresentation(name string) {
	g.manualSpell = true
	g.startSpellPresentation(name)
}

// InitializeSpellPresentation is FUN_00409280's thirteen FUN_0043e710
// calls only. The helper then creates its own three World placards and
// passes the name handle to SetSpellName, without another GUI-owned copy.
func (g *GUI) InitializeSpellPresentation() {
	g.UseExternalSpellPresentation()
	g.startSpellHUD()
}

func (g *GUI) startSpellHUD() {
	g.spellHUD.start(g.front, g.ctx.Anm)
	if s := g.ctx.Spell; s != nil {
		s.HUD = &g.spellHUD
		g.spellHUD.dim = s.Faded
	}
}

func (g *GUI) startSpellPresentation(name string) {
	g.ctx.World.Delete(g.spellName)
	g.startSpellHUD()
	g.spellName = g.ctx.World.SpawnTail(textFile(), 0x48, 15, g.ctx.Anm)
	g.SetSpellName(g.spellName, name)
	if s := g.ctx.Spell; s != nil {
		s.Placards[1] = g.spellName
	}
}

// UpdateSpellPresentation is the UI write part of FUN_00408d60. The
// caller supplies all gameplay/history data; frames is the spell timer.
func (g *GUI) UpdateSpellPresentation(bonus int64, capture bool, captures, attempts int32, frames int32) {
	g.spellHUD.update(g, bonus, capture, captures, attempts, frames)
}

// EndSpellPresentation is the GUI-side release called by FUN_00409c00.
func (g *GUI) EndSpellPresentation() {
	g.spellHUD.Active = false
	if g.ctx.Spell == nil {
		g.ctx.World.Interrupt(g.spellName, 1) // FUN_00409e50: allow the name's exit animation
	}
	g.spellName = nil
}

// UseExternalSpellPresentation disables the pre-t2 field adapter when t2
// creates its own name VM and calls SetSpellName directly instead of Begin.
func (g *GUI) UseExternalSpellPresentation() { g.manualSpell = true }

// calcSpellFallback reads existing ECL declaration/SpellCard data without
// changing it. Once t2 calls Begin/UseExternal, only its hooks drive this UI.
func (g *GUI) calcSpellFallback() {
	if g.manualSpell {
		return
	}
	c := g.ctx.Spell
	if c == nil || !c.Active || !g.info.Spell {
		g.EndSpellPresentation()
		return
	}
	name := c.Name
	if !g.spellHUD.Active || c.ID != g.spellID || name != g.spellText || c.Timer.Cur < g.spellFrame {
		g.startSpellPresentation(name)
		g.spellID = c.ID
	}
	captures, attempts := g.ctx.State.SpellCounts(c.ID)
	g.UpdateSpellPresentation(c.Bonus, c.Capture, captures, attempts, c.Timer.Cur)
	g.spellFrame = c.Timer.Cur
}

// ShowNotice is FUN_004172e0's GUI banner switch. Kind 0 also supplies
// eight animated bonus digits, hiding leading zeros rather than padding.
// It only draws the amount; it never credits GameState.Score.
func (g *GUI) ShowNotice(kind int, amount int64) {
	w, env := g.ctx.World, g.ctx.Anm
	slot := 0
	var id int
	switch kind {
	case 0:
		id = 0x47 // capture, 0x4172f9
	case 1:
		id = 0x48 // failed, 0x417441
	case 2, 3, 4:
		slot, id = 1, kind+0x47 // full power / high score / extend
	case 6:
		id = 0x4c // stage clear, 0x417520
	default:
		return // kind 5 and out-of-range values do nothing
	}
	w.Delete(g.notice[slot])
	g.notice[slot] = w.SpawnTail(g.front, id, 15, env)
	if kind != 0 {
		return
	}
	n, place, shown := min(max(amount, 0), 99999999), int64(10000000), false
	for i := range g.captureDigits {
		w.Delete(g.captureDigits[i])
		v := w.SpawnTail(g.front, i+0x27, 15, env)
		g.captureDigits[i] = v
		digit := n / place
		n %= place
		place /= 10
		shown = shown || digit != 0
		v.SetSprite(int(digit) + 8)
		if !shown {
			w.Hide(v) // FUN_004172e0 0x4173f7
		}
	}
}

// ShowCaptureBonus is FUN_00409c00 0x409cbb -> FUN_004172e0(kind 0).
func (g *GUI) ShowCaptureBonus(amount int64) { g.ShowNotice(0, amount) }

func (s *Stage) nodeSpellHUD(sv *supervisor.Supervisor) {
	if c := s.ctx.Spell; c != nil && !c.Active {
		return // FUN_004091c0 tests the helper's active bit, not the GUI's state
	}
	if s.ctx.GUI != nil {
		s.ctx.GUI.spellHUD.Draw(sv.ANM)
	}
}
