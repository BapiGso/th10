package anm

// The world list of free-standing animation objects (DAT_00491c10): bullet
// cancel sparks, graze sparks, op-88 children, the enemies' sprites (the ten
// slots at enemy+0xc0, spawned by ECL 259/262/263/264/269/272/273) and the
// other effects the game spawns. It runs once a frame at calc priority 0x1a,
// after the items.
//
// The binary refers to a world object by a handle that FUN_004491c0 looks up
// in the list; once the object has left the list (a Run removed it or Clear
// dropped it) the handle finds nothing and every helper taking one does
// nothing. Here the handle is the *VM itself, and the helpers below check
// that it is still in w, which a VM records in its world field.

import "slices"

// More VM flag bits.
const (
	// FlagAutoRotate (op 82) makes a bullet's sprite follow its heading.
	FlagAutoRotate uint32 = 0x8000000
	// FlagDelete marks a world object for deletion at its next run.
	FlagDelete uint32 = 0x4000000
)

// Layers is the number of draw lists the world sorts its objects into.
const Layers = 19

// World is the effect list. The zero value is empty and ready to use.
type World struct {
	list   []*VM
	layers [Layers][]*VM
}

// Add appends vm (FUN_004489d0) and records w as its world. Use it as
// Env.Spawn so op-88 children join the list.
func (w *World) Add(vm *VM) {
	vm.world = w
	w.list = append(w.list, vm)
}

// AddFront is FUN_00448a50: Add, but at the head of the list, so vm runs
// before every object already there. Not for use while Run walks the list
// (no ANM op calls it).
func (w *World) AddFront(vm *VM) {
	vm.world = w
	w.list = slices.Insert(w.list, 0, vm)
}

// SpawnFront is FUN_00448e30, which ECL's enemy sprite ops use: a new object
// on layer, flagged for the alternate RNG, started without a reset
// (FUN_00449870, so the flag survives the first frame; Offset stays zero)
// and put at the head of the list with AddFront. A missing script still
// inserts the object, ended, so it stays in w until the next Run removes it.
func (w *World) SpawnFront(f *File, id int, layer int32, env *Env) *VM {
	vm := w.spawn(f, id, layer, env)
	w.AddFront(vm)
	return vm
}

// SpawnTail is FUN_00448d00: SpawnFront, but appended with Add.
func (w *World) SpawnTail(f *File, id int, layer int32, env *Env) *VM {
	vm := w.spawn(f, id, layer, env)
	w.Add(vm)
	return vm
}

// spawn is the start both spawn helpers share: a new object on layer, flagged
// for the alternate RNG and started without a reset.
func (w *World) spawn(f *File, id int, layer int32, env *Env) *VM {
	vm := New()
	vm.Layer = layer
	vm.Flags |= FlagAltRNG
	vm.StartKeep(f, id, env)
	return vm
}

// eachHandle is the walk the handle helpers share: vm if it is still in w,
// and if vm is a root, the children in its chain that are still in w. Nil
// or a VM no longer in w does nothing; called on a child it does only the
// child.
func (w *World) eachHandle(vm *VM, fn func(*VM)) {
	if !w.Contains(vm) {
		return
	}
	fn(vm)
	if vm.root != nil {
		return
	}
	for _, c := range vm.chain {
		if w.Contains(c) {
			fn(c)
		}
	}
}

// Delete is FUN_004492a0: mark the object, and on a root its chain, for
// deletion at the next Run (FlagDelete).
func (w *World) Delete(vm *VM) {
	w.eachHandle(vm, func(v *VM) { v.Flags |= FlagDelete })
}

// Hide is FUN_004495e0: clear FlagDraw on the object and its chain.
func (w *World) Hide(vm *VM) {
	w.eachHandle(vm, func(v *VM) { v.Flags &^= FlagDraw })
}

// Show is FUN_00449590: set FlagDraw on the object and its chain.
func (w *World) Show(vm *VM) {
	w.eachHandle(vm, func(v *VM) { v.Flags |= FlagDraw })
}

// Place is FUN_00449350: Offset = game position pos moved into screen space
// (+224, +16; z as is) on the object and its chain.
func (w *World) Place(vm *VM, pos [3]float32) {
	off := [3]float32{pos[0] + 224, pos[1] + 16, pos[2]}
	w.eachHandle(vm, func(v *VM) { v.Offset = off })
}

// SetOffset is FUN_004492f0: Offset = pos as given, on the object and its
// chain.
func (w *World) SetOffset(vm *VM, pos [3]float32) {
	w.eachHandle(vm, func(v *VM) { v.Offset = pos })
}

// Restart is FUN_004496d0: VM.Restart(id) on the object alone (no chain),
// with its own file. Nil or a VM no longer in w does nothing.
func (w *World) Restart(vm *VM, id int, env *Env) {
	if w.Contains(vm) {
		vm.Restart(id, env)
	}
}

// Spawn is FUN_00448db0: a new object on layer 0 at game position pos, offset
// into screen space, started (its first frame runs now) and added to the
// list. The binary sets FlagAltRNG before the start, whose reset clears it.
func (w *World) Spawn(f *File, id int, pos [3]float32, env *Env) *VM {
	vm := New()
	vm.Layer = 0
	vm.Flags |= FlagAltRNG
	vm.Offset = [3]float32{pos[0] + 224, pos[1] + 16, pos[2]}
	vm.Start(f, id, env)
	w.Add(vm)
	return vm
}

// Run is FUN_00448860: run every object once, delete the marked and ended
// ones (they are no longer in w), and sort the rest into their layer's draw
// list. Each node's successor
// is read before it runs, so objects appended while the tail runs wait a
// frame; those appended earlier in the walk run this frame.
func (w *World) Run(env *Env) {
	for i := range w.layers {
		w.layers[i] = w.layers[i][:0]
	}
	dead := false
	for i := 0; i < len(w.list); i++ {
		vm := w.list[i]
		last := i == len(w.list)-1
		if vm.Flags&FlagDelete != 0 || vm.Run(env) {
			w.list[i] = nil
			vm.world = nil
			dead = true
		} else if l := vm.Layer; l >= 0 && l < Layers {
			w.layers[l] = append(w.layers[l], vm)
		}
		if last {
			break
		}
	}
	if dead {
		n := 0
		for _, vm := range w.list {
			if vm != nil {
				w.list[n] = vm
				n++
			}
		}
		clear(w.list[n:])
		w.list = w.list[:n]
	}
}

// Contains is FUN_004491c0's lookup: whether vm is still in the list, which
// a Run leaves once the object has ended or was marked for deletion. It reads
// vm's world field (set by Add and AddFront, cleared by Run's removal and by
// Clear), so it costs O(1).
func (w *World) Contains(vm *VM) bool {
	return w != nil && vm != nil && vm.world == w
}

// Len is the number of live objects.
func (w *World) Len() int { return len(w.list) }

// Clear drops every object; none of them is in w afterwards.
func (w *World) Clear() {
	for _, vm := range w.list {
		if vm != nil {
			vm.world = nil
		}
	}
	clear(w.list)
	w.list = w.list[:0]
	for i := range w.layers {
		w.layers[i] = w.layers[i][:0]
	}
}

// Each calls fn for the objects sorted into draw layer l by the last Run.
func (w *World) Each(l int, fn func(*VM)) {
	if l < 0 || l >= Layers {
		return
	}
	for _, vm := range w.layers[l] {
		fn(vm)
	}
}
