package bullet

// Per-type tables from th10.exe .rdata, indexed by bullet type 0..26.
var (
	// scriptBase is 0x474170: a bullet's script is scriptBase[type] + colour.
	scriptBase = [27]int32{0, 64, 80, 96, 112, 128, 144, 160, 176, 192, 208, 224, 275, 283,
		291, 299, 307, 347, 373, 315, 323, 331, 339, 240, 48, 374, 243}
	// sizes is 0x4741e0, the hit size (W = H).
	sizes = [27]float32{4, 6, 6, 4, 4, 4, 4, 4, 4, 0, 4, 6, 10, 8,
		8, 8, 8, 28, 6, 10, 8, 8, 8, 6, 4, 6, 4}
	// layers is 0x474250, the draw layer.
	layers = [27]int32{5, 3, 3, 4, 4, 4, 3, 4, 4, 4, 4, 3, 1, 2,
		2, 2, 2, 0, 2, 2, 2, 1, 2, 3, 5, 2, 4}
	// cancelKind is 0x4742c0, how the cancel script is picked (cancelScript).
	cancelKind = [27]int32{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1, 1,
		1, 1, 1, 2, 3, 1, 1, 1, 1, 0, 0, 4, 0}
	// cancelByColour is 0x47432c, the kind-1 cancel script per colour.
	cancelByColour = [8]int32{17, 21, 25, 29, 33, 37, 41, 47}
)

// Rect-clear spark colours (D3DCOLOR ARGB) by colour, chosen by the
// bullet's sprite width: 0x474350 up to 16px, 0x474390 up to 32px,
// 0x4743b0 above.
var (
	clearColour16 = [16]uint32{0xff808080, 0xffff1010, 0xffff1010, 0xff801080, 0xff801080,
		0xff1010ff, 0xff1010ff, 0xff108080, 0xff108080, 0xff10ff10, 0xff10ff10, 0xff10ff10,
		0xff808010, 0xff808010, 0xff808010, 0xff808080}
	clearColour32 = [8]uint32{0xff808080, 0xffff1010, 0xff801080, 0xff1010ff, 0xff108080,
		0xff10ff10, 0xff808010, 0xff808080}
	clearColourBig = [4]uint32{0xffff1010, 0xff1010ff, 0xff10ff10, 0xff808010}
)

// Effect scripts in bullet.anm.
const (
	effectGraze = 434 // the graze spark
	effectClear = 371 // the rect clear spark
)

// SE ids (table 0x4749c8).
const seGraze = 28

// ScriptID is bullet.anm's script for type t in colour c (FUN_00404f30's
// callers: 0x474170[type] + colour). The lasers use it for their beams.
func ScriptID(t int16, c int32) int32 { return scriptBase[typeIndex(t)] + c }

func typeIndex(t int16) int {
	if t < 0 || int(t) >= len(sizes) {
		return 0
	}
	return int(t)
}

// cancelScript is the switch on cancelKind in FUN_004067d0.
func cancelScript(t, colour int16) int32 {
	switch cancelKind[typeIndex(t)] {
	case 0:
		return int32(colour)*2 + 0x11
	case 1:
		if colour >= 0 && int(colour) < len(cancelByColour) {
			return cancelByColour[colour]
		}
		return -1
	case 2:
		return -1
	case 3:
		return 0x1d
	case 4:
		return 0x13
	}
	return -1
}

// clearColour picks the rect-clear spark colour for a sprite width w.
func clearColour(w float32, colour int16) (uint32, bool) {
	var tbl []uint32
	switch {
	case w <= 16:
		tbl = clearColour16[:]
	case w <= 32:
		tbl = clearColour32[:]
	default:
		tbl = clearColourBig[:]
	}
	if colour < 0 || int(colour) >= len(tbl) {
		return 0, false
	}
	return tbl[colour], true
}
