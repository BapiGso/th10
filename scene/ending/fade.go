package ending

import (
	"math"

	"th10/gfx"
	"th10/supervisor"
	"th10/zun"
)

// screenFade is only kinds 0/5 of FUN_0043c8b0/0043c970, used by ending
// MSG 13/14. It has no game RNG and does not touch a Stage camera.
type screenFade struct {
	kind, duration int32
	timer          zun.Timer
	alpha          int32
}

func newScreenFade(kind, duration int32) *screenFade {
	f := &screenFade{kind: kind, duration: duration}
	f.timer.Reset() // FUN_0043c970, 0x43caa5..0x43cace
	return f
}

func (f *screenFade) calc() bool {
	// FUN_0043bd40 (kind 0)/0043c230 (kind 5) sample pre-tick time.
	unit := math.Float32frombits(0x437f0000) // PE 0x470bf8, 255
	if f.duration != 0 {
		if f.kind == 0 {
			x := float32(f.timer.F * unit)
			x = float32(float64(x) / float64(f.duration))
			f.alpha = max(int32(unit-x), 0) // 0x43bd59..0x43bd78
		} else if f.timer.Cur < f.duration {
			x := float32(f.timer.F * unit)
			f.alpha = max(int32(float32(float64(x)/float64(f.duration))), 0)
		} else {
			f.alpha = 255 // 0x43c265
		}
	}
	limit := f.duration
	if f.kind == 5 {
		limit += 2 // 0x43c27d, two full-alpha frames before destructor
	}
	if f.timer.Cur >= limit {
		return true
	}
	f.timer.Tick(1)
	return false
}

func (f *screenFade) draw(sv *supervisor.Supervisor) {
	// FUN_0043c1a0, registered at draw 0x31: full screen, RGB zero.
	sv.Use(&sv.Screen)
	sv.ANM.Rect([4]float32{0, 0, 640, 480}, gfx.Color(uint32(f.alpha)<<24))
}

func (e *Ending) calcFades() {
	n := 0
	for _, fade := range e.fades {
		if !fade.calc() {
			e.fades[n] = fade
			n++
		}
	}
	clear(e.fades[n:])
	e.fades = e.fades[:n]
}
