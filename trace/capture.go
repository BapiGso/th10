package trace

import (
	"bufio"
	"errors"
	"fmt"
	"io"
)

// CaptureHeader is optional session metadata, never added by NewWriter.
// It records observed entry conditions rather than inventing replay hi-score.
type CaptureHeader struct {
	Chapter, Stage, Demo, Extension  int
	Character, Shot, Difficulty      int
	HiScore                          int64
	HiScoreContinues, Records, Limit int
}

const captureShape = "C _ chapter _ stage _ demo _ ext _ char _ shot _ diff _ hiscore _ hicont _ records _ limit _"
const endShape = "Q _ last _ status _"

// NewAtomicWriter serializes a whole frame before its one destination Write.
// File destinations roll back short/failed writes to the prior boundary.
// Generic streams cannot roll back and must be treated as incomplete on error.
func NewAtomicWriter(w io.Writer) *Writer {
	tw := &Writer{direct: w}
	tw.w = bufio.NewWriter(&tw.block)
	return tw
}

func (tw *Writer) commit(p []byte) error {
	type rollbackWriter interface {
		io.Writer
		io.Seeker
		Truncate(int64) error
	}
	f, rollback := tw.direct.(rollbackWriter)
	var offset int64
	if rollback {
		var err error
		if offset, err = f.Seek(0, io.SeekCurrent); err != nil {
			return err
		}
	}
	n, err := tw.direct.Write(p)
	if err == nil && n != len(p) {
		err = io.ErrShortWrite
	}
	if err != nil && rollback {
		truncateErr := f.Truncate(offset)
		_, seekErr := f.Seek(offset, io.SeekStart)
		err = errors.Join(err, truncateErr, seekErr)
	}
	if flush, ok := tw.direct.(interface{ Flush() error }); ok {
		err = errors.Join(err, flush.Flush())
	}
	return err
}

func (tw *Writer) metadata(p []byte) error {
	if tw.err != nil {
		return tw.err
	}
	if tw.direct != nil {
		tw.err = tw.commit(p)
	} else {
		_, tw.err = tw.w.Write(p)
	}
	return tw.err
}

func (tw *Writer) BeginCapture(h CaptureHeader) error {
	if tw.capture != nil || tw.frames != 0 || tw.ended {
		return fmt.Errorf("trace: capture header must precede frames")
	}
	line := fmt.Sprintf("C 1 chapter %d stage %d demo %d ext %d char %d shot %d diff %d hiscore %d hicont %d records %d limit %d\n",
		h.Chapter, h.Stage, h.Demo, h.Extension, h.Character, h.Shot, h.Difficulty, h.HiScore, h.HiScoreContinues, h.Records, h.Limit)
	if !validCaptureHeader(h) {
		return fmt.Errorf("trace: invalid capture header")
	}
	tw.capture = &h
	return tw.metadata([]byte(line))
}

func (tw *Writer) EndCapture(status string) error {
	if tw.capture == nil {
		return tw.Flush()
	}
	if tw.ended {
		return tw.Flush()
	}
	if !validEndStatus(status) {
		return fmt.Errorf("trace: invalid capture status %q", status)
	}
	tw.ended = true
	if err := tw.metadata([]byte(fmt.Sprintf("Q %d last %d status %s\n", tw.frames, tw.last, status))); err != nil {
		return err
	}
	return tw.Flush()
}

func (tw *Writer) Err() error              { return tw.err }
func (tw *Writer) Frames() int             { return tw.frames }
func (tw *Writer) LastFrame() int          { return tw.last }
func (tr *Reader) Capture() *CaptureHeader { return tr.capture }

func parseCaptureHeader(t []string) (*CaptureHeader, error) {
	h := &CaptureHeader{}
	var version int
	if err := parseShape(t, captureShape, &version, &h.Chapter, &h.Stage, &h.Demo, &h.Extension, &h.Character, &h.Shot, &h.Difficulty, &h.HiScore, &h.HiScoreContinues, &h.Records, &h.Limit); err != nil {
		return nil, err
	}
	if version != 1 || !validCaptureHeader(*h) {
		return nil, fmt.Errorf("trace: invalid capture header")
	}
	return h, nil
}

