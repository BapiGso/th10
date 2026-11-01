package std

import (
	"cmp"
	"encoding/binary"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"

	"th10/datfs"
	"th10/gamedata"
)

// embedded returns the raw bytes of an embedded .std.
func embedded(t *testing.T, name string) []byte {
	t.Helper()
	data, err := gamedata.ReadFile(datfs.EmbedPath(name))
	if err != nil {
		t.Fatal(err)
	}
	return data
}

// parseDump reads thtk's text dump (std_dump in thstd.c) into the parser's
// types. The dump has neither an entry id nor a quad record size, so both
// stay zero.
func parseDump(t *testing.T, text string) *File {
	t.Helper()
	f := &File{}
	inQuad := false
	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(line)
		if line == "SCRIPT:" {
			break
		}
		switch line {
		case "ENTRY:":
			f.Entries = append(f.Entries, Entry{})
			inQuad = false
			continue
		case "QUAD:":
			e := &f.Entries[len(f.Entries)-1]
			e.Quads = append(e.Quads, Quad{})
			inQuad = true
			continue
		}
		key, val, ok := strings.Cut(line, ": ")
		if !ok {
			continue // blank line
		}
		if key == "FACE" {
			// thtk prints the face's unknown word first, then x y z. Faces
			// appear under the entry they belong to, so the index is the
			// last ENTRY seen.
			v := dumpFloats(t, val, 4)
			f.Faces = append(f.Faces, Face{
				Entry:   uint16(len(f.Entries) - 1),
				Unknown: uint16(v[0]),
				Pos:     [3]float32{v[1], v[2], v[3]},
			})
			inQuad = false
			continue
		}
		if key == "ANM" {
			f.ANM = val
			continue
		}
		if len(f.Entries) == 0 {
			continue // Std_unknown and other header lines
		}
		e := &f.Entries[len(f.Entries)-1]
		var q *Quad
		if inQuad {
			q = &e.Quads[len(e.Quads)-1]
		}
		switch key {
		case "Unknown":
			e.Unknown = dumpUint(t, val)
		case "Type":
			q.Type = dumpUint(t, val)
		case "Script_index":
			q.ScriptIndex = dumpUint(t, val)
		case "Padding":
			q.Padding = dumpUint(t, val)
		case "Position":
			v := dumpFloats(t, val, 3)
			if q != nil {
				q.Pos = [3]float32{v[0], v[1], v[2]}
			} else {
				e.Pos = [3]float32{v[0], v[1], v[2]}
			}
		case "Depth":
			e.Size[2] = dumpFloats(t, val, 1)[0]
		case "Width":
			if q != nil {
				q.Size[0] = dumpFloats(t, val, 1)[0]
			} else {
				e.Size[0] = dumpFloats(t, val, 1)[0]
			}
		case "Height":
			if q != nil {
				q.Size[1] = dumpFloats(t, val, 1)[0]
			} else {
				e.Size[1] = dumpFloats(t, val, 1)[0]
			}
		}
	}
	return f
}

// dumpFloats parses n numbers that thtk printed with %g. Parsing to float32
// gives the nearest float32 to the printed decimal.
func dumpFloats(t *testing.T, s string, n int) []float32 {
	t.Helper()
	fields := strings.Fields(s)
	if len(fields) != n {
		t.Fatalf("dump %q: want %d numbers", s, n)
	}
	out := make([]float32, n)
	for i, f := range fields {
		v, err := strconv.ParseFloat(f, 32)
		if err != nil {
			t.Fatalf("dump %q: %v", s, err)
		}
		out[i] = float32(v)
	}
	return out
}

func dumpUint(t *testing.T, s string) uint16 {
	t.Helper()
	v, err := strconv.ParseUint(s, 10, 16)
	if err != nil {
		t.Fatalf("dump %q: %v", s, err)
	}
	return uint16(v)
}

// same is exact float32 equality, sign of zero included, since the dump
// prints -0 when a value is negative zero.
func same(a, b float32) bool {
	return a == b && math.Signbit(float64(a)) == math.Signbit(float64(b))
}

