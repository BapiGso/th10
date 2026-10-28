package anm_test

import (
	"testing"

	"th10/anm"
	"th10/gamedata"
)

func loadEnemyAnm(t *testing.T) *anm.File {
	t.Helper()
	f, err := gamedata.ANM("enemy.anm")
	if err != nil {
		t.Fatalf("load enemy.anm: %v", err)
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

// TestEnemyTexture checks the THTX block decodes to the entry's picture.
func TestEnemyTexture(t *testing.T) {
	f := loadEnemyAnm(t)
	tex := f.Entries[0].Texture
	if tex == nil {
		t.Fatal("enemy.png carries no THTX pixels")
	}
	img := f.NamedImage("enemy/enemy.png")
	if img == nil || img.Bounds().Dx() != tex.W || img.Bounds().Dy() != tex.H {
		t.Fatalf("NamedImage bounds %v, THTX %dx%d", img.Bounds(), tex.W, tex.H)
	}
	opaque := 0
	for i := 3; i < len(img.Pix); i += 4 {
		if img.Pix[i] != 0 {
			opaque++
		}
	}
	if opaque == 0 {
		t.Fatal("texture is fully transparent")
	}
}
