package datfs

import (
	"bytes"
	"encoding/binary"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"
)

// encrypt is the inverse of decrypt (th_encrypt), used to build test archives.
func encrypt(data []byte, key, step byte, block, limit int) {
	size := len(data)
	if size < block>>2 {
		size = 0
	} else {
		tail := 0
		if size%block < block>>2 {
			tail = size % block
		}
		size -= tail + size%2
	}
	if limit%block != 0 {
		limit += block - limit%block
	}
	end := min(size, limit)

	temp := make([]byte, block)
	inc := block>>1 + block&1
	for p := 0; p < end; p += block {
		if end-p < block {
			block = end - p
			inc = block>>1 + block&1
		}
		in := p + block - 1
		out := 0
		for in > p {
			temp[out] = data[in] ^ key
			in--
			temp[out+inc] = data[in] ^ (key + step*byte(inc))
			in--
			out++
			key += step
		}
		if block&1 != 0 {
			temp[out] = data[in] ^ key
			key += step
		}
		key += step * byte(inc)
		copy(data[p:p+block], temp[:block])
	}
}

func TestCryptRoundTrip(t *testing.T) {
	rng := rand.New(rand.NewSource(1))
	for _, p := range th95CryptParams {
		for _, n := range []int{0, 1, 7, 15, 16, 17, 63, 64, 65, 127, 1000, 0x2801, 0x8000} {
			orig := make([]byte, n)
			rng.Read(orig)
			buf := append([]byte(nil), orig...)
			encrypt(buf, p.key, p.step, p.block, p.limit)
			if n >= p.block && bytes.Equal(buf, orig) {
				t.Fatalf("params %+v n=%d: encrypt left data unchanged", p, n)
			}
			decrypt(buf, p.key, p.step, p.block, p.limit)
			if !bytes.Equal(buf, orig) {
				t.Fatalf("params %+v n=%d: round trip mismatch", p, n)
			}
		}
	}
}

// bitWriter packs MSB-first bit fields, mirroring bitReader.
type bitWriter struct {
	out  []byte
	acc  uint32
	bits uint
}

func (w *bitWriter) write(n uint, v int) {
	for i := int(n) - 1; i >= 0; i-- {
		w.acc = w.acc<<1 | uint32(v>>uint(i))&1
		w.bits++
		if w.bits == 8 {
			w.out = append(w.out, byte(w.acc))
			w.acc, w.bits = 0, 0
		}
	}
}

func (w *bitWriter) flush() []byte {
	for w.bits != 0 {
		w.write(1, 0)
	}
	return w.out
}

// lzssGreedy is a slow reference encoder for the decoder's format.
func lzssGreedy(src []byte) []byte {
	var w bitWriter
	for i := 0; i < len(src); {
		bestLen, bestOff := 0, 0
		for j := max(0, i-(lzssDictSize-32)); j < i; j++ {
			off := (j + 1) & lzssDictMask
			if off == 0 {
				continue
			}
			n := 0
			for n < 18 && i+n < len(src) && src[j+n] == src[i+n] {
				n++
			}
			if n > bestLen {
				bestLen, bestOff = n, off
			}
		}
		if bestLen >= lzssMinMatch {
			w.write(1, 0)
			w.write(13, bestOff)
			w.write(4, bestLen-lzssMinMatch)
			i += bestLen
		} else {
			w.write(1, 1)
			w.write(8, int(src[i]))
			i++
		}
	}
	w.write(1, 0)
	w.write(13, 0)
	return w.flush()
}

func TestUnLZSSRoundTrip(t *testing.T) {
	rng := rand.New(rand.NewSource(2))
	random := make([]byte, 3000)
	rng.Read(random)
	cases := [][]byte{
		nil,
		[]byte("a"),
		bytes.Repeat([]byte("abc"), 500),
		bytes.Repeat([]byte{0}, 100),
		random,
		append(bytes.Repeat([]byte("xyz12"), 2000), random...),
	}
	for i, c := range cases {
		z := lzssGreedy(c)
		got, err := unlzss(z, len(c))
		if err != nil {
			t.Fatalf("case %d: %v", i, err)
		}
		if !bytes.Equal(got, c) {
			t.Fatalf("case %d: mismatch", i)
		}
	}
	if _, err := unlzss(lzssGreedy([]byte("short")), 10); err == nil {
		t.Fatal("want error for stream shorter than size")
	}
}

