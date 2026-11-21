//go:build !js

package browserstore

// Open is never used by the desktop game, which retains its file backends.
func Open() (Store, error) { return nil, ErrUnavailable }
