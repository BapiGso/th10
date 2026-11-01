package std

import (
	"encoding/binary"
	"fmt"
	"reflect"
	"testing"

	"th10/anm"
	"th10/gamedata"
	"th10/zun"
)

// anmScript is an ANM script that shows sprite 0 at time 0 and ends at
// time end (op 1).
func anmScript(id, end int) *anm.Script {
	return &anm.Script{ID: id, Instrs: []anm.Instr{
		{Time: 0, Op: 3, Offset: 0, Args: binary.LittleEndian.AppendUint32(nil, 0)},
		{Time: end, Op: 1, Offset: 12},
	}}
}

func anmFile(ends map[int]int) *anm.File {
	a := &anm.File{Sprites: map[int]anm.Sprite{0: {W: 16, H: 16}}, Scripts: map[int]*anm.Script{}}
	for id, end := range ends {
		a.Scripts[id] = anmScript(id, end)
	}
	return a
}

func testEnv() *anm.Env {
	return &anm.Env{GameSpeed: 1, RNG: &zun.RNG{State: 0x1234}, Alt: &zun.RNG{State: 0x4321}}
}

// stdIns appends one STD instruction (FUN_00403c80's layout: u32 time, u16
// op, u16 size, then the arguments) to b.
func stdIns(b []byte, time int32, op uint16, args ...int32) []byte {
	b = binary.LittleEndian.AppendUint32(b, uint32(time))
	b = binary.LittleEndian.AppendUint16(b, op)
	b = binary.LittleEndian.AppendUint16(b, uint16(8+4*len(args)))
	for _, a := range args {
		b = binary.LittleEndian.AppendUint32(b, uint32(a))
	}
	return b
}

// started is the VM startVM leaves for script id of a: FUN_00404f30 on a
// fresh VM.
func started(a *anm.File, id int, env *anm.Env) anm.VM {
	vm := anm.New()
	vm.Start(a, id, env)
	return *vm
}

// TestEnable checks FUN_00404450 and the entry loop FUN_00403990 against a
// model: one VM per quad in entry then quad order, every entry active, an
// active entry's quads run each frame and the entry dropped once none is
// left running (a quadless entry on the first frame).
func TestEnable(t *testing.T) {
	f := &File{
		Entries: []Entry{
			{Quads: []Quad{{ScriptIndex: 1}, {ScriptIndex: 2}}},
			{},
			{Quads: []Quad{{ScriptIndex: 3}}},
		},
		Script: stdIns(nil, 0, 0), // holds the clock at 0
	}
	a := anmFile(map[int]int{1: 2, 2: 4, 3: 6})
	env := testEnv()
	v := New(f)
	v.Enable(a, env)

	if len(v.Quads) != 3 {
		t.Fatalf("%d quads, want 3", len(v.Quads))
	}
	ref := make([]anm.VM, 3)
	for k, id := range []int{1, 2, 3} {
		ref[k] = started(a, id, env)
	}
	if !reflect.DeepEqual(v.Quads, ref) {
		t.Fatalf("quads after Enable:\n%+v\nwant\n%+v", v.Quads, ref)
	}
	refActive := []bool{true, true, true}
	if !reflect.DeepEqual(v.Active, refActive) {
		t.Fatalf("Active after Enable = %v, want %v", v.Active, refActive)
	}

	quads := [][]int{{0, 1}, {}, {2}}
	for u := 1; u <= 10; u++ {
		v.Update(1)
		for i, ks := range quads {
			if !refActive[i] {
				continue
			}
			running := 0
			for _, k := range ks {
				ref[k].Run(env)
				if !ref[k].Ended() {
					running++
				}
			}
			refActive[i] = running != 0
		}
		if !reflect.DeepEqual(v.Quads, ref) {
			t.Fatalf("update %d: quads\n%+v\nwant\n%+v", u, v.Quads, ref)
		}
		if !reflect.DeepEqual(v.Active, refActive) {
			t.Fatalf("update %d: Active = %v, want %v", u, v.Active, refActive)
		}
		if u == 1 && !reflect.DeepEqual(v.Active, []bool{true, false, true}) {
			t.Fatalf("update 1: Active = %v, want the quadless entry dropped", v.Active)
		}
	}
	if !reflect.DeepEqual(v.Active, []bool{false, false, false}) {
		t.Errorf("Active after every script ended = %v", v.Active)
	}
}

