//go:build !js

package browserstore

import (
	"errors"
	"testing"
)

func TestDesktopDoesNotOpenBrowserStorage(t *testing.T) {
	store, err := Open()
	if store != nil || !errors.Is(err, ErrUnavailable) {
		t.Fatal("desktop unexpectedly selected a browser backend")
	}
}
