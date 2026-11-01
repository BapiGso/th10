package item

import (
	"testing"

	"th10/anm"
	"th10/game"
	"th10/zun"
)

const playerNormal = 1

// testEnv is Reimu at (0, 400), not focused, with no ANM.
func testEnv() *Env {
	return &Env{
		State:  game.NewState(game.CharReimu, game.ShotA, game.DiffNormal),
		Player: NewPlayer(0, 0, 400, playerNormal, false, false),
	}
}

// TestCancelDelays: FUN_0041bb00 delays a cancel item by its slot, longer the
// more have spawned since the last update; a slot still in use is skipped,
// the cursor moving on.
func TestCancelDelays(t *testing.T) {
	m, env := NewManager(), testEnv()
	for range 300 {
		m.Spawn(env, [3]float32{0, 100, 0}, KindCancel, -zun.HalfPi, 0.6)
	}
	for _, tc := range [][2]int32{{0, 0}, {5, 1}, {254, 2}, {255, 255&7 + 4}, {299, 299&7 + 4}} {
		if it := &m.Items[NormalSlots+tc[0]]; it.State != StateDelay || it.Delay != tc[1] {
			t.Fatalf("cancel slot %d: state %d delay %d, want delay %d", tc[0], it.State, it.Delay, tc[1])
		}
	}
	m.Update(env)
	if m.Items[NormalSlots].State != StateRise || m.Items[NormalSlots+1].State != StateDelay {
		t.Fatal("delay 0 not rising after one update, or delay 1 not waiting")
	}
	m.Items[NormalSlots+300].State = StateFall
	m.Spawn(env, [3]float32{}, KindCancel, 0, 0)
	m.Spawn(env, [3]float32{}, KindCancel, 0, 0)
	if m.Items[NormalSlots+300].Kind != 0 || m.Items[NormalSlots+301].State != StateDelay {
		t.Fatal("a slot in use was taken, or the cursor did not move past it")
	}
}

// TestFall: gravity 0.03 a frame up to 2, sideways motion stopping once the
// item goes down, lost past y 472.
func TestFall(t *testing.T) {
	m, env := NewManager(), testEnv()
	m.Spawn(env, [3]float32{100, 300, 0}, KindPoint, -zun.Pi/4, 1)
	it := &m.Items[0]
	for it.Vel[1] < 0 {
		if it.Vel[0] == 0 {
			t.Fatal("sideways motion stopped while rising")
		}
		m.Update(env)
	}
	if it.Vel[0] != 0 {
		t.Fatal("sideways motion kept while falling")
	}
	for range 200 {
		m.Update(env)
		if it.State == StateFree {
			break
		}
		if it.Vel[1] > 2 {
			t.Fatalf("falling at %v", it.Vel[1])
		}
	}
	if it.State != StateFree || m.Live != 0 {
		t.Fatalf("item not lost below the playfield: state %d at %v", it.State, it.Pos)
	}
}

// TestCollectionLine: above y 128, or during a bomb, falling items fly to
// the player at its homing speed; not while it is dying or in the deathbomb
// window, and a deathbomb drops those already flying.
func TestCollectionLine(t *testing.T) {
	for _, tc := range []struct {
		y       float32
		state   int32
		bombing bool
		want    int32
	}{
		{127, playerNormal, false, StateAuto},
		{128, playerNormal, false, StateFall},
		{400, playerNormal, true, StateAuto},
		{100, playerDying, false, StateFall},
		{100, playerDeathbomb, false, StateFall},
	} {
		m, env := NewManager(), testEnv()
		env.Player = NewPlayer(0, 0, tc.y, tc.state, false, tc.bombing)
		m.Spawn(env, [3]float32{100, 300, 0}, KindPoint, -zun.HalfPi, 0)
		m.Update(env)
		if it := &m.Items[0]; it.State != tc.want {
			t.Fatalf("player at y %v state %d bombing %v: item state %d, want %d", tc.y, tc.state, tc.bombing, it.State, tc.want)
		}
	}
	m, env := NewManager(), testEnv()
	env.Player = NewPlayer(0, 0, 100, playerNormal, false, false)
	m.Spawn(env, [3]float32{100, 300, 0}, KindPoint, -zun.HalfPi, 0)
	m.Update(env)
	env.Player.State = playerDeathbomb
	m.Update(env)
	if it := &m.Items[0]; it.State != StateFall || it.Vel != [3]float32{} {
		t.Fatalf("deathbomb left the item in state %d at %v", it.State, it.Vel)
	}
}

