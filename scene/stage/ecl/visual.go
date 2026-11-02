package ecl

// Enemy sprites (docs/research/m4_r9.md). An enemy draws nothing itself: it
// owns up to ten ANM objects in the effect world, the slots at +0xc0, which
// ECL 259/262/269 fill, FUN_0040dc80 moves with the enemy every frame and
// FUN_0040dae0 deletes with it. 263/264/272/273 spawn free objects the
// enemy does not keep. The objects run in World.Run (calc 0x1a) and draw in
// the World layer nodes.
//
// Every World call below goes through ctx.World with ctx.Anm as the
// environment. With no ctx, World or Anm the ops still do the enemy's own
// bookkeeping (the +0xe8.. fields, flags, dir, flash) but spawn nothing, so
// the slots stay nil and the size 0.

import (
	"th10/anm"
	"th10/scene/stage"
)

// slotCount is the number of ANM slots per enemy (+0xc0..+0xe4).
const slotCount = 10

// Hit flash (FUN_0040dc80 0x40e43e-0x40e538).
const (
	hitColor2 = 0xff0000ff // VM+0x300: A 255, R 0, G 0, B 255
	hitFlash  = 4          // frames the flash holds off after a flash
	seHit     = 0x13       // the hit sound
	seHitLow  = 0x23       // a boss's hit sound when its phase is nearly over
)

// fileAt is the ECL's ANM file table (DAT_00477704+0x30): index 0 is
// bullet.anm (ctx.BulletEnv.File when there is one, else loadANM), index
// i >= 1 the ECL's i-th ANM (vm.prog.AnimNames[i-1], through loadANM). Out
// of range or not embedded: nil, which spawns ended objects as a missing
// script does.
func (vm *VM) fileAt(ctx *stage.Context, i int) *anm.File {
	if i == 0 {
		if ctx != nil && ctx.BulletEnv != nil && ctx.BulletEnv.File != nil {
			return ctx.BulletEnv.File
		}
		return loadANM("bullet.anm")
	}
	if i < 0 || i > len(vm.prog.AnimNames) {
		return nil
	}
	return loadANM(vm.prog.AnimNames[i-1])
}

// effects is the stage's effect world and its environment, both nil with no
// context.
func effects(ctx *stage.Context) (*anm.World, *anm.Env) {
	if ctx == nil {
		return nil, nil
	}
	return ctx.World, ctx.Anm
}

// clearSlot empties slot i of a: its object is deleted, then the slot is nil
// (FUN_00449630).
func clearSlot(w *anm.World, a *eclActor, i int) {
	w.Delete(a.vms[i])
	a.vms[i] = nil
}

// slotOp is ECL 259, 262 and 269 (cases 0x103, 0x106 and 0x10d of
// FUN_0040e770), on the owner's actor; a slot outside 0-9 does nothing.
// Each first empties the slot (FUN_00449630: World.Delete of its object,
// then nil), then:
//
//   - 259 (slot, script): a negative script stops here. Otherwise
//     SpawnFront(fileAt(anmFile), script, layer 5) into the slot; for slot
//     0, curScript = script and bodyFile = anmFile (+0xf0, +0xec). Then
//     the tail.
//   - 262 (slot, script): SpawnFront(fileAt(anmFile), script, layer 5)
//     into the slot (no check of the script), then the tail; then, for
//     slot 0 only, flag 0x1000 (face the motion, see updateVisual), dirScript
//     = curScript = script (+0xf4, +0xf0), dir = 0 (+0xf8) and bodyFile =
//     anmFile.
//   - 269 (slot): SpawnFront(fileAt(anmFile), dirScript+5, layer 5) into
//     the slot, then the tail.
//
// The tail (0x40f103): for slot 0, mv.size = (h·Scale[1], w·Scale[0]) of
// the new object (VM+0x50 × +0x40, +0x4c × +0x3c), w and h being
// File.Sprites[vm.Sprite]'s after its first frame, 0 with no sprite; then,
// if the enemy has flag 0x10, World.Hide of the new object.
func (vm *VM) slotOp(ctx *stage.Context, t *eclTask, in *Instr) {
	dir5 := in.Opcode == opAnmSetDir5
	need := 2
	if dir5 {
		need = 1
	}
	if t.owner == nil || len(in.Params) < need {
		return
	}
	a := vm.actorFor(t.owner)
	slot := int(vm.resolveInt(t, in.Params[0]))
	script := a.dirScript + 5 // 269: the direction script plus 5
	if !dir5 {
		script = int(vm.resolveInt(t, in.Params[1]))
	}
	if slot < 0 || slot >= slotCount {
		return
	}
	w, env := effects(ctx)
	clearSlot(w, a, slot)
	if in.Opcode == opAnmSetSlot && script < 0 {
		return
	}
	if w != nil && env != nil {
		a.vms[slot] = w.SpawnFront(vm.fileAt(ctx, a.anmFile), script, 5, env)
		vm.slotTail(w, a, slot)
	}
	if slot == 0 && !dir5 {
		a.bodyFile = a.anmFile
		a.curScript = script
		if in.Opcode == opAnmSetMain {
			t.owner.ECLFlags |= 0x1000
			a.dirScript = script
			a.dir = 0
		}
	}
}

