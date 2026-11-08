package player

import (
	"slices"
	"testing"

	"th10/anm"
	"th10/game"
	"th10/input"
	"th10/zun"
)

func optionScripts() *anm.File {
	f := &anm.File{Scripts: map[int]*anm.Script{0: {ID: 0, Instrs: []anm.Instr{anmInstr(63)}}}}
	for id := 0x11; id <= 0x16; id++ {
		f.Scripts[id] = &anm.Script{ID: id, Instrs: []anm.Instr{
			anmInstr(40, 10000, 100), anmInstr(3, 100), anmInstr(63),
			anmInstr(64, 3), anmInstr(3, 103), anmInstr(63),
			anmInstr(64, 6), anmInstr(3, 106), anmInstr(63),
			anmInstr(64, 1), anmInstr(1),
		}}
	}
	return f
}

func optionsPlayer(character, shot, power int, f *anm.File) (*Player, *anm.World, *anm.Env) {
	p := New(game.NewState(character, shot, game.DiffNormal), game.FieldLeft, game.FieldTop, game.FieldRight, game.FieldBottom)
	p.GS.Power = power
	if f != nil {
		p.Anm = f
	}
	w := &anm.World{}
	env := &anm.Env{GameSpeed: 1, RNG: &zun.RNG{State: 0x1234}, Alt: &zun.RNG{}, Spawn: w.Add}
	p.AttachAnm(w, env, playerEffects())
	p.initTimer.Set(4)
	return p, w, env
}

func TestOptionWorldScriptsAndPlacement(t *testing.T) {
	for character := range 2 {
		for shot := range 3 {
			p, w, env := optionsPlayer(character, shot, 400, nil)
			if w.Len() != 4 || p.optionCount != 4 {
				t.Fatalf("character %d shot %d: world=%d options=%d", character, shot, w.Len(), p.optionCount)
			}
			var in input.State
			in.Restore(1 << input.KeyRight)
			p.move(&in)
			for i := range p.options {
				o := &p.options[i]
				want := [3]float32{float32(o.pos[0])*0.01 + 224, float32(o.pos[1])*0.01 + 16, 0}
				if !w.Contains(o.vm) || o.vm.File != p.Anm || o.vm.ScriptID != 0x11+shot ||
					o.vm.Layer != 8 || o.vm.Flags&anm.FlagAltRNG == 0 || o.vm.Offset != want {
					t.Fatalf("character %d shot %d option %d: VM %+v", character, shot, i, o.vm)
				}
			}
			w.Run(env)
			var got []*anm.VM
			w.Each(8, func(v *anm.VM) { got = append(got, v) })
			for i, v := range got {
				if i >= 4 || v != p.options[i].vm {
					t.Fatal("option main VMs are not tail-inserted in slot order")
				}
			}
			if len(got) != 4 || env.RNG.State != 0x1234 {
				t.Fatalf("world layer count %d game RNG %#x", len(got), env.RNG.State)
			}
		}
	}
}

func TestOptionFullPowerFrontOrderAndLayerArgument(t *testing.T) {
	f := optionScripts() // no op 68: this observes the original spawn argument, 15
	p := New(game.NewState(game.CharReimu, game.ShotB, game.DiffNormal), game.FieldLeft, game.FieldTop, game.FieldRight, game.FieldBottom)
	p.GS.Power, p.Anm = 500, f
	w := &anm.World{}
	env := &anm.Env{GameSpeed: 1, RNG: &zun.RNG{State: 0x1234}, Alt: &zun.RNG{}, Spawn: w.Add}
	old := w.SpawnTail(f, 0, 15, env)
	p.AttachAnm(w, env, playerEffects())
	var in input.State
	p.move(&in)
	w.Run(env)
	var got []*anm.VM
	w.Each(15, func(v *anm.VM) { got = append(got, v) })
	want := []*anm.VM{p.options[3].extra, p.options[2].extra, p.options[1].extra, p.options[0].extra, old,
		p.options[0].vm, p.options[1].vm, p.options[2].vm, p.options[3].vm}
	if !slices.Equal(got, want) {
		t.Fatal("full-power effects must precede the old list, in reverse slot order")
	}
	for i := range p.options {
		o := &p.options[i]
		if o.extra.ScriptID != 0x15 || o.vm.ScriptID != 0x12 || o.extra.Offset != o.vm.Offset {
			t.Fatalf("slot %d main/effect mismatch", i)
		}
	}
	if env.RNG.Count != 0 || env.Alt.Count != 16 {
		t.Fatalf("option starts used wrong RNG: game=%d alt=%d", env.RNG.Count, env.Alt.Count)
	}
}

