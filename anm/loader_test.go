package anm

import (
	"os"
	"testing"
)

func loadEnemyAnm(t *testing.T) *File {
	t.Helper()
	data, err := os.ReadFile("../assets/anm/enemy.anm")
	if err != nil {
		t.Fatalf("read enemy.anm: %v", err)
	}
	f, err := Load(data)
	if err != nil {
		t.Fatalf("parse enemy.anm: %v", err)
	}
	return f
}

func TestLoaderParsesEnemyAnm(t *testing.T) {
	f := loadEnemyAnm(t)
	if len(f.Entries) != 1 {
		t.Fatalf("entries=%d want 1", len(f.Entries))
	}
	if got := f.Entries[0].Name; got != "enemy/enemy.png" {
		t.Fatalf("texture name=%q want enemy/enemy.png", got)
	}
	if len(f.Sprites) != 124 {
		t.Fatalf("sprites=%d want 124", len(f.Sprites))
	}
	if len(f.Scripts) != 63 {
		t.Fatalf("scripts=%d want 63", len(f.Scripts))
	}
	// sprite0 = {x:0, y:256, w:32, h:32} per the thanm dump.
	s0 := f.Sprites[0]
	if s0.X != 0 || s0.Y != 256 || s0.W != 32 || s0.H != 32 {
		t.Fatalf("sprite0=%+v want {0,256,32,32}", s0)
	}
}

// TestSpriteResolverScript0 verifies the clock-walked sprite timeline: script0
// is ins_3(sprite0..3) each held 5 frames, looping every 20 via ins_4.
func TestSpriteResolverScript0(t *testing.T) {
	f := loadEnemyAnm(t)
	sc := f.Scripts[0]
	if sc == nil {
		t.Fatal("script0 missing")
	}
	cases := []struct {
		age, want int
	}{
		{0, 0}, {4, 0}, {5, 1}, {6, 1}, {12, 2}, {16, 3}, {20, 0}, {26, 1},
	}
	for _, c := range cases {
		if got := sc.SpriteAt(c.age); got != c.want {
			t.Errorf("SpriteAt(%d)=%d want %d", c.age, got, c.want)
		}
	}
}
