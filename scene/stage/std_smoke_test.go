package stage

import (
	"fmt"
	"strings"
	"testing"

	"th10/anm"
	"th10/std/stddraw"
	"th10/supervisor"
	"th10/supervisor/supervisortest"
	"th10/zun"
)

// TestSTDSmoke runs each convertible stage's embedded STD for a stage's
// length and draws every frame with the STD drawer (nodes 0x07 and 0x0a),
// so a render mode or an anchor anmdraw does not draw shows up here rather
// than in the game. It checks that quads are drawn, not what they look
// like. Stages 1 and 5 draw their mode 6 quads as 2D triangles, so they
// must also issue DrawTL.
func TestSTDSmoke(t *testing.T) {
	frames := 9000
	if testing.Short() {
		frames = 600
	}
	for n := 1; n <= 7; n++ {
		t.Run(fmt.Sprintf("stage%d", n), func(t *testing.T) {
			if !stdConvertible[n] {
				t.Skip("legacy background")
			}
			w := &anm.World{}
			env := &anm.Env{GameSpeed: 1, RNG: &zun.RNG{}, Alt: &zun.RNG{}, Spawn: w.Add}
			v := loadSTD(n, env)
			if v == nil || v.Active == nil {
				t.Skip("no embedded STD or its ANM")
			}
			d := &supervisortest.Rec{}
			sv := supervisor.New(d, nil)
			bg := &stddraw.Drawer{STD: v}
			var quads int32
			tl := 0
			for f := 0; f < frames; f++ {
				v.Update(1)
				w.Run(env)
				sv.BeginFrame()
				bg.Node7(sv)
				bg.Node0A(sv, func(l int) { sv.ANM.DrawLayer(w, l) })
				sv.ANM.Flush()
				quads += bg.Quads
				for _, c := range d.Take() {
					if strings.HasPrefix(c, "draw ") {
						tl++
					}
				}
			}
			t.Logf("%d STD quads, %d DrawTL calls", quads, tl)
			if quads == 0 {
				t.Errorf("no quads drawn in %d frames", frames)
			}
			if (n == 1 || n == 5) && tl == 0 {
				t.Errorf("no DrawTL in %d frames: the mode 6 quads drew nothing", frames)
			}
		})
	}
}
