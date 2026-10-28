package ecl

// Special variables (ids -10000..-9950): FUN_0044fe40 (float) and
// FUN_0044fdb0 (int), with the enemy's own getters behind them (FUN_0043db60
// and friends). Ids not listed read 0.

import "math"

// actorOf is the actor running t, or nil (tests drive bare tasks).
func (vm *VM) actorOf(t *eclTask) *eclActor {
	if t == nil || t.owner == nil {
		return nil
	}
	return vm.actors[t.owner]
}

// aim is the heading from the enemy's final position to the player
// (FUN_004073e0): exactly pi/2 (down) when the two coincide.
func (vm *VM) aim(a *eclActor) float32 {
	dx := vm.playerECL[0] - a.mv.final.pos[0]
	dy := vm.playerECL[1] - a.mv.final.pos[1]
	if dx == 0 && dy == 0 {
		return halfPi32
	}
	return float32(math.Atan2(float64(dy), float64(dx)))
}

func b2f(b bool) float32 {
	if b {
		return 1
	}
	return 0
}

// specialFloat is the float read of special var id.
func (vm *VM) specialFloat(t *eclTask, id int32) float32 {
	switch id {
	case -10000:
		return float32(vm.rngRaw())
	case -9999:
		return vm.randFloat()
	case -9998:
		return float32(vm.randSigned() * pi32)
	case -9987:
		return vm.randSigned()
	case -9991, -9965:
		return vm.playerECL[0]
	case -9990, -9964:
		return vm.playerECL[1]
	case -9963, -9962:
		if b := vm.boss0(); b != nil {
			return b.mv.final.pos[id+9963]
		}
		return 0
	case -9960:
		return float32(vm.rank)
	case -9959:
		return float32(vm.difficulty)
	case -9957:
		return 1
	case -9953, -9952, -9951, -9950:
		return b2f(vm.difficulty == int(id+9953))
	}
	a := vm.actorOf(t)
	if a == nil {
		return 0
	}
	m := &a.mv
	switch id {
	case -9997, -9977:
		return m.final.pos[0]
	case -9996, -9976:
		return m.final.pos[1]
	case -9995, -9975:
		return m.layer[0].pos[0]
	case -9994, -9974:
		return m.layer[0].pos[1]
	case -9993, -9973:
		return m.layer[1].pos[0]
	case -9992, -9972:
		return m.layer[1].pos[1]
	case -9989, -9956, -9955:
		return vm.aim(a)
	case -9988:
		return a.phase.f
	case -9986:
		return float32(a.e.ECLFlags >> 16 & 1)
	case -9985, -9984, -9983, -9982: // I0-I3
		return float32(int32(a.vars[id+9985]))
	case -9981, -9980, -9979, -9978: // F0-F3
		return math.Float32frombits(a.vars[id+9985])
	case -9971:
		return m.layer[0].angle
	case -9970:
		return m.layer[1].angle
	case -9969:
		return m.layer[0].speed
	case -9968:
		return m.layer[1].speed
	case -9967:
		return m.layer[0].radius
	case -9966:
		return m.layer[1].radius
	case -9958:
		return float32(math.Atan2(float64(m.final.delta[1]), float64(m.final.delta[0])))
	case -9954:
		return float32(a.e.HP)
	}
	return 0
}

// specialInt is the int read of special var id: the float value truncated,
// except where the int getter differs.
func (vm *VM) specialInt(t *eclTask, id int32) int32 {
	switch id {
	case -10000:
		return int32(vm.rngRaw())
	case -9999, -9987: // the draw happens, the int getter returns 0
		vm.rngRaw()
		return 0
	case -9998, -9989, -9956, -9955:
		return 0
	}
	if a := vm.actorOf(t); a != nil {
		switch id {
		case -9988:
			return a.phase.cur
		case -9985, -9984, -9983, -9982:
			return int32(a.vars[id+9985])
		case -9961:
			return int32(a.curScript)
		}
	}
	return int32(vm.specialFloat(t, id))
}

// storeSpecial writes I0-I3/F0-F3 (FUN_00412300); other ids are read-only.
func (vm *VM) storeSpecial(t *eclTask, id int32, v eVal) {
	if id < -9985 || id > -9978 {
		return
	}
	if a := vm.actorOf(t); a != nil {
		a.vars[id+9985] = v.bits
	}
}