// TestAttract: an item in the attract box (Reimu: 20 round, 50 focused)
// flies at a third of the homing speed; one in the collect box (10 round) is
// collected, except while the player is dying.
func TestAttract(t *testing.T) {
	for _, tc := range []struct {
		dx      float32
		focused bool
		want    int32
	}{
		{15, false, StateAttract},
		{40, false, StateFall},
		{40, true, StateAttract},
		{60, true, StateFall},
		{10, false, StateFree},
	} {
		m, env := NewManager(), testEnv()
		env.Player.Focused = tc.focused
		m.Spawn(env, [3]float32{tc.dx, 400, 0}, KindPoint, -zun.HalfPi, 0)
		m.Update(env)
		it := &m.Items[0]
		if it.State != tc.want {
			t.Fatalf("item %v off, focused %v: state %d, want %d", tc.dx, tc.focused, it.State, tc.want)
		}
		if tc.want == StateAttract && it.Speed != float32(5*0.33333334) {
			t.Fatalf("attract speed %v", it.Speed)
		}
	}
	m, env := NewManager(), testEnv()
	env.Player.State = playerDying
	m.Spawn(env, [3]float32{0, 400, 0}, KindPoint, -zun.HalfPi, 0)
	m.Update(env)
	if m.ActiveCount() != 1 || env.State.Score != 0 {
		t.Fatal("collected while dying")
	}
}

// TestRiseLostBelow: a cancel item lost below while still rising costs 4
// rank.
func TestRiseLostBelow(t *testing.T) {
	m, env := NewManager(), testEnv()
	m.Spawn(env, [3]float32{100, 480, 0}, KindCancel, -zun.HalfPi, 0.6)
	m.Update(env)
	m.Update(env)
	if m.ActiveCount() != 0 || env.State.Rank != -4 {
		t.Fatalf("%d items, rank %d", m.ActiveCount(), env.State.Rank)
	}
}

// TestFullPower: at full power the power items spawn as faith items and the
// boss's as full point items; the item that brings the power to full turns
// those still out into faith items tossed up where they are (FUN_0041ba50).
func TestFullPower(t *testing.T) {
	m, env := NewManager(), testEnv()
	env.State.Power = 495
	m.Spawn(env, [3]float32{0, 400, 0}, KindPower, -zun.HalfPi, 0)
	m.Spawn(env, [3]float32{100, 300, 0}, KindBigPower, -zun.HalfPi, 0)
	m.Spawn(env, [3]float32{-100, 300, 0}, KindBossPower, -zun.HalfPi, 0)
	m.Update(env)
	if env.State.Power != 500 || env.State.Rank != 0xc {
		t.Fatalf("power %d rank %d", env.State.Power, env.State.Rank)
	}
	faith := &m.Items[0]
	if faith.Kind != KindFaith || faith.State != StateFall || faith.Pos != [3]float32{100, 300, 0} || !(faith.Vel[1] < -2.19) {
		t.Fatalf("big power became kind %d state %d at %v moving %v", faith.Kind, faith.State, faith.Pos, faith.Vel)
	}
	if m.Items[1].State != StateFree || m.Items[2].Kind != KindBossPower {
		t.Fatal("boss power converted, or the big power's slot kept")
	}
	m.Spawn(env, [3]float32{}, KindPower, 0, 0)
	m.Spawn(env, [3]float32{}, KindBossBigPower, 0, 0)
	if m.Items[1].Kind != KindFaith || m.Items[3].Kind != KindFullPoint || m.Items[3].Display != KindFullPoint {
		t.Fatalf("spawned at full power as kinds %d and %d", m.Items[1].Kind, m.Items[3].Kind)
	}
}

// TestBossLooks: the boss's power items look like the player's.
func TestBossLooks(t *testing.T) {
	m, env := NewManager(), testEnv()
	m.Spawn(env, [3]float32{}, KindBossPower, 0, 0)
	m.Spawn(env, [3]float32{}, KindBossBigPower, 0, 0)
	if m.Items[0].Display != KindPower || m.Items[1].Display != KindBigPower {
		t.Fatalf("displays %d, %d", m.Items[0].Display, m.Items[1].Display)
	}
}

// TestDraw: FUN_0041b8e0 puts the item on the playfield; above it, its arrow
// at the top edge, its alpha the truncated (y-8)/32 of 255 taken to a byte.
func TestDraw(t *testing.T) {
	m, env := NewManager(), testEnv()
	m.Spawn(env, [3]float32{10, 100, 0}, KindPoint, 0, 0)
	m.Spawn(env, [3]float32{-20, -16, 0}, KindPoint, 0, 0)
	m.Items[1].VM.Alpha = 0
	var got [][2]int
	m.Draw(func(vm *anm.VM) { got = append(got, [2]int{vm.Sprite, int(vm.Alpha)}) })
	if got[0] != [2]int{int(KindPoint) + 0x157, 0xff} || m.Items[0].VM.Pos != [3]float32{234, 116, 0} {
		t.Fatalf("on the playfield: sprite, alpha %v at %v", got[0], m.Items[0].VM.Pos)
	}
	// y -16 is -8 past the edge: -63.75 truncates to -63, the byte 193.
	if got[1] != [2]int{int(KindPoint) + 0x160, 193} || m.Items[1].VM.Pos != [3]float32{204, 24, 0} {
		t.Fatalf("above the playfield: sprite, alpha %v at %v", got[1], m.Items[1].VM.Pos)
	}
}
