package anm_test

import (
	"testing"

	"th10/gamedata"
)

// FUN_00447470 ignores stored script labels while filling the runtime
// pointer table. Text.anm demonstrates why these are not the same thing.
func TestTextANMOrdinalTable(t *testing.T) {
	f, err := gamedata.ANM("text.anm")
	if err != nil {
		t.Fatal(err)
	}
	if len(f.ScriptOrder) != 73 || f.Scripts[-73] == nil || f.Scripts[0] != nil {
		t.Fatalf("raw text table: %d scripts", len(f.ScriptOrder))
	}
	runtime := f.OrdinalScripts()
	if runtime.Scripts[0] != f.Scripts[-73] || runtime.Scripts[0x48] != f.Scripts[-1] || f.Scripts[0] != nil {
		t.Fatal("runtime ordinals changed or lost the raw negative-label table")
	}
}