func validCaptureHeader(h CaptureHeader) bool {
	return h.Chapter >= 0 && h.Stage >= 1 && h.Stage <= 7 && h.Demo >= -1 && h.Demo <= 3 && h.Extension >= 0 && h.Extension <= ExtendedVersion &&
		h.Character >= 0 && h.Character <= 1 && h.Shot >= 0 && h.Shot <= 2 && h.Difficulty >= 0 && h.Difficulty <= 4 &&
		h.HiScore >= 0 && h.HiScore <= 9999999990 && h.HiScoreContinues >= 0 && h.HiScoreContinues <= 9 && h.Records >= 0 && h.Limit >= 0
}

func validateCaptureInput(f *Frame, h *CaptureHeader, last int) error {
	if f == nil || f.Index < 1 || f.Index <= last {
		return fmt.Errorf("trace: invalid capture frame index")
	}
	if h.Extension == 0 && f.Ext != nil {
		return fmt.Errorf("trace: X not declared in capture header")
	}
	if h.Extension != 0 && f.Ext == nil && f.Half == nil {
		return fmt.Errorf("trace: missing X in normal capture frame")
	}
	if f.Half == nil {
		if f.Enemies < 0 || f.Bullets < 0 || len(f.E) != min(f.Enemies, MaxEnemyRows) || len(f.B) != min(f.Bullets, MaxBulletRows) {
			return fmt.Errorf("trace: incomplete capture entity rows")
		}
		if f.Ext != nil && (f.Items != len(f.I) || f.Ext.Lasers != len(f.L)) {
			return fmt.Errorf("trace: incomplete extended entity rows")
		}
	}
	return nil
}

func readCommit(f *Frame, t []string) error {
	if f == nil || f.extCommitted {
		return fmt.Errorf("trace: unexpected Z commit")
	}
	var index, e, b, items, lasers, spell int
	if err := parseShape(t, commitShape, &index, &e, &b, &items, &lasers, &spell); err != nil {
		return err
	}
	s := 0
	if f.Ext != nil && f.Ext.Spell != nil {
		s = 1
	}
	if index != f.Index || e != len(f.E) || b != len(f.B) || items != len(f.I) || lasers != len(f.L) || spell != s {
		return fmt.Errorf("trace: frame %d: invalid Z counts", f.Index)
	}
	f.extCommitted = true
	return nil
}

func (tr *Reader) validateFrame(f *Frame) error {
	if err := validateExtended(f); err != nil {
		return err
	}
	if tr.capture == nil || f == nil {
		return nil
	}
	if !f.extCommitted {
		return fmt.Errorf("trace: frame %d: missing Z capture commit", f.Index)
	}
	return validateCaptureInput(f, tr.capture, 0)
}

func validEndStatus(s string) bool {
	switch s {
	case "complete", "interrupted", "timeout", "error", "incomplete":
		return true
	}
	return false
}

func (tr *Reader) readCaptureEnd(t []string) error {
	if tr.capture == nil {
		return fmt.Errorf("trace: Q footer without C header")
	}
	if err := tr.validateFrame(tr.held); err != nil {
		return err
	}
	var frames, last int
	var status string
	if err := parseShape(t, endShape, &frames, &last, &status); err != nil {
		return err
	}
	if !validEndStatus(status) {
		return fmt.Errorf("trace: malformed Q status")
	}
	if frames != tr.frames || last != tr.last {
		return fmt.Errorf("trace: capture footer counts do not match frames")
	}
	tr.endSeen = true
	if status != "complete" {
		tr.finalErr = fmt.Errorf("trace: capture ended with status %s", status)
	}
	if frames == 0 {
		tr.finalErr = fmt.Errorf("trace: capture contains no samples")
	}
	return nil
}
