package ecl

import (
	"testing"
	"th10/assets"
)

func loadStage01(t *testing.T) *Program {
	t.Helper()
	data, err := assets.Assets.ReadFile("ecl/stage01.ecl")
	if err != nil {
		t.Fatalf("read stage01.ecl: %v", err)
	}
	prog, err := LoadProgram(data)
	if err != nil {
		t.Fatalf("LoadProgram: %v", err)
	}
	return prog
}

func TestLoaderParsesStage01Header(t *testing.T) {
	prog := loadStage01(t)

	wantAnim := []string{"enemy.anm", "stgenm01.anm"}
	if len(prog.AnimNames) != len(wantAnim) {
		t.Fatalf("anim names = %v, want %v", prog.AnimNames, wantAnim)
	}
	for i, n := range wantAnim {
		if prog.AnimNames[i] != n {
			t.Fatalf("anim[%d] = %q, want %q", i, prog.AnimNames[i], n)
		}
	}
	if len(prog.EcliNames) != 1 || prog.EcliNames[0] != "default.ecl" {
		t.Fatalf("ecli names = %v, want [default.ecl]", prog.EcliNames)
	}
	// Known Stage 1 subs from stage01.ecl.txt.
	for _, name := range []string{"BGirl00", "Boss", "Boss1", "Boss1At1", "main"} {
		if prog.SubByName[name] == nil {
			t.Fatalf("missing sub %q (have %d subs)", name, len(prog.Subs))
		}
	}
}

func TestLoaderDecodesBGirl00(t *testing.T) {
	prog := loadStage01(t)
	sub := prog.SubByName["BGirl00"]
	// void BGirl00() { var A; ins_330(2); ins_262(1,45); @Girl00(0); ins_1(); }
	wantOps := []uint16{opStackAlloc, 330, 262, opCall, opRetBig}
	if len(sub.Instrs) != len(wantOps) {
		t.Fatalf("BGirl00 has %d instrs, want %d: %+v", len(sub.Instrs), len(wantOps), opcodeList(sub))
	}
	for i, op := range wantOps {
		if sub.Instrs[i].Opcode != op {
			t.Fatalf("BGirl00 instr[%d] = %d, want %d (%v)", i, sub.Instrs[i].Opcode, op, opcodeList(sub))
		}
	}
	// ins_330(2): one int param, value 2.
	drop := sub.Instrs[1]
	if len(drop.Params) != 1 || drop.Params[0].Int() != 2 {
		t.Fatalf("ins_330 params = %+v, want [2]", drop.Params)
	}
	// @Girl00(0): CALL with string name "Girl00" then one D arg = 0.
	call := sub.Instrs[3]
	if len(call.Params) < 1 || call.Params[0].Str != "Girl00" {
		t.Fatalf("call params[0] = %+v, want name Girl00", call.Params)
	}
}

func TestLoaderDecodesBossMovePosTime(t *testing.T) {
	prog := loadStage01(t)
	sub := prog.SubByName["Boss"]
	// Boss contains ins_281(60, 4, 0.0f, 112.0f) — movePosTime.
	var found *Instr
	for i := range sub.Instrs {
		if sub.Instrs[i].Opcode == opMovePosTime {
			found = &sub.Instrs[i]
			break
		}
	}
	if found == nil {
		t.Fatalf("Boss has no ins_281; ops=%v", opcodeList(sub))
	}
	// signature SSff: time=60, mode=4, x=0.0, y=112.0
	if len(found.Params) != 4 {
		t.Fatalf("ins_281 params = %+v, want 4", found.Params)
	}
	if found.Params[0].Int() != 60 || found.Params[1].Int() != 4 {
		t.Fatalf("ins_281 int args = %d,%d want 60,4", found.Params[0].Int(), found.Params[1].Int())
	}
	if found.Params[2].Float() != 0 || found.Params[3].Float() != 112 {
		t.Fatalf("ins_281 float args = %g,%g want 0,112", found.Params[2].Float(), found.Params[3].Float())
	}
}

func opcodeList(sub *Sub) []uint16 {
	ops := make([]uint16, len(sub.Instrs))
	for i, in := range sub.Instrs {
		ops[i] = in.Opcode
	}
	return ops
}

// TestLoaderParsesAllStages confirms every embedded stage ECL shares the SCPT V2
// container and exposes a "main" timeline sub, so the generic VM can drive them.
func TestLoaderParsesAllStages(t *testing.T) {
	for _, name := range []string{
		"ecl/stage01.ecl", "ecl/stage02.ecl", "ecl/stage03.ecl",
		"ecl/stage04.ecl", "ecl/stage05.ecl", "ecl/stage06.ecl",
		"ecl/stage07.ecl",
	} {
		data, err := assets.Assets.ReadFile(name)
		if err != nil {
			t.Fatalf("read %s: %v", name, err)
		}
		prog, err := LoadProgram(data)
		if err != nil {
			t.Fatalf("LoadProgram(%s): %v", name, err)
		}
		if prog.SubByName["main"] == nil {
			t.Fatalf("%s: missing main sub (have %d subs)", name, len(prog.Subs))
		}
	}
}
