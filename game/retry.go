package game

// RetryInitialStage is FUN_004218d0 request 10 (0x421a03..0x421a34),
// distinct from request 13's later-stage continue counter. Fresh resource
// initialization reuses the existing t5/t8 RestartStage helper.
func (s *GameState) RetryInitialStage() {
	if s.StartStage != 0 {
		s.Stage = s.StartStage
	} else {
		s.StartStage = s.Stage
	}
	s.RunFlags &^= RunFlagContinued | RunFlagReuseManagers | RunFlagKeepReplay
	s.Continued, s.ReplayState90 = false, 0
	s.RestartStage()
}
