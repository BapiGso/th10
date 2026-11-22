package ecl

import (
	"encoding/binary"
	"fmt"
	"math"
	"testing"

	"th10/assets"
	"th10/entity/enemy"
)

func rankDest(typ byte, id int32) Param {
	p := intParam(id)
	if typ == 'f' {
		p = floatParam(float32(id))
	}
	p.IsVar = true
	return p
}

func TestRankOpcodeFormats(t *testing.T) {
	for op, format := range map[uint16]string{
		349: "ffff", 350: "ffffff", 351: "fff", 352: "SSSS", 353: "SSSSSS", 354: "SSS",
	} {
		t.Run(fmt.Sprint(op), func(t *testing.T) {
			if got := th10Format(op); got != format {
				t.Fatalf("format = %q, want %q", got, format)
			}
			blob := make([]byte, 4*len(format))
			for i, typ := range format {
				v := uint32(i * 4)
				if typ == 'f' {
					v = math.Float32bits(float32(i * 4))
				}
				binary.LittleEndian.PutUint32(blob[i*4:], v)
			}
			params, err := decodeParams(blob, op, 1)
			if err != nil {
				t.Fatal(err)
			}
			if len(params) != len(format) || !params[0].IsVar || params[0].varID() != 0 {
				t.Fatalf("params = %+v", params)
			}
			for i, p := range params {
				if p.Type != format[i] {
					t.Errorf("param %d type = %c, want %c", i, p.Type, format[i])
				}
			}
		})
	}
}

func TestRankOpcodesUnusedInStages(t *testing.T) {
	for stage := 1; stage <= 7; stage++ {
		name := fmt.Sprintf("ecl/stage%02d.ecl", stage)
		data, err := assets.Assets.ReadFile(name)
		if err != nil {
			t.Fatal(err)
		}
		prog, err := loadProgram(data)
		if err != nil {
			t.Fatal(err)
		}
		var counts [6]int
		for _, sub := range prog.subs {
			for _, in := range sub.Instrs {
				if in.Opcode >= 349 && in.Opcode <= 354 {
					counts[in.Opcode-349]++
				}
			}
		}
		if counts != [6]int{} {
			t.Errorf("%s: 349-354 counts = %v, want all zero", name, counts)
		}
		t.Logf("%s: 349-354 counts = %v", name, counts)
	}
}

func TestRankSelect349And352(t *testing.T) {
	for _, op := range []uint16{349, 352} {
		for _, rank := range []int{-1024, -513, -512, -511, 0, 511, 512, 1024} {
			t.Run(fmt.Sprintf("%d/%d", op, rank), func(t *testing.T) {
				vm := newTestVM()
				vm.rank = rank
				task := &eclTask{frame: []uint32{0, 0, 0}}
				p := []Param{rankDest('S', 8), intParam(11), intParam(22), intParam(33)}
				task.frame[2] = 99
				want := intVal(99)
				if rank >= 512 {
					want = intVal(22)
				}
				if op == 349 {
					p = []Param{rankDest('f', 8), floatParam(11), floatParam(22), floatParam(33)}
					task.frame[2] = math.Float32bits(99)
					want = floatVal(want.asFloat())
				}
				vm.execGameOp(nil, task, &Instr{Opcode: op, Params: p})
				if got := task.frame[2]; got != want.bits {
					t.Fatalf("result bits = %08x, want %08x", got, want.bits)
				}
			})
		}
	}
}

func TestRankSelect350And353(t *testing.T) {
	for _, op := range []uint16{350, 353} {
		for _, tc := range []struct{ rank, want int }{
			{-1024, 99}, {-401, 99}, {-400, 11}, {-201, 11}, {-200, 22},
			{199, 22}, {200, 33}, {599, 33}, {600, 44}, {1024, 44},
		} {
			t.Run(fmt.Sprintf("%d/%d", op, tc.rank), func(t *testing.T) {
				vm := newTestVM()
				vm.rank = tc.rank
				task := &eclTask{frame: []uint32{99}}
				p := []Param{rankDest('S', 0), intParam(11), intParam(22), intParam(33), intParam(44), intParam(55)}
				want := uint32(tc.want)
				if op == 350 {
					p = []Param{rankDest('f', 0), floatParam(11), floatParam(22), floatParam(33), floatParam(44), floatParam(55)}
					task.frame[0] = math.Float32bits(99)
					want = math.Float32bits(float32(tc.want))
				}
				vm.execGameOp(nil, task, &Instr{Opcode: op, Params: p})
				if got := task.frame[0]; got != want {
					t.Fatalf("result bits = %08x, want %08x", got, want)
				}
			})
		}
	}
}