func sameVec3(a, b [3]float32) bool {
	return same(a[0], b[0]) && same(a[1], b[1]) && same(a[2], b[2])
}

func sameVec2(a, b [2]float32) bool {
	return same(a[0], b[0]) && same(a[1], b[1])
}

// compareDump checks every entry, quad and face of got against the dump.
func compareDump(t *testing.T, got, want *File) {
	t.Helper()
	if got.ANM != want.ANM {
		t.Errorf("ANM %q, dump has %q", got.ANM, want.ANM)
	}
	if len(got.Entries) != len(want.Entries) {
		t.Fatalf("%d entries, dump has %d", len(got.Entries), len(want.Entries))
	}
	for i, w := range want.Entries {
		g := got.Entries[i]
		if int(g.ID) != i {
			t.Errorf("entry %d: id %d", i, g.ID)
		}
		if g.Unknown != w.Unknown || !sameVec3(g.Pos, w.Pos) || !sameVec3(g.Size, w.Size) {
			t.Errorf("entry %d: got unknown %d pos %v size %v, dump has %d %v %v",
				i, g.Unknown, g.Pos, g.Size, w.Unknown, w.Pos, w.Size)
		}
		if len(g.Quads) != len(w.Quads) {
			t.Errorf("entry %d: %d quads, dump has %d", i, len(g.Quads), len(w.Quads))
			continue
		}
		for j, wq := range w.Quads {
			gq := g.Quads[j]
			if gq.Type != wq.Type || gq.ScriptIndex != wq.ScriptIndex || gq.Padding != wq.Padding ||
				!sameVec3(gq.Pos, wq.Pos) || !sameVec2(gq.Size, wq.Size) || gq.RecSize != quadRecSize {
				t.Errorf("entry %d quad %d: got %+v, dump has type %d script %d padding %d pos %v size %v",
					i, j, gq, wq.Type, wq.ScriptIndex, wq.Padding, wq.Pos, wq.Size)
			}
		}
	}
	// The file lists faces in its own order, which interleaves entries. thtk
	// prints them grouped by entry, keeping file order within each group, so
	// that is the order to compare in.
	grouped := slices.Clone(got.Faces)
	slices.SortStableFunc(grouped, func(a, b Face) int { return cmp.Compare(a.Entry, b.Entry) })
	if len(grouped) != len(want.Faces) {
		t.Fatalf("%d faces, dump has %d", len(grouped), len(want.Faces))
	}
	for i, w := range want.Faces {
		g := grouped[i]
		if g.Entry != w.Entry || g.Unknown != w.Unknown || !sameVec3(g.Pos, w.Pos) {
			t.Errorf("face %d: got entry %d unknown %d pos %v, dump has entry %d unknown %d pos %v",
				i, g.Entry, g.Unknown, g.Pos, w.Entry, w.Unknown, w.Pos)
		}
	}
}

// TestThtkDump compares every stage's entries, quads and faces with thtk's
// dump in .tmp/thtk_probe. The file is local (gitignored), so this skips
// without it.
func TestThtkDump(t *testing.T) {
	dir := filepath.Join("..", ".tmp", "thtk_probe")
	if _, err := os.Stat(dir); err != nil {
		t.Skipf("no thtk dumps: %v", err)
	}
	var entries, quads, faces int
	for n := 1; n <= 7; n++ {
		name := fmt.Sprintf("stage%02d.std", n)
		t.Run(name, func(t *testing.T) {
			text, err := os.ReadFile(filepath.Join(dir, name+".txt"))
			if err != nil {
				t.Fatal(err)
			}
			want := parseDump(t, string(text))
			entries += len(want.Entries)
			for _, e := range want.Entries {
				quads += len(e.Quads)
			}
			faces += len(want.Faces)

			f, err := Load(embedded(t, name))
			if err != nil {
				t.Fatal(err)
			}
			compareDump(t, f, want)
		})
	}
	// The totals thtk reports for the seven stages.
	if entries != 35 || quads != 138 || faces != 429 {
		t.Errorf("dumps hold %d entries, %d quads, %d faces; want 35, 138, 429", entries, quads, faces)
	}
}

