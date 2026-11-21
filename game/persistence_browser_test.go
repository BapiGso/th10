package game

import (
	"bytes"
	"encoding/binary"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"testing"
	"th10/replay"
)

type fakeBinaryStore struct {
	files  map[string][]byte
	writes []string
	fail   error
}

func newFakeBinaryStore() *fakeBinaryStore { return &fakeBinaryStore{files: map[string][]byte{}} }

func (s *fakeBinaryStore) Read(key string) ([]byte, error) {
	data, exists := s.files[key]
	if !exists {
		return nil, fs.ErrNotExist
	}
	return slices.Clone(data), nil
}

func (s *fakeBinaryStore) Write(key string, data []byte) error {
	if s.fail != nil {
		return s.fail
	}
	s.files[key] = slices.Clone(data)
	s.writes = append(s.writes, key)
	return nil
}

func TestBinaryConfigRoundTripAndCommitTiming(t *testing.T) {
	store := newFakeBinaryStore()
	cfg, err := loadBinaryConfig(store)
	if err != nil || len(store.writes) != 0 {
		t.Fatal("loading defaults wrote the database")
	}
	cfg.Native.Raw[0x22] = 2
	cfg.Native.SetFlags(0x91827364)
	for range 100 {
		cfg.BGMVolume, cfg.SEVolume = 0.65, 0.35
	}
	if len(store.writes) != 0 {
		t.Fatal("editing options wrote before SaveConfig")
	}
	if err := saveBinaryConfig(store, cfg); err != nil {
		t.Fatal(err)
	}
	loaded, err := loadBinaryConfig(store)
	if err != nil || len(store.writes) != 1 || loaded.BGMVolume != .65 || loaded.SEVolume != .35 || !loaded.Hint || loaded.Native.Raw[0x22] != 2 || loaded.Native.Flags() != 0x91827364 || !bytes.Equal(loaded.NativeBytes(), cfg.NativeBytes()) {
		t.Fatalf("binary cfg did not reload: %v", err)
	}
	store.files[browserConfigKey] = []byte("invalid cfg")
	if _, err := loadBinaryConfig(store); err == nil || len(store.writes) != 1 {
		t.Fatal("bad cfg was accepted or overwritten on load")
	}
}

func TestBinarySaveCheckpointsNotPerFrame(t *testing.T) {
	store := newFakeBinaryStore()
	g := &Game{Scores: binaryScoreStore{store}, Replays: binaryReplayStore{store}}
	g.LoadProgress()
	if len(store.writes) != 0 {
		t.Fatal("fresh database wrote on load")
	}
	state := g.NewState(CharReimu, ShotC, DiffEasy)
	g.BeginRun(state)
	if len(store.writes) != 1 {
		t.Fatal("existing new-run checkpoint did not commit once")
	}
	for range 600 {
		state.RecordPlayFrame()
	}
	if len(store.writes) != 1 {
		t.Fatal("play-time accounting became a per-frame database write")
	}
	g.RecordMusicPlayed(2)
	for range 100 {
		g.RecordMusicPlayed(2)
	}
	if len(store.writes) != 2 {
		t.Fatal("music unlock wrote more than once")
	}
	g.MarkGameOver(state)
	if len(store.writes) != 3 {
		t.Fatal("existing game-over checkpoint did not commit")
	}
	state.ReplayMode = 1
	g.BeginRun(state)
	g.MarkGameOver(state)
	g.MarkClear(state)
	if len(store.writes) != 3 {
		t.Fatal("playback wrote shared progress")
	}
	if slots := g.ListReplays(); len(slots) != 25 || slots[0].Number != 1 {
		t.Fatal("binary backend changed replay slots")
	}
	if len(store.writes) != 3 {
		t.Fatal("replay listing wrote the database")
	}
	if err := g.SaveReplay(25, storedReplay()); err != nil || len(store.writes) != 4 || store.writes[3] != "replay/th10_25.rpy" {
		t.Fatalf("replay was not written only on explicit save: %v", err)
	}
}

func TestScoreImportRejectsWithoutOverwrite(t *testing.T) {
	store := newFakeBinaryStore()
	g := &Game{Scores: binaryScoreStore{store}}
	valid, err := EncodeScoreFile(DefaultScoreFile(123))
	if err != nil {
		t.Fatal(err)
	}
	if err := g.ImportScore(valid); err != nil {
		t.Fatal(err)
	}
	badLength := slices.Clone(valid)
	binary.LittleEndian.PutUint32(badLength[4:], uint32(len(valid)+1))
	payload := encodeScorePayload(DefaultScoreFile(123))
	badChecksum := slices.Clone(payload)
	badChecksum[0x10] ^= 1
	duplicate := slices.Clone(payload)
	copy(duplicate[scoreBlockSize:2*scoreBlockSize], duplicate[:scoreBlockSize])
	for name, bad := range map[string][]byte{
		"empty": nil, "truncated": valid[:10], "oversized": make([]byte, maxScoreBytes+1),
		"length": badLength, "trailing": append(slices.Clone(valid), 0),
		"checksum": scoreImportPayload(t, badChecksum), "duplicate": scoreImportPayload(t, duplicate),
		"missing": scoreImportPayload(t, payload[:len(payload)-scoreStatsSize]),
	} {
		t.Run(name, func(t *testing.T) {
			if err := g.ImportScore(bad); err == nil {
				t.Fatal("accepted malformed score")
			}
			if len(store.writes) != 1 || !bytes.Equal(store.files[browserScoreKey], valid) {
				t.Fatal("invalid import overwrote the existing save")
			}
		})
	}
	store.fail = errors.New("quota exceeded")
	if err := g.ImportScore(valid); err == nil || !bytes.Equal(store.files[browserScoreKey], valid) {
		t.Fatal("failed commit was acknowledged or damaged the old save")
	}
}

