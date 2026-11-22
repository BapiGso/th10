package gamedata

import "testing"

func TestSharedANMIsOneFilePerName(t *testing.T) {
	a, err := SharedANM("ascii.anm")
	if err != nil {
		t.Fatal(err)
	}
	b, err := SharedANM("ascii.anm")
	if err != nil || a != b {
		t.Fatal("SharedANM parsed the file twice")
	}
	if c, err := ANM("ascii.anm"); err != nil || c == a {
		t.Fatal("ANM returned the shared file instead of a private copy")
	}
	o, err := SharedOrdinalANM("text.anm", false)
	if err != nil {
		t.Fatal(err)
	}
	if p, err := SharedOrdinalANM("text.anm", false); err != nil || p != o {
		t.Fatal("SharedOrdinalANM rebuilt its table")
	}
	text, _ := SharedANM("text.anm")
	if len(o.Scripts) != len(text.ScriptOrder) {
		t.Fatal("ordinal table does not follow the file's script order")
	}
	if _, err := SharedANM("missing.anm"); err == nil {
		t.Fatal("missing file accepted")
	}
}
