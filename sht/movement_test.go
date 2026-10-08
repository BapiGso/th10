package sht

import (
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"math"
	"testing"
	"th10/assets"
)

func TestOriginalMovementHeaders(t *testing.T) {
	cases := []struct {
		name, hash     string
		fast, diagonal int
	}{
		{"pl00a", "1619341f5f7a86f0f620d676906bc215dd84f0382d98e8bebed11d45cd3a1b11", 450, 318},
		{"pl00b", "d01eeae2336f97e7ac037546d2e5f23318c05e62c40914db2153288994a2892a", 450, 318},
		{"pl00c", "f9ea0eab644885636884e8d2e1ef9231f46a2a5ded53d7f94afcf56c384e5c13", 450, 318},
		{"pl01a", "0e6c76430a70ebe8262fb4c065c88b53a28c2fb0d1d2e6b29c34c42a8a06d67e", 500, 353},
		{"pl01b", "8c312dcf9bccdb2d095f0a6b35839f583722f7b1054d2d1d4240167bd262c31a", 500, 353},
		{"pl01c", "bb91ba008f3e92f940b7db4c6b002b754f514b1dbcf9c6dd8f336e322c048da5", 500, 353},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			data, err := assets.Assets.ReadFile("sht/" + tc.name + ".sht")
			if err != nil {
				t.Fatal(err)
			}
			if got := fmt.Sprintf("%x", sha256.Sum256(data)); got != tc.hash {
				t.Fatalf("original asset changed: %s", got)
			}
			got, err := ParseMovement(data)
			if err != nil {
				t.Fatal(err)
			}
			want := Movement{Fast: tc.fast, Slow: 200, FastDiagonal: tc.diagonal, SlowDiagonal: 141}
			if got != want {
				t.Fatalf("movement = %+v, want %+v", got, want)
			}
		})
	}
}

func TestInvalidMovementHeaders(t *testing.T) {
	valid := make([]byte, 32)
	binary.LittleEndian.PutUint16(valid, 3)
	binary.LittleEndian.PutUint32(valid[16:], math.Float32bits(4.5))
	binary.LittleEndian.PutUint32(valid[20:], math.Float32bits(2))
	for n := 0; n < 32; n++ {
		if _, err := ParseMovement(valid[:n]); err == nil {
			t.Fatalf("accepted %d bytes", n)
		}
	}
	badVersion := append([]byte(nil), valid...)
	badVersion[0] = 2
	if _, err := ParseMovement(badVersion); err == nil {
		t.Fatal("accepted version 2")
	}
	for _, off := range []int{16, 20} {
		for _, value := range []float32{0, -1, 101, float32(math.NaN()), float32(math.Inf(1))} {
			data := append([]byte(nil), valid...)
			binary.LittleEndian.PutUint32(data[off:], math.Float32bits(value))
			if _, err := ParseMovement(data); err == nil {
				t.Fatalf("accepted speed %v at %x", value, off)
			}
		}
	}
}
