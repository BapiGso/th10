package player

import (
	"testing"

	"th10/entity/playerbullet"
	"th10/game"
	"th10/input"
)

func TestAllSixSHTGroupsFireAtOriginalPhase(t *testing.T) {
	for character := range 2 {
		for shot := range 3 {
			for _, power := range []int{0, 100, 200, 300, 400, 500} {
				for _, focus := range []bool{false, true} {
					state := game.NewState(character, shot, game.DiffNormal)
					state.Power = power
					p := New(state, game.FieldLeft, game.FieldTop, game.FieldRight, game.FieldBottom)
					p.initTimer.Set(4)
					var in input.State
					keys := uint16(1 << input.KeyShot)
					if focus {
						keys |= 1 << input.KeyFocus
					}
					in.Restore(keys)
					p.UpdateInput(&in)
					var want []int
					for _, s := range p.shotFile.Shots(power, focus) {
						if s.Delay == 0 {
							want = append(want, s.Script)
						}
					}
					got := newborns(p)
					if len(got) != len(want) {
						t.Fatalf("character %d shot %d power %d focus %v: births=%d want=%d", character, shot, power, focus, len(got), len(want))
					}
					for i, b := range got {
						if b.ScriptID != want[i] {
							t.Fatalf("character %d shot %d power %d focus %v: script %d want %d", character, shot, power, focus, b.ScriptID, want[i])
						}
					}
				}
			}
		}
	}
}

func TestSHTClockSlowFramesDoNotRepeatShots(t *testing.T) {
	p := New(game.NewState(game.CharReimu, game.ShotB, game.DiffNormal), game.FieldLeft, game.FieldTop, game.FieldRight, game.FieldBottom)
	p.GameSpeed = func() float32 { return 0.5 }
	var in input.State
	in.Restore(1 << input.KeyShot)
	for frame := range 7 {
		p.UpdateInput(&in)
		want := 2
		if frame == 6 {
			want = 4
		}
		if got := p.Bullets.ActiveCount(); got != want {
			t.Fatalf("frame %d shots=%d want=%d", frame, got, want)
		}
	}
	first := true
	p.Bullets.Each(func(b *playerbullet.Bullet) {
		if first {
			first = false
			if b.Y != game.FieldTop+400-8-6*24 || b.Age != 3 {
				t.Fatalf("slow timer incorrectly scaled shot motion: Y=%v age=%d", b.Y, b.Age)
			}
		}
	})
}

func TestMarisaBLasersStayOnePerSourceAndEndWithBurst(t *testing.T) {
	p, w, env := optionsPlayer(game.CharMarisa, game.ShotB, 400, nil)
	var in input.State
	in.Restore(1 << input.KeyShot)
	beams := 0
	for frame := range 15 {
		p.UpdateInput(&in)
		w.Run(env)
		beams = 0
		p.Bullets.Each(func(b *playerbullet.Bullet) {
			if b.ScriptID == 10 {
				beams++
				if !w.Contains(b.VM) || !w.Contains(b.ExtraVM) || b.ExtraVM.ScriptID != 0x10 || b.VM.Layer != 8 {
					t.Fatal("laser did not retain both original World handles")
				}
			}
		})
		if beams != 4 {
			t.Fatalf("frame %d beams=%d want=4", frame, beams)
		}
	}
	in.Restore(0)
	p.UpdateInput(&in) // burst clock 15 -> -1, both laser VMs take label 1
	beams = 0
	p.Bullets.Each(func(b *playerbullet.Bullet) {
		if b.ScriptID == 10 {
			beams++
		}
	})
	if beams != 0 {
		t.Fatal("released burst left damaging lasers alive")
	}
}

func TestSHTNoAttackResetsBurstButMovesExistingShots(t *testing.T) {
	p := New(game.NewState(game.CharReimu, game.ShotC, game.DiffNormal), game.FieldLeft, game.FieldTop, game.FieldRight, game.FieldBottom)
	var in input.State
	in.Restore(1 << input.KeyShot)
	p.UpdateInput(&in)
	count := p.Bullets.ActiveCount()
	p.NoAttack = true
	p.UpdateInput(&in)
	if p.reimuA.timer.Cur != -1 || p.Bullets.ActiveCount() != count {
		t.Fatal("dialog gate failed to reset the burst or removed existing shots")
	}
	p.Bullets.Each(func(b *playerbullet.Bullet) {
		if b.Age != 2 {
			t.Fatal("dialog gate froze an existing shot")
		}
	})
}
