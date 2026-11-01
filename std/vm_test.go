package std

import (
	"testing"

	"th10/datfs"
	"th10/gamedata"
)

func load(t *testing.T, name string) *File {
	t.Helper()
	data, err := gamedata.ReadFile(datfs.EmbedPath(name))
	if err != nil {
		t.Fatal(err)
	}
	f, err := Load(data)
	if err != nil {
		t.Fatal(err)
	}
	return f
}

// stage04.std at t=0: op 2 puts the camera at (-0, 0, -550), then op 3 moves
// it linearly to (0, 100, -450) over 200 frames.
func TestStage04Camera(t *testing.T) {
	v := New(load(t, "stage04.std"))
	v.Update(1)
	if want := [3]float32{0, 0, 50}; v.Cam.Delta != want {
		t.Errorf("frame 1 delta = %v, want %v", v.Cam.Delta, want)
	}
	if want := [3]float32{0, 0.5, -549.5}; v.Cam.Pos != want {
		t.Errorf("frame 1 pos = %v, want %v", v.Cam.Pos, want)
	}
	for range 31 {
		v.Update(1)
	}
	if v.Cam.Delta != ([3]float32{}) {
		t.Errorf("frame 32 delta = %v, want 0", v.Cam.Delta)
	}
	if v.Cam.Pos[1] != 16 {
		t.Errorf("frame 32 y = %v, want 16", v.Cam.Pos[1])
	}
}

// With the initial camera, a point on the line of sight lands mid-field.
func TestProjectCentre(t *testing.T) {
	v := New(load(t, "stage04.std"))
	c := v.Cam
	at := add3(c.Pos, c.Facing)
	if x, y := c.Project(at); abs(x) > 1e-3 || abs(y-224) > 1e-3 {
		t.Errorf("Project(eye+facing) = (%v, %v), want (0, 224)", x, y)
	}
}

func abs(v float32) float32 {
	if v < 0 {
		return -v
	}
	return v
}
