package datfs

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// ErrNoInstall means Locate found no TH10 install.
var ErrNoInstall = errors.New("datfs: th10.dat not found; set TH10_DIR to your Touhou 10 (v1.00a) directory")

// Locate finds a TH10 install for development tools and tests (the game
// itself only reads embedded copies): $TH10_DIR, else a ".tmp/[th10]"
// directory found walking up from the working directory.
func Locate() (string, error) {
	if dir := os.Getenv("TH10_DIR"); dir != "" {
		if hasDat(dir) {
			return dir, nil
		}
		return "", fmt.Errorf("datfs: TH10_DIR=%q has no th10.dat", dir)
	}
	d, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		if dir := filepath.Join(d, ".tmp", "[th10]"); hasDat(dir) {
			return dir, nil
		}
		parent := filepath.Dir(d)
		if parent == d {
			return "", ErrNoInstall
		}
		d = parent
	}
}

func hasDat(dir string) bool {
	st, err := os.Stat(filepath.Join(dir, "th10.dat"))
	return err == nil && !st.IsDir()
}
