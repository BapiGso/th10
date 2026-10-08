package ecl

import (
	"testing"
	"th10/entity/enemy"
	"th10/entity/item"
)

func TestOriginalPowerDropIDs(t *testing.T) {
	ctx := newTestContext()
	vm := NewStage01VM()
	e := &enemy.Enemy{X: 100, Y: 100}
	vm.addDrop(e, 1, 2)
	vm.addDrop(e, 4, 3) // large P, NOT an extend
	vm.addDrop(e, 5, 4) // large point, NOT a bomb
	vm.addDrop(e, 7, 1) // extend
	vm.spawnDrops(ctx, e)
	counts := map[item.ItemType]int{}
	ctx.Items.Each(func(it *item.Item) { counts[it.Type]++ })
	for kind, want := range map[item.ItemType]int{item.PowerItem: 2, item.BigPower: 3, item.BigPoint: 4, item.LifeFragment: 1} {
		if counts[kind] != want {
			t.Fatalf("drop %d: %d, want %d", kind, counts[kind], want)
		}
	}
	if e.DropPower+e.DropBigPower+e.DropBigPoint+e.DropLife != 0 {
		t.Fatal("drop counts not cleared")
	}
	vm.spawnDrops(ctx, e)
	total := 0
	ctx.Items.Each(func(it *item.Item) { total++ })
	if total != 10 {
		t.Fatal("dropItems duplicated rewards")
	}
}
