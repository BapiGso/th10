package ecl

// execRankOp ports FUN_0040e770 cases 349-354 (0x4118bb..0x411b6b).
// The selects use the parameter indices in the binary, not the apparent
// output-plus-choices layout: 349/352 read p0 below rank 512, and 350/353
// read p0..p4 rather than p1..p5. Only the selected source is evaluated.
func (vm *VM) execRankOp(t *eclTask, in *Instr) {
	p := in.Params
	n := len(th10Format(in.Opcode))
	if len(p) < n {
		return
	}

	switch in.Opcode {
	case opRankSelectF3: // 0x4118bb: the two lower tiers both select p0.
		i := 0
		if vm.rank >= 512 {
			i = 2
		}
		value := vm.resolveFloat(t, p[i]) // FUN_0044fe40 before FUN_00450070(0).
		vm.storeRankResult(t, p[0], floatVal(value))
	case opRankSelectF5: // 0x4118e2: unlike 423, the lowest boundary is -400.
		value := vm.resolveFloat(t, p[rankSelectTier5(vm.rank)])
		vm.storeRankResult(t, p[0], floatVal(value))
	case opRankLerpF: // 0x41198b: p1, p2, output; float32 steps, 0x470d48/4c.
		lo := vm.resolveFloat(t, p[1])
		hi := vm.resolveFloat(t, p[2])
		vm.storeRankResult(t, p[0], floatVal(speedBlend(lo, hi, vm.rank)))
	case opRankSelectI3: // 0x4119fc: output address first, then p0 or p2.
		out := p[0] // FUN_00412a10(0) resolves an address without reading it.
		i := 0
		if vm.rank >= 512 {
			i = 2
		}
		value := vm.resolveInt(t, p[i])
		vm.storeRankResult(t, out, intVal(value))
	case opRankSelectI5: // 0x411a3e: output address first, then the one source.
		out := p[0]
		value := vm.resolveInt(t, p[rankSelectTier5(vm.rank)])
		vm.storeRankResult(t, out, intVal(value))
	case opRankLerpI: // 0x411b23: p1, p2, output; signed 32-bit product and /2048.
		lo := vm.resolveInt(t, p[1])
		hi := vm.resolveInt(t, p[2])
		vm.storeRankResult(t, p[0], intVal(rankBlend(hi-lo, vm.rank)+lo))
	}
}

// rankSelectTier5 is 350/353's source index (0x4118e2/0x411a3e).
func rankSelectTier5(rank int) int {
	switch {
	case rank >= 600:
		return 4
	case rank >= 200:
		return 3
	case rank >= -200:
		return 2
	case rank >= -400:
		return 1
	default:
		return 0
	}
}

// storeRankResult uses the output reference of FUN_00450070/00412a10.
// An immediate is not a writable destination; those helpers return null.
func (vm *VM) storeRankResult(t *eclTask, out Param, value eVal) {
	if out.IsVar {
		vm.storeVar(t, out, value)
	}
}
