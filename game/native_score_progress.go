package game

import "encoding/binary"

// StageRecord is the eight-byte entry at score object +0x4dc + index*8.
// FUN_004236f0 saves practice score at +0. FUN_00415e90 sets both +4/+5
// flags (0x416c08 / 0x416c39); +6/+7 and nonzero flag bytes are retained.
type StageRecord struct {
	Score              int32
	Available, Cleared byte
	Raw                [8]byte
}

type ShotProgress struct {
	Runs, Frames int32    // object +0x4c8 / +0x4cc, FUN_00417870 / FUN_00418190
	Clears       [5]int32 // object +0x4d0, FUN_00415e90's final clear counter
	Stages       [4][6]StageRecord
}

type ScoreProgress [7]ShotProgress // six shot blocks plus the otherwise preserved total block

func (s *ScoreFile) progress() (p ScoreProgress) {
	for shot := range p {
		b := s.Blocks[shot][:]
		p[shot].Runs = int32(binary.LittleEndian.Uint32(b[0x4c0:]))
		p[shot].Frames = int32(binary.LittleEndian.Uint32(b[0x4c4:]))
		for difficulty := range 5 {
			p[shot].Clears[difficulty] = int32(binary.LittleEndian.Uint32(b[0x4c8+difficulty*4:]))
		}
		for difficulty := range 4 {
			for stage := range 6 {
				at := scoreProgressAt + (stage+1+difficulty*6)*8
				r := &p[shot].Stages[difficulty][stage]
				copy(r.Raw[:], b[at:at+8])
				r.Score, r.Available, r.Cleared = int32(binary.LittleEndian.Uint32(b[at:])), b[at+4], b[at+5]
			}
		}
	}
	return p
}

func (s *ScoreFile) setProgress(p *ScoreProgress) {
	for shot := range p {
		b := s.Blocks[shot][:]
		binary.LittleEndian.PutUint32(b[0x4c0:], uint32(p[shot].Runs))
		binary.LittleEndian.PutUint32(b[0x4c4:], uint32(p[shot].Frames))
		for difficulty, n := range p[shot].Clears {
			binary.LittleEndian.PutUint32(b[0x4c8+difficulty*4:], uint32(n))
		}
		for difficulty := range 4 {
			for stage := range 6 {
				at := scoreProgressAt + (stage+1+difficulty*6)*8
				r := &p[shot].Stages[difficulty][stage]
				copy(b[at:at+8], r.Raw[:])
				binary.LittleEndian.PutUint32(b[at:], uint32(r.Score))
				b[at+4], b[at+5] = r.Available, r.Cleared
			}
		}
	}
}

func (s *GameState) scoreShot() int {
	return s.Character*3 + s.ShotType
}

func (s *GameState) shotProgress() *ShotProgress {
	shot := s.scoreShot()
	if s.ScoreProgress == nil || shot < 0 || shot >= 6 {
		return nil
	}
	return &s.ScoreProgress[shot]
}

// RecordPlayFrame is FUN_00418190 (0x418727..0x41873e), after Numbers and
// before faith decay. Exactly mode 1 suppresses this saturated counter.
func (s *GameState) RecordPlayFrame() {
	if s.ReplayMode != 1 {
		if p := s.shotProgress(); p != nil && p.Frames < 215999999 {
			p.Frames++
		}
	}
}

// RecordStageClear is the non-Extra part of MSG 0x14. It also applies to
// practice and replay: the binary writes these flags before testing modes.
// Game.BindState supplies a private progress view for read-only playback.
func (s *GameState) RecordStageClear() {
	if s.Stage < 1 || s.Stage > 6 || s.Difficulty < 0 || s.Difficulty > 3 {
		return
	}
	if p := s.shotProgress(); p != nil {
		r := &p.Stages[s.Difficulty][s.Stage-1]
		r.Available, r.Cleared = 1, 1
	}
}

// RecordFinalClear is FUN_00415e90 (0x416dd2), reached only after a normal
// six-stage/Extra clear, not the practice or replay branches.
func (s *GameState) RecordFinalClear() {
	if s.ReplayMode == 1 || s.RunFlags&RunFlagPractice != 0 || (s.Stage != StageCount && s.Stage != StageExtra) || s.Difficulty < 0 || s.Difficulty > 4 {
		return
	}
	if p := s.shotProgress(); p != nil && p.Clears[s.Difficulty] < 99999 {
		p.Clears[s.Difficulty]++
	}
}

func (g *Game) RecordPracticeScore(s *GameState) {
	// FUN_004236f0 case 7, 0x4237d3: no ranking for bit 0x10, instead
	// replace only this stage's practice best if the stored score is higher.
	if s.ReplayMode == 1 || s.RunFlags&RunFlagPractice == 0 || s.Stage < 1 || s.Stage > 6 || s.Difficulty < 0 || s.Difficulty > 3 {
		return
	}
	if g.Save == nil {
		g.Save = DefaultSaveData()
	}
	p := &g.Save.Progress[min(max(s.scoreShot(), 0), 5)]
	r := &p.Stages[s.Difficulty][s.Stage-1]
	r.Score = max(r.Score, int32(s.Score/10))
}
