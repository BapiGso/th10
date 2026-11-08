package player

import (
	"math"
	"testing"
	"th10/entity/enemy"
	"th10/entity/playerbullet"
	"th10/game"
	"th10/input"
)

func newborns(p *Player) []playerbullet.Bullet {
	var out []playerbullet.Bullet
	p.Bullets.Each(func(b *playerbullet.Bullet) {
		if b.Age == 1 {
			out = append(out, *b)
		}
	})
	return out
}

func TestReimuASHTBurstClock(t *testing.T) {
	for _, hold := range []bool{false, true} {
		p := testPlayer(game.CharReimu)
		p.GS.Power = 400
		var in input.State
		for frame := 0; frame < 32; frame++ {
			bits := uint16(0)
			if hold || frame == 0 {
				bits = 1 << input.KeyShot
			}
			in.Restore(bits)
			p.UpdateInput(&in)
			want := 0
			if hold || frame < 16 {
				clock := frame % 16
				if clock%3 == 0 {
					want += 2
				}
				if clock%5 == 0 {
					want += 4
				}
			}
			if got := len(newborns(p)); got != want {
				t.Fatalf("hold=%v frame=%d births=%d want=%d", hold, frame, got, want)
			}
		}
	}
}

func TestReimuAPowerLevelsAndBirths(t *testing.T) {
	for _, power := range []int{0, 100, 200, 300, 400, 500} {
		p := testPlayer(game.CharReimu)
		p.GS.Power = power
		var in input.State
		in.Restore(1 << input.KeyShot)
		p.UpdateInput(&in)
		bullets := newborns(p)
		count := min(power/100, 4)
		if len(bullets) != 2+count || len(p.Options()) != count {
			t.Fatalf("power %d bullets=%d options=%d", power, len(bullets), len(p.Options()))
		}
		if math.Abs(bullets[0].X-(p.X-10)) > 1e-4 || math.Abs(bullets[0].Y-(p.Y-8)) > 1e-4 {
			t.Fatal("main muzzle offset incorrect")
		}
		for i, b := range bullets[2:] {
			o := p.Options()[i]
			// FUN_00427e90 (0x427ff2) compensates speed 12 before the first
			// FUN_00428b10 update adds 0.1 (DAT_00470c18 = 0x3dcccccd).
			// FUN_00428280 (0x4283c5) then floors to the 0.01 game grid;
			// at x=0 the negative residual becomes -0.01, not zero.
			angle := float64(math.Float32frombits(0xbfc90fdb))
			initialVX := float32(math.Cos(angle) * 12)
			firstSpeed := float32(float32(12) + math.Float32frombits(0x3dcccccd))
			firstVX := float32(math.Cos(angle) * float64(firstSpeed))
			gx := float32(float32(-initialVX+float32(o.X-p.originX())) + firstVX)
			gx = float32(float32(math.Floor(float64(float32(gx*100)))) * math.Float32frombits(0x3c23d70a))
			wantX := p.originX() + float64(gx)
			if !b.Homing || math.Abs(b.X-wantX) > 1e-4 || math.Abs(b.Y-(o.Y-0.1)) > 1e-4 {
				t.Fatalf("homing muzzle %+v at option %+v", b, o)
			}
		}
	}
}

func TestReimuAOptionFocusInterpolation(t *testing.T) {
	p := testPlayer(game.CharReimu)
	p.SetPos(200, 200)
	p.GS.Power = 400
	var in input.State
	for i := 0; i < 4; i++ { // focus is held off for 4 frames
		p.UpdateInput(&in)
	}
	if p.Options()[0] != (Option{168, 208}) {
		t.Fatal(p.Options())
	}
	in.Restore(1 << input.KeyFocus)
	p.UpdateInput(&in)
	if p.Options()[0] != (Option{172.8, 211.6}) {
		t.Fatal(p.Options())
	}
	for i := 0; i < 50; i++ {
		p.UpdateInput(&in)
	}
	if p.Options()[0] != (Option{184, 220}) {
		t.Fatal(p.Options())
	}
	p.GS.Power = 100
	p.UpdateInput(&in)
	if len(p.Options()) != 1 || p.Options()[0] != (Option{200, 220}) {
		t.Fatal(p.Options())
	}
}

func TestReimuAStickyTargetSelection(t *testing.T) {
	p := testPlayer(game.CharReimu)
	a := &enemy.Enemy{X: 100, Y: 100, Active: true}
	b := &enemy.Enemy{X: 200, Y: 100, Active: true}
	p.SetHomingCandidates([]*enemy.Enemy{a, b})
	p.SetHomingCandidates([]*enemy.Enemy{b, a})
	if p.reimuA.target != a {
		t.Fatal("target changed to nearer or reordered enemy")
	}
	a.Active = false
	p.SetHomingCandidates([]*enemy.Enemy{a, b})
	if p.reimuA.target != b {
		t.Fatal("did not release deleted enemy")
	}
}

func TestExistingShotsAdvanceThroughDeathAndBomb(t *testing.T) {
	p := testPlayer(game.CharReimu)
	var in input.State
	in.Restore(1 << input.KeyShot)
	p.UpdateInput(&in)
	p.Hit()
	in.Restore(0)
	p.UpdateInput(&in)
	p.Bullets.Each(func(b *playerbullet.Bullet) {
		if b.Age != 2 {
			t.Fatal("dead player froze live bullets")
		}
	})
	p.GS.Power = 100
	in.Restore(1<<input.KeyBomb | 1<<input.KeyShot)
	p.UpdateInput(&in)
	if p.State != StateNormal || !p.Bomb.Active() {
		t.Fatal("expected deathbomb")
	}
	p.Bullets.Each(func(b *playerbullet.Bullet) {
		if b.Age != 3 {
			t.Fatal("bomb update advanced bullet twice or not at all")
		}
	})
}

func TestOtherShotsRemainLegacy(t *testing.T) {
	for character := 0; character < 2; character++ {
		for shot := 0; shot < 3; shot++ {
			if character == game.CharReimu && shot == game.ShotA {
				continue
			}
			p := New(game.NewState(character, shot, game.DiffNormal), game.FieldLeft, game.FieldTop, game.FieldRight, game.FieldBottom)
			var in input.State
			in.Restore(1 << input.KeyShot)
			p.UpdateInput(&in)
			// FUN_004281d0: 0x428201 sets tick 0, then 0x428223 runs
			// the SHT group immediately. All five power-0 groups have two
			// main shots at phase 0 with interval 3 (FUN_00428160).
			if got := len(newborns(p)); got != 2 {
				t.Fatalf("character %d shot %d: SHT tick 0 births=%d, want 2", character, shot, got)
			}
			p.UpdateInput(&in)
			if got := len(newborns(p)); got != 0 {
				t.Fatalf("character %d shot %d: SHT tick 1 births=%d, want 0", character, shot, got)
			}
			p.UpdateInput(&in)
			if got := len(newborns(p)); got != 0 {
				t.Fatalf("character %d shot %d: SHT tick 2 births=%d, want 0", character, shot, got)
			}
			p.UpdateInput(&in)
			if got := len(newborns(p)); got != 2 {
				t.Fatalf("character %d shot %d: SHT tick 3 births=%d, want 2", character, shot, got)
			}
		}
	}
}