// TestStage01Values checks a few values by hand against stage01.std's dump.
func TestStage01Values(t *testing.T) {
	f, err := Load(embedded(t, "stage01.std"))
	if err != nil {
		t.Fatal(err)
	}
	if f.ANM != "stage01.anm" {
		t.Errorf("ANM %q", f.ANM)
	}
	if len(f.Entries) != 8 || len(f.Faces) != 56 {
		t.Fatalf("%d entries, %d faces; want 8, 56", len(f.Entries), len(f.Faces))
	}
	// The second face in the file is entry 3's, not entry 1's: the list is
	// not grouped by entry.
	if f.Faces[1].Entry != 3 {
		t.Errorf("face 1 is for entry %d, want 3", f.Faces[1].Entry)
	}
	// ENTRY 1: width 1024, two quads at x = -256 and 256, faces from y = 256 up.
	e := f.Entries[1]
	if e.Size != [3]float32{1024, 256, 0} {
		t.Errorf("entry 1 size %v", e.Size)
	}
	if want := (Quad{RecSize: 0x1c, Pos: [3]float32{256, 0, 0}, Size: [2]float32{512, 256}}); e.Quads[1] != want {
		t.Errorf("entry 1 quad 1 = %+v, want %+v", e.Quads[1], want)
	}
	if want := (Face{Entry: 1, Unknown: 256, Pos: [3]float32{0, 256, 0}}); f.Faces[0] != want {
		t.Errorf("face 0 = %+v, want %+v", f.Faces[0], want)
	}
	// ENTRY 2 sits at z = -64 and its quads run script 1.
	if e := f.Entries[2]; e.Unknown != 2 || e.Pos != [3]float32{0, 0, -64} || e.Quads[0].ScriptIndex != 1 {
		t.Errorf("entry 2 = %+v", e)
	}
}

// synthetic builds a small .std in thstd.c's layout. entries lists each
// entry's quads, and faces holds the entry index of each face. The script is
// four bytes.
func synthetic(entries [][]Quad, faces []uint16) []byte {
	var b []byte
	u16 := func(v uint16) { b = binary.LittleEndian.AppendUint16(b, v) }
	u32 := func(v uint32) { b = binary.LittleEndian.AppendUint32(b, v) }
	f32 := func(v float32) { u32(math.Float32bits(v)) }

	facesOff := headerSize + 4*len(entries)
	nQuads := 0
	for _, qs := range entries {
		facesOff += entryRecSize + quadRecSize*len(qs) + 4
		nQuads += len(qs)
	}
	scriptOff := facesOff + faceRecSize*len(faces) + 16

	u16(uint16(len(entries)))
	u16(uint16(nQuads))
	u32(uint32(facesOff))
	u32(uint32(scriptOff))
	u32(0)
	name := make([]byte, 128)
	copy(name, "stage99.anm")
	b = append(b, name...)

	off := headerSize + 4*len(entries)
	for _, qs := range entries {
		u32(uint32(off))
		off += entryRecSize + quadRecSize*len(qs) + 4
	}
	for i, qs := range entries {
		u16(uint16(i))
		u16(0)
		for _, v := range [3]float32{float32(i), 2, 3} {
			f32(v)
		}
		for _, v := range [3]float32{4, 5, 6} {
			f32(v)
		}
		for _, q := range qs {
			u16(q.Type)
			u16(quadRecSize)
			u16(q.ScriptIndex)
			u16(q.Padding)
			for _, v := range q.Pos {
				f32(v)
			}
			for _, v := range q.Size {
				f32(v)
			}
		}
		u32(quadEnd)
	}
	for i, id := range faces {
		u16(id)
		u16(256)
		f32(float32(i))
		f32(5)
		f32(6)
	}
	for range 4 {
		u32(0xffffffff)
	}
	return append(b, 1, 2, 3, 4)
}

