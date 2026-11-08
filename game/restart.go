package game

// RestartStage ports mode 13 of FUN_004218d0 (0x421ac7), its
// FUN_00417c80 teardown and the fresh-run branch of FUN_00417870. The
// requested initial stage (DAT_00474c80) is not advanced by FUN_004175e0,
// so restarting after progressing beyond it counts as a continue.
func (s *GameState) RestartStage() {
	if s.StartStage != 0 && s.StartStage != s.Stage {
		s.ReplayState90 = min(s.ReplayState90+1, 9) // FUN_00418a90, 0x417d92
		s.Continued = true
		s.RunFlags |= RunFlagContinued | RunFlagReuseManagers // FUN_00417c80, 0x417d9c
	}
	if s.RunFlags&RunFlagContinued == 0 {
		s.ReplayState90 = 0
	}
	s.Frame, s.Score, s.Faith = 0, 0, 50000
	s.RestoreFaithHold(0)
	s.ReplayState9C = 0
	s.ResetRank(s.RunFlags&RunFlagContinued != 0) // FUN_00417870, 0x4179cc
	s.Continued = s.RunFlags&RunFlagContinued != 0
	s.Power = 0
	if s.Stage != 1 {
		s.Power = 0x50 * 5
	}
	s.Life = 2
	if s.RunFlags&RunFlagPractice != 0 {
		s.Life = s.practiceLife()
	} // practice, DAT_00474cac = 0 default
	s.RunFlags &^= RunFlagHighScore
	s.FreshRun, s.StageMode, s.ReplayMode = true, 0, 0 // FUN_004218d0 mode 13
	s.Paused, s.Recording = false, nil
}
