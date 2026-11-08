//go:build !js && !wasm

package game

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
)

type fileReplayStore struct{ dir string }

func newReplayStore() ReplayStore {
	return &fileReplayStore{dir: filepath.Join(dataDir(), "replay")}
}

func (s *fileReplayStore) path(slot int) (string, error) {
	name, err := ReplayFilename(slot)
	if err != nil {
		return "", err
	}
	return filepath.Join(s.dir, name), nil
}

func (s *fileReplayStore) Read(slot int) ([]byte, error) {
	path, err := s.path(slot)
	if err != nil {
		return nil, err
	}
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer func() { _ = f.Close() }()
	data, err := io.ReadAll(io.LimitReader(f, maxReplayBytes+1))
	if err != nil {
		return nil, err
	}
	if len(data) > maxReplayBytes {
		return nil, fmt.Errorf("game: replay exceeds %d bytes", maxReplayBytes)
	}
	return data, nil
}

// Write stores FUN_00429b60's bytes under the original slot filename. The
// temporary file is created in the same user-data directory, synced and
// renamed, so a failed write cannot truncate an existing replay.
func (s *fileReplayStore) Write(slot int, data []byte) error {
	path, err := s.path(slot)
	if err != nil {
		return err
	}
	if len(data) > maxReplayBytes {
		return fmt.Errorf("game: replay exceeds %d bytes", maxReplayBytes)
	}
	if err := os.MkdirAll(s.dir, 0o755); err != nil {
		return err
	}
	f, err := os.CreateTemp(s.dir, ".th10-replay-*.tmp")
	if err != nil {
		return err
	}
	tmp, saved := f.Name(), false
	defer func() {
		_ = f.Close()
		if !saved {
			_ = os.Remove(tmp)
		}
	}()
	if _, err := f.Write(data); err != nil {
		return err
	}
	if err := f.Sync(); err != nil {
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	if err := os.Rename(tmp, path); err != nil {
		return err
	}
	saved = true
	return nil
}
