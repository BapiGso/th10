package game

// DAT_00474ca0 is a controller bitset, not a single practice/replay mode.
const (
	RunFlagReuseManagers uint32 = 0x01 // FUN_00417c80: same-stage reload keeps shared managers
	RunFlagKeepReplay    uint32 = 0x02 // FUN_00417c80: scene 0x0b keeps the replay manager across stages
	RunFlagHighScore     uint32 = 0x04 // FUN_00417040: this run improved DAT_00474c40
	RunFlagContinued     uint32 = 0x08 // FUN_00417c80: later-stage restart; FUN_00417870 starts rank at -512
	RunFlagPractice      uint32 = 0x10 // FUN_0042d420: stage-practice selection, not boss practice
	RunFlagDemo          uint32 = 0x20 // FUN_0042cdf0: title idle demo; suppress titles/BGM/pause
)

// Practice is written by the practice menu (FUN_0042d420, 0x42d7ce).
// SingleStage remains the host's stop-after-one-stage policy, not a binary flag.
func (s *GameState) Practice(stage int) {
	s.RunFlags |= RunFlagPractice
	s.Stage, s.StartStage, s.SingleStage = stage, stage, true
	s.Life = s.practiceLife()
	if stage != 1 {
		s.Power = 0x50 * 5 // FUN_00417870, 0x417975
	}
}

func (s *GameState) practiceLife() int {
	// FUN_00417870, 0x41794f..0x417974: DAT_00474cac == 0 means 9.
	if s.PracticeLives == 0 {
		return 9
	}
	return s.PracticeLives - 1
}

// BeginStageTransition is FUN_00417c80's scene 0x0b path followed by
// FUN_00417870's successful initialization (0x417c34 clears bits 0/1/3).
// Scene 0x0b carries stage +0x5c forward, and clears DAT_00491fc4.
func (s *GameState) BeginStageTransition() {
	s.RunFlags &^= RunFlagReuseManagers | RunFlagKeepReplay | RunFlagContinued
	s.FreshRun, s.Continued = false, false
}
