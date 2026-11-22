package game

import (
	"errors"
	"fmt"
	"io/fs"
	"th10/browserstore"
)

const (
	browserScoreKey  = "score.dat"
	browserConfigKey = "th10.cfg"
)

type binaryScoreStore struct{ store browserstore.Store }

func (s binaryScoreStore) Read() ([]byte, error) {
	data, err := s.store.Read(browserScoreKey)
	if err == nil && len(data) > maxScoreBytes {
		return nil, fmt.Errorf("game: score exceeds %d bytes", maxScoreBytes)
	}
	return data, err
}

func (s binaryScoreStore) Write(data []byte) error {
	if len(data) > maxScoreBytes {
		return fmt.Errorf("game: score exceeds %d bytes", maxScoreBytes)
	}
	return s.store.Write(browserScoreKey, data)
}

func (s binaryScoreStore) LegacyJSON() ([]byte, error) { return s.store.Read("save.json") }

type binaryReplayStore struct{ store browserstore.Store }

func (s binaryReplayStore) Read(slot int) ([]byte, error) {
	name, err := ReplayFilename(slot)
	if err != nil {
		return nil, err
	}
	data, err := s.store.Read("replay/" + name)
	if err == nil && len(data) > maxReplayBytes {
		return nil, fmt.Errorf("game: replay exceeds %d bytes", maxReplayBytes)
	}
	return data, err
}

func (s binaryReplayStore) Write(slot int, data []byte) error {
	name, err := ReplayFilename(slot)
	if err != nil {
		return err
	}
	if len(data) > maxReplayBytes {
		return fmt.Errorf("game: replay exceeds %d bytes", maxReplayBytes)
	}
	return s.store.Write("replay/"+name, data)
}

func loadBinaryConfig(store browserstore.Store) (*Config, error) {
	cfg := DefaultConfig()
	return cfg, applyNativeConfig(cfg, store)
}

// applyNativeConfig sets cfg from the stored th10.cfg; a missing file
// leaves cfg as it is.
func applyNativeConfig(cfg *Config, store browserstore.Store) error {
	data, err := store.Read(browserConfigKey)
	if errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	native, err := DecodeNativeConfig(data)
	if err != nil {
		return err
	}
	cfg.Native = native
	cfg.BGMVolume, cfg.SEVolume = float64(native.Raw[0x20])/100, float64(native.Raw[0x21])/100
	return nil
}

func saveBinaryConfig(store browserstore.Store, cfg *Config) error {
	if cfg == nil {
		return nil
	}
	return store.Write(browserConfigKey, cfg.NativeBytes())
}
