package zun

// RNG is the binary's 16-bit generator {state u16, count u32}. The game RNG
// lives at DAT_004918b0 and is shared by ECL and every ANM VM without flag
// 0x40000000; VMs with the flag draw from DAT_004918a8. FUN_00448db0 sets the
// flag on its effects, but the reset inside FUN_0043e710 clears it again, so
// bullet cancel/graze effects draw from the game RNG. Only children spawned by
// ANM op 88 (FUN_0043e7e0, no reset) keep the flag.
type RNG struct {
	State uint16
	Count uint32
}

// step is one 16-bit draw: rotl16((w ^ 0x9630) - 0x6553, 2).
func (r *RNG) step() uint16 {
	w := (r.State ^ 0x9630) - 0x6553
	w = w<<2 | w>>14
	r.State = w
	return w
}

// Raw is two steps, hi<<16 | lo (the body of FUN_0044bb20/FUN_0044bb90).
func (r *RNG) Raw() uint32 {
	hi := r.step()
	lo := r.step()
	r.Count += 2
	return uint32(hi)<<16 | uint32(lo)
}

// Float is FUN_0044bb20: [0, 1), raw * 2^-32.
func (r *RNG) Float() float32 {
	return float32(float64(r.Raw()) * (1.0 / 4294967296.0))
}

// Signed is FUN_0044bb90: [-1, 1), raw * 2^-31 - 1 at 24-bit precision.
func (r *RNG) Signed() float32 {
	return float32(float64(r.Raw())*(1.0/2147483648.0)) - 1
}

// Intn is FUN_0043cb80: raw % n, or 0 without a draw when n is 0.
func (r *RNG) Intn(n uint32) uint32 {
	if n == 0 {
		return 0
	}
	return r.Raw() % n
}
