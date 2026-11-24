package audio

import "math"

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
