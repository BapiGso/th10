package anm_test

import (
	"testing"

	"th10/anm"
	"th10/gamedata"
	"th10/zun"
)

func TestWorldFindChildBeforeLayerSort(t *testing.T) {
	f, err := gamedata.ANM("pl00.anm")
	if err != nil {
		t.Fatal(err)
	}
	w := &anm.World{}
	env := &anm.Env{GameSpeed: 1, RNG: &zun.RNG{}, Alt: &zun.RNG{}, Spawn: w.Add}
	root := w.SpawnTail(f, 29, 15, env)
	for _, id := range []int{23, 26} {
		v := w.FindChild(root, id)
		if v == nil || v.ScriptID != id || !w.Contains(v) {
			t.Fatalf("new portrait child %d = %v before World.Run", id, v)
		}
	}
	if w.FindChild(root, 999) != nil {
		t.Fatal("found an absent child")
	}
	w.Clear()
	if w.FindChild(root, 23) != nil {
		t.Fatal("cleared handle still finds its child")
	}
}
