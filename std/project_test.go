package std

import (
	"math"
	"testing"
)

// TestFlatMatrices checks FUN_00421480's matrices by what they do: a point
// on the plane z = 0 lands on its own pixel of the viewport (relative to
// the viewport's corner), and the eye distance and projection terms are
// the formula's.
func TestFlatMatrices(t *testing.T) {
	for _, c := range []struct{ w, h uint32 }{{640, 480}, {384, 448}} {
		view, proj := FlatMatrices(c.w, c.h)
		m := mul4(mat4(view), mat4(proj))
		fw, fh := float32(c.w), float32(c.h)
		for _, p := range [][2]float32{{0, 0}, {fw / 2, fh / 2}, {fw, fh}, {13.25, fh - 7.5}, {fw - 0.5, 1}} {
			v := transformCoord([3]float32{p[0], p[1], 0}, m)
			sx := float64(1+v[0]) * float64(fw) / 2
			sy := float64(1-v[1]) * float64(fh) / 2
			if math.Abs(sx-float64(p[0])) > 1e-3 || math.Abs(sy-float64(p[1])) > 1e-3 {
				t.Errorf("%dx%d: (%g, %g, 0) lands on (%g, %g)", c.w, c.h, p[0], p[1], sx, sy)
			}
			if v[2] <= 0 || v[2] >= 1 {
				t.Errorf("%dx%d: (%g, %g, 0) has depth %g, outside (0, 1)", c.w, c.h, p[0], p[1], v[2])
			}
		}
		// Nearer the eye is greater z.
		near := transformCoord([3]float32{fw / 2, fh / 2, 100}, m)
		far := transformCoord([3]float32{fw / 2, fh / 2, 0}, m)
		if !(near[2] < far[2]) {
			t.Errorf("%dx%d: depth at z = 100 is %g, not less than %g at z = 0", c.w, c.h, near[2], far[2])
		}
		d := float32(float64(float32(fh*0.5)) / math.Tan(0.15707963705062866))
		if view[3][2] != d {
			t.Errorf("%dx%d: eye distance %g, want %g", c.w, c.h, view[3][2], d)
		}
		if view[3][0] != -fw/2 || view[3][1] != fh/2 {
			t.Errorf("%dx%d: view translation (%g, %g), want (%g, %g)", c.w, c.h, view[3][0], view[3][1], -fw/2, fh/2)
		}
		if math.Abs(float64(proj[1][1])-1/math.Tan(math.Pi/20)) > 1e-5 {
			t.Errorf("%dx%d: proj[1][1] = %g, want 1/tan(π/20)", c.w, c.h, proj[1][1])
		}
		if proj[2][2] != float32(10000)/9999 || proj[2][3] != 1 {
			t.Errorf("%dx%d: proj z column (%g, %g)", c.w, c.h, proj[2][2], proj[2][3])
		}
	}
}

// TestLookMatrices checks FUN_004215a0's matrices against the slot-0 pair
// Project uses, the eye offset, and Right.
func TestLookMatrices(t *testing.T) {
	c := Camera{Pos: [3]float32{10, -20, 300}, Facing: [3]float32{0, 0.5, -1}, Up: [3]float32{0, 1, 0}, FOV: 0.7}
	view, proj := LookMatrices(c.Pos, c.Facing, c.Up, [3]float32{}, c.FOV, 384, 448)
	if v, p := c.Matrices(); v != view || p != proj {
		t.Errorf("LookMatrices(384, 448) differs from Camera.Matrices")
	}
	tan := float32(math.Tan(float64(c.FOV / 2)))
	if proj[2][2] != float32(1800)/1770 || proj[0][0] != 1/float32(float32(384)/float32(448)*tan) {
		t.Errorf("projection %v", proj)
	}

	// The eye offset moves the eye, not the point looked at.
	eye := [3]float32{0, 0, -50}
	view, _ = LookMatrices(c.Pos, c.Facing, c.Up, eye, c.FOV, 640, 480)
	want := lookAtLH([3]float32{10, -20, 250}, [3]float32{10, -19.5, 299}, c.Up)
	if mat4(view) != want {
		t.Errorf("eye offset: view %v, want %v", view, want)
	}

	// A zero Facing looks from Pos at Pos: no axes, only w.
	view, _ = LookMatrices([3]float32{0, 0, 1000}, [3]float32{}, [3]float32{0, 1, 0}, [3]float32{}, 0.5236, 640, 480)
	for i := range 3 {
		for j := range 3 {
			if view[i][j] != 0 {
				t.Fatalf("degenerate view %v", view)
			}
		}
	}

	if r := Right([3]float32{0, 0, 2}, [3]float32{0, 1, 0}); r != [3]float32{-1, 0, 0} {
		t.Errorf("Right(+z, +y) = %v, want (-1, 0, 0)", r)
	}
}
