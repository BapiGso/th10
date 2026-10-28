package datfs

// decrypt undoes ZUN's block cipher in place: each block of up to `block`
// bytes is de-interleaved (second half reversed against first half) and XORed
// with a running key. Only the first `limit` bytes (rounded up to a block
// multiple) are encrypted, and a short tail block is left in plain text.
//
// Port of th_decrypt (thtk/thcrypt.c).
func decrypt(data []byte, key, step byte, block, limit int) {
	size := len(data)
	if size < block>>2 {
		size = 0
	} else {
		tail := 0
		if size%block < block>>2 {
			tail = size % block
		}
		size -= tail + size%2
	}
	if limit%block != 0 {
		limit += block - limit%block
	}
	end := min(size, limit)

	temp := make([]byte, block)
	inc := block>>1 + block&1
	for p := 0; p < end; p += block {
		if end-p < block {
			block = end - p
			inc = block>>1 + block&1
		}
		in := p
		out := block - 1
		for out > 0 {
			temp[out] = data[in] ^ key
			out--
			temp[out] = data[in+inc] ^ (key + step*byte(inc))
			out--
			in++
			key += step
		}
		if block&1 != 0 {
			temp[out] = data[in] ^ key
			key += step
		}
		key += step * byte(inc)
		copy(data[p:p+block], temp[:block])
	}
}

type cryptParams struct {
	key, step    byte
	block, limit int
}

// th95CryptParams is the per-file key table shared by TH09.5, TH10 and TH11
// (thtk/dattypes.h), selected by the low 3 bits of the file name's byte sum.
var th95CryptParams = [8]cryptParams{
	{0x1b, 0x37, 0x40, 0x2800},
	{0x51, 0xe9, 0x40, 0x3000},
	{0xc1, 0x51, 0x80, 0x3200},
	{0x03, 0x19, 0x400, 0x7800},
	{0xab, 0xcd, 0x200, 0x2800},
	{0x12, 0x34, 0x80, 0x3200},
	{0x35, 0x97, 0x80, 0x2800},
	{0x99, 0x37, 0x400, 0x2000},
}

func cryptParamsFor(name string) cryptParams {
	var sum byte
	for i := 0; i < len(name); i++ {
		sum += name[i]
	}
	return th95CryptParams[sum&7]
}
