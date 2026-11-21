// Package browserstore is the binary user-data boundary for the web build.
package browserstore

import "errors"

const MaxBlobBytes = 16 << 20

var ErrUnavailable = errors.New("browserstore: IndexedDB is unavailable")

// Store returns owned byte slices and acknowledges writes only after commit.
// It is synchronous to Go callers; a js event callback must use a goroutine.
type Store interface {
	Read(key string) ([]byte, error)
	Write(key string, data []byte) error
}
