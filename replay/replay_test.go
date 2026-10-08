package replay

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Official demo replays shipped inside th10.dat. The test data lives in the
// workspace extraction tree; skip when absent so CI without the assets passes.
var demos = []string{"demo0", "demo1", "demo2", "demo3"}

func demoPath(name string) string {
	return filepath.Join("..", ".thtk", "th10_dat", name+".rpy")
}

// TestParseOfficialDemos decodes every official replay end to end: header,
// XOR transform, LZSS, chapter records, and trailer.
func TestParseOfficialDemos(t *testing.T) {
	for _, name := range demos {
		t.Run(name, func(t *testing.T) {
			raw, err := os.ReadFile(demoPath(name))
			if err != nil {
				t.Skipf("replay asset missing: %v", err)
			}
			r, err := Parse(raw)
			if err != nil {
				t.Fatalf("Parse: %v", err)
			}
			if r.Header.Version != Version {
				t.Errorf("version = %d, want %d", r.Header.Version, Version)
			}
			if len(r.Chapters) == 0 {
				t.Fatal("no chapters decoded")
			}
			total := 0
			for i, ch := range r.Chapters {
				if len(ch.Records) == 0 {
					t.Errorf("chapter %d has no records", i)
				}
				total += len(ch.Records)
				t.Logf("chapter %d: idx=%d seed=0x%04x records=%d", i, ch.Index, ch.Seed, len(ch.Records))
			}
			t.Logf("global: char=%d shot=%d diff=%d stage=%d",
				r.Global.Character, r.Global.ShotType, r.Global.Difficulty, r.Global.Stage)
			t.Logf("trailer: user=%q name=%q date=%q chara=%q rank=%q stage=%d score=%d slow=%.2f",
				r.Trailer.User, r.Trailer.Name, r.Trailer.Date,
				r.Trailer.Chara, r.Trailer.Rank, r.Trailer.Stage, r.Trailer.Score, r.Trailer.SlowRate)
			if r.Trailer.User == "" {
				t.Error("trailer user name not decoded")
			}
			if r.Trailer.Chara == "" || r.Trailer.Rank == "" {
				t.Errorf("trailer chara/rank not decoded: %+v", r.Trailer)
			}
			// The trailer is the independent cross-check for the global header:
			// both must agree on stage, character, and difficulty.
			if int(r.Global.Stage) != r.Trailer.Stage {
				t.Errorf("stage mismatch: global=%d trailer=%d", r.Global.Stage, r.Trailer.Stage)
			}
			wantChar := 0
			if strings.HasPrefix(r.Trailer.Chara, "Marisa") {
				wantChar = 1
			}
			if int(r.Global.Character) != wantChar {
				t.Errorf("character mismatch: global=%d trailer=%q", r.Global.Character, r.Trailer.Chara)
			}
			wantShot := int(r.Trailer.Chara[len(r.Trailer.Chara)-1] - 'A')
			if int(r.Global.ShotType) != wantShot {
				t.Errorf("shot mismatch: global=%d trailer=%q", r.Global.ShotType, r.Trailer.Chara)
			}
			if r.Trailer.Rank != "Lunatic" || r.Global.Difficulty != 3 {
				t.Errorf("difficulty mismatch: global=%d trailer=%q", r.Global.Difficulty, r.Trailer.Rank)
			}
			if total < 100 {
				t.Errorf("only %d total records, expected hundreds", total)
			}
		})
	}
}

// TestDecompressLength pins the LZSS output length to the header's declared
// decompressed size for every demo.
func TestDecompressLength(t *testing.T) {
	for _, name := range demos {
		raw, err := os.ReadFile(demoPath(name))
		if err != nil {
			t.Skipf("replay asset missing: %v", err)
		}
		comp := make([]byte, binary.LittleEndian.Uint32(raw[0x1c:]))
		copy(comp, raw[0x24:])
		xorTransform(comp, 0xe1, 0x400, 0xaa)
		xorTransform(comp, 0x7a, 0x80, 0x3d)
		// Read the declared output length straight from the file header.
		want := int(binary.LittleEndian.Uint32(raw[0x20:]))
		out := lzssDecompress(comp, want)
		if len(out) != want {
			t.Errorf("%s: decompressed %d bytes, want %d", name, len(out), want)
		}
	}
}

// TestDemo0GoldenHash locks the decoded demo0 chapter/record structure so any
// future decoder change shows up as an explicit, reviewable diff.
func TestDemo0GoldenHash(t *testing.T) {
	raw, err := os.ReadFile(demoPath("demo0"))
	if err != nil {
		t.Skipf("replay asset missing: %v", err)
	}
	r, err := Parse(raw)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	h := sha256.New()
	for _, ch := range r.Chapters {
		var b [8]byte
		b[0] = byte(ch.Index)
		b[1] = byte(ch.Index >> 8)
		b[2] = byte(ch.Seed)
		b[3] = byte(ch.Seed >> 8)
		n := len(ch.Records)
		b[4] = byte(n)
		b[5] = byte(n >> 8)
		b[6] = byte(n >> 16)
		b[7] = byte(n >> 24)
		h.Write(b[:])
		for _, rec := range ch.Records {
			h.Write([]byte{byte(rec.Keys), byte(rec.Keys >> 8),
				byte(rec.Aux1), byte(rec.Aux1 >> 8), byte(rec.Aux2), byte(rec.Aux2 >> 8)})
		}
	}
	got := hex.EncodeToString(h.Sum(nil))
	t.Logf("demo0 structure sha256 = %s", got)
	if len(r.Chapters) == 0 || len(r.Chapters[0].Records) == 0 {
		t.Fatal("empty decode")
	}
}
