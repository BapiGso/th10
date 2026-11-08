package ecl

import (
	"testing"

	"th10/anm"
	"th10/game"
	"th10/render"
	"th10/scene/stage"

	"golang.org/x/text/encoding/japanese"
)

func spellGUIFixture(t *testing.T) (*VM, *stage.Context, *eclActor) {
	t.Helper()
	vm, ctx, boss, _ := visualVM(t)
	ctx.State.Stage = 1
	ctx.GUI = stage.NewGUI(ctx)
	vm.bossSlots[0] = boss.e
	return vm, ctx, boss
}

func TestSpellGUIInitializesEveryDeclaration(t *testing.T) {
	vm, ctx, boss := spellGUIFixture(t)
	ctx.State.Character, ctx.State.ShotType = game.CharMarisa, game.ShotC
	ctx.State.SpellHistory = &game.SpellHistory{}
	ctx.State.SpellHistory.Shots[5][2] = game.SpellRecord{Captures: 17, Attempts: 101}
	raw, err := japanese.ShiftJIS.NewEncoder().String("Test \u7b26\u540d")
	if err != nil {
		t.Fatal(err)
	}
	for range 2 {
		gameOp(vm, ctx, boss, opSpellBg, intParam(2), intParam(2700), intParam(999), stringParam(raw))
		s := &vm.card
		if s.HUD == nil || !s.HUD.Active || ctx.Spell != s || s.Name != raw {
			t.Fatalf("declaration did not bind the helper HUD: %+v", s)
		}
		for i, v := range s.HUD.Bonus {
			if v.ScriptID != i+0x3a || v.T.Cur != 1 || v.Flags&anm.FlagDraw == 0 {
				t.Errorf("bonus VM %d: script %#x timer %d flags %#x", i, v.ScriptID, v.T.Cur, v.Flags)
			}
		}
		for i, v := range s.HUD.History {
			if v.ScriptID != i+0x42 || v.T.Cur != 1 {
				t.Errorf("history VM %d: script %#x timer %d", i, v.ScriptID, v.T.Cur)
			}
		}
		name := s.Placards[1]
		if name == nil {
			t.Fatal("no name texture VM")
		}
		want := *name
		render.WriteVMText(&want, "Test \u7b26\u540d", render.TextStyle{Right: true, Color: 0xffffff})
		if name.File != want.File {
			t.Fatal("GUI did not decode the raw SJIS name at the font boundary")
		}
		vm.LateUpdate(ctx)
		if s.HUD.Bonus[0].T.Cur != 2 || s.HUD.History[0].T.Cur != 2 {
			t.Fatal("spell calc did not step the inline VMs exactly once")
		}
		ctx.GUI.Calc()
		if s.HUD.Bonus[0].T.Cur != 2 || s.HUD.History[0].T.Cur != 2 {
			t.Fatal("GUI calc also stepped the externally driven spell VMs")
		}
		if got := [5]int{s.HUD.History[0].Sprite, s.HUD.History[1].Sprite, s.HUD.History[2].Sprite,
			s.HUD.History[3].Sprite, s.HUD.History[4].Sprite}; got != [5]int{31, 37, 40, 39, 39} {
			t.Fatalf("shot-specific history sprites %v, want 17/99", got)
		}
		if caps, attempts := ctx.State.SpellCounts(2); caps != 17 || attempts < 102 {
			t.Fatalf("display changed the raw history: %d/%d", caps, attempts)
		}
		s.Timer.Set(60)
		s.Bombed()
	}
}

func TestSpellGUIFailureStepsOnlyThirdBonusVM(t *testing.T) {
	vm, ctx, boss := spellGUIFixture(t)
	gameOp(vm, ctx, boss, opSpellBg, intParam(2), intParam(2700), intParam(0), stringParam("card"))
	s := &vm.card
	s.Timer.Set(60)
	s.Bombed()
	before := [8]int32{}
	for i := range before {
		before[i] = s.HUD.Bonus[i].T.Cur
	}
	vm.LateUpdate(ctx)
	for i, v := range s.HUD.Bonus {
		want := before[i]
		if i == 2 {
			want++
			if v.Sprite != 0x29 || v.Flags&anm.FlagDraw == 0 {
				t.Errorf("failed marker sprite %#x flags %#x", v.Sprite, v.Flags)
			}
		} else if v.Flags&anm.FlagDraw != 0 {
			t.Errorf("failed bonus VM %d is still drawable", i)
		}
		if v.T.Cur != want {
			t.Errorf("bonus VM %d timer %d, want %d", i, v.T.Cur, want)
		}
	}
	for i, v := range s.HUD.History {
		if v.T.Cur != 2 {
			t.Errorf("history VM %d timer %d, want 2", i, v.T.Cur)
		}
	}
}

func TestSpellGUITimeoutCaptureGate(t *testing.T) {
	for _, tc := range []struct {
		timer    int32
		survival bool
		lost     bool
	}{{59, false, false}, {60, false, true}, {60, true, false}} {
		vm, ctx, boss := spellGUIFixture(t)
		gameOp(vm, ctx, boss, opSpellBg, intParam(2), intParam(2700), intParam(0), stringParam("card"))
		vm.card.Timer.Set(tc.timer)
		vm.card.Survival = tc.survival
		boss.slots[0] = interruptSlot{hp: 0, time: 1, timeoutSub: "end"}
		boss.phase.set(1)
		vm.checkInterrupt(boss.e, boss)
		for i, v := range vm.card.HUD.Bonus {
			if want := !tc.lost || i == 2; (v.Flags&anm.FlagDraw != 0) != want {
				t.Errorf("timer %d survival %v VM %d draw %v, want %v", tc.timer, tc.survival, i,
					v.Flags&anm.FlagDraw != 0, want)
			}
		}
	}
}

func TestSpellGUIResultWorldScripts(t *testing.T) {
	for _, capture := range []bool{false, true} {
		vm, ctx, boss := spellGUIFixture(t)
		gameOp(vm, ctx, boss, opSpellBg, intParam(2), intParam(2700), intParam(0), stringParam("card"))
		vm.card.Capture, vm.card.Bonus = capture, 1230
		vm.endSpell(ctx)
		if vm.card.HUD.Active {
			t.Fatal("ended spell still enables the digit node")
		}
		ctx.UpdateWorld()
		want, count, digits := 0x48, 0, 0
		if capture {
			want = 0x47
		}
		ctx.World.Each(15, func(v *anm.VM) {
			if v.ScriptID == want && v.File != vm.card.Placards[1].File {
				count++
				if v.Flags&anm.FlagAltRNG == 0 {
					t.Fatal("result banner uses the main RNG")
				}
			}
			if v.ScriptID >= 0x27 && v.ScriptID <= 0x2e {
				digits++
			}
		})
		if count != 1 || capture && digits != 8 || !capture && digits != 0 {
			t.Fatalf("capture %v: result banners %d bonus digits %d", capture, count, digits)
		}
		before := ctx.World.Len()
		vm.endSpell(ctx)
		if ctx.World.Len() != before {
			t.Fatal("repeated 343 created another result")
		}
	}
}
