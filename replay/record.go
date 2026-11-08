package replay

import (
	"errors"
	"slices"
)

// Recorder is the recording side of DAT_00477838 (mode +0x10 = 0).
// FUN_0042aa50 uses 3600-record chunks; contiguous slices produce the same
// writer order without the original's allocation pointers.
type Recorder struct {
	replay  Replay
	current int
}

// NewRecorder is the mode-0 initialization of FUN_00428f60. Its caller
// captures the global configuration before the stage's objects run.
func NewRecorder(global GlobalHeader) *Recorder {
	global.Raw = slices.Clone(global.Raw)
	global.Chapters = 0
	return &Recorder{replay: Replay{Header: Header{Version: Version}, Global: global}, current: -1}
}

// BeginChapter is FUN_0042a450's recording branch (the earlier research
// called it 0x42a460, which is inside the instruction at 0x42a45f). It
// resets that chapter's records and the +0x1c8 frame/sample cursor.
func (r *Recorder) BeginChapter(ch Chapter) error {
	if ch.Index < 1 || ch.Index > 7 {
		return errors.New("replay: chapter index must be 1..7")
	}
	ch.Raw = slices.Clone(ch.Raw)
	ch.FrameCount = 0
	ch.Records, ch.SlowRates, ch.Padding = nil, nil, nil
	for i := range r.replay.Chapters {
		if r.replay.Chapters[i].Index == ch.Index {
			r.replay.Chapters[i], r.current = ch, i
			return nil
		}
	}
	r.replay.Chapters = append(r.replay.Chapters, ch)
	r.current = len(r.replay.Chapters) - 1
	r.replay.Global.Chapters = uint32(len(r.replay.Chapters))
	return nil
}

// Append is FUN_004297d0 -> FUN_0042a8a0. The slow-rate byte is appended
// before record 0, 30, 60, ...; aux words are press and release edges, not
// menu keycodes. It does not mask keys: that happens in the input producer.
func (r *Recorder) Append(rec Record, slowRate byte) {
	if r == nil || r.current < 0 {
		return
	}
	ch := &r.replay.Chapters[r.current]
	if len(ch.Records)%slowRateStride == 0 {
		ch.SlowRates = append(ch.SlowRates, slowRate)
	}
	ch.Records = append(ch.Records, rec)
}

// Replay exposes the recorded state for inspection. Callers must not alter
// it while frames are being appended.
func (r *Recorder) Replay() *Replay { return &r.replay }

// PrepareSave is FUN_00429b60's count accumulation. Preserve its failure to
// clear +0x04: saving a second time doubles every chapter's playback limit.
// Encode itself stays pure so Decode -> Encode can be byte-exact.
func (r *Recorder) PrepareSave() *Replay {
	for i := range r.replay.Chapters {
		ch := &r.replay.Chapters[i]
		ch.FrameCount += uint32(len(ch.Records))
	}
	r.replay.Global.Chapters = uint32(len(r.replay.Chapters))
	return &r.replay
}

// Stream is FUN_004297d0's shared per-chapter record/sample cursor. Both the
// headless simulator and the in-game playback use it; it never polls input.
type Stream struct {
	chapter Chapter
	at      int
}

func NewStream(ch Chapter) *Stream { return &Stream{chapter: ch} }

// Prime reconstructs the held keys before the first frame from its press
// edge. The original retains these keys over a chapter boundary.
func (s *Stream) Prime() Record {
	if len(s.chapter.Records) == 0 {
		return Record{}
	}
	r := s.chapter.Records[0]
	return Record{Keys: r.Keys &^ r.Aux1}
}

func (s *Stream) Next() (Record, byte, bool) {
	if s.at >= len(s.chapter.Records) {
		return Record{}, 0, false
	}
	rate := byte(0xff)
	if sample := s.at / slowRateStride; sample < len(s.chapter.SlowRates) {
		rate = s.chapter.SlowRates[sample]
	}
	rec := s.chapter.Records[s.at]
	s.at++
	return rec, rate, true
}

func (s *Stream) Position() int { return s.at }
func (s *Stream) Done() bool    { return s.at >= len(s.chapter.Records) }
