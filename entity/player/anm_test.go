package player

import (
	"encoding/binary"
	"math"
	"slices"
	"testing"

	"th10/anm"
	"th10/game"
	"th10/gamedata"
	"th10/gfx"
	"th10/input"
	"th10/zun"
)

func anmInstr(op int, args ...int32) anm.Instr {
	in := anm.Instr{Op: op, Args: make([]byte, len(args)*4)}
	for i, arg := range args {
		binary.LittleEndian.PutUint32(in.Args[i*4:], uint32(arg))
	}
	return in
}

// These parked scripts isolate handle order and interrupts from the shipped
// scripts' child effects. Jump offsets are not needed for label interrupts.
func playerEffects() *anm.File {
	f := &anm.File{Scripts: map[int]*anm.Script{}}
	for _, id := range []int{0x160, 0x162, 0x163} {
		f.Scripts[id] = &anm.Script{ID: id, Instrs: []anm.Instr{
			anmInstr(68, 13), anmInstr(3, int32(id)), anmInstr(63),
			anmInstr(64, 1), anmInstr(1),
		}}
	}
	return f
}

func animatedPlayer(t *testing.T) (*Player, *anm.World, *anm.Env) {
	t.Helper()
	p := testPlayer(game.CharReimu)
	w := &anm.World{}
	env := &anm.Env{GameSpeed: 1, RNG: &zun.RNG{State: 0x1234}, Alt: &zun.RNG{}, Spawn: w.Add}
	p.AttachAnm(w, env, playerEffects())
	p.initTimer.Set(4)
	return p, w, env
}

func TestPlayerAnmFiles(t *testing.T) {
	for character := range 2 {
		var file *anm.File
		for shot := range 3 {
			p := New(game.NewState(character, shot, game.DiffNormal), game.FieldLeft, game.FieldTop, game.FieldRight, game.FieldBottom)
			if p.Anm == nil || (file != nil && p.Anm != file) {
				t.Fatalf("character %d shot %d: file is nil or not shared", character, shot)
			}
			file = p.Anm
			w, env := &anm.World{}, &anm.Env{GameSpeed: 1}
			p.AttachAnm(w, env, nil)
			if p.VM.File != file || p.VM.ScriptID != 0 || p.VM.T.Cur != 1 || w.Len() != 0 {
				t.Fatalf("character %d shot %d: script %d clock %d world len %d", character, shot, p.VM.ScriptID, p.VM.T.Cur, w.Len())
			}
		}
	}
	if playerFile(0) == playerFile(1) {
		t.Fatal("characters share an animation file")
	}
}

func TestMoveAnmTurns(t *testing.T) {
	for _, tc := range []struct {
		name string
		prev int32
		keys uint16
		id   int
		vx   int32
	}{
		{"left from still", 0, 1 << input.KeyLeft, 1, -450},
		{"left from right", 450, 1 << input.KeyLeft, 1, -450},
		{"right from still", 0, 1 << input.KeyRight, 3, 450},
		{"right from left", -450, 1 << input.KeyRight, 3, 450},
		{"stop from left", -450, 0, 2, 0},
		{"stop from right", 450, 0, 4, 0},
		{"conflicting diagonals", 0, 1<<input.KeyUp | 1<<input.KeyLeft | 1<<input.KeyRight, 1, -318},
	} {
		t.Run(tc.name, func(t *testing.T) {
			p, _, env := animatedPlayer(t)
			p.xStep = tc.prev
			p.GameSpeed = func() float32 { return 0.001 }
			x := p.IX
			var in input.State
			in.Restore(tc.keys)
			p.move(&in)
			if p.VM.ScriptID != tc.id || p.VM.T.Cur != 1 || p.xStep != tc.vx || p.IX != x {
				t.Fatalf("script %d clock %d raw step %d moved %d", p.VM.ScriptID, p.VM.T.Cur, p.xStep, p.IX-x)
			}
			if env.RNG.State != 0x1234 || env.Alt.State != 0 {
				t.Fatal("body animation consumed RNG")
			}
		})
	}
}

