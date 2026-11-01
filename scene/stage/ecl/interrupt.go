package ecl

// Boss phases: the interrupt slots armed by ins_334/341 (FUN_00412720), the
// slot check FUN_004127a0, and the parts of the enemy update FUN_0040dc80
// around it (damage, invulnerability, the phase timer).
//
// A phase is not a task of its own. When a slot fires, the enemy's primary
// ECL context is pointed at the slot's sub: its async tasks are freed, its
// instruction pointer and clock restart, its call stack is kept. The check
// runs twice per frame: after damage (and the new sub then runs at once),
// and again unconditionally (the new sub starts next frame).

import (
	"th10/entity/enemy"
	"th10/scene/stage"
)

// interruptSlot is one of an enemy's 8 slots (B+0x2494+0x10*i). hp < 0 means
// unarmed.
type interruptSlot struct {
	hp, time        int32
	sub, timeoutSub string
}

// eclTimer is the binary's frame timer: a whole-frame count and its float
// accumulator. The enemy's phase timer (B+0x1158, reset by ins_333, counted
// against an interrupt's time limit) and invulnerability timer are ones.
type eclTimer struct {
	cur int32
	f   float32
}

// set is FUN_00405410.
func (t *eclTimer) set(n int32) { t.cur, t.f = n, float32(n) }

// add adds delta frames at the game speed.
func (t *eclTimer) add(delta, speed float32) {
	if speed > 0.99 && speed < 1.01 {
		t.f = delta + t.f
	} else {
		t.f = float32(delta*speed) + t.f
	}
	t.cur = int32(t.f)
}

// tick advances by the game speed: within 1% of 1 it counts whole frames,
// otherwise it accumulates and truncates.
func (t *eclTimer) tick(speed float32) {
	if speed > 0.99 && speed < 1.01 {
		t.f++
		t.cur++
		return
	}
	t.f += speed
	t.cur = int32(t.f)
}

// timed reports whether an armed slot carries a time limit (the boss timer
// is then on screen).
func (a *eclActor) timed() bool {
	for _, s := range a.slots {
		if s.hp >= 0 && s.time >= 1 {
			return true
		}
	}
	return false
}

// checkInterrupt is FUN_004127a0. The first armed slot fires when the HP has
// fallen to its threshold (the HP is raised to exactly that); otherwise the
// first armed slot with a time limit updates the boss timer and fires once
// the phase timer reaches it, taking the HP to its threshold, flagging the
// timeout (0x10000) and costing faith.
func (vm *VM) checkInterrupt(e *enemy.Enemy, a *eclActor) (string, bool) {
	for i := range a.slots {
		s := &a.slots[i]
		if s.hp < 0 {
			continue
		}
		if e.HP > int(s.hp) {
			break
		}
		e.HP = int(s.hp)
		s.hp = -1
		a.phase.set(0)
		e.ECLFlags &^= 0x10000
		return s.sub, true
	}
	for i := range a.slots {
		s := &a.slots[i]
		if s.hp < 0 || s.time < 1 {
			continue
		}
		vm.hudTimer = min(99, int((s.time-a.phase.cur+59)/60))
		if a.phase.cur < s.time {
			return "", false
		}
		e.HP = int(s.hp)
		s.hp = -1
		a.phase.set(0)
		e.ECLFlags |= 0x10000
		// DAT_00474c4c -= 3000, floored at 5000 (this project keeps faith x10).
		if vm.ctx != nil && vm.ctx.State != nil {
			vm.ctx.State.Faith = max(vm.ctx.State.Faith-30000, 50000)
		}
		return s.timeoutSub, true
	}
	return "", false
}

// interrupt points e's primary context at sub name: the async contexts are
// freed and the instruction pointer and clock restart, keeping the call
// stack (FUN_0040c6e0 + FUN_0040c730). An unknown name leaves no sub, so the
// enemy is deleted on its next run.
func (vm *VM) interrupt(e *enemy.Enemy, a *eclActor, name string) {
	for _, t := range a.asyncs {
		t.done = true
	}
	a.asyncs = nil
	t := a.main
	if t == nil {
		return
	}
	t.sub = vm.prog.SubByName[name]
	t.pc, t.clock, t.done, t.blocked = 0, 0, false, false
	if t.sub != nil {
		t.frame = newFrame(t.sub, nil)
	}
}

// Results of hurt, as FUN_0040dc80 returns them.
const (
	hurtAlive   = 0
	hurtKilled  = 1
	hurtRemoved = -1 // the phase the damage started ended at once
)

// hurt is FUN_0040dc80's damage step for dmg != 0: a fifth (at least 1) for
// a boss (0x8000) while a spell card is up (DAT_004776f4+0x378c bit 0), the
// HP unless invulnerable (0x8, ins_335), a phase the damage triggers runs at
// once, and at HP < 1 the defeat unless the enemy is immortal (0x40).
func (vm *VM) hurt(a *eclActor, dmg int) int {
	e := a.e
	if vm.spell.active && e.ECLFlags&0x8000 != 0 {
		dmg = max(dmg/5, 1)
	}
	if e.ECLFlags&0x8 == 0 && a.invuln.cur < 1 {
		e.HP -= dmg
		if vm.ctx != nil {
			vm.ctx.EnemyHurt(dmg)
		}
	}
	if name, ok := vm.checkInterrupt(e, a); ok {
		vm.interrupt(e, a, name)
		if !vm.runActor(a) {
			return hurtRemoved
		}
	}
	if e.HP < 1 && e.ECLFlags&0x40 == 0 {
		vm.kill(a)
		return hurtKilled
	}
	e.ECLFlags |= 0x2000
	return hurtAlive
}

// damaged is installed as Enemy.Damaged for damage dealt outside the enemy's
// update (the placeholder bomb). It reports whether the hit destroyed e.
func (vm *VM) damaged(e *enemy.Enemy, dmg int) bool {
	a := vm.actors[e]
	if a == nil || e.ECLFlags&0x11 != 0 || dmg == 0 {
		return false
	}
	r := vm.hurt(a, dmg)
	if r != hurtAlive {
		e.Active = false
	}
	return r == hurtKilled
}

// LateUpdate runs after the stage's collisions: it forgets the enemies
// deleted this frame.
func (vm *VM) LateUpdate(ctx *stage.Context) {
	vm.ctx = ctx
	vm.prune()
}

// prune forgets deleted enemies and empties their boss slots.
func (vm *VM) prune() {
	n := 0
	for _, a := range vm.order {
		if a.e.Active {
			vm.order[n] = a
			n++
		} else {
			delete(vm.actors, a.e)
		}
	}
	clear(vm.order[n:])
	vm.order = vm.order[:n]
	for i, e := range vm.bossSlots {
		if e != nil && !e.Active {
			vm.bossSlots[i] = nil
		}
	}
}
