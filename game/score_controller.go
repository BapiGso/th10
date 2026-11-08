package game

// Original stored-point tables, PE 0x474474 / 0x474488. The last entry is
// unreachable under FUN_00409d90's 999999999 cap, not a repeating extend.
var scoreExtends = [...]int32{2000000, 4000000, 8000000, 15000000, 1000000000}
var extraExtends = [...]int32{3000000, 10000000, 1000000000}

// BindScoreController selects the original staged scoring path. Unbound
// standalone state helpers retain the old immediate HiScore API for callers
// that have no stage/GUI controller; a real Stage always binds before play.
func (s *GameState) BindScoreController() { s.scoreControllerBound = true }

// UpdateScoreController ports FUN_00417040's non-visual side effects. The
// caller supplies GUI+0x9e78 after rolling, not DAT_00474c44's real score.
// Return true only when FUN_004188a0 would emit the one-up sound/notice.
func (s *GameState) UpdateScoreController(shown int32, noExtend bool) bool {
	if int64(shown) > s.HiScore/10 {
		s.RunFlags |= RunFlagHighScore       // 0x417118
		s.HiScoreContinues = s.ReplayState90 // DAT_00474c94, 0x417110
		s.HiScore = int64(shown) * 10
	}
	if noExtend { // GUI+0x9eb4 bit 0x20, 0x417288
		return false
	}
	table := scoreExtends[:]
	if s.Difficulty == DiffExtra {
		table = extraExtends[:]
	}
	i := s.ReplayState9C // DAT_00474c9c, chapter +0x1b8 is the extend-table cursor
	// Original replays use 0..sentinel; guard untrusted out-of-range cursors.
	if i < 0 || int64(i) >= int64(len(table)) || shown < table[i] {
		return false
	}
	s.ReplayState9C++ // 0x4172d5: exactly one threshold per controller tick
	n := s.Life + 1   // FUN_004188a0; natural extends do not add rank
	s.Life = min(n, 9)
	return n <= 9 // 0x4188a9..0x4188af: cap skips both sound and World notice
}