// slotTail is the end of 259/262/269 (0x40f103): slot 0 takes the new
// object's size, and a hidden enemy's new object is hidden.
func (vm *VM) slotTail(w *anm.World, a *eclActor, slot int) {
	v := a.vms[slot]
	if slot == 0 {
		var sw, sh float32
		if v.File != nil {
			if sp, ok := v.File.Sprites[v.Sprite]; ok {
				sw, sh = sp.W, sp.H
			}
		}
		a.mv.size = [2]float32{float32(sh * v.Scale[1]), float32(sw * v.Scale[0])}
	}
	if a.e.ECLFlags&0x10 != 0 {
		w.Hide(v)
	}
}

// effectOp is ECL 263, 264, 272 and 273 (cases 0x107, 0x108, 0x110 and
// 0x111), (file, script[, angle]): a free object of fileAt(file) on layer 6
// the enemy does not keep.
//
//   - 263: SpawnFront, then its Offset (the object's alone, not the chain's:
//     a direct write, not World.Place) = the enemy's final position + (224,
//     16, 0), or the position as is with flag 0x40000.
//   - 264: SpawnFront and nothing else.
//   - 272: as 263 with SpawnTail.
//   - 273: as 263, then Rot[2] = the angle (param 2) and FlagRotDirty.
//
// They need no owner for the spawn, but 263/272/273 read the owner's
// position; with no owner they do nothing.
func (vm *VM) effectOp(ctx *stage.Context, t *eclTask, in *Instr) {
	need := 2
	if in.Opcode == opAnmPlayRot {
		need = 3
	}
	if len(in.Params) < need {
		return
	}
	file := int(vm.resolveInt(t, in.Params[0]))
	script := int(vm.resolveInt(t, in.Params[1]))
	var angle float32
	if need == 3 {
		angle = vm.resolveFloat(t, in.Params[2])
	}
	w, env := effects(ctx)
	if w == nil || env == nil {
		return
	}
	if in.Opcode != opAnmPlayAbs && t.owner == nil {
		return
	}
	f := vm.fileAt(ctx, file)
	var v *anm.VM
	if in.Opcode == opAnmPlayTail {
		v = w.SpawnTail(f, script, 6, env)
	} else {
		v = w.SpawnFront(f, script, 6, env)
	}
	if in.Opcode == opAnmPlayAbs {
		return
	}
	a := vm.actorFor(t.owner)
	pos := a.mv.final.pos
	if a.e.ECLFlags&0x40000 == 0 {
		pos[0] += 224
		pos[1] += 16
	}
	v.Offset = pos
	if in.Opcode == opAnmPlayRot {
		v.Rot[2] = angle
		v.Flags |= anm.FlagRotDirty
	}
}

// showSlots is the tail of 322/323 (cases 0x142, 0x143): World.Show (show)
// or World.Hide of all ten slots. 322 calls it to hide when the flags it
// leaves have 0x10, 323 to show when they lack it.
func (vm *VM) showSlots(a *eclActor, show bool) {
	w, _ := effects(vm.ctx)
	for _, v := range a.vms {
		if show {
			w.Show(v)
		} else {
			w.Hide(v)
		}
	}
}

// faceDir is the direction switch of FUN_0040dc80 (0x40e2e4-0x40e374,
// flag 0x1000): dx, the final position's per-frame step in x (+0x38),
// gives dir -1 below -0.1, 1 above 0.1 and 0 otherwise (NaN included).
// When dir differs from old, ok is true and k is the offset from
// dirScript of the script slot 0 restarts with:
//
//	old  0 → -1: 1    old  0 → 1: 2
//	old -1 →  0: 3    old -1 → 1: 2
//	old  1 →  0: 4    old  1 → -1: 1
//	any other old: 0
func faceDir(old int32, dx float32) (dir, k int32, ok bool) {
	switch {
	case dx < -0.1:
		dir = -1
	case dx > 0.1:
		dir = 1
	}
	if dir == old {
		return dir, 0, false
	}
	switch {
	case old == 0 && dir == -1:
		k = 1
	case old == 0 && dir == 1:
		k = 2
	case old == -1 && dir == 0:
		k = 3
	case old == -1 && dir == 1:
		k = 2
	case old == 1 && dir == 0:
		k = 4
	case old == 1 && dir == -1:
		k = 1
	}
	return dir, k, true
}

