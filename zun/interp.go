package zun

// Interp is the timed tween FUN_00404610 (3 components) / FUN_00412ac0 (2),
// shared by ECL movement and the ANM VM. It is live while Dur != 0; a
// negative Dur never finishes. Modes: 7 adds End to Start every frame, 17
// adds Tan1 (then End to Tan1), 8 is a cubic Hermite through Tan0/Tan1, the
// rest scale End-Start by Ease.
type Interp struct {
	Start, End, Tan0, Tan1 [3]float32
	T                      Timer
	Dur, Mode              int32
}

// HermiteWeights are the cubic Hermite weights mode 8 computes from t, the
// same code in FUN_00404610 (0x4047ff-0x404846), FUN_00412ac0 (from
// 0x412c38) and FUN_004049a0 (0x404b21-0x404b7a):
//
//	a   = t − 1
//	h00 = ((1 + (t + t))·a)·a
//	h01 = ((3 − (t + t))·t)·t
//	h10 = ((1 − t)·(1 − t))·t
//	h11 = (a·t)·t
//
// with every operation rounded to float32 (x87 at 24-bit precision). h00
// doubles t before adding the 1 (FLD ST0; FADD ST0,ST1, then FLD 1.0;
// FADD ST0,ST1): the decompiler's "1 + t + t" reads as (1 + t) + t, which
// differs in the last bit for t = 1/5, 1/6, 1/7 and many more.
func HermiteWeights(t float32) (h00, h01, h10, h11 float32) {
	a := t - 1
	h00 = float32(float32((1+(t+t))*a) * a)
	h01 = float32(float32((3-(t+t))*t) * t)
	h10 = float32(float32((1-t)*(1-t)) * t)
	h11 = float32(float32(a*t) * t)
	return h00, h01, h10, h11
}

// Begin restarts the tween's timer (FUN_004050d0 / FUN_00412db0) and zeroes
// the tangents, as every caller does by storing the zero vector DAT_00491c14.
// Callers set Start and End themselves.
func (p *Interp) Begin(dur, mode int32) {
	p.Dur, p.Mode = dur, mode
	p.Tan0, p.Tan1 = [3]float32{}, [3]float32{}
	p.T.Reset()
}

// Step advances one frame at game speed gs and returns the current value; n
// is the component count (2 or 3).
func (p *Interp) Step(gs float32, n int) (out [3]float32) {
	if p.Dur > 0 {
		p.T.Tick(gs)
		if p.Dur <= p.T.Cur {
			p.T.Set(p.Dur)
			p.Dur = 0
			if p.Mode == 7 {
				return p.Start
			}
			return p.End
		}
	}
	switch p.Mode {
	case 7:
		for i := range n {
			p.Start[i] += p.End[i]
		}
		return p.Start
	case 17:
		for i := range n {
			out[i] = p.Start[i] + p.Tan1[i]
			p.Start[i] = out[i]
			p.Tan1[i] += p.End[i]
		}
		return out
	case 8:
		h00, h01, h10, h11 := HermiteWeights(p.T.F / float32(p.Dur))
		for i := range n {
			out[i] = float32(float32(float32(h00*p.Start[i])+float32(h01*p.End[i]))+float32(h10*p.Tan0[i])) +
				float32(h11*p.Tan1[i])
		}
		return out
	}
	f := Ease(p.Mode, p.T.F, float32(p.Dur))
	for i := range n {
		out[i] = float32(float32(p.End[i]-p.Start[i])*f) + p.Start[i]
	}
	return out
}
