package datfs

import "errors"

const (
	lzssDictSize = 0x2000
	lzssDictMask = lzssDictSize - 1
	lzssMinMatch = 3
)

var errLZSSShort = errors.New("datfs: lzss stream ended early")

// bitReader reads MSB-first bit fields. Reads past the end yield zero bits,
// as thtk's bitstream does, so a truncated stream decodes into the
// terminating (offset 0) token and is reported as short output.
type bitReader struct {
	src  []byte
	pos  int
	acc  uint32
	bits uint
}

func (b *bitReader) read(n uint) int {
	for n > b.bits {
		var c byte
		if b.pos < len(b.src) {
			c = b.src[b.pos]
		}
		b.pos++
		b.acc = b.acc<<8 | uint32(c)
		b.bits += 8
	}
	b.bits -= n
	return int(b.acc>>b.bits) & (1<<n - 1)
}

// unlzss decodes ZUN's LZSS: a 1 bit is followed by a literal byte, a 0 bit by
// a 13-bit dictionary offset and a 4-bit length (+3). Offset 0 ends the
// stream. The 8 KiB dictionary starts zeroed with the write head at 1.
//
// Port of th_unlzss (thtk/thlzss.c).
func unlzss(src []byte, size int) ([]byte, error) {
	out := make([]byte, 0, size+18)
	var dict [lzssDictSize]byte
	head := 1
	br := bitReader{src: src}
	for len(out) < size {
		if br.read(1) != 0 {
			c := byte(br.read(8))
			out = append(out, c)
			dict[head] = c
			head = (head + 1) & lzssDictMask
			continue
		}
		off := br.read(13)
		if off == 0 {
			break
		}
		n := br.read(4) + lzssMinMatch
		for i := 0; i < n; i++ {
			c := dict[(off+i)&lzssDictMask]
			out = append(out, c)
			dict[head] = c
			head = (head + 1) & lzssDictMask
		}
	}
	if len(out) < size {
		return nil, errLZSSShort
	}
	return out[:size], nil
}
