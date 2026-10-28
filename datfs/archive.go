// Package datfs reads the original game's archives in place: th10.dat (THA1)
// as an fs.FS, and the thbgm.dat/thbgm.fmt pair as PCM spans with loop
// points. It is a development-time source: tools/th10extract copies files
// from it into assets/, and tests check the embedded copies against it.
//
// THA1 layout (port of thtk/thdat95.c):
//
//	0x00 header (16 B, encrypted with key 0x1b step 0x37 block 16):
//	     "THA1", u32 listSize+123456789, u32 listZSize+987654321,
//	     u32 count+135792468
//	...  file bodies, back to back, each encrypted with a key picked by name
//	     and LZSS-compressed unless zsize == size
//	EOF-listZSize  file list (encrypted 0x3e/0x9b/0x80, then LZSS):
//	     per file: NUL-padded name (to the next multiple of 4, at least one
//	     NUL), u32 offset, u32 size, u32 zero
package datfs

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"sort"
	"strings"
	"time"
)

const (
	headerSize = 16

	sizeBias  = 123456789
	zsizeBias = 987654321
	countBias = 135792468
)

type entry struct {
	name   string
	offset int64
	size   int64
	zsize  int64
}

// Archive is an open THA1 archive. It is safe for concurrent use as long as
// the underlying io.ReaderAt is (an *os.File is).
type Archive struct {
	r       io.ReaderAt
	closer  io.Closer
	entries []entry
	byName  map[string]int // lower-cased name -> index
	modTime time.Time
}

// Open opens a THA1 archive file such as th10.dat or th10c.dat.
func Open(name string) (*Archive, error) {
	f, err := os.Open(name)
	if err != nil {
		return nil, err
	}
	st, err := f.Stat()
	if err != nil {
		f.Close()
		return nil, err
	}
	a, err := New(f, st.Size())
	if err != nil {
		f.Close()
		return nil, fmt.Errorf("%s: %w", name, err)
	}
	a.closer = f
	a.modTime = st.ModTime()
	return a, nil
}

// New reads the file list of a THA1 archive of the given total size.
func New(r io.ReaderAt, size int64) (*Archive, error) {
	if size < headerSize {
		return nil, errors.New("datfs: file too small for a THA1 header")
	}
	hdr := make([]byte, headerSize)
	if _, err := r.ReadAt(hdr, 0); err != nil {
		return nil, err
	}
	decrypt(hdr, 0x1b, 0x37, headerSize, headerSize)
	if string(hdr[:4]) != "THA1" {
		return nil, errors.New("datfs: not a THA1 archive")
	}
	listSize := int64(binary.LittleEndian.Uint32(hdr[4:]) - sizeBias)
	listZSize := int64(binary.LittleEndian.Uint32(hdr[8:]) - zsizeBias)
	count := int(binary.LittleEndian.Uint32(hdr[12:]) - countBias)
	if listZSize <= 0 || listZSize > size-headerSize || listSize <= 0 || listSize > 1<<24 || count <= 0 || count > 1<<16 {
		return nil, errors.New("datfs: corrupt THA1 header")
	}

	zlist := make([]byte, listZSize)
	if _, err := r.ReadAt(zlist, size-listZSize); err != nil {
		return nil, err
	}
	decrypt(zlist, 0x3e, 0x9b, 0x80, int(listZSize))
	list, err := unlzss(zlist, int(listSize))
	if err != nil {
		return nil, fmt.Errorf("datfs: file list: %w", err)
	}

	a := &Archive{r: r, entries: make([]entry, 0, count), byName: make(map[string]int, count)}
	p := 0
	for i := 0; i < count; i++ {
		n := bytes.IndexByte(list[p:], 0)
		if n < 0 {
			return nil, errors.New("datfs: corrupt file list")
		}
		name := string(list[p : p+n])
		p += n + (4 - n%4)
		if p+12 > len(list) {
			return nil, errors.New("datfs: corrupt file list")
		}
		e := entry{
			name:   name,
			offset: int64(binary.LittleEndian.Uint32(list[p:])),
			size:   int64(binary.LittleEndian.Uint32(list[p+4:])),
		}
		p += 12
		if i > 0 {
			prev := &a.entries[i-1]
			prev.zsize = e.offset - prev.offset
		}
		a.entries = append(a.entries, e)
		a.byName[strings.ToLower(name)] = i
	}
	last := &a.entries[count-1]
	last.zsize = size - listZSize - last.offset
	for _, e := range a.entries {
		if e.offset < headerSize || e.zsize < 0 || e.offset+e.zsize > size-listZSize {
			return nil, fmt.Errorf("datfs: entry %q out of bounds", e.name)
		}
	}
	return a, nil
}

// Close releases the underlying file if the archive was opened by Open.
func (a *Archive) Close() error {
	if a.closer == nil {
		return nil
	}
	return a.closer.Close()
}

// Names lists the archived file names in archive order.
func (a *Archive) Names() []string {
	names := make([]string, len(a.entries))
	for i, e := range a.entries {
		names[i] = e.name
	}
	return names
}

