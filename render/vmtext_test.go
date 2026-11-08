package render

import (
	"testing"

	"th10/anm"
	"th10/gamedata"
	"th10/zun"
)

func TestVMTextKeepsTimelineAndSpriteSize(t *testing.T) {
	f, err := gamedata.ANM("text.anm")
	if err != nil {
		t.Fatal(err)
	}
	f = f.OrdinalScripts()
	v := anm.New()
	env := &anm.Env{GameSpeed: 1, RNG: &zun.RNG{}, Alt: &zun.RNG{}}
	if !v.Start(f, 0, env) {
		t.Fatal("text.anm runtime script 0 did not start")
	}
	clock, pos, sprite := v.T, v.Pos, v.Sprite
	sp := f.Sprites[sprite]
	WriteVMText(v, "Test", TextStyle{Size: 16, Color: 0xf8f08f})
	if v.File == f || v.T != clock || v.Pos != pos || v.Sprite != sprite {
		t.Fatalf("text write changed VM timeline/position/sprite: %+v", v)
	}
	got := v.File.Sprites[sprite]
	if got.W != sp.W || got.H != sp.H || !v.Visible() {
		t.Fatalf("text sprite %+v, original %+v", got, sp)
	}
	img := v.File.Entries[0].Texture.Image()
	ink := false
	for i := 0; i < len(img.Pix); i += 4 {
		if img.Pix[i+3] > 0 && img.Pix[i] != 0 {
			ink = true
			break
		}
	}
	if !ink {
		t.Fatal("embedded font produced no visible text")
	}
	first := v.File
	WriteVMText(v, "Test", TextStyle{Size: 16, Color: 0xf8f08f})
	if v.File != first {
		t.Fatal("repeated embedded line allocated another sheet")
	}
	if f.Entries[0].Texture != nil {
		t.Fatal("text write mutated the original @ texture")
	}
}

func TestTextByteWidth(t *testing.T) {
	if textByteWidth("AB") != 2 || textByteWidth("日本") != 4 || textByteWidth("A日") != 3 {
		t.Fatal("texture alignment used UTF-8 bytes rather than original character width")
	}
}
