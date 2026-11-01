package player

import (
	"math"
	"testing"
	"th10/game"
	"th10/input"
)

func testPlayer(character int) *Player {
	return New(game.NewState(character, game.ShotA, game.DiffNormal), game.FieldLeft, game.FieldTop, game.FieldRight, game.FieldBottom)
}

func TestBombConsumesPower(t *testing.T) {
	// States 1 and 4 bomb; the respawn (before its last frame) and dying
	// states do not.
	for _, state := range []State{StateNormal, StateDeathbomb, StateRespawn, StateDying} {
		for _, power := range []int{0, 5, 95, 99, 100, 105, 400, 500} {
			p := testPlayer(game.CharReimu)
			p.State, p.GS.Power = state, power
			life := p.GS.Life
			var in input.State
			in.Restore(1 << input.KeyBomb)
			p.UpdateInput(&in)
			want := power
			bombs := power >= 100 && (state == StateNormal || state == StateDeathbomb)
			if bombs {
				want -= 100
			}
			if p.GS.Power != want || (p.BombTimer > 0) != bombs {
				t.Fatalf("state %d power %d: power=%d state=%d", state, power, p.GS.Power, p.State)
			}
			if bombs && p.State != StateNormal {
				t.Fatalf("state %d: bomb left state %d", state, p.State)
			}
			if p.GS.Life != life {
				t.Fatal("bomb changed life")
			}
			if bombs {
				in.Restore(1 << input.KeyBomb)
				p.UpdateInput(&in)
				if p.GS.Power != want {
					t.Fatal("active bomb charged again")
				}
			}
		}
	}
}

func TestBombUsesHeldInput(t *testing.T) {
	p := testPlayer(game.CharReimu)
	var in input.State
	in.Restore(1 << input.KeyBomb)
	p.UpdateInput(&in) // no power yet
	p.AddPower(100)
	in.Restore(1 << input.KeyBomb)
	if in.JustPressed(input.KeyBomb) {
		t.Fatal("test must use held input")
	}
	p.UpdateInput(&in)
	if p.BombTimer == 0 || p.GS.Power != 0 {
		t.Fatal("original held-X trigger not honored")
	}
}

func TestDeathbombWindow(t *testing.T) {
	for _, frame := range []int{1, 8, 9} {
		p := testPlayer(game.CharReimu)
		p.GS.Power = 500
		life := p.GS.Life
		p.Hit()
		var in input.State
		for i := 1; i <= frame; i++ {
			bits := uint16(0)
			if i == frame {
				bits = 1 << input.KeyBomb
			}
			in.Restore(bits)
			p.UpdateInput(&in)
		}
		if frame <= 8 {
			// FUN_00425730 state 4: StateTimer 60, 200 frames invulnerable.
			if p.State != StateNormal || p.GS.Life != life || p.GS.Power != 400 ||
				p.BombTimer == 0 || p.StateTimer.Cur != 61 || p.Invinc.Cur != 199 {
				t.Fatalf("deathbomb at %d: %+v", frame, p)
			}
			continue
		}
		// The power goes 3 frames into the dying state.
		if p.State != StateDying || p.GS.Life != life-1 || p.GS.Power != 500 {
			t.Fatalf("late deathbomb should commit death: %+v", p)
		}
		in.Restore(0)
		for i := 0; i < 3; i++ {
			p.UpdateInput(&in)
		}
		if p.GS.Power != 180 {
			t.Fatalf("power %d after the drop", p.GS.Power)
		}
	}
}

// TestDeathAndRespawn follows FUN_00425730 from the hit: 8 frames to
// deathbomb, 30 dying, then 60 rising from y 480 to 400.
func TestDeathAndRespawn(t *testing.T) {
	p := testPlayer(game.CharReimu)
	p.SetFixed(1000, 20000)
	p.GS.Faith, p.GS.Rank = 80000, 100
	life := p.GS.Life
	p.Hit()
	var in input.State
	n := 0
	for p.State != StateDying {
		p.UpdateInput(&in)
		n++
	}
	// FUN_004269d0: a third of the faith above 5000 and 1024 rank go.
	if n != 9 || p.GS.Life != life-1 || p.GS.Faith != 70000 || p.GS.Rank != -924 || p.Invinc.Cur != 179 {
		t.Fatalf("death after %d: life %d faith %d rank %d invinc %d", n, p.GS.Life, p.GS.Faith, p.GS.Rank, p.Invinc.Cur)
	}
	for p.State != StateRespawn {
		p.UpdateInput(&in)
		n++
	}
	a := p.areas[0]
	if n != 39 || p.IX != 0 || p.IY != 48000 || p.Invinc.Cur != 279 ||
		!a.active || a.pos != [2]float32{10, 200} || a.radius != 48 || a.timer.Cur != 29 {
		t.Fatalf("respawn after %d: %d,%d invinc %d area %+v", n, p.IX, p.IY, p.Invinc.Cur, a)
	}
	for p.State != StateNormal {
		p.UpdateInput(&in)
		n++
	}
	if n != 99 || p.IY != 40000 || p.Invinc.Cur != 219 || p.areas[0].active {
		t.Fatalf("back after %d: %d invinc %d", n, p.IY, p.Invinc.Cur)
	}
}

