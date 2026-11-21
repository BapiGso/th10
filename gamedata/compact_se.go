// Compact SE envelopes are lossless and decoded only at the loading boundary.
package gamedata

import (
	"bytes"
	"compress/flate"
	"encoding/binary"
	"fmt"
	"io"
)

// compactWAV reverses the development-time lossless SE envelope. It runs
// during resource loading, before Ebiten's WAV decode and PCM preloading.
func compactWAV(data []byte) ([]byte, error) {
	if len(data) < 8 {
		return nil, fmt.Errorf("gamedata: truncated compact SE")
	}
	n := uint64(binary.LittleEndian.Uint32(data[4:]))
	if n < 12 || n > 16<<20 {
		return nil, fmt.Errorf("gamedata: invalid compact SE length")
	}
	r := flate.NewReader(bytes.NewReader(data[8:]))
	defer func() { _ = r.Close() }()
	pcm, err := io.ReadAll(io.LimitReader(r, int64(n)+1))
	if err != nil {
		return nil, err
	}
	if uint64(len(pcm)) != n || string(pcm[:4]) != "RIFF" || string(pcm[8:12]) != "WAVE" {
		return nil, fmt.Errorf("gamedata: invalid decoded SE")
	}
	return pcm, nil
}
