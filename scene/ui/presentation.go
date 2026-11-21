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

// Run keeps FUN_00448860's list walk before ANM-91 front-child emission.
// A nil emitter retains callers whose common VM still treats 91 as a no-op.
func (a TitleAnimations) Run(emitter *LoadingEmitter) {
	var before int32
	if emitter != nil && emitter.Parent != nil {
		before = emitter.Parent.T.Cur
	}
	a.World.Run(a.Env)
	if emitter != nil {
		emitter.afterRun(a, before)
	}
}

// LoadingEmitter is the existing ascii.anm script-8 subset of FUN_0043ee30
// at 0x43ef55/0x43f00d: sixteen front children, then one every three frames.
// It is deliberately opt-in; enabling it on title-A would change that scene.
type LoadingEmitter struct {
	Parent *anm.VM
	File   *anm.File
	Burst  bool
}

func (e *LoadingEmitter) afterRun(a TitleAnimations, before int32) {
	if !a.World.Contains(e.Parent) {
		return
	}
	count := 0
	if !e.Burst && before == 1 && e.Parent.T.Cur == 2 {
		count, e.Burst = 16, true
	} else if before == 4 && e.Parent.T.Cur == 2 {
		count = 1
	}
	for range count {
		v := a.World.SpawnFront(e.File, 7, e.Parent.Layer, a.Env)
		v.Pos2, v.Offset = e.Parent.Pos, e.Parent.Offset
	}
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