func synthQuads() [][]Quad {
	q := func(script uint16, x float32) Quad {
		return Quad{ScriptIndex: script, Pos: [3]float32{x, 1, 2}, Size: [2]float32{3, 4}}
	}
	return [][]Quad{{q(7, 10)}, {}, {q(8, 20), q(9, 30)}}
}

func TestSynthetic(t *testing.T) {
	data := synthetic(synthQuads(), []uint16{2, 0})
	f, err := Load(data)
	if err != nil {
		t.Fatal(err)
	}
	if f.ANM != "stage99.anm" || len(f.Entries) != 3 || len(f.Faces) != 2 {
		t.Fatalf("ANM %q, %d entries, %d faces", f.ANM, len(f.Entries), len(f.Faces))
	}
	if n := len(f.Entries[1].Quads); n != 0 {
		t.Errorf("entry 1 has %d quads, want 0", n)
	}
	if got := f.Entries[2].Quads[1]; got.ScriptIndex != 9 || got.Pos[0] != 30 || got.RecSize != quadRecSize {
		t.Errorf("entry 2 quad 1 = %+v", got)
	}
	if want := (Face{Entry: 2, Unknown: 256, Pos: [3]float32{0, 5, 6}}); f.Faces[0] != want {
		t.Errorf("face 0 = %+v, want %+v", f.Faces[0], want)
	}
	if string(f.Script) != "\x01\x02\x03\x04" {
		t.Errorf("script %v", f.Script)
	}
}

// Every cut before the script must fail, not panic. The header's
// script_offset is patched to the cut so the parser gets as far as it can.
func TestTruncatedFails(t *testing.T) {
	for _, data := range [][]byte{embedded(t, "stage04.std"), synthetic(synthQuads(), []uint16{2, 0})} {
		script := int(binary.LittleEndian.Uint32(data[8:]))
		for n := headerSize; n < script; n++ {
			cut := append([]byte(nil), data[:n]...)
			binary.LittleEndian.PutUint32(cut[8:], uint32(n))
			if _, err := Load(cut); err == nil {
				t.Fatalf("%d of %d bytes: no error", n, script)
			}
		}
	}
}

func TestMalformed(t *testing.T) {
	entryOff := func(b []byte, i int) int { return int(binary.LittleEndian.Uint32(b[headerSize+4*i:])) }
	facesOff := func(b []byte) int { return int(binary.LittleEndian.Uint32(b[4:])) }
	put16 := func(b []byte, off int, v uint16) { binary.LittleEndian.PutUint16(b[off:], v) }
	put32 := func(b []byte, off int, v uint32) { binary.LittleEndian.PutUint32(b[off:], v) }

	cases := []struct {
		name  string
		patch func(b []byte)
		want  string // substring of the error
	}{
		{"quad count", func(b []byte) { put16(b, 2, 4) }, "header counts 4 quads"},
		{"quad record size", func(b []byte) { put16(b, entryOff(b, 0)+entryRecSize+2, 0x20) }, "record size 0x20"},
		{"quad terminator", func(b []byte) { put32(b, entryOff(b, 1)+entryRecSize, 0) }, "record size 0x0"},
		{"entry offset", func(b []byte) { put32(b, headerSize+4, 0xfffffff0) }, "entry header"},
		{"face entry", func(b []byte) { put16(b, facesOff(b), 3) }, "names entry 3"},
		{"face list open", func(b []byte) { put32(b, facesOff(b)+2*faceRecSize, 0) }, "face list"},
		{"face terminator", func(b []byte) { put32(b, 8, binary.LittleEndian.Uint32(b[8:])+4) }, "face terminator"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			data := synthetic(synthQuads(), []uint16{2, 0})
			c.patch(data)
			_, err := Load(data)
			if err == nil || !strings.Contains(err.Error(), c.want) {
				t.Errorf("error %v, want one containing %q", err, c.want)
			}
		})
	}
}
