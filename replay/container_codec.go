package replay

import "fmt"

// CompressLZSS exposes FUN_004359b0's codec, shared by .rpy and scoreth10.dat.
func CompressLZSS(src []byte) []byte { return lzssCompress(src) }

// DecompressLZSS is FUN_00435dc0 with a host-side allocation bound. Both
// callers validate their own container length before reaching this helper.
func DecompressLZSS(src []byte, outLen int) ([]byte, error) {
	if outLen < 0 || outLen > 64<<20 {
		return nil, fmt.Errorf("replay: invalid LZSS output length %d", outLen)
	}
	out := lzssDecompress(src, outLen)
	if len(out) != outLen {
		return nil, fmt.Errorf("replay: short LZSS output %d, want %d", len(out), outLen)
	}
	return out, nil
}

// EncryptBlocks and DecryptBlocks are FUN_0044b220 / FUN_0044b0d0. The
// length limit passed by both native container writers is the buffer size.
func EncryptBlocks(data []byte, keyInc byte, modulus int, initKey byte) {
	if modulus < 2 || modulus&1 != 0 {
		panic("replay: invalid XOR block size")
	}
	xorEncrypt(data, keyInc, modulus, initKey)
}

func DecryptBlocks(data []byte, keyInc byte, modulus int, initKey byte) {
	if modulus < 2 || modulus&1 != 0 {
		panic("replay: invalid XOR block size")
	}
	xorTransform(data, keyInc, modulus, initKey)
}
