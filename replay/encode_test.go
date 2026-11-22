package replay

import (
	"bytes"
	"encoding/binary"
	"math/rand/v2"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"th10/datfs"
)

func originalReplayPaths() []string {
	var paths []string
	for i := 1; i <= 9; i++ {
		paths = append(paths, filepath.Join("..", ".tmp", "[th10]", "replay", "th10_0"+string(rune('0'+i))+".rpy"))
	}
	for _, demo := range demos {
		paths = append(paths, filepath.Join("..", ".tmp", "th10_dat_extract", demo+".rpy"))
	}
	return paths
}

func TestEncodeOriginalReplays(t *testing.T) {
	for _, path := range originalReplayPaths() {
		t.Run(filepath.Base(path), func(t *testing.T) {
			src, err := os.ReadFile(path)
			if err != nil {
				t.Skipf("original replay unavailable: %v", err)
			}
			r, err := Decode(src)
			if err != nil {
				t.Fatal(err)
			}
			got, err := Encode(r)
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(got, src) {
				at := 0
				for at < len(got) && at < len(src) && got[at] == src[at] {
					at++
				}
				t.Fatalf("first byte difference at %#x; lengths %d / %d", at, len(got), len(src))
			}
			roundtrip, err := Decode(got)
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(roundtrip, r) {
				t.Fatal("Decode(Encode(x)) differs from x")
			}
			t.Logf("%d bytes identical, %d chapters", len(src), len(r.Chapters))
		})
	}
}

func TestLZSSCompress(t *testing.T) {
	rng := rand.New(rand.NewPCG(1, 2))
	for _, size := range []int{0, 1, 2, 3, 17, 18, 19, 8191, 8192, 8193, 30000} {
		for kind := 0; kind < 3; kind++ {
			src := make([]byte, size)
			for i := range src {
				switch kind {
				case 1:
					src[i] = byte(i % 13)
				case 2:
					src[i] = byte(rng.Uint32())
				}
			}
			got := datfs.Unlzss(lzssCompress(src), len(src))
			if !bytes.Equal(got, src) {
				t.Fatalf("size %d kind %d: compressed roundtrip differs", size, kind)
			}
		}
	}
}

func TestXOREncryptInverse(t *testing.T) {
	for _, modulus := range []int{0x80, 0x400} {
		for n := 0; n < 3*modulus; n++ {
			src := make([]byte, n)
			for i := range src {
				src[i] = byte(i*19 + n)
			}
			got := bytes.Clone(src)
			xorEncrypt(got, 0x7a, modulus, 0x3d)
			DecryptBlocks(got, 0x7a, modulus, 0x3d)
			if !bytes.Equal(got, src) {
				t.Fatalf("modulus %#x size %d", modulus, n)
			}
		}
	}
}

func TestEncodeNewReplay(t *testing.T) {
	r := &Replay{
		Global:   GlobalHeader{User: "PLAYER", Timestamp: 1190992440, Score: 123456, Stage: 8},
		Chapters: []Chapter{{Index: 1, Seed: 0x4a4d, Faith: 5000, Lives: 2, Records: make([]Record, 31)}},
	}
	r.Chapters[0].Records[30] = Record{Keys: KeyRight | KeyShot, Aux1: KeyRight, Aux2: 3}
	data, err := Encode(r)
	if err != nil {
		t.Fatal(err)
	}
	got, err := Decode(data)
	if err != nil {
		t.Fatal(err)
	}
	if got.Global.User != "PLAYER" || got.Global.Score != 123456 || got.Trailer.Stage != 8 || got.Trailer.Version != "1.00a" {
		t.Fatalf("metadata: %+v / %+v", got.Global, got.Trailer)
	}
	if len(got.Chapters) != 1 || !reflect.DeepEqual(got.Chapters[0].Records, r.Chapters[0].Records) {
		t.Fatal("records changed")
	}
	if !bytes.Equal(got.Chapters[0].SlowRates, []byte{255, 255}) {
		t.Fatalf("slow-rate samples: %v", got.Chapters[0].SlowRates)
	}
	if binary.LittleEndian.Uint32(data[0x10:]) != 0x100 || got.Header.TrailerOff != 36+got.Header.Compressed {
		t.Fatal("writer header defaults/offset differ")
	}
	again, err := Encode(got)
	if err != nil || !bytes.Equal(again, data) {
		t.Fatal("new replay is not stable after decoding")
	}
}

func TestEncodeRejectsInvalid(t *testing.T) {
	for _, r := range []*Replay{nil, {Header: Header{Version: 6}}, {Global: GlobalHeader{User: "123456789"}}, {Global: GlobalHeader{Raw: []byte{1}}}, {Chapters: []Chapter{{Raw: []byte{1}}}}, {Chapters: []Chapter{{Records: make([]Record, 31), SlowRates: []byte{255}}}}} {
		if _, err := Encode(r); err == nil {
			t.Fatalf("accepted invalid replay: %+v", r)
		}
	}
}