// TestOp14 checks op 14 and the slot step after the script: a slot started
// in a frame is run again in that frame, a negative script only hides it,
// and a restart zeroes its Offset.
func TestOp14(t *testing.T) {
	var s []byte
	s = stdIns(s, 0, 14, 2, 3)
	s = stdIns(s, 0, 14, 7, 2)
	s = stdIns(s, 3, 14, 2, -1)
	s = stdIns(s, 5, 14, 2, 3)
	s = stdIns(s, 6, 0)
	f := &File{Script: s}
	a := anmFile(map[int]int{2: 100, 3: 100})
	env := testEnv()

	idle := New(f)
	for range 10 {
		idle.Update(1)
	}
	for i := range idle.Slots {
		if !reflect.DeepEqual(idle.Slots[i], *anm.New()) {
			t.Fatalf("without Enable, slot %d = %+v", i, idle.Slots[i])
		}
	}

	v := New(f)
	v.Enable(a, env)
	var ref [8]anm.VM
	for i := range ref {
		ref[i] = *anm.New()
	}
	if !reflect.DeepEqual(v.Slots, ref) {
		t.Fatalf("slots after Enable:\n%+v\nwant every one idle", v.Slots)
	}
	for u := 1; u <= 10; u++ {
		// STD time u-1 runs in update u.
		switch u {
		case 1:
			ref[2] = started(a, 3, env)
			ref[7] = started(a, 2, env)
		case 4:
			ref[2].Flags &^= anm.FlagVisible
		case 6:
			v.Slots[2].Offset = [3]float32{1, 2, 3}
			ref[2] = started(a, 3, env)
		}
		v.Update(1)
		for i := range ref {
			ref[i].Run(env)
		}
		if !reflect.DeepEqual(v.Slots, ref) {
			t.Fatalf("update %d: slots\n%+v\nwant\n%+v", u, v.Slots, ref)
		}
		switch u {
		case 4, 5:
			if v.Slots[2].Visible() || v.Slots[2].Ended() {
				t.Fatalf("update %d: slot 2 visible %v, ended %v; want hidden and running", u, v.Slots[2].Visible(), v.Slots[2].Ended())
			}
		case 6:
			if !v.Slots[2].Visible() || v.Slots[2].Offset != ([3]float32{}) {
				t.Fatalf("update 6: slot 2 visible %v, Offset %v; want a fresh start", v.Slots[2].Visible(), v.Slots[2].Offset)
			}
		}
	}
}

// TestEnableShipped enables every stage's objects on its own .anm: one VM
// per quad, in order, on the quad's script; the op 14 calls of stages 4
// and 6 land in the frames their script times say; and none of it draws
// from either RNG.
func TestEnableShipped(t *testing.T) {
	for n := 1; n <= 7; n++ {
		f := load(t, fmt.Sprintf("stage%02d.std", n))
		a, err := gamedata.ANM(f.ANM)
		if err != nil {
			t.Fatalf("stage %d: %v", n, err)
		}
		env := testEnv()
		rng, alt := *env.RNG, *env.Alt
		v := New(f)
		v.Enable(a, env)
		k := 0
		for i, e := range f.Entries {
			for j, q := range e.Quads {
				if a.Scripts[int(q.ScriptIndex)] == nil {
					t.Fatalf("stage %d entry %d quad %d: no script %d in %s", n, i, j, q.ScriptIndex, f.ANM)
				}
				if k >= len(v.Quads) || v.Quads[k].File != a || v.Quads[k].ScriptID != int(q.ScriptIndex) {
					t.Fatalf("stage %d: quad %d is not entry %d quad %d (script %d)", n, k, i, j, q.ScriptIndex)
				}
				k++
			}
		}
		if len(v.Quads) != k {
			t.Fatalf("stage %d: %d quad VMs, want %d", n, len(v.Quads), k)
		}

		// update is the update in which op 14 starts the slots in want
		// (slot: script); 0 checks only the RNGs over 600 updates.
		update, want := 600, map[int]int{}
		switch n {
		case 4:
			update, want = 8689, map[int]int{0: 4}
		case 6:
			update, want = 3369, map[int]int{0: 0, 1: 10}
		}
		for u := 1; u <= update; u++ {
			v.Update(1)
			if u == update-1 {
				for slot := range want {
					if !reflect.DeepEqual(v.Slots[slot], *anm.New()) {
						t.Fatalf("stage %d: slot %d started before update %d", n, slot, update)
					}
				}
			}
		}
		for slot, script := range want {
			s := &v.Slots[slot]
			if s.File != a || s.ScriptID != script || s.Ended() {
				t.Errorf("stage %d update %d: slot %d has script %d (ended %v), want %d running", n, update, slot, s.ScriptID, s.Ended(), script)
			}
		}
		if *env.RNG != rng || *env.Alt != alt {
			t.Errorf("stage %d: RNGs moved: %+v %+v, were %+v %+v", n, *env.RNG, *env.Alt, rng, alt)
		}
	}
}
