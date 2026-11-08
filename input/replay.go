package input

import "th10/replay"

// ReplayBits is the input.State mapping previously local to sim. Movement
// resolves contradictory directions in FUN_004250b0's order, while the
// original raw words are retained separately for a recording observer.
func ReplayBits(rec replay.Record) uint16 {
	var bits uint16
	set := func(k Key, on bool) {
		if on {
			bits |= 1 << uint(k)
		}
	}
	set(KeyUp, rec.Up())
	set(KeyDown, rec.Down())
	set(KeyLeft, rec.Left())
	set(KeyRight, rec.Right())
	set(KeyShot, rec.Keys&replay.KeyShot != 0)
	set(KeyBomb, rec.Keys&replay.KeyBomb != 0)
	set(KeyFocus, rec.Keys&replay.KeyFocus != 0)
	set(KeyOk, rec.Keys&replay.KeyShot != 0)
	set(KeySkip, rec.Keys&replay.KeySkip != 0)
	return bits
}

// RestoreReplay is the playback-input branch of FUN_004297d0. Aux1/Aux2
// retain DAT_00474e62/64 exactly, including the first chapter frame.
func (s *State) RestoreReplay(rec replay.Record) {
	s.Restore(ReplayBits(rec))
	s.replayRecord, s.fromReplay = rec, true
}

// ApplyReplayOptions is FUN_004297d0, 0x429812..0x429842: cfg +0x30 bit
// 0x200 synthesizes focus after eight frames holding shot. Recorded input
// already contains that bit, so host configuration must not alter playback.
func (s *State) ApplyReplayOptions(flags uint32) {
	if !s.fromReplay && flags&0x200 != 0 && s.HoldFrames(KeyShot) >= 8 {
		s.pressed[KeyFocus] = true
	}
}

// ReplayRecord is the live-input branch of FUN_004297d0 and FUN_0040ac20:
// DAT_00474e5c = raw & 0x1f7; +0x62 = current &^ previous; +0x64 is the
// release edge, previous &^ current (0x40ac76..0x40ac86).
func (s *State) ReplayRecord() replay.Record {
	if s.fromReplay {
		return s.replayRecord
	}
	keys, prev := replayKeys(s.pressed), replayKeys(s.prev)
	return replay.Record{Keys: keys, Aux1: keys &^ prev, Aux2: prev &^ keys}
}

func replayKeys(state [keyCount]bool) uint16 {
	var keys uint16
	for _, pair := range [...]struct {
		key Key
		bit uint16
	}{
		{KeyShot, replay.KeyShot}, {KeyBomb, replay.KeyBomb}, {KeyFocus, replay.KeyFocus},
		{KeyUp, replay.KeyUp}, {KeyDown, replay.KeyDown}, {KeyLeft, replay.KeyLeft},
		{KeyRight, replay.KeyRight}, {KeySkip, replay.KeySkip},
	} {
		if state[pair.key] {
			keys |= pair.bit
		}
	}
	return keys
}
