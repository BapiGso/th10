package stage

import (
	"slices"
	"strings"
	"testing"

	"th10/anm"
	"th10/std"
)

func TestSpellPlacardsWorldFilesAndOrigin(t *testing.T) {
	g := newTestGUI()
	c := g.ctx
	c.Spell = &SpellCard{}
	c.Spell.Declare(c.State, 2, "card", 2700)
	g.InitializeSpellPresentation()
	before, rng := c.World.Len(), c.RNG.Count
	c.Spell.SpawnPlacards(c)
	if c.World.Len() != before+3 {
		t.Fatal("declaration did not create exactly three World placards")
	}
	wantScript := [3]int{1, 0x48, 2}
	wantPos := [3][3]float32{{416, 384, 0}, {412, 400, 0}, {368, 51, 0}}
	for i, v := range c.Spell.Placards {
		if v == nil || !c.World.Contains(v) || v.ScriptID != wantScript[i] || v.Layer != 14 || v.Pos != wantPos[i] ||
			v.Offset != [3]float32{} || v.Pos2 != [3]float32{} || v.Flags&anm.FlagAltRNG == 0 || v.T.Cur != 1 {
			t.Fatalf("placard %d = %+v", i, v)
		}
	}
	if a, b := c.Spell.Placards[0], c.Spell.Placards[2]; a.File != b.File || a.File.Entries[0].Name != "ascii/ascii.png" {
		t.Fatal("the two side placards do not share ascii.anm")
	}
	if c.Spell.Placards[1].File.Entries[0].Name != "@" || g.spellName != c.Spell.Placards[1] {
		t.Fatal("name VM did not use the text.anm texture or GUI retained another name")
	}
	if c.RNG.Count != rng {
		t.Fatal("placards consumed the primary RNG")
	}
	c.UpdateWorld()
	var order []*anm.VM
	c.World.Each(14, func(v *anm.VM) {
		if slices.Contains(c.Spell.Placards[:], v) {
			order = append(order, v)
		}
	})
	if !slices.Equal(order, c.Spell.Placards[:]) {
		t.Fatal("placards did not retain the original World insertion/draw order")
	}
}

func TestSpellBackgroundDrawNodeOrderAndActive(t *testing.T) {
	c := &Context{Spell: &SpellCard{Active: true}, STD: std.New(&std.File{})}
	c.STD.BackgroundVisible = false
	vms := []*anm.VM{&c.Spell.Background[0], &c.Spell.Background[1]}
	sv, d := guiFixture(t, vms)
	s := &Stage{ctx: c}
	s.nodeSpellBackground(sv)
	sv.ANM.Flush()
	var got []string
	for _, call := range d.Take() {
		if strings.HasPrefix(call, "tex ") {
			got = append(got, call)
		}
	}
	if !slices.Equal(got, []string{"tex 00", "tex 01"}) || len(d.vertices) != 2 {
		t.Fatalf("cdbg draw order %v with %d VMs", got, len(d.vertices))
	}
	c.Spell.Active = false
	s.nodeSpellBackground(sv)
	sv.ANM.Flush()
	if len(d.Take()) != 0 {
		t.Fatal("inactive helper drew cdbg")
	}
}

func TestSpellHUDDrawUsesHelperActiveBit(t *testing.T) {
	g := &GUI{ctx: &Context{Spell: &SpellCard{}}}
	g.ctx.GUI = g
	g.spellHUD.Active = true
	vms := []*anm.VM{&g.spellHUD.Bonus[0]}
	sv, d := guiFixture(t, vms)
	(&Stage{ctx: g.ctx}).nodeSpellHUD(sv)
	sv.ANM.Flush()
	if len(d.Take()) != 0 {
		t.Fatal("GUI state overrode the inactive helper's draw gate")
	}
}
