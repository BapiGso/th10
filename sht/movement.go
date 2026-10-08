// Package sht reads the original TH10 shot-data format.
package sht

import (
	"encoding/binary"
	"fmt"
	"math"
)

// Movement holds per-axis steps in hundredths of a pixel, as stored by
// th10.exe 0x4248dc..0x424926. This is not yet a complete SHT shot interpreter.
type Movement struct {
	Fast, Slow                 int
	FastDiagonal, SlowDiagonal int
}

// ParseMovement reads the version-3 SHT movement header. The original loader
// recomputes diagonal speeds from the cardinal speeds (FUN_00426520), then
// truncates speed*100 to integer steps (FUN_004247f0).
func ParseMovement(data []byte) (Movement, error) {
	if len(data) < 0x20 {
		return Movement{}, fmt.Errorf("SHT movement header truncated: %d bytes", len(data))
	}
	if version := binary.LittleEndian.Uint16(data); version != 3 {
		return Movement{}, fmt.Errorf("unsupported SHT version %d", version)
	}
	fast := math.Float32frombits(binary.LittleEndian.Uint32(data[0x10:]))
	slow := math.Float32frombits(binary.LittleEndian.Uint32(data[0x14:]))
	for _, speed := range []float32{fast, slow} {
		if math.IsNaN(float64(speed)) || math.IsInf(float64(speed), 0) || speed <= 0 || speed > 100 {
			return Movement{}, fmt.Errorf("invalid SHT movement speed %g", speed)
		}
	}
	return Movement{
		Fast: int(float64(fast) * 100), Slow: int(float64(slow) * 100),
		FastDiagonal: int(float64(float32(float64(fast)*math.Sqrt(0.5))) * 100),
		SlowDiagonal: int(float64(float32(float64(slow)*math.Sqrt(0.5))) * 100),
	}, nil
}
