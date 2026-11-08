package ecl

import (
	"math"

	"th10/game"
	"th10/scene/stage"
)

// FUN_00408d60's constants, read from th10.exe's .rdata.
var (
	spellFadeY  = math.Float32frombits(0x42c00000) // DAT_00470c80: 96
	spellShowY  = math.Float32frombits(0x43000000) // DAT_00470bf4: 128
	spellFollow = math.Float32frombits(0x3d4ccccd) // DAT_00470c84: 0.05
)

// spawnSpellEffects is the bullet.anm and cut-in part of FUN_00409280.
// FUN_00449870 preserves flag 0x40000000; children are linked first, then
// the root joins the active World's tail (FUN_004489d0). Positions set by
// the ANM script are not shifted a second time by World.Spawn.
func (vm *VM) spawnSpellEffects(ctx *stage.Context) {
	w, env := effects(ctx)
	if w == nil || env == nil {
		return
	}
	s := &vm.card
	s.SpawnPlacards(ctx)                    // FUN_00409280 0x409422-0x409681, before the ring
	ctx.PlaySEID(0xe)                       // FUN_0043dc90, EDI=14 at 0x40968a-0x409694
	f := vm.fileAt(ctx, 0)                  // bullet.anm: scripts 0x19f-0x1ab
	s.Ring = w.SpawnTail(f, 0x1a1, 15, env) // FUN_00409280, handle +0x774
	if boss := vm.boss0(); boss != nil {
		s.Position = boss.mv.final.pos // +0x379c, copied at the declaration
	}
	w.Place(s.Ring, s.Position) // FUN_00449350
	for _, id := range []int{0x19f, 0x1a0} {
		if child := s.Ring.Child(id); child != nil {
			child.I[2] = s.Duration // VM+0x314, FUN_00409280 0x4097e5/0x409868
		}
	}
	w.SpawnTail(f, 0x1ab, 15, env) // free declaration flash, no handle or offset
	if ctx.State == nil {
		return
	}
	// FUN_00409280 0x409964-0x409b95: FUN_00404f30 starts inline cdbg
	// VMs from manager +0x38 (file table index 2), then a World cut-in.
	id := -1
	background := [2]int{-1, -1}
	switch ctx.State.Stage {
	case 1:
		background = [2]int{0xc, 0xb} // 0x409971/0x40998b: b before a
		id = 0xe
		if s.ID < 2 {
			id = 0xf
		}
	case 2:
		background = [2]int{0xe, 0xf} // 0x4099d8/0x4099f2
		id = 0x11
	case 3:
		background = [2]int{0x12, 0x13} // 0x409a23/0x409a3c
		id = 0x15
	case 4:
		background = [2]int{0x13, 0x14} // 0x409a6d/0x409a87
		id = 0x16
	case 5:
		background = [2]int{0xc, 0xd} // 0x409ab9/0x409ad3
		id = 0xf
	case 6:
		background = [2]int{0x21, 0x22} // 0x409b04/0x409b1d
		id = 0x24
	case 7:
		if ctx.BossSection < 0x18 {
			background[0], id = 0x26, 0x28 // 0x409b54/0x409b72: no write to VM 1
		} else {
			background, id = [2]int{0x1b, 0x1c}, 0x1e // 0x409b76/0x409b90/0x409bae
		}
	}
	f = vm.fileAt(ctx, 2)
	for i, script := range background {
		if script >= 0 {
			v := &s.Background[i]
			// FUN_00404f30: reset, zero +0x334/+0x340/+0x34c,
			// then FUN_0043e710. No World insertion or alternate flag.
			v.Reset()
			v.Pos, v.Offset, v.Pos2 = [3]float32{}, [3]float32{}, [3]float32{}
			v.Start(f, script, env)
		}
	}
	if id >= 0 {
		w.SpawnTail(f, id, 15, env) // FUN_00449870 / FUN_004489d0, 0x409bb0-0x409bd7
	}
}

// updateSpell is FUN_00408d60 (calc 0x16). The pre-tick timer hides STD
// after 60 frames; cdbg steps before the timer; placard fading and the
// smoothed ring position follow it. The bonus/history display is t3's.
func (vm *VM) updateSpell(ctx *stage.Context) {
	s := &vm.card
	if !s.Active {
		return
	}
	if ctx != nil && s.Timer.Cur >= 60 && ctx.STD != nil {
		ctx.STD.BackgroundVisible = false // FUN_00408d60 0x408d71-0x408d7f
	}
	s.DecayBonus() // 0x408d86-0x408ddc
	if ctx != nil && ctx.Anm != nil {
		for i := range s.Background {
			if s.Background[i].File != nil {
				s.Background[i].Run(ctx.Anm) // 0x408de2-0x408df2
			}
		}
	}
	s.Timer.Tick(vm.gameSpeed) // unchanged 0x408df7-0x408e55
	if ctx == nil {
		return
	}
	w := ctx.World
	// FUN_00408d60 0x408e5b-0x408f76: threshold hysteresis, tested after
	// the timer has advanced. All three roots and their chains take it.
	if s.Timer.Cur >= 120 && ctx.Player != nil {
		y := float32(ctx.Player.Y - float64(game.FieldTop))
		var label int16
		if !s.Faded && y < spellFadeY {
			label, s.Faded = 3, true
		} else if s.Faded && y > spellShowY {
			label, s.Faded = 2, false
		}
		if label != 0 {
			for _, v := range s.Placards {
				w.Interrupt(v, label) // FUN_00449470
			}
			if ctx.GUI != nil {
				ctx.GUI.InterruptSpellPresentation(label) // inline +0x304 writes, 0x408e9c/0x408f32
			}
		}
	}
	if ctx.GUI != nil {
		captures, attempts := ctx.State.SpellCounts(s.ID)
		ctx.GUI.UpdateSpellPresentation(s.Bonus, s.Capture, captures, attempts, s.Timer.Cur)
	}
	if boss := vm.boss0(); boss != nil {
		for i := range s.Position {
			d := float32(boss.mv.final.pos[i] - s.Position[i])
			s.Position[i] = float32(float32(d*spellFollow) + s.Position[i]) // 0x40910e-0x409166
		}
		w.Place(s.Ring, s.Position) // FUN_00449350, 0x409169-0x409170
	}
}
