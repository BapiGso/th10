package ui

import (
	"math"

	"th10/gfx"
	"th10/supervisor"
	"th10/zun"
)

// Fade is FUN_0043c8b0/0043c970 kinds 0/2/3/5. Kind 1 remains t7's
// screen shake; it must not be treated as this draw-chain rectangle.
type Fade struct {
	Kind, Duration int32     // +0x10/+0x1c
	Color          uint32    // +0x20, RGB (the draw ORs it with Alpha << 24)
	Alpha          int32     // +0x18
	Timer          zun.Timer // +0x30
	Active         bool
}

func NewFade(kind, frames int32, rgb uint32) *Fade {
	if kind != 0 && kind != 2 && kind != 3 && kind != 5 {
		panic("ui: unsupported native fade kind")
	}
	f := &Fade{Kind: kind, Duration: frames, Color: rgb, Active: true}
	if kind == 3 {
		f.Alpha = 255
	} // FUN_0043c970, 0x43c9e0
	f.Timer.Reset() // 0x43caa5..0x43cace
	return f
}

// Calc is FUN_0043bd40 (in) / FUN_0043c230 (out). Alpha reads the
// pre-tick accumulator; unlike t7's shake, the timer ticks afterward.
func (f *Fade) Calc(speed float32, exiting bool, stageFlags uint32) {
	if !f.Active {
		return
	}
	in := f.Kind == 0 || f.Kind == 3
	if f.Duration != 0 {
		v := float32(f.Timer.F * math.Float32frombits(0x437f0000)) // PE 0x470bf8: 255
		v = float32(float64(v) / float64(f.Duration))
		if in {
			v = math.Float32frombits(0x437f0000) - v
		}
		if !in && f.Timer.Cur >= f.Duration {
			v = 255
		}
		f.Alpha = max(int32(v), 0)
	}
	end := f.Duration
	if !in {
		end += 2
	} // 0x43c27a..0x43c283
	if exiting || f.Timer.Cur >= end {
		f.Active = false // native calc return 7 -> FUN_0043c870
		return
	}
	if in || stageFlags&5 == 0 {
		f.Timer.Tick(speed)
	} // 0x43c296..0x43c2a8
}

// Draw is FUN_0043c1a0 (full screen) / FUN_0043c2c0 (field).
func (f *Fade) Draw(sv *supervisor.Supervisor) {
	if !f.Active {
		return
	}
	rect := [4]float32{0, 0, 640, 480}
	if f.Kind == 2 || f.Kind == 3 {
		rect = [4]float32{32, 16, 416, 464}
	}
	sv.Use(&sv.Screen)
	sv.Fog(false)
	sv.ANM.Rect(rect, gfx.Color(uint32(f.Alpha)<<24|f.Color))
}
