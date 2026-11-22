package game

import (
	"bytes"
	"encoding/json"
	"errors"
	"io/fs"
	"testing"
)

func TestScoreJSONMigrationOnlyWhenAbsent(t *testing.T) {
	mem := memoryStore{}
	store := binaryScoreStore{store: mem}
	old := DefaultSaveData()
	old.HighScore, old.TotalRuns, old.TotalClears, old.TotalExtraClears = 90000000, 7, 2, 1
	old.LastCharacter, old.LastShotType, old.LastDifficulty = 1, 2, DiffNormal
	old.LastReplayName, old.ExtraUnlocked = "MIGRATED", true
	old.Rankings[5][1][0] = ScoreEntry{Score: 9000000, Name: "OLD", Stage: 6, Continues: 2, Timestamp: 123, SlowRate: 0.5}
	legacy, err := json.Marshal(old)
	if err != nil {
		t.Fatal(err)
	}
	mem["save.json"] = legacy
	g := &Game{Scores: store}
	g.LoadProgress()
	raw, err := store.Read()
	if err != nil || g.Save.HighScore != old.HighScore || g.Save.Progress[5].Runs != 7 || g.Save.Progress[5].Clears[1] != 2 || g.Save.Progress[5].Clears[4] != 1 || g.Save.LastReplayName != "MIGRATED" {
		t.Fatalf("legacy import differs: %v", err)
	}
	if kept, err := store.LegacyJSON(); err != nil || !bytes.Equal(kept, legacy) {
		t.Fatal("migration rewrote legacy JSON")
	}
	mem["save.json"] = []byte(`{"high_score":9999999990,"total_runs":99}`)
	fresh := &Game{Scores: store}
	fresh.LoadProgress()
	if fresh.Save.HighScore != old.HighScore || fresh.Save.Progress[5].Runs != 7 || fresh.Save.Rankings[5][1][0].Name != "OLD" {
		t.Fatal("existing score reimported stale JSON")
	}
	if err := fresh.SaveProgress(); err != nil {
		t.Fatal(err)
	}
	again, err := store.Read()
	if err != nil || !bytes.Equal(raw, again) {
		t.Fatal("loaded native save was not byte-identical after saving")
	}
	if err := store.Write([]byte("damaged score")); err != nil {
		t.Fatal(err)
	}
	fresh.LoadProgress()
	if fresh.Save.LoadError == "" || fresh.Save.HighScore == 9999999990 {
		t.Fatal("damaged existing score silently fell back to stale JSON")
	}
}

func TestMemoryScoreBytesDoNotAlias(t *testing.T) {
	s := NewMemoryScoreStore()
	if _, err := s.Read(); !errors.Is(err, fs.ErrNotExist) {
		t.Fatal("missing score did not report not-exist")
	}
	b := []byte{1, 2, 3}
	if err := s.Write(b); err != nil {
		t.Fatal(err)
	}
	b[0] = 4
	a, err := s.Read()
	if err != nil || a[0] != 1 {
		t.Fatal("write aliases the caller's bytes")
	}
	a[0] = 9
	c, err := s.Read()
	if err != nil || c[0] != 1 {
		t.Fatal("read aliases the stored bytes")
	}
	if err := s.Write(make([]byte, maxScoreBytes+1)); err == nil {
		t.Fatal("oversized score accepted")
	}
}
