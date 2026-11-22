package playerbullet

import (
	"encoding/binary"
	"math"
	"slices"
	"testing"

	"th10/anm"
	"th10/game"
	"th10/gamedata"
	"th10/sht"
	"th10/zun"
)

func shotInstr(op int, args ...int32) anm.Instr {
	in := anm.Instr{Op: op, Args: make([]byte, len(args)*4)}
	for i, v := range args {
		binary.LittleEndian.PutUint32(in.Args[i*4:], uint32(v))
	}
	return in
}

func shotPool(t *testing.T, capacity int) (*Pool, *anm.World, *anm.Env, sht.Shot) {
	t.Helper()
	f, err := gamedata.ANM("pl00.anm")
	if err != nil {
		t.Fatal(err)
	}
	p, w := NewPool(capacity), &anm.World{}
	env := &anm.Env{GameSpeed: 1, RNG: &zun.RNG{State: 0x1234}, Alt: &zun.RNG{}, Spawn: w.Add}
	p.AttachAnm(w, env, f)
	s := sht.Shot{Damage: 15, Width: 18, Height: 48, Angle: float64(-zun.HalfPi), Speed: 24, Script: 5, HitScript: 6}
	return p, w, env, s
}

func TestShotWorldTailAndRotation(t *testing.T) {
	p, w, env, s := shotPool(t, 2)
	old := w.SpawnTail(p.file, 0x11, 15, env)
	s.Angle = 0.5
	p.FireSHT(200, 300, s, nil)
	p.FireSHT(210, 300, s, nil)
	p.Update()
	w.Run(env)
	var got []*anm.VM
	w.Each(8, func(v *anm.VM) { got = append(got, v) })
	if !slices.Equal(got, []*anm.VM{old, p.bullets[0].VM, p.bullets[1].VM}) {
		t.Fatal("shot VMs were not tail-inserted after the options")
	}
	for _, b := range p.bullets {
		if b.VM.File != p.file || b.VM.ScriptID != 5 || b.VM.Layer != 8 || b.VM.Flags&anm.FlagAltRNG == 0 {
			t.Fatalf("shot VM %+v", b.VM)
		}
		want := [3]float32{float32(b.X), float32(b.Y), 0}
		if b.VM.Offset != want || b.VM.Rot[2] != float32(b.Angle) { // 0x428568: heading as is, no quarter turn
			t.Fatalf("offset %v rotation %v", b.VM.Offset, b.VM.Rot)
		}
	}
	if env.RNG.State != 0x1234 {
		t.Fatal("shot starts consumed the game RNG")
	}
}

func TestShotHitSparkKeepsSlotUntilWorldEnd(t *testing.T) {
	p, w, env, s := shotPool(t, 1)
	p.FireSHT(200, 300, s, nil)
	p.Update()
	b := &p.bullets[0]
	flight, rot := b.VM, b.VM.Rot[2]
	if got := p.HitEnemy(b.X, b.Y, 10, 10); got != 15 {
		t.Fatalf("hit damage %d", got)
	}
	spark := b.VM
	if b.Active || b.state != 2 || spark == flight || !w.Contains(flight) || flight.Flags&anm.FlagDelete == 0 ||
		!w.Contains(spark) || spark.ScriptID != 6 || spark.Layer != 8 || spark.Rot[2] != rot ||
		spark.Flags&anm.FlagRotDirty == 0 || b.Speed != 3 || math.Float32bits(b.Z) != 0x3dcccccd {
		t.Fatal("hit did not replace the flight handle with the original fading spark")
	}
	if got := p.HitEnemy(b.X, b.Y, 100, 100); got != 0 || p.ActiveCount() != 0 {
		t.Fatal("fading shot still damages enemies or counts as a live shot")
	}
	p.FireSHT(200, 300, s, nil)
	if b.VM != spark || b.state != 2 {
		t.Fatal("fading slot reused before its World VM ended")
	}
	y := b.Y
	p.Update()
	closeTo(t, b.Y, y-3)
	if spark.Offset[2] != b.Z || spark.Offset[1] != float32(b.Y) {
		t.Fatal("hit spark did not follow the slowed record")
	}
	for range 40 {
		w.Run(env)
		p.Update()
	}
	if w.Contains(flight) || w.Contains(spark) || b.state != 0 || b.VM != nil {
		t.Fatal("ended hit VM did not release its record")
	}
	p.FireSHT(210, 300, s, nil)
	if !b.Active || b.state != 1 || b.VM == spark || b.Z != 0 || b.Age != 0 {
		t.Fatal("freed record retained hit-animation metadata")
	}
}

func TestShotSpriteCullAndStaleHandle(t *testing.T) {
	p, w, env, s := shotPool(t, 1)
	f := &anm.File{Scripts: map[int]*anm.Script{5: {ID: 5, Instrs: []anm.Instr{
		shotInstr(68, 8), shotInstr(3, 24), shotInstr(63),
	}}}, Sprites: map[int]anm.Sprite{24: {W: 2, H: 4}}}
	p.AttachAnm(w, env, f)
	s.Width, s.Height, s.Speed = 1000, 1000, 0
	p.FireSHT(game.FieldRight+3, 300, s, nil) // W=2: +0x34, not H=4 at +0x30
	flight := p.bullets[0].VM
	for range 10 {
		p.Update()
	}
	if !p.bullets[0].Active {
		t.Fatal("sprite culling ran before age 10")
	}
	p.Update()
	if p.bullets[0].Active || p.bullets[0].state != 0 || flight.Flags&anm.FlagDelete == 0 {
		t.Fatal("culling used SHT collision size instead of the scaled VM sprite")
	}
	w.Run(env)
	p.FireSHT(200, 300, s, nil)
	w.Clear()
	p.Update()
	if p.bullets[0].Active || p.bullets[0].VM != nil || p.bullets[0].ExtraVM != nil {
		t.Fatal("World.Clear left a dangling shot record")
	}
}

func TestShotSpawnLayerArgumentAndAlternateRNG(t *testing.T) {
	p, w, env, s := shotPool(t, 1)
	f := &anm.File{Scripts: map[int]*anm.Script{5: {ID: 5, Instrs: []anm.Instr{
		shotInstr(40, 10000, 100), shotInstr(63),
	}}}}
	p.AttachAnm(w, env, f)
	p.FireSHT(200, 300, s, nil)
	if p.bullets[0].VM.Layer != 15 || env.Alt.Count != 2 || env.RNG.Count != 0 {
		t.Fatal("shot spawn did not pass layer 15 or selected the wrong RNG")
	}
}
