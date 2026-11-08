package item

import (
	"slices"
	"testing"

	"th10/anm"
	"th10/gamedata"
	"th10/zun"
)

func effectsEnv(t *testing.T) (*Env, *anm.World) {
	t.Helper()
	f, err := gamedata.ANM("bullet.anm")
	if err != nil {
		t.Fatal(err)
	}
	e, w := testEnv(), &anm.World{}
	e.Anm = &anm.Env{GameSpeed: 1, RNG: &zun.RNG{State: 0x1234}, Alt: &zun.RNG{}, Spawn: w.Add}
	e.File, e.Effects = f, w
	return e, w
}

func TestFaithEffectUsesPreviousSlotKindAndClampedPosition(t *testing.T) {
	e, w := effectsEnv(t)
	m := NewManager()
	m.Spawn(e, [3]float32{0, 200, 0}, KindBigFaith, 0, 0)
	if w.Len() != 0 {
		t.Fatal("new big-faith kind incorrectly triggered the previous-kind branch")
	}
	m.Items[0].State = StateFree
	m.Spawn(e, [3]float32{500, 250, 0.25}, KindPoint, 0, 0)
	if w.Len() != 1 || m.Items[0].Kind != KindPoint {
		t.Fatal("reusing a previous big-faith slot did not spawn one effect")
	}
	w.Run(e.Anm)
	var fx []*anm.VM
	w.Each(11, func(v *anm.VM) { fx = append(fx, v) })
	if len(fx) != 1 || fx[0].ScriptID != 0x189 || fx[0].File != e.File ||
		fx[0].Offset != [3]float32{416, 266, 0.25} || fx[0].Flags&anm.FlagAltRNG != 0 {
		t.Fatal("0x189 did not use original Spawn, layer 11 and the clamped item position")
	}
	if e.Anm.RNG.Count != 0 || e.Anm.Alt.Count != 0 {
		t.Fatal("original 0x189/item scripts unexpectedly consumed RNG")
	}
}

func TestFullPowerEffectsKeepItemConversionAndSlotOrder(t *testing.T) {
	e, w := effectsEnv(t)
	m := NewManager()
	kinds := []int32{KindPower, KindBigPower, KindBossPower, KindBossBigPower}
	for i, kind := range kinds {
		m.Spawn(e, [3]float32{float32(i) * 40, 200, 0}, kind, 0, 0)
	}
	e.State.Power = 500
	m.fullPower(e)
	if m.ActiveCount() != 4 || m.Items[0].Kind != KindFaith || m.Items[1].Kind != KindFaith ||
		m.Items[2].Kind != KindBossPower || m.Items[3].Kind != KindBossBigPower {
		t.Fatal("effect wiring changed the original conversion/spared boss items")
	}
	w.Run(e.Anm)
	var offsets [][3]float32
	w.Each(11, func(v *anm.VM) {
		if v.ScriptID != 0x189 {
			t.Fatal("wrong full-power effect script")
		}
		offsets = append(offsets, v.Offset)
	})
	if !slices.Equal(offsets, [][3]float32{{224, 216, 0}, {264, 216, 0}}) {
		t.Fatalf("full-power effect order/positions %v", offsets)
	}
}

func TestFaithEffectEndsThroughANMWithoutRNG(t *testing.T) {
	e, w := effectsEnv(t)
	e.faithEffect([3]float32{-12.25, 234.5, 0})
	for range 19 {
		w.Run(e.Anm)
	}
	if w.Len() != 1 {
		t.Fatal("faith effect ended before script time 20")
	}
	w.Run(e.Anm)
	if w.Len() != 0 || e.Anm.RNG.State != 0x1234 || e.Anm.RNG.Count != 0 || e.Anm.Alt.Count != 0 {
		t.Fatal("faith effect did not end at time 20 or changed RNG")
	}
}

func TestFaithEffectNilAnimationAndCancelPath(t *testing.T) {
	for _, nilPart := range []string{"world", "env", "file"} {
		e, w := effectsEnv(t)
		switch nilPart {
		case "world":
			e.Effects = nil
		case "env":
			e.Anm = nil
		case "file":
			e.File = nil
		}
		e.faithEffect([3]float32{})
		if w.Len() != 0 {
			t.Fatal("nil effect attachment inserted an ended VM")
		}
	}
	e, w := effectsEnv(t)
	m := NewManager()
	m.Items[NormalSlots].Kind = KindBigFaith
	m.Spawn(e, [3]float32{0, 100, 0}, KindCancel, -zun.HalfPi, 0.6)
	if w.Len() != 0 || m.Items[NormalSlots].Kind != KindCancel {
		t.Fatal("cancel-slot path incorrectly entered the normal previous-kind effect branch")
	}
}
