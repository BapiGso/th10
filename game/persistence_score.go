package game

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"slices"
)

// ScoreStore is the user-data boundary, like ReplayStore. Both platforms
// store the same native bytes; LegacyJSON is only read on first migration.
type ScoreStore interface {
	Read() ([]byte, error)
	Write([]byte) error
	LegacyJSON() ([]byte, error)
}

type MemoryScoreStore struct {
	data, legacy []byte
}

func NewMemoryScoreStore() *MemoryScoreStore { return &MemoryScoreStore{} }

func (s *MemoryScoreStore) Read() ([]byte, error) {
	if s.data == nil {
		return nil, fs.ErrNotExist
	}
	return slices.Clone(s.data), nil
}

func (s *MemoryScoreStore) Write(data []byte) error {
	if len(data) > maxScoreBytes {
		return fmt.Errorf("game: score exceeds %d bytes", maxScoreBytes)
	}
	s.data = slices.Clone(data)
	return nil
}

func (s *MemoryScoreStore) LegacyJSON() ([]byte, error) {
	if s.legacy == nil {
		return nil, fs.ErrNotExist
	}
	return slices.Clone(s.legacy), nil
}

func (s *MemoryScoreStore) SetLegacyJSON(data []byte) { s.legacy = slices.Clone(data) }

func (g *Game) scoreStore() ScoreStore {
	if g.Scores == nil {
		g.Scores = newScoreStore()
	}
	return g.Scores
}

// DefaultSaveData keeps the empty host view useful to tests and callers that
// supply their own tables. LoadSaveData uses the binary's initialized CR/STs.
func DefaultSaveData() *SaveData { return &SaveData{LastDifficulty: DiffNormal} }

func saveView(native *ScoreFile) *SaveData {
	s := DefaultSaveData()
	s.Native, s.SpellHistory, s.Progress = native, native.history(), native.progress()
	s.LastReplayName = scoreName(native.Stats[0xc:0x15])
	for shot := range 6 {
		for difficulty := range 5 {
			s.Rankings[shot][difficulty] = native.rankings(shot, difficulty)
		}
	}
	s.refreshTotals()
	return s
}

func (s *SaveData) refreshTotals() {
	s.HighScore, s.TotalRuns, s.TotalClears, s.TotalExtraClears = 0, 0, 0, 0
	s.ExtraUnlocked = false
	for shot := range 6 {
		s.TotalRuns += int(s.Progress[shot].Runs)
		for difficulty := range 5 {
			s.HighScore = max(s.HighScore, int64(s.Rankings[shot][difficulty][0].Score)*10)
			if difficulty < 4 {
				s.TotalClears += int(s.Progress[shot].Clears[difficulty])
			} else {
				s.TotalExtraClears += int(s.Progress[shot].Clears[difficulty])
			}
			if difficulty >= DiffNormal && difficulty <= DiffLuna && s.Progress[shot].Clears[difficulty] > 0 {
				s.ExtraUnlocked = true
			}
		}
	}
	// Game-over and last-selection totals have no native storage slots.
	// Keep them host-only rather than inventing a CR/ST extension.
}

func (s *SaveData) nativeBytes() ([]byte, error) {
	if s.Native == nil {
		s.Native = DefaultScoreFile(0)
	}
	for shot := range 6 {
		for difficulty := range 5 {
			s.Native.setRankings(shot, difficulty, s.Rankings[shot][difficulty])
		}
	}
	s.Native.setHistory(s.SpellHistory)
	s.Native.setProgress(&s.Progress)
	if scoreName(s.Native.Stats[0xc:0x15]) != s.LastReplayName {
		clear(s.Native.Stats[0xc:0x15])
		copy(s.Native.Stats[0xc:0x14], s.LastReplayName)
	}
	return EncodeScoreFile(s.Native)
}

// importLegacy maps only existing native fields. Aggregate JSON counts are
// assigned to its last selected shot/difficulty; native counters saturate at
// 99999. Unmappable host metrics are not hidden in original reserved bytes.
func importLegacy(old *SaveData, seed uint16) *SaveData {
	s := saveView(DefaultScoreFile(seed))
	for shot := range 6 {
		for difficulty := range 5 {
			for _, r := range old.Rankings[shot][difficulty] {
				if r != (ScoreEntry{}) {
					s.Rankings[shot][difficulty] = old.Rankings[shot][difficulty]
					break
				}
			}
		}
	}
	shot := min(max(old.LastCharacter*3+old.LastShotType, 0), 5)
	diff := min(max(old.LastDifficulty, 0), 4)
	s.Rankings[shot][diff][0].Score = max(s.Rankings[shot][diff][0].Score, int32(min(old.HighScore/10, 999999999)))
	s.Progress[shot].Runs = int32(min(max(old.TotalRuns, 0), 99999))
	if diff < 4 {
		s.Progress[shot].Clears[diff] = int32(min(max(old.TotalClears, 0), 99999))
	}
	s.Progress[shot].Clears[4] = int32(min(max(old.TotalExtraClears, 0), 99999))
	if old.ExtraUnlocked && s.Progress[shot].Clears[DiffNormal] == 0 {
		s.Progress[shot].Clears[DiffNormal] = 1
	}
	s.LastReplayName = old.LastReplayName
	s.LastCharacter, s.LastShotType, s.LastDifficulty = old.LastCharacter, old.LastShotType, old.LastDifficulty
	s.TotalGameOvers = old.TotalGameOvers
	s.refreshTotals()
	return s
}

func loadSaveData(store ScoreStore) *SaveData {
	return loadSaveDataWithSeed(store, 0)
}

func loadSaveDataWithSeed(store ScoreStore, seed uint16) *SaveData {
	data, err := store.Read()
	if err == nil {
		if native, err := decodeScoreFile(data, seed); err == nil {
			return saveView(native)
		} else {
			s := saveView(DefaultScoreFile(seed))
			s.LoadError = err.Error()
			return s // an existing but damaged score never imports stale JSON
		}
	}
	s := saveView(DefaultScoreFile(seed))
	if !errors.Is(err, fs.ErrNotExist) {
		s.LoadError = err.Error()
		return s
	}
	legacy, err := store.LegacyJSON()
	if err != nil {
		return s
	}
	old := DefaultSaveData()
	if err := json.Unmarshal(legacy, old); err != nil {
		s.LoadError = err.Error()
		return s
	}
	s = importLegacy(old, seed)
	encoded, err := s.nativeBytes()
	if err == nil {
		err = store.Write(encoded)
	}
	if err != nil {
		s.LoadError = err.Error()
	}
	return s
}

func LoadSaveData() *SaveData { return loadSaveData(newScoreStore()) }

// LoadProgress supports injected stores without initializing audio/windowing.
func (g *Game) LoadProgress() {
	g.Save = loadSaveData(g.scoreStore())
	g.SpellHistory = g.Save.SpellHistory
}

func (g *Game) SaveProgress() error {
	if g.Save == nil {
		return nil
	}
	if g.SpellHistory != nil {
		g.Save.SpellHistory = g.SpellHistory
	}
	data, err := g.Save.nativeBytes()
	if err != nil {
		return err
	}
	return g.scoreStore().Write(data)
}
