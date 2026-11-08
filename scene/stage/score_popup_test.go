package stage

import (
	"fmt"
	"math"
	"slices"
	"strings"
	"testing"

	"th10/anm"
	"th10/anm/anmdraw"
	"th10/gamedata"
	"th10/gfx"
	"th10/supervisor/supervisortest"
)

func TestScorePopupDigits(t *testing.T) {
	for _, tc := range []struct {
		value  int32
		digits []uint8
	}{
		{0, []uint8{0}}, {-1, []uint8{10}}, {-123, []uint8{10}},
		{1230, []uint8{0, 3, 2, 1}}, {math.MaxInt32, []uint8{7, 4, 6, 3, 8, 4, 7, 4, 1, 2}},
	} {
		p := NewScorePopups()
		pos := [3]float32{10, 20, 30}
		p.Spawn(pos, tc.value, 0xff123456)
		s := &p.Slots[0]
		if !s.Live || s.Pos != pos || s.Color != 0xff123456 || !slices.Equal(s.Digits[:s.Count], tc.digits) {
			t.Errorf("value %d: %+v, want digits %v", tc.value, s, tc.digits)
		}
	}
}

func TestScorePopupCursorAndLifetime(t *testing.T) {
	p := NewScorePopups()
	for i := range popupRing + 1 {
		p.Spawn([3]float32{float32(i), 100, 0}, int32(i), 0xffffffff)
	}
	if p.cursor != 1 || p.Slots[0].Pos[0] != popupRing {
		t.Fatalf("cursor %d slot 0 x %g, want wrap before slot 720", p.cursor, p.Slots[0].Pos[0])
	}
	for i := popupRing; i < popupSlots; i++ {
		if p.Slots[i].Live {
			t.Errorf("reserved slot %d is live", i)
		}
	}
	p = NewScorePopups()
	p.Spawn([3]float32{0, 100, 0}, 1, 0xffffffff)
	for range 60 {
		p.Calc(1)
	}
	if !p.Slots[0].Live || p.Slots[0].T.Cur != 60 || p.Slots[0].Pos[1] != 70 {
		t.Fatalf("age 60: %+v", p.Slots[0])
	}
	p.Calc(1)
	if p.Slots[0].Live || p.Slots[0].Pos[1] != 69.5 {
		t.Fatalf("age 61: %+v", p.Slots[0])
	}
	p.Spawn([3]float32{0, 100, 0}, 1, 0xffffffff)
	for range 4 {
		p.Calc(0.5)
	}
	if p.Slots[1].T.Cur != 2 || p.Slots[1].Pos[1] != 99 {
		t.Fatalf("half-speed popup: %+v", p.Slots[1])
	}
}

func TestScorePopupAlphaAndPalette(t *testing.T) {
	for _, tc := range []struct {
		x    float32
		want uint8
	}{{0, 80}, {32, 80}, {48, 133}, {64, 208}, {100, 208}} {
		if got := popupAlpha([3]float32{}, [2]float32{tc.x, 0}); got != tc.want {
			t.Errorf("distance %g alpha %d, want %d", tc.x, got, tc.want)
		}
	}
	for _, tc := range []struct {
		age  int32
		want int
	}{{0, 0xc7}, {51, 0xc7}, {52, 0xd2}, {55, 0xd2}, {56, 0xdc}, {60, 0xdc}} {
		if got := popupSprite(3, tc.age); got != tc.want {
			t.Errorf("age %d palette sprite %#x, want %#x", tc.age, got, tc.want)
		}
		if got := popupSprite(10, tc.age); got != 0xce {
			t.Errorf("negative-value glyph age %d sprite %#x, want 0xce", tc.age, got)
		}
	}
}

func TestScorePopupDrawSequence(t *testing.T) {
	p := NewScorePopups()
	f := &anm.File{}
	p.vm.File = f
	sh := &anmdraw.Sheet{Sprites: make([]anmdraw.Sprite, 235)}
	d := &guiRec{Rec: supervisortest.Rec{Names: map[*gfx.Texture]string{}}}
	for i := range sh.Sprites {
		tex := &gfx.Texture{W: 8, H: 8}
		d.Names[tex] = fmt.Sprint(i)
		sh.Sprites[i] = anmdraw.Sprite{Tex: tex, W: 8, H: 8, U1: 1, V1: 1}
	}
	r := anmdraw.NewRenderer(d, func(file *anm.File) *anmdraw.Sheet {
		if file != f {
			t.Fatal("popup VM used another file")
		}
		return sh
	})
	r.View = [4]uint32{32, 16, 384, 448}
	p.Spawn([3]float32{0, 100, 0}, 123, 0xff112233)
	// Spawn occurs after calc 0x0f. At age zero the first digit is -Inf and
	// culls, then -Inf+Inf makes NaN. The original's ordered cull comparisons
	// do not reject NaN, so the following two quads reach the device.
	p.Draw(r, [2]float32{0, 200})
	r.Flush()
	if len(d.vertices) != 2 {
		t.Fatalf("birth-frame quads %d, want two NaN quads", len(d.vertices))
	}
	for _, v := range d.vertices {
		if !math.IsNaN(float64(v[0].X)) {
			t.Fatal("birth-frame infinity was repaired rather than preserving the original arithmetic")
		}
	}
	d.Take()
	d.vertices = nil
	p.Calc(1)
	p.Draw(r, [2]float32{0, 200})
	r.Flush()
	var got []string
	for _, c := range d.Take() {
		if strings.HasPrefix(c, "tex ") {
			got = append(got, c)
		}
	}
	if !slices.Equal(got, []string{"tex 197", "tex 198", "tex 199"}) || len(d.vertices) != 3 {
		t.Fatalf("digit order %v, %d draws, want MSD first", got, len(d.vertices))
	}
	for _, v := range d.vertices {
		if v[0].Diffuse != 0xd0112233 {
			t.Fatalf("popup ink %#x, want distance alpha replacing Spawn alpha", v[0].Diffuse)
		}
	}
}

func TestASCIIPopupGlyphDimensions(t *testing.T) {
	f, err := gamedata.ANM("ascii.anm")
	if err != nil {
		t.Fatal(err)
	}
	base := f.Sprites[0xc4]
	for id := 0xc4; id <= 0xe2; id++ {
		sp, ok := f.Sprites[id]
		if !ok || sp.H != base.H {
			t.Errorf("popup sprite %#x height %g, want initial VM height %g", id, sp.H, base.H)
		}
	}
}
