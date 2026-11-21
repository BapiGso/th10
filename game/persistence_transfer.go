package game

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io/fs"
	"slices"
	"th10/replay"
)

const (
	MaxScoreImportBytes  = maxScoreBytes
	MaxReplayImportBytes = maxReplayBytes
)

// ImportScore validates before writing, retaining the user's original bytes.
// DecodeScoreFile follows the original skip/default policy for bad sections;
// an explicit file import must instead reject those sections without replacing
// an existing save. Call LoadProgress only after a successful import, before play.
func (g *Game) ImportScore(data []byte) error {
	if err := validateScoreImport(data); err != nil {
		return err
	}
	return g.scoreStore().Write(slices.Clone(data))
}

func validateScoreImport(data []byte) error {
	native, err := DecodeScoreFile(data)
	if err != nil {
		return err
	}
	if len(native.Rejected) != 0 {
		return fmt.Errorf("game: score import rejected: %s", native.Rejected[0])
	}
	if uint64(binary.LittleEndian.Uint32(data[4:])) != uint64(len(data)) ||
		uint64(binary.LittleEndian.Uint32(data[0x10:])) != uint64(len(data)-scoreHeaderSize) ||
		binary.LittleEndian.Uint32(data[0x14:]) != uint32(len(native.Blocks)*scoreBlockSize+scoreStatsSize) {
		return fmt.Errorf("game: score import has inconsistent file/section lengths")
	}
	// Reuse the codecs, not a second score parser. The native loader accepts
	// missing/duplicate sections; imports require all seven CRs and one ST.
	compressed := slices.Clone(data[scoreHeaderSize:])
	replay.DecryptBlocks(compressed, scoreKeyInc, 0x10, scoreKey)
	payload, err := replay.DecompressLZSS(compressed, len(native.Blocks)*scoreBlockSize+scoreStatsSize)
	if err != nil {
		return err
	}
	var seen [8]bool
	for at := 0; at < len(payload); {
		section := payload[at:]
		index := 7
		if string(section[:2]) == "CR" {
			index = int(binary.LittleEndian.Uint32(section[0xc:]))
		}
		if seen[index] {
			return fmt.Errorf("game: score import has a duplicate section")
		}
		seen[index] = true
		at += int(binary.LittleEndian.Uint32(section[8:]))
	}
	for _, present := range seen {
		if !present {
			return fmt.Errorf("game: score import has a missing section")
		}
	}
	return nil
}

// ImportReplay uses t5's container decoder and playback bounds validator.
func (g *Game) ImportReplay(slot int, data []byte) error {
	if _, err := ReplayFilename(slot); err != nil {
		return err
	}
	if err := validateReplayImport(data); err != nil {
		return err
	}
	return g.replayStore().Write(slot, slices.Clone(data))
}

func validateReplayImport(data []byte) error {
	if len(data) > maxReplayBytes {
		return fmt.Errorf("game: replay exceeds %d bytes", maxReplayBytes)
	}
	r, err := replay.Decode(data)
	if err != nil {
		return err
	}
	return r.ValidatePlayback()
}

// ExportScore returns the last committed bytes. Before the first save it
// provides a valid default snapshot without causing an implicit write.
func (g *Game) ExportScore() ([]byte, error) {
	data, err := g.scoreStore().Read()
	if !errors.Is(err, fs.ErrNotExist) {
		return data, err
	}
	if g.Save != nil {
		return g.Save.nativeBytes()
	}
	return EncodeScoreFile(DefaultScoreFile(0))
}

func (g *Game) ExportReplay(slot int) ([]byte, error) {
	if _, err := ReplayFilename(slot); err != nil {
		return nil, err
	}
	data, err := g.replayStore().Read(slot)
	if err != nil {
		return nil, err
	}
	if err := validateReplayImport(data); err != nil {
		return nil, err
	}
	return data, nil
}
