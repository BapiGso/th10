package playerbullet

import (
	"math"
	"testing"

	"th10/anm"
	"th10/entity/enemy"
	"th10/gamedata"
	"th10/sht"
	"th10/zun"
)

func laserPool(t *testing.T) (*Pool, *anm.World, *anm.Env, sht.Shot) {
	t.Helper()
	f, err := gamedata.ANM("pl01.anm")
	if err != nil {
		t.Fatal(err)
	}
	p, w := NewPool(8), &anm.World{}
	env := &anm.Env{GameSpeed: 1, RNG: &zun.RNG{State: 0x1234}, Alt: &zun.RNG{}, Spawn: w.Add}
	p.AttachAnm(w, env, f)
	s := sht.Shot{Damage: 10, Width: 16, Height: 480, Y: -240, Angle: float64(-zun.HalfPi),
		Source: 1, Kind: 3, Script: 10, HitScript: 11, Sound: -1, Callbacks: [4]uint32{0, 2, 0, 0}}
	return p, w, env, s
}

func TestInitCallbackOneDoesNotImplyHoming(t *testing.T) {
	p, _, _, s := shotPool(t, 1)
	s.Kind, s.Callbacks = 2, [4]uint32{1, 0, 0, 0}
	target := &enemy.Enemy{X: 100, Y: 100, Active: true}
	p.FireSHT(200, 300, s, target)
	if p.bullets[0].target != target || p.bullets[0].Homing {
		t.Fatal("init 1 was mistaken for update 1 on the Marisa rocket record")
	}
	p.Update()
	if p.bullets[0].Angle != s.Angle || p.bullets[0].Speed != s.Speed {
		t.Fatal("rocket with only init callback 1 steered or accelerated")
	}
}

func TestHomingCallbackHitStateAndShortestWrap(t *testing.T) {
	b := Bullet{X: 200, Y: 200, pos: gamePos(200, 200), Angle: 1, Speed: 7, Age: 4, state: 2,
		target: &enemy.Enemy{X: 100, Y: 100, Active: true}}
	b.tickHoming()
	if b.Angle != 1 || b.Speed != 7 {
		t.Fatal("update callback 1 changed a hit-animation record")
	}
	b.state, b.Angle = 1, float64(zun.Pi)-0.1
	b.target.X, b.target.Y = b.X-100, b.Y-10
	aim := float32(math.Atan2(-10, -100))
	want := zun.NormAngle(float32(zun.AngleDiff(aim, float32(b.Angle))*0.2) + float32(b.Angle))
	b.tickHoming()
	if math.Float32bits(float32(b.Angle)) != math.Float32bits(want) {
		t.Fatal("homing wrap used a different float32 operand order from FUN_00428ce0")
	}
}

func TestHomingBirthGridNegativeZeroSide(t *testing.T) {
	p := NewPool(1)
	s := homingShot()
	p.FireSHT(gameOriginX, gameOriginY+432, s, nil)
	p.Update()
	b := &p.bullets[0]
	// FUN_0044c2a0: acceleration 12 -> 12.1 makes cos(-pi/2)*speed
	// slightly more negative than the compensated initial velocity.
	// The negative residue floors to -0.01, not 0 (bits 0xbc23d70a).
	if math.Float32bits(b.pos[0]) != 0xbc23d70a || b.X != gameOriginX+float64(math.Float32frombits(0xbc23d70a)) {
		t.Fatalf("grid corner: x=%v game=%#x", b.X, math.Float32bits(b.pos[0]))
	}
}

func TestLaserMuzzleCallbackUsesCurrentSource(t *testing.T) {
	p, w, env, s := laserPool(t)
	pos := [3]float32{-30, 350, 0}
	p.SourcePosition = func(source int) [3]float32 {
		if source != 1 {
			t.Fatal("wrong SHT source passed to callback 2")
		}
		return pos
	}
	p.FireSHT(200, 300, s, nil)
	p.Update()
	b := &p.bullets[0]
	if b.X != gameOriginX-30 || b.Y != gameOriginY+110 || b.VM.Offset != [3]float32{194, 126, 0} || b.ExtraVM.Offset != b.VM.Offset {
		t.Fatal("laser callback did not reset both handles to the current option")
	}
	w.Run(env)
	pos = [3]float32{50, 360, 0}
	p.Update()
	if b.X != gameOriginX+50 || b.Y != gameOriginY+120 {
		t.Fatal("laser remained bound to its birth muzzle")
	}
}