func TestOptionRefreshSameCountAndDrop(t *testing.T) {
	p, w, env := optionsPlayer(game.CharReimu, game.ShotA, 500, optionScripts())
	var mains, extras [4]*anm.VM
	for i, o := range p.options {
		mains[i], extras[i] = o.vm, o.extra
	}
	p.RefreshOptions()
	for i, o := range p.options {
		if o.vm != mains[i] || o.extra == extras[i] || extras[i].Flags&anm.FlagDelete == 0 {
			t.Fatalf("same-count refresh failed at %d", i)
		}
		extras[i] = o.extra
	}
	w.Run(env)
	if w.Len() != 8 {
		t.Fatalf("refresh retained %d objects, want 8", w.Len())
	}
	p.GS.Power = 100
	p.RefreshOptions()
	if mains[0].Flags&anm.FlagDelete == 0 || p.options[0].vm == mains[0] || p.optionCount != 1 {
		t.Fatal("remaining slot was not rebuilt after a count change")
	}
	for i := 1; i < 4; i++ {
		if mains[i].Flags&anm.FlagDelete != 0 || p.options[i].state != 0 {
			t.Fatal("removed options must interrupt, not mark their main VMs for deletion")
		}
	}
	w.Run(env)
	for i := range 4 {
		if w.Contains(mains[i]) || w.Contains(extras[i]) {
			t.Fatal("label 1 did not end the removed main/effect scripts")
		}
	}
	if w.Len() != 1 || !w.Contains(p.options[0].vm) {
		t.Fatal("power drop removed the newly rebuilt main VM")
	}
}

func TestOptionDeathInterruptsBothHandles(t *testing.T) {
	p, w, env := optionsPlayer(game.CharReimu, game.ShotC, 500, optionScripts())
	old := p.options
	p.die()
	if p.optionCount != 0 || p.Options() != nil || w.Len() != 8 {
		t.Fatal("death must clear logical options before the exiting scripts run")
	}
	for _, o := range old {
		if o.vm.Flags&anm.FlagDelete != 0 || o.extra.Flags&anm.FlagDelete != 0 {
			t.Fatal("death deleted an option instead of interrupting label 1")
		}
	}
	w.Run(env)
	if w.Len() != 0 {
		t.Fatal("death did not interrupt both handles")
	}
}

func TestPositionHistoryMovementGates(t *testing.T) {
	p := testPlayer(game.CharReimu)
	p.optionCount, p.IX, p.IY = 2, -120, 40001
	for i := range p.positionHistory {
		p.positionHistory[i] = [2]int32{int32(i), -int32(i)}
	}
	before := p.positionHistory
	p.updatePositionHistory(true)
	if p.positionHistory[0] != [2]int32{p.IX, p.IY} {
		t.Fatal("history head is not the current fixed-point player position")
	}
	for i := 1; i <= 16; i++ {
		if p.positionHistory[i] != before[i-1] {
			t.Fatalf("history shift at %d", i)
		}
	}
	for i := 17; i <= 32; i++ {
		if p.positionHistory[i] != before[16] {
			t.Fatalf("history tail at %d does not repeat the terminal sample", i)
		}
	}
	for _, focused := range []bool{false, true} {
		before = p.positionHistory
		p.focus = focused
		p.updatePositionHistory(focused)
		if p.positionHistory != before {
			t.Fatal("stationary or focused movement shifted history")
		}
	}
}

func TestMarisaATrailInterpolationWeights(t *testing.T) {
	p := testPlayer(game.CharMarisa)
	p.IX, p.IY, p.focus = 1000, 2000, true
	p.positionHistory[0] = [2]int32{1000, 2000}
	o := &p.options[0]
	o.fast = [2]int32{800, -1600}
	p.trailOption(o)
	for i := 1; i <= 8; i++ {
		want := [2]int32{1000 + int32(i)*100, 2000 - int32(i)*200}
		if p.positionHistory[i] != want {
			t.Fatalf("history %d=%v want=%v", i, p.positionHistory[i], want)
		}
	}
	if !o.lastFocus || o.target != [2]int32{1800, 400} {
		t.Fatal("focused trail callback did not retain its offset")
	}
	p.focus = false
	p.positionHistory[8] = [2]int32{2000, 3000}
	p.trailOption(o)
	if o.fast != [2]int32{1000, 1000} || o.target != p.positionHistory[8] || o.lastFocus {
		t.Fatal("unfocused trail callback did not use the terminal history entry")
	}
}

func TestMarisaCLockInterruptsAndReleaseDelay(t *testing.T) {
	p, w, env := optionsPlayer(game.CharMarisa, game.ShotC, 100, optionScripts())
	var in input.State
	p.move(&in)
	p.move(&in) // lockOption saves the previous position before the snap/easing step
	o := &p.options[0]
	saved := o.pos
	in.Restore(1<<input.KeyFocus | 1<<input.KeyRight)
	p.move(&in)
	w.Run(env)
	if o.vm.Sprite != 103 || o.target != saved || o.pos != saved || p.optionEase != 0 {
		t.Fatalf("focus lock: sprite %d target %v pos %v ease %d", o.vm.Sprite, o.target, o.pos, p.optionEase)
	}
	in.Restore(1 << input.KeyRight)
	p.move(&in)
	w.Run(env)
	if o.vm.Sprite != 106 || p.optionEase != 1 || o.pos != saved {
		t.Fatal("focus release did not interrupt label 6 or hold the option")
	}
	for range 28 {
		p.move(&in)
	}
	if p.optionEase != 29 || o.pos != saved {
		t.Fatal("option eased before the thirtieth release tick")
	}
	p.move(&in)
	if p.optionEase != 30 || o.pos == saved {
		t.Fatal("option did not resume easing on the thirtieth release tick")
	}
}
