package game

import "th10/zun"

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
	s.FreshRun = false
}

// RestartStage ports mode 13 of FUN_004218d0 (0x421ac7), its
// FUN_00417c80 teardown and the fresh-run branch of FUN_00417870. The
// requested initial stage (DAT_00474c80) is not advanced by FUN_004175e0,
// so restarting after progressing beyond it counts as a continue.
func (s *GameState) RestartStage() {
	if s.StartStage != 0 && s.StartStage != s.Stage {
		s.ReplayState90 = min(s.ReplayState90+1, 9)           // FUN_00418a90, 0x417d92
		s.RunFlags |= RunFlagContinued | RunFlagReuseManagers // FUN_00417c80, 0x417d9c
	}
	if s.RunFlags&RunFlagContinued == 0 {
		s.ReplayState90 = 0
	}
	s.Frame, s.Score, s.Faith = 0, 0, 50000
	s.RestoreFaithHold(0)
	s.ReplayState9C = 0
	s.ResetRank(s.RunFlags&RunFlagContinued != 0) // FUN_00417870, 0x4179cc
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
	s.ReplayState90 = 0
	s.RestartStage()
}

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
