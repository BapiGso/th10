//go:build js && wasm

package game

// Match persistence_wasm.go: user data lives for this browser-page session.
var browserReplays = NewMemoryReplayStore()

func newReplayStore() ReplayStore { return browserReplays }
