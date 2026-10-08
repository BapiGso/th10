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
	for _, state := range []State{StateNormal, StateRespawn, StateDead} {
		for _, power := range []int{0, 5, 95, 99, 100, 105, 400, 500} {
			p := testPlayer(game.CharReimu)
			p.State, p.GS.Power = state, power
			life := p.GS.Life
			var in input.State
			in.Restore(1 << input.KeyBomb)
			p.UpdateInput(&in)
			want := power
			if power >= 100 {
				want -= 100
			}
			if p.GS.Power != want || (p.State == StateBomb) != (power >= 100) {
				t.Fatalf("state %d power %d: power=%d state=%d", state, power, p.GS.Power, p.State)
			}
			if p.GS.Life != life {
				t.Fatal("bomb changed life")
			}
			if p.State == StateBomb {
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
	if p.State != StateBomb || p.GS.Power != 0 {
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
			if p.State != StateBomb || p.GS.Life != life || p.GS.Power != 400 || p.DeathTimer != 0 {
				t.Fatalf("deathbomb at %d: %+v", frame, p)
			}
		} else if p.State != StateRespawn || p.GS.Life != life-1 || p.GS.Power != 180 {
			t.Fatalf("late deathbomb should commit death: %+v", p)
		}
	}
}

func TestDeathPowerLossClampsAtZero(t *testing.T) {
	for _, initial := range []int{0, 100, 315, 320, 325, 500} {
		p := testPlayer(game.CharReimu)
		p.GS.Power = initial
		p.Hit()
		var in input.State
		for i := 0; i < 9; i++ {
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
					p.X, p.Y = 200, 200
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
					wantY := 200.0
					if diagonal {
						wantY += step
					}
					if p.X != 200+step || p.Y != wantY {
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
		if math.Abs(p.X*100-math.Round(p.X*100)) > 1e-8 {
			t.Fatal("position left fixed-point grid")
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
