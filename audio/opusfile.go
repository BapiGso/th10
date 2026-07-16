package audio

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"

	"github.com/pion/opus"
	"github.com/pion/opus/pkg/oggreader"
)

// streamPCM 按需解码 Ogg/Opus 为 S16LE PCM，并提供 Ebiten 循环播放需要的 Seek。
type streamPCM struct {
	data         []byte
	channels     int
	totalBytes   int64
	preSkipBytes int64
	skipBytes    int64

	reader  *oggreader.OggReader
	decoder opus.Decoder

	frame []int16
	pcm   []byte
	pos   int64
}

func (s *streamPCM) Read(p []byte) (int, error) {
	total := 0
	for len(p) > 0 {
		if len(s.pcm) == 0 {
			if err := s.decodeNext(); err != nil {
				if err == io.EOF && total > 0 {
					return total, nil
				}
				return total, err
			}
			if len(s.pcm) == 0 {
				continue
			}
		}
		n := copy(p, s.pcm)
		s.pcm = s.pcm[n:]
		s.pos += int64(n)
		total += n
		p = p[n:]
	}
	return total, nil
}

func (s *streamPCM) Seek(offset int64, whence int) (int64, error) {
	var abs int64
	switch whence {
	case io.SeekStart:
		abs = offset
	case io.SeekCurrent:
		abs = s.pos + offset
	case io.SeekEnd:
		abs = s.totalBytes + offset
	default:
		return 0, fmt.Errorf("invalid whence %d", whence)
	}
	if abs < 0 {
		return 0, fmt.Errorf("negative seek position %d", abs)
	}
	if abs > s.totalBytes {
		abs = s.totalBytes
	}
	if abs == s.pos {
		return abs, nil
	}
	if err := s.reset(); err != nil {
		return 0, err
	}
	if err := s.discard(abs); err != nil {
		return 0, err
	}
	return abs, nil
}

// scanOggOpusTotalSamples 主线程快扫 Ogg 页头，返回 (channels, totalPCMBytes, preSkipBytes, err)。
// 不做任何解码，只读 page header 累计 granule。
func scanOggOpusTotalSamples(data []byte) (channels int, totalBytes int64, preSkipBytes int64, err error) {
	reader, header, err := oggreader.NewWith(bytes.NewReader(data))
	if err != nil {
		return 0, 0, 0, fmt.Errorf("open opus stream: %w", err)
	}
	if header.Channels != bgmChannels {
		return 0, 0, 0, fmt.Errorf("unsupported opus channel count: %d", header.Channels)
	}

	var lastGranule uint64
	for {
		_, pageHeader, err := reader.ParseNextPage()
		if err == io.EOF {
			break
		}
		if err != nil {
			return 0, 0, 0, fmt.Errorf("read opus page: %w", err)
		}
		if pageHeader.GranulePosition > 0 {
			lastGranule = pageHeader.GranulePosition
		}
	}
	if lastGranule == 0 {
		return 0, 0, 0, fmt.Errorf("opus stream did not contain audio packets")
	}

	preSkip := int64(header.PreSkip)
	validSamplesPerChannel := int64(lastGranule) - preSkip
	if validSamplesPerChannel < 0 {
		return 0, 0, 0, fmt.Errorf("invalid opus granule position")
	}
	ch := int64(header.Channels)
	totalBytes = validSamplesPerChannel * ch * pcmBytesPerSample
	preSkipBytes = preSkip * ch * pcmBytesPerSample
	return int(header.Channels), totalBytes, preSkipBytes, nil
}

// newOggOpusStream 返回按需解码的 ReadSeeker + 总 PCM 字节数。
func newOggOpusStream(data []byte) (*streamPCM, int64, error) {
	channels, totalBytes, preSkipBytes, err := scanOggOpusTotalSamples(data)
	if err != nil {
		return nil, 0, err
	}

	s := &streamPCM{
		data:         data,
		channels:     channels,
		totalBytes:   totalBytes,
		preSkipBytes: preSkipBytes,
		frame:        make([]int16, maxOpusSamplesPerChannel*channels),
	}
	if err := s.reset(); err != nil {
		return nil, 0, err
	}
	return s, totalBytes, nil
}

func (s *streamPCM) reset() error {
	reader, _, err := oggreader.NewWith(bytes.NewReader(s.data))
	if err != nil {
		return fmt.Errorf("open opus stream: %w", err)
	}
	decoder, err := opus.NewDecoderWithOutput(bgmPlaybackSampleRate, s.channels)
	if err != nil {
		return fmt.Errorf("create opus decoder: %w", err)
	}
	s.reader = reader
	s.decoder = decoder
	s.pcm = s.pcm[:0]
	s.pos = 0
	s.skipBytes = s.preSkipBytes
	return nil
}

func (s *streamPCM) decodeNext() error {
	for {
		packet, _, err := s.reader.ParseNextPacket()
		if err != nil {
			if err == io.EOF {
				return io.EOF
			}
			return fmt.Errorf("read opus packet: %w", err)
		}
		if len(packet) == 0 || bytes.HasPrefix(packet, []byte("OpusHead")) || bytes.HasPrefix(packet, []byte("OpusTags")) {
			continue
		}
		n, err := s.decoder.DecodeToInt16(packet, s.frame)
		if err != nil {
			return fmt.Errorf("decode opus packet: %w", err)
		}
		s.setPCM(s.frame[:n*s.channels])
		if len(s.pcm) > 0 {
			return nil
		}
	}
}

func (s *streamPCM) setPCM(samples []int16) {
	if cap(s.pcm) < len(samples)*pcmBytesPerSample {
		s.pcm = make([]byte, len(samples)*pcmBytesPerSample)
	}
	bs := s.pcm[:len(samples)*pcmBytesPerSample]
	for i, v := range samples {
		binary.LittleEndian.PutUint16(bs[i*pcmBytesPerSample:], uint16(v))
	}

	if s.skipBytes > 0 {
		drop := int64(len(bs))
		if drop > s.skipBytes {
			drop = s.skipBytes
		}
		bs = bs[drop:]
		s.skipBytes -= drop
	}
	remaining := int(s.totalBytes - s.pos)
	if remaining <= 0 {
		s.pcm = s.pcm[:0]
		return
	}
	if len(bs) > remaining {
		bs = bs[:remaining]
	}
	s.pcm = bs
}

func (s *streamPCM) discard(bytesToDiscard int64) error {
	buf := make([]byte, 32*1024)
	for bytesToDiscard > 0 {
		n := int64(len(buf))
		if n > bytesToDiscard {
			n = bytesToDiscard
		}
		read, err := s.Read(buf[:n])
		bytesToDiscard -= int64(read)
		if err != nil {
			if err == io.EOF && bytesToDiscard == 0 {
				return nil
			}
			return err
		}
		if read == 0 {
			return io.ErrUnexpectedEOF
		}
	}
	return nil
}
