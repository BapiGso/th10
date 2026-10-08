package playerbullet

import (
	"math"
	"testing"
	"th10/entity/enemy"
	"th10/game"
	"th10/sht"
)

func closeTo(t *testing.T, got, want float64) {
	t.Helper()
	if math.Abs(got-want) > 1e-4 {
		t.Fatalf("got %.9f want %.9f", got, want)
	}
}

func homingShot() sht.Shot {
	return sht.Shot{Damage: 6, Width: 16, Height: 16, Angle: float64(float32(-math.Pi / 2)), Speed: 12, Source: 1, Script: 7, Callbacks: [4]uint32{1, 1, 0, 0}}
}

func TestSHTBirthCompensation(t *testing.T) {
	p := NewPool(1)
	s := sht.Shot{X: -10, Y: -8, Damage: 15, Width: 18, Height: 48, Angle: float64(float32(-math.Pi / 2)), Speed: 24, Script: 5}
	p.FireSHT(200, 300, s, nil)
	p.Update()
	b := p.bullets[0]
	closeTo(t, b.X, 190)
	closeTo(t, b.Y, 292)
	if b.Width != 18 || b.Height != 48 || b.Damage != 15 || b.Age != 1 {
		t.Fatal(b)
	}
	p.Update()
	closeTo(t, p.bullets[0].Y, 268)
}

func TestHomingTurnSpeedAndLifetime(t *testing.T) {
	e := &enemy.Enemy{X: 100, Y: 100, Active: true}
	b := Bullet{X: 100, Y: 200, Angle: 0, Speed: 12, Homing: true, target: e}
	b.tickHoming()
	closeTo(t, b.Angle, -math.Pi/10)
	closeTo(t, b.Speed, 11.7)
	b.Angle = -math.Pi / 2
	b.Speed = 12
	b.tickHoming()
	closeTo(t, b.Speed, 12.1)
	b.Angle = -math.Pi/2 + math.Pi/6
	b.Speed = 12
	b.tickHoming()
	closeTo(t, b.Speed, 12)
	b.Age = 120
	b.Angle = 1
	b.Speed = 16
	b.tickHoming()
	closeTo(t, b.Angle, 1)
	closeTo(t, b.Speed, 16.2)
}

func TestHomingDoesNotReacquire(t *testing.T) {
	e := &enemy.Enemy{X: 200, Y: 100, Active: true}
	p := NewPool(2)
	p.FireSHT(200, 300, homingShot(), e)
	p.Update()
	e.Active = false
	p.Update()
	if p.bullets[0].target != nil {
		t.Fatal("dead target retained")
	}
	angle := p.bullets[0].Angle
	e.Active = true
	e.X = 400
	p.Update()
	if p.bullets[0].target != nil || p.bullets[0].Angle != angle {
		t.Fatal("lost target reacquired")
	}
}

func TestHomingRejectsOriginalFlagsAndOffscreenBirthTarget(t *testing.T) {
	for _, flag := range []int{1, 0x10, 0x40000, 0x80000} {
		e := &enemy.Enemy{Active: true, ECLFlags: flag}
		if HomingTargetable(e) {
			t.Fatalf("accepted flag %x", flag)
		}
	}
	p := NewPool(1)
	p.FireSHT(200, 300, homingShot(), &enemy.Enemy{Active: true, X: 449, Y: 100})
	if p.bullets[0].target != nil {
		t.Fatal("accepted X outside original +/-224")
	}
}

func TestHomingWrapAndClamps(t *testing.T) {
	closeTo(t, normAngle(-math.Pi+0.1-(math.Pi-0.1)), 0.2)
	b := Bullet{Speed: 15.99, Angle: 0}
	b.tickHoming()
	closeTo(t, b.Speed, 16)
	b = Bullet{X: 100, Y: 100, Speed: 4.1, Angle: 0, target: &enemy.Enemy{Active: true, X: 0, Y: 100}}
	b.tickHoming()
	closeTo(t, b.Speed, 4)
}

func TestSHTCollisionExtentAndPoolReuse(t *testing.T) {
	p := NewPool(1)
	s := homingShot()
	s.Width = 18
	s.Height = 48
	s.Damage = 15
	s.Callbacks = [4]uint32{}
	p.FireSHT(200, 300, s, nil)
	p.Update()
	if got := p.HitEnemy(208, 300, 2, 2); got != 15 {
		t.Fatalf("SHT width not used: damage %d", got)
	}
	if got := p.HitEnemy(200, 300, 30, 30); got != 0 {
		t.Fatal("hit twice")
	}
	p.Fire(200, 300, 0, -12, 8)
	if p.bullets[0].Homing || p.bullets[0].fromSHT || p.bullets[0].ScriptID != -1 || p.bullets[0].target != nil {
		t.Fatal("stale slot metadata")
	}
}

func TestHomingLeavesAnyFieldEdge(t *testing.T) {
	p := NewPool(1)
	p.FireSHT(game.FieldRight+100, 300, homingShot(), nil)
	for i := 0; i < 11; i++ {
		p.Update()
	}
	if p.bullets[0].Active {
		t.Fatal("sideways homing bullet leaked pool slot")
	}
}
