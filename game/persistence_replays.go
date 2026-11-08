package game

import (
	"errors"
	"fmt"
	"io/fs"
	"slices"
	"th10/replay"
)

// ReplaySlots is FUN_004236f0's 0x19 slots, th10_01.rpy through th10_25.rpy.
const ReplaySlots = 25

const maxReplayBytes = 16 << 20 // host-side bound on untrusted user data, not an original format limit

// ReplayStore is the only user-replay I/O boundary. Resources are still
// embedded; desktop files and browser storage are implemented in persistence*.
type ReplayStore interface {
	Read(slot int) ([]byte, error)
	Write(slot int, data []byte) error
}

type ReplaySlot struct {
	Number int
	Replay *replay.Replay // metadata only; LoadReplay fetches the full input stream
	Error  error          // nil for a vacant slot; corrupt slots remain visible
}

func ReplayFilename(slot int) (string, error) {
	if slot < 1 || slot > ReplaySlots {
		return "", fmt.Errorf("game: replay slot %d outside 1..%d", slot, ReplaySlots)
	}
	return fmt.Sprintf("th10_%02d.rpy", slot), nil
}

func (g *Game) replayStore() ReplayStore {
	if g.Replays == nil {
		g.Replays = newReplayStore()
	}
	return g.Replays
}

func (g *Game) LoadReplay(slot int) (*replay.Replay, error) {
	if _, err := ReplayFilename(slot); err != nil {
		return nil, err
	}
	data, err := g.replayStore().Read(slot)
	if err != nil {
		return nil, err
	}
	r, err := replay.Decode(data)
	if err != nil {
		return nil, err
	}
	if err := r.ValidatePlayback(); err != nil {
		return nil, err
	}
	return r, nil
}

func (g *Game) SaveReplay(slot int, r *replay.Replay) error {
	if _, err := ReplayFilename(slot); err != nil {
		return err
	}
	if err := r.ValidatePlayback(); err != nil {
		return err
	}
	data, err := replay.Encode(r)
	if err != nil {
		return err
	}
	return g.replayStore().Write(slot, data)
}

func (g *Game) ListReplays() []ReplaySlot {
	slots := make([]ReplaySlot, ReplaySlots)
	for i := range slots {
		slots[i].Number = i + 1
		r, err := g.LoadReplay(i + 1)
		if err != nil && !isReplayMissing(err) {
			slots[i].Error = err
		} else {
			if r != nil {
				slots[i].Replay = &replay.Replay{Header: r.Header, Global: r.Global, Trailer: r.Trailer}
			}
		}
	}
	return slots
}

func isReplayMissing(err error) bool { return errors.Is(err, fs.ErrNotExist) }

// MemoryReplayStore is the current browser persistence policy and an
// injectable backend for tests; returned data never aliases stored bytes.
type MemoryReplayStore struct{ files map[int][]byte }

func NewMemoryReplayStore() *MemoryReplayStore {
	return &MemoryReplayStore{files: make(map[int][]byte)}
}

func (s *MemoryReplayStore) Read(slot int) ([]byte, error) {
	if _, err := ReplayFilename(slot); err != nil {
		return nil, err
	}
	data, ok := s.files[slot]
	if !ok {
		return nil, fs.ErrNotExist
	}
	return slices.Clone(data), nil
}

func (s *MemoryReplayStore) Write(slot int, data []byte) error {
	if _, err := ReplayFilename(slot); err != nil {
		return err
	}
	if len(data) > maxReplayBytes {
		return fmt.Errorf("game: replay exceeds %d bytes", maxReplayBytes)
	}
	if s.files == nil {
		s.files = make(map[int][]byte)
	}
	s.files[slot] = slices.Clone(data)
	return nil
}
