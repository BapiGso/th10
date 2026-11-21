package ending

import (
	sceneui "th10/scene/ui"
	"th10/supervisor"
)

// screenFade is only kinds 0/5 of FUN_0043c8b0/0043c970, used by ending
// MSG 13/14. It has no game RNG and does not touch a Stage camera.
type screenFade struct {
	*sceneui.Fade
	alpha int32 // existing private test observation, not a second calculation
}

func newScreenFade(kind, duration int32) *screenFade {
	return &screenFade{Fade: sceneui.NewFade(kind, duration, 0)}
}

func (f *screenFade) calc() bool {
	f.Calc(1, false, 0)
	f.alpha = f.Alpha
	return !f.Active
}

func (f *screenFade) draw(sv *supervisor.Supervisor) {
	// FUN_0043c1a0, registered at draw 0x31: full screen, RGB zero.
	sv.Use(&sv.Screen)
	f.DrawRect(sv)
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