func TestMoveAnmSameSign(t *testing.T) {
	p, _, _ := animatedPlayer(t)
	p.xStep = -450
	p.VM.T.Set(12)
	p.SetFixed(-fixedMaxX, 40000)
	var in input.State
	in.Restore(1<<input.KeyLeft | 1<<input.KeyFocus)
	p.move(&in)
	if p.VM.ScriptID != 0 || p.VM.T.Cur != 12 || p.xStep != -200 || p.IX != -fixedMaxX {
		t.Fatalf("same sign/clamp restarted animation: script %d clock %d raw %d x %d", p.VM.ScriptID, p.VM.T.Cur, p.xStep, p.IX)
	}
}

func TestFocusHitbox(t *testing.T) {
	p, w, env := animatedPlayer(t)
	p.initTimer.Set(3)
	var in input.State
	in.Restore(1 << input.KeyFocus)
	p.move(&in)
	if p.hitbox != nil || w.Len() != 0 {
		t.Fatal("focus was not held off during init")
	}
	p.initTimer.Set(4)
	in.Restore(1<<input.KeyFocus | 1<<input.KeyRight | 1<<input.KeyUp)
	p.move(&in)
	hitbox := p.hitbox
	x, y := p.gamePos()
	if !w.Contains(hitbox) || hitbox.ScriptID != 0x160 || hitbox.Layer != 13 ||
		hitbox.Flags&anm.FlagAltRNG == 0 || hitbox.Offset != [3]float32{x + 224, y + 16, 0} {
		t.Fatalf("focus hitbox: %+v", hitbox)
	}
	p.move(&in)
	if p.hitbox != hitbox || w.Len() != 1 {
		t.Fatal("held focus spawned another hitbox")
	}
	x, y = p.gamePos()
	if hitbox.Offset != [3]float32{x + 224, y + 16, 0} {
		t.Fatal("hitbox did not follow the updated float position")
	}
	in.Restore(0)
	p.move(&in)
	if p.hitbox != nil || !w.Contains(hitbox) || hitbox.Flags&anm.FlagDelete != 0 {
		t.Fatal("focus release must interrupt, not delete the world handle")
	}
	w.Run(env)
	if w.Contains(hitbox) {
		t.Fatal("label 1 was not taken by the hitbox script")
	}
	in.Restore(1 << input.KeyFocus)
	p.move(&in)
	w.Clear()
	p.move(&in) // the stale handle is cleared after syncPos
	if p.hitbox != nil {
		t.Fatal("stale handle survived Contains")
	}
	p.move(&in)
	if !w.Contains(p.hitbox) {
		t.Fatal("focus did not replace the cleared handle on the next move")
	}
}

func TestInvincAnmBlink(t *testing.T) {
	for _, tc := range []struct {
		invinc, cur, prev int32
		blink             bool
	}{
		{1, 3, 2, true}, {6, 0, -1, true}, {6, 3, 3, false},
		{6, 4, 3, false}, {0, 3, 2, false}, {-1, 3, 2, false},
	} {
		p, _, _ := animatedPlayer(t)
		p.Invinc.Set(tc.invinc)
		p.StateTimer = zun.Timer{Cur: tc.cur, Prev: tc.prev, F: float32(tc.cur)}
		p.VM.Flags |= 0x8000
		p.VM.Color2, p.VM.Alpha2 = [3]uint8{10, 20, 30}, 42
		var in input.State
		p.tail(&in)
		if got := p.VM.Flags&0x8000 != 0; got != tc.blink {
			t.Fatalf("invinc %d state clock %d/%d: blink %v, want %v", tc.invinc, tc.prev, tc.cur, got, tc.blink)
		}
		wantColor, wantAlpha := [3]uint8{10, 20, 30}, uint8(42)
		if tc.blink {
			wantColor, wantAlpha = [3]uint8{0, 0, 255}, 255
		}
		if p.VM.Color2 != wantColor || p.VM.Alpha2 != wantAlpha || p.VM.T.Cur != 2 || p.StateTimer.Cur != tc.cur+1 {
			t.Fatalf("color %v alpha %d VM clock %d state clock %d", p.VM.Color2, p.VM.Alpha2, p.VM.T.Cur, p.StateTimer.Cur)
		}
		wantInvinc := tc.invinc
		if wantInvinc >= 1 {
			wantInvinc--
		}
		if p.Invinc.Cur != wantInvinc {
			t.Fatalf("invinc %d, want %d", p.Invinc.Cur, wantInvinc)
		}
	}
}

