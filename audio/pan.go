package audio

import (
	"encoding/binary"
	"errors"
	"io"
	"math"
)

// SEPan is FUN_0043dd10 (0x43dd10..0x43dd24): x * float32(0x40a6aaab),
// then FUN_00463b2c truncates toward zero. DirectSound SetPan receives this
// value in hundredths of a dB (0x43e2c7); x +/-192 produces pan +/-1000.
func SEPan(x float32) int32 {
	const scale float32 = 0x1.4d5556p+2 // 0x470b20, PE bits 0x40a6aaab
	return int32(float32(x * scale))
}

// sePanGains is DirectSound's attenuation-only pan law, not linear balance:
// positive pan attenuates the left channel, negative pan the right channel.
func sePanGains(pan int32) (left, right float64) {
	left, right = 1, 1
	if pan > 0 {
		left = math.Pow(10, -float64(pan)/2000)
	} else if pan < 0 {
		right = math.Pow(10, float64(pan)/2000)
	}
	return left, right
}

// panStream wraps 16-bit little-endian stereo PCM for Ebitengine on every
// target, including WASM. It applies the table volume and the pan together;
// the Player's volume remains the independently adjustable master SE gain.
// Buffer complete stereo frames so short/odd reads do not swap channels or
// split a sample's gain operation. The cached source PCM is never modified.
type panStream struct {
	src         io.ReadSeeker
	left, right float64
	pos         int64
	buf         [4096]byte
	start, end  int
	skip        int
	err         error
}

func newPanStream(src io.ReadSeeker, gain float64, pan int32) *panStream {
	l, r := sePanGains(pan)
	return &panStream{src: src, left: gain * l, right: gain * r}
}

func (s *panStream) Read(p []byte) (n int, err error) {
	for n < len(p) {
		if s.start < s.end {
			k := copy(p[n:], s.buf[s.start:s.end])
			s.start += k
			s.pos += int64(k)
			n += k
			continue
		}
		if s.err != nil {
			return n, s.err
		}
		count, readErr := io.ReadFull(s.src, s.buf[:])
		if readErr == io.ErrUnexpectedEOF && count%pcmBytesPerFrame == 0 {
			readErr = io.EOF
		}
		if count%pcmBytesPerFrame != 0 {
			count -= count % pcmBytesPerFrame
			readErr = io.ErrUnexpectedEOF
		}
		for i := 0; i < count; i += pcmBytesPerFrame {
			l := int16(binary.LittleEndian.Uint16(s.buf[i:]))
			r := int16(binary.LittleEndian.Uint16(s.buf[i+2:]))
			binary.LittleEndian.PutUint16(s.buf[i:], uint16(int16(float64(l)*s.left)))
			binary.LittleEndian.PutUint16(s.buf[i+2:], uint16(int16(float64(r)*s.right)))
		}
		s.start, s.end, s.err = min(s.skip, count), count, readErr
		s.skip = 0
	}
	return n, nil
}

func (s *panStream) Seek(offset int64, whence int) (int64, error) {
	target := offset
	switch whence {
	case io.SeekStart:
	case io.SeekCurrent:
		target += s.pos // the source can be ahead because of the frame buffer.
	case io.SeekEnd:
		var err error
		if target, err = s.src.Seek(offset, io.SeekEnd); err != nil {
			return s.pos, err
		}
	default:
		return s.pos, errors.New("audio: invalid seek whence")
	}
	if target < 0 {
		return s.pos, errors.New("audio: negative PCM position")
	}
	if _, err := s.src.Seek(target/pcmBytesPerFrame*pcmBytesPerFrame, io.SeekStart); err != nil {
		return s.pos, err
	}
	s.pos, s.skip = target, int(target%pcmBytesPerFrame)
	s.start, s.end, s.err = 0, 0, nil
	return target, nil
}
