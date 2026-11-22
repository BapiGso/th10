package ui

import (
	"math"

	"th10/anm"
	"th10/supervisor"
)

// TitleAnimations is FUN_0042c670/0042c6d0/0042c770's common title-object
// facade. Callers retain their existing World, file choice and lifetime.
// Roots is optional for scene-B pages that already keep their own handles.
type TitleAnimations struct {
	World *anm.World
	Env   *anm.Env
	File  *anm.File
	Roots map[int]*anm.VM
}

func (a TitleAnimations) Root(script int) *anm.VM {
	vm := a.Roots[script]
	if !a.World.Contains(vm) {
		if a.Roots != nil {
			a.Roots[script] = nil
		}
		return nil
	}
	return vm
}

func (a TitleAnimations) Spawn(script int) *anm.VM {
	vm := a.World.SpawnTail(a.File, script, 15, a.Env)
	if a.Roots != nil {
		a.Roots[script] = vm
	}
	return vm
}

func (a TitleAnimations) Leave(script int) {
	a.Interrupt(script, 1)
	if a.Roots != nil {
		a.Roots[script] = nil
	}
}

func (a TitleAnimations) InterruptVM(vm *anm.VM, label int16) {
	a.World.Interrupt(vm, label)
}

func (a TitleAnimations) Interrupt(script int, label int16) {
	a.InterruptVM(a.Root(script), label)
}

func (a TitleAnimations) Child(root, script int) *anm.VM {
	return a.World.FindChild(a.Root(root), script)
}

func (a TitleAnimations) InterruptChild(root, script int, label int16) {
	a.InterruptVM(a.Child(root, script), label)
}

// Run is FUN_00448860's list walk. The common VM now emits ascii script 8's
// ANM-91 front children itself (0x43ef32), so no scene-side emitter remains.
func (a TitleAnimations) Run() {
	a.World.Run(a.Env)
}

// GlyphTextFile is FUN_00401760's native small-font pass. Keeping the file
// as an argument preserves raw/presentation resource selection at each caller.
func GlyphTextFile(sv *supervisor.Supervisor, file *anm.File, text string, x, y float32, colour uint32) {
	vm := anm.New()
	vm.File = file
	vm.Flags = vm.Flags&0xffd7ffff | 0x140001
	vm.Color, vm.Alpha = [3]uint8{uint8(colour >> 16), uint8(colour >> 8), uint8(colour)}, uint8(colour>>24)
	start := x
	for i := range len(text) {
		ch := text[i]
		if ch == '\n' {
			x, y = start, y+16
			continue
		}
		if ch < 0x20 || ch > 0x81 {
			ch = '?'
		}
		vm.SetSprite(int(ch) + 66)
		vm.Pos = [3]float32{x, y, 0}
		sv.ANM.Draw(vm)
		x += math.Float32frombits(0x41100000) // 9, PE 0x470c10.
	}
}
