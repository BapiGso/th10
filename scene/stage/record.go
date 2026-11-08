package stage

import (
	"encoding/binary"
	"slices"
	"th10/replay"
)

// ReplaySnapshot ports FUN_00428f60 / FUN_0042a450 in their write order.
// The seed is captured before STD/ECL initialization; player fields are
// captured after initialization, just before the first recorded input.
func (s *Stage) ReplaySnapshot() replay.Chapter {
	c, state := s.ctx, s.ctx.State
	raw := make([]byte, 0x1c4)
	copy(raw, s.replayTemplate)
	if c.CaptureReplayPlayer != nil {
		c.CaptureReplayPlayer(raw)
	}
	flags := binary.LittleEndian.Uint32(raw[0x1c0:])
	if s.replayTemplate == nil {
		// FUN_00428f60: bit 0 is DAT_00491fc4, not an inferred stage number.
		if state.FreshRun {
			flags |= 1
		} else {
			flags &^= 1
		}
	}
	return replay.Chapter{
		Index: uint16(state.Stage), Seed: s.replaySeed,
		Score: int32(state.Score / 10), Power: int16(state.Power / 5),
		Faith: int32(state.Faith / 10), Hold: int32(state.FaithHold),
		Lives: int32(state.Life), Rank: int32(state.Rank),
		PlayerX: c.Player.IX, PlayerY: c.Player.IY,
		State90: state.ReplayState90, State9C: state.ReplayState9C,
		Focus: c.Player.ReplayFocus(), Flags: flags, Raw: raw,
	}
}

// RestoreReplayChapter is FUN_00428f60's state restore followed by
// FUN_0042a450 (mode 1). The RNG was seeded before stage initialization;
// do not rewind draws already made by STD/ECL time-0 instructions here.
func (s *Stage) RestoreReplayChapter(ch replay.Chapter) {
	c, state := s.ctx, s.ctx.State
	state.Score, state.Faith = int64(ch.Score)*10, int64(ch.Faith)*10
	state.Power, state.Life, state.Rank = int(ch.Power)*5, int(ch.Lives), int(ch.Rank)
	state.RestoreFaithHold(int(ch.Hold))
	state.ReplayState90, state.ReplayState9C = ch.State90, ch.State9C
	state.FreshRun = ch.Flags&1 != 0 // FUN_0042a6a0, chapter +0x1c0
	c.Player.SetFixed(ch.PlayerX, ch.PlayerY)
	c.Player.RestoreReplayFocus(ch.Focus)
	s.replayTemplate = slices.Clone(ch.Raw)
	if c.RestoreReplayPlayer != nil {
		c.RestoreReplayPlayer(ch.Raw)
	}
}

// StartRecording attaches an explicit observer for sim's replay re-recording
// acceptance test. Interactive mode calls it only for original mode 0.
func (s *Stage) StartRecording(rec *replay.Recorder) error {
	s.ctx.State.Recording = rec
	return rec.BeginChapter(s.ReplaySnapshot())
}

func (s *Stage) beginRecording() {
	state := s.ctx.State
	if state.StageMode != 0 || state.ReplayMode != 0 || state.DebugMode {
		return
	}
	if state.Recording == nil {
		raw := make([]byte, 100)
		copy(raw[0x14:0x48], state.ReplayConfig[:])                            // FUN_00428f60's 13 config words
		binary.LittleEndian.PutUint32(raw[0x60:], uint32(state.ReplayState90)) // FUN_00428f60: initial continue count
		state.Recording = replay.NewRecorder(replay.GlobalHeader{
			Character: uint32(state.Character), ShotType: uint32(state.ShotType),
			Difficulty: uint32(state.Difficulty), Stage: uint32(state.Stage),
			Raw: raw,
		})
	}
	// FUN_00429610 receives only stage +0x5c (mode); neither it nor
	// FUN_00428f60 excludes DAT_00474ca0 bit 0x10 (practice). FUN_004236f0
	// also offers replay saving in that branch. Practice is recorded too.
	_ = s.StartRecording(state.Recording)
}