// hitSE is the sound a hit flash plays (0x40e490-0x40e509): seHit, except
// for a boss (flag 0x8000) whose phase is nearly over, seHitLow: during a
// card (card bit 0) that is not a survival card (bit 3) when phaseHP <
// 300, and outside a card when phaseHP < 900.
func hitSE(boss, card, survival bool, phaseHP int32) int {
	if boss && (card && !survival && phaseHP < 300 || !card && phaseHP < 900) {
		return seHitLow
	}
	return seHit
}

// updateVisual is the part of FUN_0040dc80 between the body collision and
// the timers; updateActor calls it after the interrupt check.
//
//  1. Direction (flag 0x1000): dir, k, ok := faceDir(a.dir,
//     mv.final.delta[0]); if ok, a.dir = dir and World.Restart(slot 0,
//     dirScript + k).
//  2. Slots 0-7 follow the enemy: World.Place(slot, mv.final.pos), or
//     World.SetOffset(slot, mv.final.pos) with flag 0x40000.
//  3. Hit flash on slot 0. First the lookup: a slot 0 whose object has
//     left the world (!World.Contains) is emptied (nil). The binary then
//     dereferences the object unchecked, so it never gets here without
//     one; here a nil slot 0 skips only the writes to the object, the
//     counter and the sound still go. With flash != 0: clear the object's
//     0x8000 (colour 2 select) and flash-- (a boss also clears the GUI's
//     +0x9da4 0x8000; that is the boss GUI, not ported). Otherwise, if the
//     enemy was hit this frame (0x2000): the object's Color2/Alpha2 =
//     hitColor2, its 0x8000 set, flash = hitFlash, and
//     ctx.PlaySEID(hitSE(boss, vm.card.Active, vm.card.Survival,
//     a.phaseHP)).
//
// Restart resets the object's flags, so step 1 comes before step 3: a hit
// on the frame the direction changes still flashes.
//
// So a steady stream of hits shows one frame of flash every five frames.
func (vm *VM) updateVisual(a *eclActor) {
	e := a.e
	ctx := vm.ctx
	w, env := effects(ctx)

	// Direction (0x40e2e4): a change restarts slot 0 on the new script.
	if e.ECLFlags&0x1000 != 0 {
		if dir, k, ok := faceDir(a.dir, a.mv.final.delta[0]); ok {
			a.dir = dir
			w.Restart(a.vms[0], a.dirScript+int(k), env)
		}
	}

	// Slots 0-7 follow the enemy (0x40e38b).
	for i := range 8 {
		if e.ECLFlags&0x40000 != 0 {
			w.SetOffset(a.vms[i], a.mv.final.pos)
		} else {
			w.Place(a.vms[i], a.mv.final.pos)
		}
	}

	// Hit flash on slot 0 (0x40e447). An object that left the world is
	// forgotten first.
	if !w.Contains(a.vms[0]) {
		a.vms[0] = nil
	}
	body := a.vms[0]
	switch {
	case a.flash != 0:
		if body != nil {
			body.Flags &^= 0x8000
		}
		a.flash--
	case e.ECLFlags&0x2000 != 0:
		if body != nil {
			c := uint32(hitColor2)
			body.Color2 = [3]uint8{uint8(c >> 16), uint8(c >> 8), uint8(c)}
			body.Alpha2 = uint8(c >> 24)
			body.Flags |= 0x8000
		}
		a.flash = hitFlash
		if ctx != nil {
			ctx.PlaySEID(hitSE(e.ECLFlags&0x8000 != 0, vm.card.Active, vm.card.Survival, a.phaseHP))
		}
	}
}

// releaseSlots is FUN_0040dae0's sprite part, when the manager deletes the
// enemy: World.Delete of all ten slots, then nil. Update calls it where it
// deletes an enemy (updateActor false or flag 0x20000) and damaged where a
// hit removes one.
func (vm *VM) releaseSlots(a *eclActor) {
	w, _ := effects(vm.ctx)
	for i := range a.vms {
		clearSlot(w, a, i)
	}
}
