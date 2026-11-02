package stage

import (
	"fmt"
	"slices"
	"strings"
	"testing"

	"th10/anm"
	"th10/supervisor"
	"th10/supervisor/supervisortest"
	"th10/zun"
)

// TestChainTable checks drawChain against m4_render_re.md §1.3.
func TestChainTable(t *testing.T) {
	want := []struct {
		prio uint8
		fn   uint32
	}{
		{0x01, 0x420000}, {0x02, 0x41fef0}, {0x03, 0x42d2f0}, {0x06, 0x40ba80},
		{0x07, 0x403060}, {0x09, 0x4485f0}, {0x0a, 0x403070}, {0x0b, 0x448600},
		{0x0d, 0x448610}, {0x0e, 0x409230}, {0x0f, 0x448620}, {0x10, 0x4486a0},
		{0x11, 0x4486b0}, {0x12, 0x4486c0}, {0x13, 0x4486d0}, {0x14, 0x40d820},
		{0x15, 0x4486e0}, {0x16, 0x426510}, {0x17, 0x4486f0}, {0x18, 0x448700},
		{0x19, 0x41ba30}, {0x1a, 0x448710}, {0x1b, 0x41c4e0}, {0x1c, 0x448720},
		{0x1d, 0x4067a0}, {0x20, 0x40b060}, {0x21, 0x448730}, {0x22, 0x405850},
		{0x23, 0x42b9b0}, {0x24, 0x448740}, {0x25, 0x409270}, {0x26, 0x401520},
		{0x28, 0x4200c0}, {0x29, 0x448770}, {0x2a, 0x4487c0}, {0x2b, 0x415af0},
		{0x2c, 0x4198b0}, {0x2d, 0x448810}, {0x2e, 0x422aa0}, {0x30, 0x401510},
		{0x32, 0x4200d0},
	}
	if len(drawChain) != len(want) {
		t.Fatalf("%d nodes, want %d", len(drawChain), len(want))
	}
	for i, n := range drawChain {
		if n.prio != want[i].prio || n.fn != want[i].fn {
			t.Errorf("node %d: prio %#x fn %#x, want prio %#x fn %#x", i, n.prio, n.fn, want[i].prio, want[i].fn)
		}
	}
}

// TestChainDraws checks which nodes draw: those with stage content, ported
// or legacy (m4_render.md §2.2), and no others.
func TestChainDraws(t *testing.T) {
	drawing := []uint8{
		0x07, 0x09, 0x0a, 0x0b, 0x0d, // background and layers 0-2, 17, 18
		0x0f, 0x10, 0x11, 0x12, 0x13, // layers 3-7
		0x15, 0x16, 0x17, 0x18, 0x19, 0x1a, 0x1b, 0x1c, 0x1d,
		0x21, 0x24, 0x29, 0x2a, // layers 13-16
		0x2b, // GUI
		0x2e, // pause (a guess)
	}
	// Of those, the ones still drawing with Ebiten. Node 0x14 (0x40d820,
	// MOV EAX,1; RET) draws nothing: the enemies are world objects.
	legacy := []uint8{0x16, 0x2b, 0x2e}
	for _, n := range drawChain {
		if want := slices.Contains(drawing, n.prio); (n.draw != nil) != want {
			t.Errorf("node %#x (%#x): draws %v, want %v", n.prio, n.fn, n.draw != nil, want)
		}
		if want := slices.Contains(legacy, n.prio); n.legacy != want {
			t.Errorf("node %#x (%#x): legacy %v, want %v", n.prio, n.fn, n.legacy, want)
		}
	}
}

// TestLayersANM checks that the World layer nodes draw through anmdraw: a
// bomb (bullet.anm 400, whose mode 9 children make the ring) drawn by the
// chain's non-legacy nodes, but for the background, on a supervisor with
// no back buffer, which a legacy draw would need. The ring reaches the
// device as a triangle strip.
func TestLayersANM(t *testing.T) {
	w := &anm.World{}
	env := &anm.Env{GameSpeed: 1, RNG: &zun.RNG{}, Alt: &zun.RNG{}, Spawn: w.Add}
	w.Spawn(bulletFile(), 400, [3]float32{0, 300, 0}, env)
	s := &Stage{ctx: &Context{World: w}}
	d := &supervisortest.Rec{}
	sv := supervisor.New(d, nil)
	draws, strips, frames := 0, 0, 0
	for ; frames < 600 && w.Len() > 0; frames++ {
		w.Run(env)
		sv.BeginFrame()
		for _, n := range drawChain {
			if n.enter != nil {
				n.enter(sv)
			}
			if n.draw != nil && !n.legacy && !slices.Contains([]uint8{0x07, 0x19, 0x1b, 0x1d}, n.prio) { // the layer nodes
				n.draw(s, sv)
			}
		}
		sv.EndFrame()
		for _, c := range d.Take() {
			if strings.HasPrefix(c, "draw ") {
				draws++
			}
			if strings.HasPrefix(c, "draw 5 ") {
				strips++
			}
		}
	}
	t.Logf("%d frames, %d DrawTL, %d strips", frames, draws, strips)
	if strips == 0 {
		t.Errorf("no triangle strip in %d frames (%d DrawTL): the ring was not drawn", frames, draws)
	}
}

// TestChainEnter runs every node's enter on a recording supervisor and
// checks the device calls each makes and the camera it leaves in use.
func TestChainEnter(t *testing.T) {
	d := &supervisortest.Rec{}
	sv := supervisor.New(d, nil)
	sv.ClearColor = 0xff000000
	sv.Field.Shift = [2]float32{3, -2}
	sv.BeginFrame()
	d.Take()

	screen := "transform 2; transform 3; viewport 0 0 640 480 0 1"
	field := "transform 2; transform 3; viewport 32 16 384 448 0 1"
	calls := map[uint8]string{
		0x01: screen + "; clear 1 ff000000 1 []",
		0x0f: field, // the fog is already off
		0x29: screen,
		0x2a: screen,
		0x2d: screen,
	}
	for _, n := range drawChain {
		if n.enter != nil {
			n.enter(sv)
		}
		if got := strings.Join(d.Take(), "; "); got != calls[n.prio] {
			t.Errorf("node %#x: calls %q, want %q", n.prio, got, calls[n.prio])
		}
		cur, is2D := &sv.Screen, true
		if n.prio >= 0x0f && n.prio < 0x29 {
			cur, is2D = &sv.Field, false
		}
		if sv.Cur != cur || sv.Is2D != is2D {
			t.Errorf("node %#x: Cur %s, Is2D %v", n.prio, camName(sv, sv.Cur), sv.Is2D)
		}

		// The field is shaken from node 0xf until node 0x24.
		shift := [2]float32{}
		if n.prio >= 0x0f && n.prio < 0x24 {
			shift = [2]float32{3, -2}
		}
		if sv.ANM.Shift != shift {
			t.Errorf("node %#x: ANM.Shift %v, want %v", n.prio, sv.ANM.Shift, shift)
		}
	}
	if sv.Field.Shift != ([2]float32{}) {
		t.Errorf("Field.Shift %v after the chain", sv.Field.Shift)
	}
}

func camName(sv *supervisor.Supervisor, c *supervisor.Camera) string {
	switch c {
	case &sv.Screen:
		return "Screen"
	case &sv.Field:
		return "Field"
	}
	return fmt.Sprintf("%p", c)
}
