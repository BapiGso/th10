package stddraw

import (
	"slices"
	"testing"

	"th10/std"
)

func TestSpellHiddenSTDSlotsAndPasses(t *testing.T) {
	sv, rec, af := fixture(t)
	v := newSTD(&std.File{})
	v.BackgroundVisible = false
	drawable(&v.Slots[0], af, 0, 0)
	d := &Drawer{STD: v, Faces: 123, Culled: 456, Quads: 789}
	d.Node7(sv)
	if calls := only(rec.Take(), "draw "); len(calls) != 0 {
		t.Fatalf("hidden STD drew a slot: %v", calls)
	}
	if d.Faces != 0 || d.Culled != 0 || d.Quads != 0 {
		t.Fatalf("node 7 did not clear counters: %d/%d/%d", d.Faces, d.Culled, d.Quads)
	}
	var layers []int
	d.Node0A(sv, func(l int) { layers = append(layers, l) })
	if !slices.Equal(layers, []int{17, 18}) {
		t.Fatalf("hidden STD skipped World layers 17/18: %v", layers)
	}
	if calls := only(rec.Take(), "draw "); len(calls) != 0 {
		t.Fatalf("hidden STD drew a pass: %v", calls)
	}
	before := v.T.Cur
	v.Update(1)
	if v.T.Cur != before+1 {
		t.Fatal("hiding STD stopped the camera script")
	}
}