func TestHittable(t *testing.T) {
	p := testPlayer(game.CharReimu)
	for _, c := range []struct {
		s    State
		want bool
	}{{StateRespawn, true}, {StateNormal, true}, {StateDying, false}, {State3, false}, {StateDeathbomb, false}} {
		p.State = c.s
		if p.Hittable() != c.want {
			t.Fatalf("state %d hittable %v", c.s, !c.want)
		}
	}
}

func TestAreaDamage(t *testing.T) {
	p := testPlayer(game.CharReimu)
	p.addArea(0, 0, 32, 16, 30, 150)
	if d := p.AreaDamage(0, 0); d != 150 {
		t.Fatalf("damage %d on the spawn frame", d)
	}
	p.updateAreas(1) // timer 29, r 48
	if d := p.AreaDamage(48, 0); d != 150 {
		t.Fatalf("damage %d at the edge", d)
	}
	if d := p.AreaDamage(48.01, 0); d != 0 {
		t.Fatalf("damage %d outside", d)
	}
	p.updateAreas(1) // timer 28, a multiple of the interval: skipped
	if d := p.AreaDamage(0, 0); d != 0 {
		t.Fatalf("damage %d on a skipped frame", d)
	}
}

func TestDeathPowerLossClampsAtZero(t *testing.T) {
	for _, initial := range []int{0, 100, 315, 320, 325, 500} {
		p := testPlayer(game.CharReimu)
		p.GS.Power = initial
		p.Hit()
		var in input.State
		for i := 0; i < 12; i++ {
			p.UpdateInput(&in)
		}
		if p.GS.Power != max(initial-320, 0) {
			t.Fatalf("power %d -> %d", initial, p.GS.Power)
		}
	}
}

func TestPowerPickups(t *testing.T) {
	p := testPlayer(game.CharReimu)
	p.AddPower(game.SmallPowerValue)
	if p.GS.Power != 5 {
		t.Fatal(p.GS.Power)
	}
	p.AddPower(game.BigPowerValue)
	if p.GS.Power != 105 {
		t.Fatal(p.GS.Power)
	}
	p.AddPower(1000)
	if p.GS.Power != 500 {
		t.Fatal(p.GS.Power)
	}
	p.AddPower(-1000)
	if p.GS.Power != 0 {
		t.Fatal(p.GS.Power)
	}
}

func TestSHTMovement(t *testing.T) {
	for character := 0; character < 2; character++ {
		for shot := 0; shot < 3; shot++ {
			for _, focused := range []bool{false, true} {
				for _, diagonal := range []bool{false, true} {
					p := New(game.NewState(character, shot, game.DiffNormal), game.FieldLeft, game.FieldTop, game.FieldRight, game.FieldBottom)
					p.SetPos(200, 200)
					var idle input.State
					for i := 0; i < 4; i++ { // focus is held off for 4 frames
						p.UpdateInput(&idle)
					}
					x0, y0 := p.IX, p.IY
					bits := uint16(1 << input.KeyRight)
					step := 4.5
					if character == game.CharMarisa {
						step = 5
					}
					if diagonal {
						bits |= 1 << input.KeyDown
						step = 3.18
						if character == game.CharMarisa {
							step = 3.53
						}
					}
					if focused {
						bits |= 1 << input.KeyFocus
						step = 2
						if diagonal {
							step = 1.41
						}
					}
					var in input.State
					in.Restore(bits)
					p.UpdateInput(&in)
					stepInt := int32(math.Round(step * 100))
					wantY := y0
					if diagonal {
						wantY += stepInt
					}
					if p.IX != x0+stepInt || p.IY != wantY {
						t.Fatalf("%d/%d focus=%v diagonal=%v: %v,%v", character, shot, focused, diagonal, p.X, p.Y)
					}
				}
			}
		}
	}
}

func TestMovementBoundsAndRepeatability(t *testing.T) {
	p, q := testPlayer(game.CharReimu), testPlayer(game.CharReimu)
	var a, b input.State
	for frame := 0; frame < 600; frame++ {
		bits := uint16(1<<input.KeyUp | 1<<input.KeyLeft)
		if frame%60 >= 30 {
			bits = 1<<input.KeyDown | 1<<input.KeyRight | 1<<input.KeyFocus
		}
		a.Restore(bits)
		b.Restore(bits)
		p.UpdateInput(&a)
		q.UpdateInput(&b)
		if p.X != q.X || p.Y != q.Y {
			t.Fatal("same snapshots diverged")
		}
		if p.X != 224+float64(float32(p.IX)*float32(0.01)) {
			t.Fatal("float position not derived from the fixed-point one")
		}
	}
	a.Restore(1<<input.KeyUp | 1<<input.KeyLeft)
	for i := 0; i < 200; i++ {
		p.UpdateInput(&a)
	}
	if p.X != game.FieldLeft+8 || p.Y != game.FieldTop+32 {
		t.Fatalf("bounds %v,%v", p.X, p.Y)
	}
}