func TestHitAnm(t *testing.T) {
	p, w, env := animatedPlayer(t)
	p.SetFixed(-12325, 40012)
	p.VM.Start(p.Anm, 3, env)
	p.Hit()
	if p.State != StateDeathbomb || p.StateTimer.Cur != 0 || p.StateTimer.Prev != -1 ||
		p.Invinc.Cur != 6 || p.VM.ScriptID != 0 || p.VM.T.Cur != 1 || w.Len() != 33 {
		t.Fatalf("state %d clock %+v invinc %+v script %d world len %d", p.State, p.StateTimer, p.Invinc, p.VM.ScriptID, w.Len())
	}
	w.Run(env)
	var ids []int
	x, y := p.gamePos()
	w.Each(13, func(v *anm.VM) {
		ids = append(ids, v.ScriptID)
		if v.Offset != [3]float32{x + 224, y + 16, 0} || v.Flags&anm.FlagAltRNG == 0 {
			t.Errorf("effect %d offset %v flags %#x", len(ids), v.Offset, v.Flags)
		}
	})
	want := make([]int, 33)
	want[0] = 0x162
	for i := 1; i < len(want); i++ {
		want[i] = 0x163
	}
	if !slices.Equal(ids, want) {
		t.Fatalf("effect order %v", ids)
	}
}

func TestHitAnmBeforeClockReset(t *testing.T) {
	p, w, env := animatedPlayer(t)
	p.StateTimer.Set(20)
	p.Invinc.Set(30)
	for _, id := range []int{0x162, 0x163} {
		p.bulletAnm.Scripts[id].Instrs = []anm.Instr{anmInstr(88, 0x160), anmInstr(63)}
	}
	n := 0
	env.Spawn = func(v *anm.VM) {
		n++
		if p.State != StateDeathbomb || p.StateTimer.Cur != 20 || p.Invinc.Cur != 30 {
			t.Fatalf("effect %d spawned after the clock reset", n)
		}
		w.Add(v)
	}
	p.Hit()
	if n != 33 {
		t.Fatalf("%d effect starts, want 33", n)
	}
}

func TestPlayerEffectsUseAltRNG(t *testing.T) {
	p, w, env := animatedPlayer(t)
	f, err := gamedata.ANM("bullet.anm")
	if err != nil {
		t.Fatal(err)
	}
	p.bulletAnm = f
	var in input.State
	in.Restore(1 << input.KeyFocus)
	p.move(&in)
	p.Hit()
	for range 40 {
		w.Run(env)
	}
	if env.RNG.State != 0x1234 {
		t.Fatalf("player effects changed game RNG to %#x", env.RNG.State)
	}
	if env.Alt.State == 0 {
		t.Fatal("shipped death sparks did not consume alternate RNG")
	}
}