// Has reports whether the archive contains name (case-insensitive).
func (a *Archive) Has(name string) bool {
	_, ok := a.byName[strings.ToLower(name)]
	return ok
}

// ReadFile decrypts and decompresses one archived file. Names are flat and
// matched case-insensitively.
func (a *Archive) ReadFile(name string) ([]byte, error) {
	i, ok := a.lookup(name)
	if !ok {
		return nil, &fs.PathError{Op: "read", Path: name, Err: fs.ErrNotExist}
	}
	data, err := a.read(&a.entries[i])
	if err != nil {
		return nil, &fs.PathError{Op: "read", Path: name, Err: err}
	}
	return data, nil
}

func (a *Archive) lookup(name string) (int, bool) {
	if !fs.ValidPath(name) || strings.Contains(name, "/") {
		return 0, false
	}
	i, ok := a.byName[strings.ToLower(name)]
	return i, ok
}

func (a *Archive) read(e *entry) ([]byte, error) {
	z := make([]byte, e.zsize)
	if _, err := a.r.ReadAt(z, e.offset); err != nil {
		return nil, err
	}
	p := cryptParamsFor(e.name)
	decrypt(z, p.key, p.step, p.block, p.limit)
	if e.zsize == e.size {
		return z, nil
	}
	return unlzss(z, int(e.size))
}

// Open implements fs.FS. The archive is a single flat directory ".".
func (a *Archive) Open(name string) (fs.File, error) {
	if name == "." {
		return &dirFile{a: a}, nil
	}
	i, ok := a.lookup(name)
	if !ok {
		return nil, &fs.PathError{Op: "open", Path: name, Err: fs.ErrNotExist}
	}
	e := &a.entries[i]
	data, err := a.read(e)
	if err != nil {
		return nil, &fs.PathError{Op: "open", Path: name, Err: err}
	}
	return &file{Reader: bytes.NewReader(data), info: fileInfo{name: e.name, size: e.size, mod: a.modTime}}, nil
}

// Stat implements fs.StatFS without decompressing.
func (a *Archive) Stat(name string) (fs.FileInfo, error) {
	if name == "." {
		return fileInfo{name: ".", dir: true, mod: a.modTime}, nil
	}
	i, ok := a.lookup(name)
	if !ok {
		return nil, &fs.PathError{Op: "stat", Path: name, Err: fs.ErrNotExist}
	}
	e := &a.entries[i]
	return fileInfo{name: e.name, size: e.size, mod: a.modTime}, nil
}

// ReadDir implements fs.ReadDirFS for the root directory.
func (a *Archive) ReadDir(name string) ([]fs.DirEntry, error) {
	if name != "." {
		if _, ok := a.lookup(name); ok {
			return nil, &fs.PathError{Op: "readdir", Path: name, Err: errors.New("not a directory")}
		}
		return nil, &fs.PathError{Op: "readdir", Path: name, Err: fs.ErrNotExist}
	}
	return a.dirEntries(), nil
}

func (a *Archive) dirEntries() []fs.DirEntry {
	out := make([]fs.DirEntry, len(a.entries))
	for i, e := range a.entries {
		out[i] = fs.FileInfoToDirEntry(fileInfo{name: e.name, size: e.size, mod: a.modTime})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name() < out[j].Name() })
	return out
}

type fileInfo struct {
	name string
	size int64
	mod  time.Time
	dir  bool
}

func (fi fileInfo) Name() string       { return path.Base(fi.name) }
func (fi fileInfo) Size() int64        { return fi.size }
func (fi fileInfo) ModTime() time.Time { return fi.mod }
func (fi fileInfo) IsDir() bool        { return fi.dir }
func (fi fileInfo) Sys() any           { return nil }
func (fi fileInfo) Mode() fs.FileMode {
	if fi.dir {
		return fs.ModeDir | 0o555
	}
	return 0o444
}

type file struct {
	*bytes.Reader
	info fileInfo
}

func (f *file) Stat() (fs.FileInfo, error) { return f.info, nil }
func (f *file) Close() error               { return nil }

type dirFile struct {
	a       *Archive
	entries []fs.DirEntry
	pos     int
}

func (d *dirFile) Stat() (fs.FileInfo, error) {
	return fileInfo{name: ".", dir: true, mod: d.a.modTime}, nil
}
func (d *dirFile) Read([]byte) (int, error) {
	return 0, &fs.PathError{Op: "read", Path: ".", Err: errors.New("is a directory")}
}
func (d *dirFile) Close() error { return nil }

func (d *dirFile) ReadDir(n int) ([]fs.DirEntry, error) {
	if d.entries == nil {
		d.entries = d.a.dirEntries()
	}
	rest := d.entries[d.pos:]
	if n <= 0 {
		d.pos = len(d.entries)
		return rest, nil
	}
	if len(rest) == 0 {
		return nil, io.EOF
	}
	n = min(n, len(rest))
	d.pos += n
	return rest[:n], nil
}
