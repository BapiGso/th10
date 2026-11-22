package gamedata

import (
	"bytes"
	"strings"

	"th10/anm"
)

// PresentationANM uses th10c.dat's traditional-Chinese texture variant when
// embedded. The original file remains available to logic and resource tests.
// The patch changes pictures, not the GUI's FUN_0043ee30 timelines.
func PresentationANM(name string) (*anm.File, error) {
	if data, err := readANM(presentationPath(name)); err == nil {
		return anm.Load(bytes.Clone(data))
	}
	return ANM(name)
}

// presentationPath is name's th10c.dat variant in the embedded tree.
func presentationPath(name string) string {
	return "anm/" + strings.TrimSuffix(name, ".anm") + ".ant"
}
