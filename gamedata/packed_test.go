package gamedata

import (
	"bytes"
	"encoding/binary"
	"sync"
	"testing"

	"th10/anm"
	"th10/assets"
)

func TestTextureIndexDoesNotDecodeANMs(t *testing.T) {
	previous := anmFiles
	anmFiles = &sync.Map{}
	defer func() { anmFiles = previous }()
	index := textureIndex()
	if index["background/stg3bg.png"] != "stage03.anm" {
		t.Fatal("generated texture index has no stage-3 owner")
	}
	count := 0
	anmFiles.Range(func(_, _ any) bool { count++; return true })
	if count != 0 {
		t.Fatal("building the texture index eagerly decoded ANMs")
	}
	if _, err := Image("anm/background/stg3bg.png"); err != nil {
		t.Fatal(err)
	}
	anmFiles.Range(func(key, _ any) bool {
		count++
		if key != "anm/stage03.anm" {
			t.Errorf("loaded unrelated ANM %v", key)
		}
		return true
	})
	if count != 1 {
		t.Fatalf("loaded %d ANMs for one image", count)
	}
}

func TestANMReadsKeepPrivateMutableData(t *testing.T) {
	a, err := ReadFile("anm/enemy.anm")
	if err != nil {
		t.Fatal(err)
	}
	want := bytes.Clone(a)
	a[0] ^= 255
	b, err := ReadFile("anm/enemy.anm")
	if err != nil || !bytes.Equal(b, want) {
		t.Fatal("ReadFile exposed mutable cached ANM bytes")
	}
	x, err := ANM("enemy.anm")
	if err != nil {
		t.Fatal(err)
	}
	y, err := ANM("enemy.anm")
	if err != nil {
		t.Fatal(err)
	}
	x.Entries[0].Texture.Pixels[0] ^= 255
	if bytes.Equal(x.Entries[0].Texture.Pixels, y.Entries[0].Texture.Pixels) {
		t.Fatal("ANM returned shared mutable pixels")
	}
}

func BenchmarkANMDecode(b *testing.B) {
	for _, name := range []string{"ascii.anm", "bullet.anm", "stage04.anm", "stgenm06.anm", "title.anm", "staff.ant"} {
		b.Run(name, func(b *testing.B) {
			data, err := assets.Assets.ReadFile("anm/" + name)
			if err != nil {
				b.Fatal(err)
			}
			b.SetBytes(int64(binary.LittleEndian.Uint32(data[4:])))
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				raw, err := anm.Restore(data, assets.ReadTexture)
				if err != nil {
					b.Fatal(err)
				}
				if _, err := anm.Load(raw); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}
