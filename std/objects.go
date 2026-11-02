package std

import "th10/anm"

// Enable is FUN_00404450, run once at stage start (FUN_00418190) after the
// stage's .anm a is loaded: it gives the objects env to run with, sets every
// entry's Active (a whole-byte store of 1 in the original, quadless entries
// too), and starts one VM per quad in order, startVM(&Quads[k],
// quad.ScriptIndex), recording each entry's first index.
func (v *VM) Enable(a *anm.File, env *anm.Env) {
	v.anm, v.env = a, env
	n := 0
	for _, e := range v.f.Entries {
		n += len(e.Quads)
	}
	v.Quads = make([]anm.VM, n)
	v.Active = make([]bool, len(v.f.Entries))
	v.first = make([]int, len(v.f.Entries))
	k := 0
	for i, e := range v.f.Entries {
		v.Active[i] = true
		v.first[i] = k
		for _, q := range e.Quads {
			v.startVM(&v.Quads[k], q.ScriptIndex)
			k++
		}
	}
}

// File is the .std v runs.
func (v *VM) File() *File { return v.f }

// Quad is the VM of entry e's quad j, the one FUN_00403a30 finds through
// the quad's +6 index into +0x17c: &Quads[first[e]+j]. Valid after Enable.
func (v *VM) Quad(e, j int) *anm.VM { return &v.Quads[v.first[e]+j] }

// startVM is FUN_00404f30 on vm: reset it (FUN_00401de0), zero Pos,
// Offset and Pos2 (anm.VM.Reset keeps Offset, so zero it here), then start
// script of v's .anm (FUN_0043e710, anm.VM.Start with v.env), which runs
// its first frame. The bytes it also sets to 0x10 (+0x3a0, +0x3a1) are not
// modelled until something is found to read them.
func (v *VM) startVM(vm *anm.VM, script uint16) {
	vm.Reset()
	vm.Pos, vm.Offset, vm.Pos2 = [3]float32{}, [3]float32{}, [3]float32{}
	vm.Start(v.anm, int(script), v.env)
}

// stepEntries is FUN_00403990: for each entry in order whose Active is set,
// Run(env) each of its quads' VMs in order and count those not Ended
// afterwards (VM+0x390 != 0); with none, clear Active. Entries whose
// Active is clear are skipped, quads and all.
func (v *VM) stepEntries() {
	for i, e := range v.f.Entries {
		if !v.Active[i] {
			continue
		}
		running := 0
		for j := range e.Quads {
			vm := &v.Quads[v.first[i]+j]
			vm.Run(v.env)
			if !vm.Ended() {
				running++
			}
		}
		v.Active[i] = running != 0
	}
}
