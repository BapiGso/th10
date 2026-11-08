package replay

import "fmt"

// RunConfig is the chapter-start state shared by sim and interactive
// playback. GameState's score/faith are restored separately after stage
// initialization, in FUN_00428f60 / FUN_0042a450 order.
type RunConfig struct {
	Stage, Character, ShotType, Difficulty int
	Power, Lives                           int
	Seed                                   uint16
	Rank                                   int
	PlayerX, PlayerY                       int32
	Config                                 [0x34]byte
	FreshRun                               bool // DAT_00491fc4, chapter +0x1c0 bit 0
}

// ConfigFor maps the original chapter/header fields, including its
// difficulty fallback (Extra=4 only on stage 7). It was formerly in sim.
func ConfigFor(rp *Replay, chapter int) (RunConfig, error) {
	if chapter < 0 || chapter >= len(rp.Chapters) {
		return RunConfig{}, fmt.Errorf("chapter %d out of range (replay has %d)", chapter, len(rp.Chapters))
	}
	ch := rp.Chapters[chapter]
	stage := int(ch.Index)
	if stage < 1 || stage > 7 {
		stage = int(rp.Global.Stage)
	}
	if stage < 1 || stage > 7 {
		return RunConfig{}, fmt.Errorf("stage %d not simulatable (1..7)", stage)
	}
	cfg := RunConfig{
		Stage: stage, Character: int(rp.Global.Character), ShotType: int(rp.Global.ShotType),
		Difficulty: int(rp.Global.Difficulty), Power: int(ch.Power) * 5, Lives: int(ch.Lives),
		Seed: ch.Seed, Rank: int(ch.Rank), PlayerX: ch.PlayerX, PlayerY: ch.PlayerY,
		FreshRun: ch.Flags&1 != 0,
	}
	if cfg.Difficulty > 3 && (stage != 7 || cfg.Difficulty != 4) {
		cfg.Difficulty = 1
	}
	if len(rp.Global.Raw) == 100 {
		copy(cfg.Config[:], rp.Global.Raw[0x14:0x48])
	}
	return cfg, nil
}
