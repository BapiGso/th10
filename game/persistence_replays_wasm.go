//go:build js && wasm

package game

func newReplayStore() ReplayStore { return binaryReplayStore{store: browserAccess{}} }
