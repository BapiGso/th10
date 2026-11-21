package ui

import (
	"testing"

	"th10/anm"
	"th10/game"
)

func TestSharedTitleAnimationsKeepOptionalRootOwnership(t *testing.T) {
	g := &game.Game{}
	c := NewTitleController(g, 0)
	a := c.animations()
	v := a.Spawn(0x5e)
	if c.Root(0x5e) != v || a.Root(0x5e) != v {
		t.Fatal("the title controller and shared facade do not own the same root")
	}
	a.Leave(0x5e)
	if c.Root(0x5e) != nil {
		t.Fatal("shared leave did not clear the controller's indexed handle")
	}
	a.Roots = nil
	v = a.Spawn(0x5f)
	if !c.World.Contains(v) || a.Root(0x5f) != nil {
		t.Fatal("scene-B's unindexed background changed handle ownership")
	}
	a.Leave(0x5f)
	c.Dispose()
}

func TestSharedLoadingEmitterStaysOptInAndSkipsBirthPass(t *testing.T) {
	for _, enabled := range []bool{false, true} {
		g := &game.Game{}
		w := &anm.World{}
		env := &anm.Env{GameSpeed: 1, Alt: g.AlternateRNG(), Spawn: w.Add}
		file := titleASCII()
		parent := w.SpawnTail(file, 8, 15, env)
		a := TitleAnimations{World: w, Env: env, File: file}
		var emitter *LoadingEmitter
		if enabled {
			emitter = &LoadingEmitter{Parent: parent, File: file}
		}
		a.Run(emitter)
		count := 0
		for l := range anm.Layers {
			w.Each(l, func(v *anm.VM) {
				if v.File == file && v.ScriptID == 7 {
					count++
				}
			})
		}
		if count != 0 {
			t.Fatal("front-born particles were visited on their birth pass")
		}
		a.Run(emitter)
		count = 0
		for l := range anm.Layers {
			w.Each(l, func(v *anm.VM) {
				if v.File == file && v.ScriptID == 7 {
					count++
				}
			})
		}
		want := 0
		if enabled {
			want = 16
		}
		if count != want {
			t.Fatalf("enabled=%v loader particles=%d, want %d", enabled, count, want)
		}
		w.Clear()
	}
}
