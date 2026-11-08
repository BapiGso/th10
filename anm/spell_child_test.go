package anm

import "testing"

func TestSpellChildLookup(t *testing.T) {
	w := &World{}
	env := testEnv()
	env.Spawn = w.Add
	f := &File{Scripts: map[int]*Script{
		1: newScript(1, []Instr{ins(0, 88, 0, 2), ins(0, 88, 0, 3), ins(0, 63, 0)}),
		2: newScript(2, []Instr{ins(0, 63, 0)}),
		3: newScript(3, []Instr{ins(0, 63, 0)}),
	}}
	p := w.SpawnTail(f, 1, 15, env)
	a, b := p.Child(2), p.Child(3)
	if a == nil || b == nil || a == b || a.ScriptID != 2 || b.ScriptID != 3 || p.Child(1) != nil || p.Child(4) != nil {
		t.Fatalf("child lookup = %p/%p", a, b)
	}
	w.Delete(a)
	w.Run(env)
	if p.Child(2) != nil || p.Child(3) != b {
		t.Fatal("removed child remained in the lookup")
	}
	w.Clear()
	if p.Child(3) != nil {
		t.Fatal("cleared child remained in the lookup")
	}
	var missing *VM
	if missing.Child(2) != nil {
		t.Fatal("nil root found a child")
	}
}
