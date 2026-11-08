package game

import "th10/zun"

// CarryStageRNG keeps the process-global DAT_004918b0 through the live
// scene-0x0b replacement (FUN_00417c80). Stage construction must not undo
// ECL/shake RNG consumption; only replay restoration (FUN_0042a6a0)
// deliberately overrides a chapter's seed.
func (s *GameState) CarryStageRNG(r *zun.RNG) {
	s.stageRNG = r
	if r != nil {
		s.RNGSeed, s.HasSeed = r.State, true
	}
}

// StageRNG is the live continuation reference, nil before a transfer.
// NewContext ignores it for playback, whose chapter seed is authoritative.
func (s *GameState) StageRNG() *zun.RNG { return s.stageRNG }
