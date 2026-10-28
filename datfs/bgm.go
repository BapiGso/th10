package datfs

import (
	"bytes"
	"encoding/binary"
	"errors"
	"io"
)

const bgmFmtRecordSize = 52

// BGMTrack is one record of thbgm.fmt: where a track's raw PCM lives inside
// thbgm.dat and where it loops. Playback runs [0, Size) once, then repeats
// [Intro, Size) forever.
//
//	0x00 char name[16]   e.g. "th10_02.wav"
//	0x10 u32  offset     byte offset of the PCM in thbgm.dat
//	0x14 u32  unknown    larger than size; not used by playback
//	0x18 u32  intro      bytes before the loop start
//	0x1c u32  size       PCM bytes stored (loop end)
//	0x20 WAVEFORMATEX    format tag, channels, rate, byte rate, align, bits, cbSize
//	0x32 2 bytes padding
type BGMTrack struct {
	Name          string
	Offset        int64
	Unknown       uint32
	Intro         int64
	Size          int64
	Channels      int
	SampleRate    int
	BitsPerSample int
}

// ParseBGMFmt decodes thbgm.fmt (stored inside th10.dat). Records are read in
// file order until a record with an empty name.
func ParseBGMFmt(data []byte) ([]BGMTrack, error) {
	var tracks []BGMTrack
	for p := 0; p+bgmFmtRecordSize <= len(data); p += bgmFmtRecordSize {
		rec := data[p : p+bgmFmtRecordSize]
		name, _, _ := bytes.Cut(rec[:16], []byte{0})
		if len(name) == 0 {
			break
		}
		u32 := func(off int) uint32 { return binary.LittleEndian.Uint32(rec[off:]) }
		u16 := func(off int) int { return int(binary.LittleEndian.Uint16(rec[off:])) }
		tracks = append(tracks, BGMTrack{
			Name:          string(name),
			Offset:        int64(u32(0x10)),
			Unknown:       u32(0x14),
			Intro:         int64(u32(0x18)),
			Size:          int64(u32(0x1c)),
			Channels:      u16(0x22),
			SampleRate:    int(u32(0x24)),
			BitsPerSample: u16(0x2e),
		})
	}
	if len(tracks) == 0 {
		return nil, errors.New("datfs: thbgm.fmt has no tracks")
	}
	return tracks, nil
}

// PCM returns a reader over the track's raw little-endian PCM in thbgm.dat
// (which starts with a 16-byte "ZWAV" header that the offsets already skip).
func (t BGMTrack) PCM(bgm io.ReaderAt) *io.SectionReader {
	return io.NewSectionReader(bgm, t.Offset, t.Size)
}
