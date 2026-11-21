package trace

import (
	"bufio"
	"bytes"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type writeSpy struct {
	bytes.Buffer
	calls int
}

func (s *writeSpy) Write(p []byte) (int, error) { s.calls++; return s.Buffer.Write(p) }

func TestAtomicWriterCommitsWholeFrameWithOneWrite(t *testing.T) {
	s := &writeSpy{}
	w := NewAtomicWriter(s)
	f := &Frame{Index: 1, Bullets: MaxBulletRows}
	for range MaxBulletRows {
		f.B = append(f.B, BulletRow{X: 10, Y: 20, Angle: .5, Speed: 3, Type: 6, Color: 2})
	}
	w.WriteFrame(f)
	if err := w.Flush(); err != nil {
		t.Fatal(err)
	}
	if s.calls != 1 || s.Len() < 4096 {
		t.Fatalf("not one complete large commit: calls=%d bytes=%d", s.calls, s.Len())
	}
	got, err := NewReader(bytes.NewReader(s.Bytes())).Next()
	if err != nil || got == nil || len(got.B) != MaxBulletRows {
		t.Fatalf("partial frame: frame=%v err=%v", got, err)
	}
}

func TestAtomicWriterFlushesExternalBufferEveryFrame(t *testing.T) {
	var dest bytes.Buffer
	b := bufio.NewWriter(&dest)
	w := NewAtomicWriter(b)
	w.WriteFrame(&Frame{Index: 1})
	if dest.Len() == 0 {
		t.Fatal("complete frame retained in user-space buffer")
	}
}

type partialFile struct {
	*os.File
	fail     bool
	nilError bool
}

func (f *partialFile) Write(p []byte) (int, error) {
	if !f.fail {
		return f.File.Write(p)
	}
	n, err := f.File.Write(p[:min(7, len(p))])
	if err != nil {
		return n, err
	}
	if f.nilError {
		return n, nil
	}
	return n, io.ErrClosedPipe
}

func TestAtomicFileWriteRollsBackPartialFrame(t *testing.T) {
	for _, nilError := range []bool{false, true} {
		f, err := os.Create(filepath.Join(t.TempDir(), "capture.txt"))
		if err != nil {
			t.Fatal(err)
		}
		defer func() { _ = f.Close() }()
		fault := &partialFile{File: f, nilError: nilError}
		w := NewAtomicWriter(fault)
		w.WriteFrame(&Frame{Index: 1})
		before, _ := f.Stat()
		fault.fail = true
		w.WriteFrame(&Frame{Index: 2})
		if w.Flush() == nil {
			t.Fatal("short/failed write was lost")
		}
		after, _ := f.Stat()
		if after.Size() != before.Size() || w.Frames() != 1 || w.LastFrame() != 1 {
			t.Fatalf("partial frame remained: before=%d after=%d frames=%d", before.Size(), after.Size(), w.Frames())
		}
		data, err := os.ReadFile(f.Name())
		if err != nil {
			t.Fatal(err)
		}
		if bytes.Contains(data, []byte("F 2")) || !bytes.HasSuffix(data, []byte("\n")) {
			t.Fatal("rollback left a torn row")
		}
	}
}

func TestFlushStillDrainsBeforeAnEarlierError(t *testing.T) {
	var b bytes.Buffer
	w := NewWriter(&b)
	w.WriteFrame(&Frame{Index: 1})
	sentinel := errors.New("earlier error")
	w.err = sentinel
	if !errors.Is(w.Flush(), sentinel) || b.Len() == 0 {
		t.Fatal("first error prevented draining previous complete data")
	}
}

func captureText(t *testing.T, ext bool, status string) string {
	t.Helper()
	var b bytes.Buffer
	w := NewAtomicWriter(&b)
	h := CaptureHeader{Stage: 1, Demo: -1, Records: 2}
	if ext {
		h.Extension = ExtendedVersion
	}
	if err := w.BeginCapture(h); err != nil {
		t.Fatal(err)
	}
	for i := 1; i <= 2; i++ {
		f := &Frame{Index: i}
		if ext {
			f.Ext = &Extended{}
		}
		w.WriteFrame(f)
	}
	if err := w.EndCapture(status); err != nil {
		t.Fatal(err)
	}
	return b.String()
}

func TestCaptureFooterReturnsTheEntireLastFrame(t *testing.T) {
	for _, ext := range []bool{false, true} {
		r := NewReader(strings.NewReader(captureText(t, ext, "complete")))
		for i := 1; i <= 2; i++ {
			f, err := r.Next()
			if err != nil || f == nil || f.Index != i {
				t.Fatalf("frame %d: frame=%v err=%v", i, f, err)
			}
		}
		if f, err := r.Next(); f != nil || err != nil {
			t.Fatalf("footer EOF: frame=%v err=%v", f, err)
		}
		if h := r.Capture(); h == nil || h.Stage != 1 || h.Records != 2 {
			t.Fatalf("missing entry metadata: %+v", h)
		}
	}
}

func TestCaptureReaderRejectsMissingWholeTailAndFooter(t *testing.T) {
	text := captureText(t, false, "complete")
	second := strings.Index(text, "F 2 ")
	footer := strings.Index(text, "Q ")
	for _, broken := range []string{text[:footer], text[:second] + text[footer:], strings.Replace(text, "Q 2 last 2", "Q 1 last 2", 1), strings.Replace(text, "Z 2 e 0 b 0 i 0 l 0 s 0\n", "", 1)} {
		r := NewReader(strings.NewReader(broken))
		gotError := false
		for range 4 {
			f, err := r.Next()
			if err != nil {
				gotError = true
				break
			}
			if f == nil {
				break
			}
		}
		if !gotError {
			t.Fatalf("accepted incomplete capture: %s", broken)
		}
	}
}

func TestCaptureInterruptedFooterPreservesCompletePrefix(t *testing.T) {
	r := NewReader(strings.NewReader(captureText(t, true, "interrupted")))
	for i := 1; i <= 2; i++ {
		f, err := r.Next()
		if f == nil || err != nil {
			t.Fatalf("complete prefix lost at %d: %v", i, err)
		}
	}
	if _, err := r.Next(); err == nil || !strings.Contains(err.Error(), "interrupted") {
		t.Fatalf("interrupted capture accepted: %v", err)
	}
}