func scoreImportPayload(t *testing.T, payload []byte) []byte {
	t.Helper()
	compressed := replay.CompressLZSS(payload)
	replay.EncryptBlocks(compressed, scoreKeyInc, 0x10, scoreKey)
	header := DefaultScoreFile(123).Header
	binary.LittleEndian.PutUint32(header[4:], uint32(scoreHeaderSize+len(compressed)))
	binary.LittleEndian.PutUint32(header[0x10:], uint32(len(compressed)))
	binary.LittleEndian.PutUint32(header[0x14:], uint32(len(payload)))
	return append(header[:], compressed...)
}

func TestReplayImportRejectsWithoutOverwrite(t *testing.T) {
	store := newFakeBinaryStore()
	g := &Game{Replays: binaryReplayStore{store}}
	valid, err := replay.Encode(storedReplay())
	if err != nil {
		t.Fatal(err)
	}
	if err := g.ImportReplay(1, valid); err != nil {
		t.Fatal(err)
	}
	unsafe := storedReplay()
	unsafe.Global.Character = 99
	unsafeBytes, err := replay.Encode(unsafe)
	if err != nil {
		t.Fatal(err)
	}
	for _, bad := range [][]byte{nil, []byte("invalid"), valid[:35], unsafeBytes, make([]byte, maxReplayBytes+1)} {
		if err := g.ImportReplay(1, bad); err == nil || len(store.writes) != 1 || !bytes.Equal(store.files["replay/th10_01.rpy"], valid) {
			t.Fatal("invalid replay import replaced an existing slot")
		}
	}
	if err := g.ImportReplay(26, valid); err == nil || len(store.writes) != 1 {
		t.Fatal("invalid slot wrote to the database")
	}
	got, err := g.ExportReplay(1)
	if err != nil || !bytes.Equal(got, valid) || len(store.writes) != 1 {
		t.Fatal("export changed bytes or caused a save")
	}
}

func TestBrowserAdaptersReloadAndExportWithoutWrite(t *testing.T) {
	store := newFakeBinaryStore()
	g := &Game{Scores: binaryScoreStore{store}}
	initial, err := g.ExportScore()
	if err != nil || len(store.writes) != 0 {
		t.Fatal("exporting defaults wrote a new save")
	}
	if _, err := DecodeScoreFile(initial); err != nil {
		t.Fatal(err)
	}
	g.LoadProgress()
	g.Save.LastReplayName = "BROWSER"
	if err := g.SaveProgress(); err != nil {
		t.Fatal(err)
	}
	reloaded := &Game{Scores: binaryScoreStore{store}}
	reloaded.LoadProgress()
	if reloaded.Save.LastReplayName != "BROWSER" || len(store.writes) != 1 {
		t.Fatal("new application instance did not reload the committed save")
	}
	exported, err := reloaded.ExportScore()
	if err != nil || !bytes.Equal(exported, store.files[browserScoreKey]) || len(store.writes) != 1 {
		t.Fatal("export altered committed bytes or wrote again")
	}
}

func TestImportOriginalFilesUnchanged(t *testing.T) {
	store := newFakeBinaryStore()
	g := &Game{Scores: binaryScoreStore{store}, Replays: binaryReplayStore{store}}
	for _, file := range []struct {
		path, key  string
		importFile func([]byte) error
	}{
		{filepath.Join("..", ".tmp", "[th10]", "scoreth10.dat"), browserScoreKey, g.ImportScore},
		{filepath.Join("..", ".tmp", "[th10]", "replay", "th10_05.rpy"), "replay/th10_05.rpy", func(data []byte) error { return g.ImportReplay(5, data) }},
	} {
		data, err := os.ReadFile(file.path)
		if errors.Is(err, fs.ErrNotExist) {
			t.Logf("original sample unavailable: %s", file.path)
			continue
		}
		if err != nil {
			t.Fatal(err)
		}
		if err := file.importFile(data); err != nil || !bytes.Equal(store.files[file.key], data) {
			t.Fatalf("original bytes were rejected or changed: %s: %v", file.path, err)
		}
	}
}