// buildArchive writes a THA1 archive the way th95_write/th95_close do.
func buildArchive(files map[string][]byte, order []string) []byte {
	out := make([]byte, headerSize)
	var list []byte
	for _, name := range order {
		data := files[name]
		body := lzssGreedy(data)
		if len(body) >= len(data) {
			body = append([]byte(nil), data...)
		}
		p := cryptParamsFor(name)
		encrypt(body, p.key, p.step, p.block, p.limit)
		offset := len(out)
		out = append(out, body...)

		pad := 4 - len(name)%4
		list = append(list, name...)
		list = append(list, make([]byte, pad)...)
		list = binary.LittleEndian.AppendUint32(list, uint32(offset))
		list = binary.LittleEndian.AppendUint32(list, uint32(len(data)))
		list = binary.LittleEndian.AppendUint32(list, 0)
	}
	zlist := lzssGreedy(list)
	encrypt(zlist, 0x3e, 0x9b, 0x80, len(list))
	out = append(out, zlist...)

	hdr := out[:headerSize]
	copy(hdr, "THA1")
	binary.LittleEndian.PutUint32(hdr[4:], uint32(len(list))+sizeBias)
	binary.LittleEndian.PutUint32(hdr[8:], uint32(len(zlist))+zsizeBias)
	binary.LittleEndian.PutUint32(hdr[12:], uint32(len(order))+countBias)
	encrypt(hdr, 0x1b, 0x37, headerSize, headerSize)
	return out
}

func TestArchiveSynthetic(t *testing.T) {
	rng := rand.New(rand.NewSource(3))
	noise := make([]byte, 5000)
	rng.Read(noise)
	files := map[string][]byte{
		"stage01.ecl": bytes.Repeat([]byte("ECLH\x01\x00\x00\x00"), 900),
		"abc.msg":     noise,
		"x.anm":       []byte("tiny"),
		"Mixed.TXT":   append([]byte("header"), bytes.Repeat([]byte{7, 7, 7, 1}, 300)...),
	}
	order := []string{"stage01.ecl", "abc.msg", "x.anm", "Mixed.TXT"}
	raw := buildArchive(files, order)

	a, err := New(bytes.NewReader(raw), int64(len(raw)))
	if err != nil {
		t.Fatal(err)
	}
	if got := strings.Join(a.Names(), ","); got != strings.Join(order, ",") {
		t.Fatalf("names = %s", got)
	}
	for name, want := range files {
		got, err := a.ReadFile(name)
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		if !bytes.Equal(got, want) {
			t.Fatalf("%s: content mismatch", name)
		}
	}
	if got, err := a.ReadFile("mixed.txt"); err != nil || !bytes.Equal(got, files["Mixed.TXT"]) {
		t.Fatalf("case-insensitive lookup failed: %v", err)
	}
	if _, err := a.ReadFile("missing.ecl"); err == nil {
		t.Fatal("want error for missing file")
	}
	if err := fstest.TestFS(a, order...); err != nil {
		t.Fatal(err)
	}
}

// gameDir finds the user's original install for integration tests.
func gameDir(t *testing.T) string {
	t.Helper()
	dir := os.Getenv("TH10_DIR")
	if dir == "" {
		dir = filepath.Join("..", ".tmp", "[th10]")
	}
	if _, err := os.Stat(filepath.Join(dir, "th10.dat")); err != nil {
		t.Skip("th10.dat not found; set TH10_DIR to the original game directory")
	}
	return dir
}

func TestRealArchive(t *testing.T) {
	dir := gameDir(t)
	a, err := Open(filepath.Join(dir, "th10.dat"))
	if err != nil {
		t.Fatal(err)
	}
	defer a.Close()
	if len(a.Names()) < 100 {
		t.Fatalf("only %d entries", len(a.Names()))
	}
	for _, name := range []string{"stage01.ecl", "stage01.std", "pl00.anm", "st01_00.msg", "thbgm.fmt", "se_gun00.wav"} {
		if !a.Has(name) {
			t.Fatalf("missing %s", name)
		}
	}

	// Compare against an independent extraction when one is available.
	extract := filepath.Join("..", ".tmp", "th10_dat_extract")
	if _, err := os.Stat(extract); err == nil {
		for _, name := range a.Names() {
			want, err := os.ReadFile(filepath.Join(extract, name))
			if err != nil {
				continue
			}
			got, err := a.ReadFile(name)
			if err != nil {
				t.Fatalf("%s: %v", name, err)
			}
			if !bytes.Equal(got, want) {
				t.Fatalf("%s: differs from extracted copy", name)
			}
		}
	}

	fmtData, err := a.ReadFile("thbgm.fmt")
	if err != nil {
		t.Fatal(err)
	}
	tracks, err := ParseBGMFmt(fmtData)
	if err != nil {
		t.Fatal(err)
	}
	if len(tracks) != 18 {
		t.Fatalf("%d BGM tracks, want 18", len(tracks))
	}
	st, err := os.Stat(filepath.Join(dir, "thbgm.dat"))
	if err != nil {
		t.Skip("thbgm.dat not found")
	}
	for _, tr := range tracks {
		if tr.SampleRate != 44100 || tr.Channels != 2 || tr.BitsPerSample != 16 {
			t.Fatalf("%s: unexpected format %+v", tr.Name, tr)
		}
		if tr.Intro >= tr.Size || tr.Offset+tr.Size > st.Size() {
			t.Fatalf("%s: bad span %+v (thbgm.dat is %d bytes)", tr.Name, tr, st.Size())
		}
	}
}