func TestLaserSourceGatesAndEndReasons(t *testing.T) {
	for _, reason := range []string{"idle", "lost source", "dialog", "missing manager"} {
		t.Run(reason, func(t *testing.T) {
			p, w, env, s := laserPool(t)
			p.FireSHT(200, 300, s, nil)
			p.FireSHT(210, 300, s, nil)
			if p.ActiveCount() != 1 || w.Len() != 2 {
				t.Fatal("kind 3 ignored the per-source gate")
			}
			b := &p.bullets[0]
			flight, extra := b.VM, b.ExtraVM
			switch reason {
			case "idle":
				p.ShotClock = func() int32 { return -1 }
			case "lost source":
				p.OptionCount = func() int { return 0 }
			case "dialog":
				p.NoAttack = func() bool { return true }
			case "missing manager":
				p.ManagerPresent = func() bool { return false }
			}
			p.Update()
			if b.state != 2 || b.Active || p.laserSources[1] || flight.Flags&anm.FlagDelete != 0 || extra.Flags&anm.FlagDelete != 0 {
				t.Fatal("laser end must release its source, then interrupt both handles")
			}
			p.FireSHT(210, 300, s, nil)
			if p.ActiveCount() != 1 || p.bullets[1].VM == nil || p.bullets[0].VM != flight {
				t.Fatal("released source could not allocate a new record beside the fading one")
			}
			for range 24 {
				w.Run(env)
				p.Update()
			}
			if w.Contains(flight) || w.Contains(extra) || b.state != 0 {
				t.Fatal("laser exit scripts did not release their record")
			}
		})
	}
}

func TestLaserDamageFourTickGateAndBrightness(t *testing.T) {
	p, w, env, s := laserPool(t)
	p.FireSHT(200, 300, s, nil)
	b := &p.bullets[0]
	main := b.VM
	for frame := 1; frame <= 8; frame++ {
		p.Update()
		want := 0
		if frame%4 == 0 {
			want = 10
		}
		if got := p.HitEnemy(200, 100, 16, 16); got != want {
			t.Fatalf("tick %d laser damage=%d want=%d", frame, got, want)
		}
		w.Run(env)
		if b.state != 1 || !b.Active || b.VM != main || !b.bright || main.Flags&anm.FlagDelete != 0 {
			t.Fatal("laser hit became a spark or did not brighten")
		}
	}
	p.Update() // retains brightness because the preceding enemy pass hit it
	w.Run(env)
	p.Update() // preceding pass did not hit, so label 3 dims the beam now
	w.Run(env)
	if b.bright || !b.Active || b.VM != main {
		t.Fatal("non-hitting laser did not dim without being deleted")
	}
	b.timer.Cur, b.timer.Prev = 12, 12
	if got := p.HitEnemy(200, 100, 16, 16); got != 0 {
		t.Fatal("frozen timer applied laser damage")
	}
}

func TestKindTwoHitAddsOriginalDamageArea(t *testing.T) {
	p, _, _, s := shotPool(t, 1)
	s.Kind, s.Damage = 2, 17
	calls := 0
	p.HitArea = func(x, y, r, growth float32, frames, damage int32) {
		calls++
		b := &p.bullets[0]
		if x != b.pos[0] || y != b.pos[1] || r != 32 || math.Float32bits(growth) != 0x3fb33333 || frames != 13 || damage != 5 || b.state != 2 {
			t.Fatal("kind 2 area parameters/order differ from FUN_00428630")
		}
	}
	p.FireSHT(200, 300, s, nil)
	p.Update()
	b := &p.bullets[0]
	if p.HitEnemy(b.X, b.Y, 4, 4) != 17 || calls != 1 {
		t.Fatal("rocket hit did not create one area")
	}
	p.HitEnemy(b.X, b.Y, 4, 4)
	if calls != 1 {
		t.Fatal("fading rocket spawned another area")
	}
}

func TestSHTSoundAndInclusiveTopGate(t *testing.T) {
	p := NewPool(2)
	s := sht.Shot{Damage: 10, Width: 4, Height: 4, Speed: 0, Script: 5, HitScript: 6, Sound: 7}
	calls := 0
	p.PlaySE = func(id int, x float32) {
		calls++
		if id != 7 || x != 10 {
			t.Fatal("SHT sound did not use the original id and compensated game x")
		}
	}
	p.FireSHT(gameOriginX+10, gameOriginY+2, s, nil)
	p.Update()
	if calls != 1 || p.HitEnemy(gameOriginX+14, gameOriginY+2, 4, 4) != 10 {
		t.Fatal("shot at game top 0 or touching rectangle edges was rejected")
	}
	p.FireSHT(gameOriginX+10, gameOriginY+1, s, nil)
	p.Update()
	if p.HitEnemy(gameOriginX+10, gameOriginY+1, 4, 4) != 0 {
		t.Fatal("ordinary shot whose top is above game y=0 hit an enemy")
	}
}
