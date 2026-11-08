package game

// AwardClearBonus is MSG 0x14's FUN_00409d90 call sequence in
// FUN_00415e90 (0x416c64..0x416e44). Practice exits before these calls;
// replay still scores the bonus, but does not write final-clear counters.
func (s *GameState) AwardClearBonus() {
	if s.RunFlags&RunFlagPractice != 0 || (s.Stage != StageCount && s.Stage != StageExtra) {
		return
	}
	// Preserve the native int32 multiplications and individual /10/cap steps.
	s.AddPoints(int64(int32(s.Faith/10) * 1000))
	life, power := int32(40000000), int32(400000)
	if s.Stage != StageExtra {
		switch s.Difficulty {
		case DiffEasy:
			life, power = 20000000, 100000
		case DiffNormal:
			life, power = 25000000, 100000
		case DiffHard:
			life, power = 35000000, 200000
		case DiffLuna:
			life, power = 40000000, 300000
		case DiffExtra:
		default:
			return // native switch default has already added the faith bonus
		}
	}
	s.AddPoints(int64(int32(s.Life) * life))
	s.AddPoints(int64(int32(int16(s.Power/5)) * power)) // MOVSX DAT_00474c48
}
