package replay

const (
	lzWindow = 0x2000 // FUN_004359b0, 0x4359f2: zero 0x800 dwords
	lzMask   = lzWindow - 1
	lzMatch  = 18 // FUN_004359b0, 0x435a51
)

type lzNode struct{ parent, left, right int }

// lzTree is DAT_00477858 (three dwords per node) and DAT_0048f868.
// Index 0 is the null link, 0x2000 is the sentinel whose right link is the
// root. The original also writes node 0's parent when splicing null links.
type lzTree struct {
	nodes  [lzWindow + 1]lzNode
	window [lzWindow]byte
}

// insert ports FUN_00436000. Equal-length matches replace the previous best
// (0x436131 J L, not J LE); an 18-byte match replaces the old tree node.
// Both details affect the compressed bytes, even though either match would
// decode to the same input.
func (t *lzTree) insert(at int) (length, offset int) {
	if at == 0 {
		return 0, 0
	}
	for p := t.nodes[lzWindow].right; ; {
		n, diff := 0, 0
		for n < lzMatch {
			diff = int(t.window[(at+n)&lzMask]) - int(t.window[(p+n)&lzMask])
			if diff != 0 {
				break
			}
			n++
		}
		if n >= length {
			length, offset = n, p
			if n == lzMatch {
				t.replace(p, at)
				return length, offset
			}
		}
		link := &t.nodes[p].right
		if diff < 0 {
			link = &t.nodes[p].left
		}
		if *link == 0 {
			*link = at
			t.nodes[at] = lzNode{parent: p}
			return length, offset
		}
		p = *link
	}
}

// remove ports FUN_00436210 and the inline copy at 0x435bd4..0x435cf5.
func (t *lzTree) remove(at int) {
	n := t.nodes[at]
	if n.parent == 0 {
		return
	}
	if n.right == 0 {
		t.unlink(at, n.left)
	} else if n.left == 0 {
		t.unlink(at, n.right)
	} else {
		p := t.predecessor(at)
		t.remove(p)
		t.replace(at, p)
	}
}

// unlink is FUN_00436260, replacing a node with its sole child.
func (t *lzTree) unlink(at, child int) {
	p := t.nodes[at].parent
	t.nodes[child].parent = p
	if t.nodes[p].right == at {
		t.nodes[p].right = child
	} else {
		t.nodes[p].left = child
	}
	t.nodes[at].parent = 0
}

// replace is FUN_004362b0, also inlined by FUN_00436000's full-match branch.
func (t *lzTree) replace(at, with int) {
	n := t.nodes[at]
	if t.nodes[n.parent].left == at {
		t.nodes[n.parent].left = with
	} else {
		t.nodes[n.parent].right = with
	}
	t.nodes[with] = n
	t.nodes[n.left].parent = with
	t.nodes[n.right].parent = with
	t.nodes[at].parent = 0
}

// predecessor is FUN_00436330: the rightmost node of the left subtree.
func (t *lzTree) predecessor(at int) int {
	p := t.nodes[at].left
	for t.nodes[p].right != 0 {
		p = t.nodes[p].right
	}
	return p
}

type bitWriter struct {
	data []byte
	cur  byte
	mask byte
}

// write is the MSB-first bit emission in FUN_004359b0 (0x435aa3 onwards).
func (w *bitWriter) write(n int, value int) {
	for bit := n - 1; bit >= 0; bit-- {
		if value&(1<<uint(bit)) != 0 {
			w.cur |= w.mask
		}
		w.mask >>= 1
		if w.mask == 0 {
			w.data = append(w.data, w.cur)
			w.cur, w.mask = 0, 0x80
		}
	}
}

// lzssCompress ports FUN_004359b0, including its initially unsearched node
// 1, an 18-byte lookahead, per-byte tree updates, and the incomplete-byte
// truncation at the end. This is not thtk's hash-based compressor.
func lzssCompress(src []byte) []byte {
	t := &lzTree{}
	t.nodes[lzWindow].right = 1  // 0x435a62
	t.nodes[1].parent = lzWindow // 0x435a6c
	waiting := min(len(src), lzMatch)
	copy(t.window[1:], src[:waiting])
	read, head, matchLen, matchOff := waiting, 1, 0, 0
	w := bitWriter{data: make([]byte, 0, len(src)), mask: 0x80}
	for waiting > 0 {
		matchLen = min(matchLen, waiting)
		n := matchLen
		if n < 3 { // 0x435a9a: lengths 0..2 are emitted as one literal
			n = 1
			w.write(1, 1)
			w.write(8, int(t.window[head]))
		} else {
			w.write(1, 0)
			w.write(13, matchOff)
			w.write(4, n-3)
		}
		for i := 0; i < n; i++ {
			tail := (head + lzMatch) & lzMask
			t.remove(tail)
			if read < len(src) {
				t.window[tail] = src[read]
				read++
			} else {
				waiting--
			}
			head = (head + 1) & lzMask
			if waiting > 0 {
				matchLen, matchOff = t.insert(head)
			}
		}
	}
	// 0x435d6e..0x435da9: only flag 0 and offset 0, not a length, and no
	// flush of the last partial byte. The declared decompressed length lets
	// the reader stop before this truncated terminator.
	w.write(14, 0)
	return w.data
}

// xorEncrypt ports FUN_0044b220, the inverse of FUN_0044b0d0. The block
// permutation and key progression are the same but source/destination swap.
func xorEncrypt(data []byte, keyInc byte, modulus int, initKey byte) {
	n := len(data)
	rem, corr := n%modulus, 0
	if rem < modulus/4 {
		corr = rem
	}
	toDo := n - ((n & 1) + corr)
	src := append([]byte(nil), data...)
	key, pos, write := initKey, 0, 0
	for toDo > 0 {
		size := min(modulus, toDo)
		for i := size - 1; i >= 0; i -= 2 {
			data[write] = src[pos+i] ^ key
			key += keyInc
			write++
		}
		for i := size - 2; i >= 0; i -= 2 {
			data[write] = src[pos+i] ^ key
			key += keyInc
			write++
		}
		pos += size
		toDo -= size
	}
}
