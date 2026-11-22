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

func TestLoadingParticlesAreFrontBornAndSkipBirthPass(t *testing.T) {
	g := &game.Game{}
	w := &anm.World{}
	env := &anm.Env{GameSpeed: 1, Alt: g.AlternateRNG(), Spawn: w.Add}
	file := titleASCII()
	w.SpawnTail(file, 8, 15, env)
	a := TitleAnimations{World: w, Env: env, File: file}
	count := func() int {
		n := 0
		for l := range anm.Layers {
			w.Each(l, func(v *anm.VM) {
				if v.File == file && v.ScriptID == 7 {
					n++
				}
			})
		}
		return n
	}
	a.Run() // SpawnTail ran frame 0; frame 1 puts sixteen op-91 children at the list head
	if count() != 0 || w.Len() != 17 {
		t.Fatalf("front-born particles visited on their birth pass (drawn %d, live %d)", count(), w.Len())
	}
	a.Run()
	if count() != 16 {
		t.Fatalf("loader particles=%d, want 16", count())
	}
	w.Clear()
}