func TestRankLerp351(t *testing.T) {
	for _, tc := range []struct {
		rank     int
		lo, hi   float32
		wantBits uint32
	}{
		{-1024, 2, 7, 0x40000000}, {0, 2, 7, 0x40900000}, {1024, 2, 7, 0x40e00000},
		{0, 7, 2, 0x40900000}, {333, 0.1, 0.9, 0x3f214ccd},
	} {
		t.Run(fmt.Sprintf("%d/%g/%g", tc.rank, tc.lo, tc.hi), func(t *testing.T) {
			vm := newTestVM()
			vm.rank = tc.rank
			task := &eclTask{frame: []uint32{0}}
			vm.execGameOp(nil, task, &Instr{Opcode: 351, Params: []Param{
				rankDest('f', 0), floatParam(tc.lo), floatParam(tc.hi),
			}})
			if got := task.frame[0]; got != tc.wantBits {
				t.Fatalf("result = %g (%08x), want bits %08x", math.Float32frombits(got), got, tc.wantBits)
			}
		})
	}
}

func TestRankLerp354(t *testing.T) {
	for _, tc := range []struct {
		rank         int
		lo, hi, want int32
	}{
		{-1024, 2, 7, 2}, {0, 2, 7, 4}, {1024, 2, 7, 7}, {0, 7, 2, 5},
		{0, -4, 1, -2}, {0, 1, -4, -1}, {0, 0, 0x400000, 0},
	} {
		t.Run(fmt.Sprintf("%d/%d/%d", tc.rank, tc.lo, tc.hi), func(t *testing.T) {
			vm := newTestVM()
			vm.rank = tc.rank
			task := &eclTask{frame: []uint32{0}}
			vm.execGameOp(nil, task, &Instr{Opcode: 354, Params: []Param{
				rankDest('S', 0), intParam(tc.lo), intParam(tc.hi),
			}})
			if got := int32(task.frame[0]); got != tc.want {
				t.Fatalf("result = %d, want %d", got, tc.want)
			}
		})
	}
}

func TestRankOpcodeSourceEvaluation(t *testing.T) {
	for _, op := range []uint16{349, 350, 351, 352, 353, 354} {
		t.Run(fmt.Sprint(op), func(t *testing.T) {
			vm := newTestVM()
			vm.rank = 512
			task := &eclTask{frame: []uint32{0}}
			task.push(intVal(8))
			task.push(intVal(4))
			format := th10Format(op)
			p := make([]Param, len(format))
			p[0] = rankDest(format[0], 0)
			for i := 1; i < len(p); i++ {
				p[i] = rankDest(format[i], -1)
			}
			vm.execGameOp(nil, task, &Instr{Opcode: op, Params: p})
			want, left := int32(4), 1
			if op == 351 || op == 354 { // p1 pops 4, p2 pops 8; blend at 3/4.
				want, left = 7, 0
			}
			bits := uint32(want)
			if format[0] == 'f' {
				bits = math.Float32bits(float32(want))
			}
			if task.frame[0] != bits || len(task.stack) != left {
				t.Fatalf("result bits = %08x, stack left = %d; want %08x, %d", task.frame[0], len(task.stack), bits, left)
			}
		})
	}
}

func TestRankOpcodeSpecialOutput(t *testing.T) {
	for _, tc := range []struct {
		op   uint16
		id   int32
		want uint32
	}{{351, -9981, 0x40900000}, {354, -9985, 4}} {
		vm := newTestVM()
		e := &enemy.Enemy{Active: true}
		a := vm.actorFor(e)
		task := &eclTask{owner: e}
		p := []Param{rankDest('S', tc.id), intParam(2), intParam(7)}
		if tc.op == 351 {
			p = []Param{rankDest('f', tc.id), floatParam(2), floatParam(7)}
		}
		vm.execGameOp(nil, task, &Instr{Opcode: tc.op, Params: p})
		if got := a.vars[tc.id+9985]; got != tc.want {
			t.Errorf("%d: special output bits = %08x, want %08x", tc.op, got, tc.want)
		}
	}
}