func TestPlayerDraw(t *testing.T) {
	p, _, _ := animatedPlayer(t)
	p.SetFixed(-12325, 40012)
	for _, state := range []State{StateRespawn, StateNormal, StateDying, State3, StateDeathbomb} {
		p.State = state
		p.VM.Offset = [3]float32{-1, -2, -3}
		calls := 0
		p.Draw(func(v *anm.VM) {
			calls++
			x, y := p.gamePos()
			if v != &p.VM || v.Offset != [3]float32{x + 224, y + 16, 0} {
				t.Fatalf("draw offset %v", v.Offset)
			}
		}, func([4]float32, gfx.Color) { calls++ }, 130, true)
		if state == StateDying {
			if calls != 0 || p.VM.Offset != [3]float32{-1, -2, -3} {
				t.Fatal("state 2 must skip the entire draw")
			}
		} else if calls != 3 {
			t.Fatalf("state %d: %d draws, want 3", state, calls)
		}
	}
}

func TestFaithHoldRectFloat32(t *testing.T) {
	for _, tc := range []struct {
		x, y float32
		hold int32
		bits [4]uint32
	}{
		{-123.25, 400.125, 130, [4]uint32{0x42aa0000, 0x43c48000, 0x42f60000, 0x43c58000}},
		{0.125, 32.75, 1, [4]uint32{0x43510000, 0x41c80000, 0x43514ad5, 0x41d80000}},
		{183.99, 431.99, 17, [4]uint32{0x43c40000, 0x43d40000, 0x43c67c10, 0x43d50000}},
		{-209.00001, 7.999999, 13, [4]uint32{0xbf800000, 0x3f800000, 0x40333334, 0x40400000}},
	} {
		p, _, _ := animatedPlayer(t)
		p.X, p.Y = p.originX()+float64(tc.x), p.FieldTop+float64(tc.y)
		var order []string
		var rects [][4]float32
		var colors []gfx.Color
		p.Draw(func(*anm.VM) { order = append(order, "VM") }, func(v [4]float32, c gfx.Color) {
			order = append(order, "rect")
			rects, colors = append(rects, v), append(colors, c)
		}, tc.hold, true)
		if !slices.Equal(order, []string{"VM", "rect", "rect"}) || !slices.Equal(colors, []gfx.Color{0x80000000, 0xffffffff}) {
			t.Fatalf("draw order %v colors %v", order, colors)
		}
		for i, want := range tc.bits {
			if got := math.Float32bits(rects[0][i]); got != want {
				t.Errorf("(%v,%v) hold %d component %d: bits %#x, want %#x", tc.x, tc.y, tc.hold, i, got, want)
			}
			if rects[1][i] != rects[0][i]-1 {
				t.Errorf("second rectangle component %d was not translated by -1", i)
			}
		}
	}
}

func TestFaithHoldRectGates(t *testing.T) {
	p, _, _ := animatedPlayer(t)
	for _, tc := range []struct {
		hold int32
		bar  bool
	}{{0, true}, {130, false}, {0, false}} {
		n := 0
		p.Draw(func(*anm.VM) { n++ }, func([4]float32, gfx.Color) { t.Fatal("gated rectangle drawn") }, tc.hold, tc.bar)
		if n != 1 {
			t.Fatalf("%d body draws, want 1", n)
		}
	}
}

func TestPlayerNilAnm(t *testing.T) {
	for _, tc := range []struct {
		w   *anm.World
		env *anm.Env
	}{{nil, nil}, {&anm.World{}, nil}, {nil, &anm.Env{GameSpeed: 1}}} {
		p := testPlayer(game.CharReimu)
		p.AttachAnm(tc.w, tc.env, playerEffects())
		p.initTimer.Set(4)
		var in input.State
		in.Restore(1<<input.KeyFocus | 1<<input.KeyLeft)
		p.UpdateInput(&in)
		p.Hit()
		p.UpdateInput(&in)
		p.Draw(func(*anm.VM) { t.Fatal("unattached VM drawn") }, func([4]float32, gfx.Color) { t.Fatal("unattached rectangle drawn") }, 130, true)
		if p.hitbox != nil || (tc.w != nil && tc.w.Len() != 0) || p.StateTimer.Cur != 1 || p.Invinc.Cur != 5 {
			t.Fatal("nil animation environment affected logical updates")
		}
	}
}
