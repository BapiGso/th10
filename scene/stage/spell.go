package stage

import "th10/game"

// SpellBonus is the integer formula in FUN_00409280: the stage factor,
// stored faith, then *10. Convert the project's x10 faith before any
// multiplication. Only +0x3794 is capped; +0x3790 keeps the raw result.
// Both fit int32 for every original stage and faith value.
func SpellBonus(stage int, faith int64) (bonus, initial int32) {
	bonus = (int32(stage)*3 + 10) * int32(faith/10) * 10
	return bonus, min(bonus, 99999999)
}

// Declare is the non-visual part of FUN_00409280. ECL arg 2 is evaluated
// by its caller but is not the reward; arg 1 becomes Duration. Capture
// opens, the survival bit clears and the timer resets to {prev:-1,cur:0,f:0}.
func (s *SpellCard) Declare(state *game.GameState, id int32, name string, duration int32) {
	s.Active, s.Capture, s.Survival = true, true, false
	s.ID, s.Name, s.Duration = id, name, duration
	s.Timer.Reset()
	var stage int
	var faith int64
	if state != nil {
		stage, faith = state.Stage, state.Faith
	}
	bonus, initial := SpellBonus(stage, faith)
	s.Bonus, s.InitialBonus = int64(bonus), initial
	state.RecordSpellDeclaration(id, name)
}

// Tick is FUN_00408d60's bonus and timer step (0x408d86-0x408e55).
// Decay starts at the pre-tick timer's 300th frame, once per calc even
// when the game-speed timer does not advance. All divisions are signed
// integer divisions, followed by clearing the units digit. No capture or
// zero clamp is present: a failed card's invisible bonus can go negative.
func (s *SpellCard) Tick(speed float32) {
	if !s.Active {
		return
	}
	s.DecayBonus()
	s.Timer.Tick(speed)
}

// DecayBonus is the pre-ANM part of FUN_00408d60. The ECL caller steps
// the two owned cdbg VMs between this and the same Timer.Tick as above.
func (s *SpellCard) DecayBonus() {
	if !s.Active {
		return
	}
	if s.Timer.Cur >= 300 && !s.Survival {
		loss := (s.InitialBonus - s.InitialBonus/10) / (s.Duration - 300)
		next := int32(s.Bonus) - loss
		s.Bonus = int64(next - next%10)
	}
}

// SpawnPlacards is FUN_00409280 0x409422-0x409681: two ascii.anm
// placards around text.anm ordinal 0x48. The initial layer is 15; each
// script changes it to 14 and sets Pos itself, so Offset must stay zero.
// All three use FUN_00449870, preserving the alternate-RNG flag.
func (s *SpellCard) SpawnPlacards(ctx *Context) {
	if ctx == nil || ctx.World == nil || ctx.Anm == nil {
		return
	}
	w, env := ctx.World, ctx.Anm
	s.Placards[0] = w.SpawnTail(asciiFile(), 1, 15, env)   // +0x8994, 0x40942d/0x409457
	s.Placards[1] = w.SpawnTail(textFile(), 0x48, 15, env) // +0x899c, 0x4094d5/0x409500
	s.Placards[2] = w.SpawnTail(asciiFile(), 2, 15, env)   // +0x8994, 0x409580/0x4095ab
	if ctx.GUI != nil {
		ctx.GUI.SetSpellName(s.Placards[1], s.Name) // FUN_00447ae0, raw SJIS, 0x409681
	}
}

// Finish is FUN_00409c00 including its result-GUI calls (0x409c04-0x409d3c).
// Only bit 1 decides the result, even for a zero bonus or a survival card
// ended by HP. The timer, id, bonus and capture flag survive the end; an
// already-ended card cannot pay or record a second time.
func (s *SpellCard) Finish(ctx *Context) bool {
	if !s.Active {
		return false
	}
	if ctx != nil {
		if ctx.STD != nil {
			ctx.STD.BackgroundVisible = true // 0x409c11-0x409c18
		}
		for _, v := range s.Placards {
			ctx.World.Interrupt(v, 1) // FUN_00409e50: root and chain, 0x409c1f-0x409c3b
		}
	}
	s.Active = false
	if ctx != nil {
		ctx.World.Delete(s.Ring) // FUN_004492a0, 0x409c55-0x409c5e
	}
	s.Ring = nil // +0x774, 0x409c63
	if ctx != nil && ctx.GUI != nil {
		ctx.GUI.EndSpellPresentation() // helper bit 0 also gates draw 0x25
	}
	if !s.Capture {
		if ctx != nil && ctx.GUI != nil {
			ctx.GUI.ShowNotice(1, 0) // FUN_004492a0 / FUN_00448d00, 0x409d49-0x409d76
		}
		return false
	}
	if ctx != nil && ctx.State != nil {
		ctx.State.AddPoints(s.Bonus)
	}
	if ctx != nil && ctx.GUI != nil {
		ctx.GUI.ShowCaptureBonus(s.Bonus) // FUN_004172e0(kind 0), before history, 0x409cbb
	}
	if ctx != nil && ctx.State != nil {
		ctx.State.RecordSpellCapture(s.ID)
	}
	return true
}
