package anm

// The world list of free-standing animation objects (DAT_00491c10): bullet
// cancel sparks, graze sparks, op-88 children and the other effects the game
// spawns without an owner. It runs once a frame at calc priority 0x1a, after
// the items.

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

// Add appends vm (FUN_004489d0). Use it as Env.Spawn so op-88 children join
// the list.
func (w *World) Add(vm *VM) { w.list = append(w.list, vm) }

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
// ones, and sort the rest into their layer's draw list. Each node's successor
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
// a Run leaves once the object has ended or was marked for deletion.
func (w *World) Contains(vm *VM) bool {
	return vm != nil && slices.Contains(w.list, vm)
}

// Len is the number of live objects.
func (w *World) Len() int { return len(w.list) }

// Clear drops every object.
func (w *World) Clear() {
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
