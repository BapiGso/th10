package sprite

import (
	"sync"

	"th10/anm"

	"github.com/hajimehoshi/ebiten/v2"
)

// sheets caches the AnmSheet of each parsed .anm the VMs draw from.
var sheets sync.Map // *anm.File -> *AnmSheet

// SheetFor returns the textures of f, built on first use.
func SheetFor(f *anm.File) *AnmSheet {
	if v, ok := sheets.Load(f); ok {
		return v.(*AnmSheet)
	}
	v, _ := sheets.LoadOrStore(f, NewAnmSheet(f))
	return v.(*AnmSheet)
}

// DrawVM draws an ANM object's current sprite onto dst, its world position
// shifted by (dx, dy): centred, scaled (flips negate the scale), turned by the
// z rotation, tinted by its colour and alpha, additive in blend mode 1.
func DrawVM(dst *ebiten.Image, vm *anm.VM, dx, dy float64) {
	if vm == nil || vm.File == nil || !vm.Visible() || vm.Sprite < 0 {
		return
	}
	img := SheetFor(vm.File).Sprite(vm.Sprite)
	if img == nil {
		return
	}
	b := img.Bounds()
	sx, sy := float64(vm.Scale[0]), float64(vm.Scale[1])
	if vm.Flags&anm.FlagFlipX != 0 {
		sx = -sx
	}
	if vm.Flags&anm.FlagFlipY != 0 {
		sy = -sy
	}
	op := &ebiten.DrawImageOptions{}
	// The anchor (flag bits 18-19 across, 20-21 down): 0 centres the sprite
	// on the position, 1 puts its left/top edge there, 2 its right/bottom.
	ax, ay := -float64(b.Dx())/2, -float64(b.Dy())/2
	switch vm.Flags >> 18 & 3 {
	case 1:
		ax = 0
	case 2:
		ax = -float64(b.Dx())
	}
	switch vm.Flags >> 20 & 3 {
	case 1:
		ay = 0
	case 2:
		ay = -float64(b.Dy())
	}
	op.GeoM.Translate(ax, ay)
	op.GeoM.Scale(sx, sy)
	op.GeoM.Rotate(float64(vm.Rot[2]))
	op.GeoM.Translate(
		float64(vm.Pos[0]+vm.Offset[0]+vm.Pos2[0])+dx,
		float64(vm.Pos[1]+vm.Offset[1]+vm.Pos2[1])+dy)
	a := float32(vm.Alpha) / 255
	op.ColorScale.Scale(
		float32(vm.Color[0])/255*a, float32(vm.Color[1])/255*a, float32(vm.Color[2])/255*a, a)
	if vm.Blend() == 1 {
		op.Blend = ebiten.BlendLighter
	}
	op.Filter = ebiten.FilterLinear
	dst.DrawImage(img, op)
}
