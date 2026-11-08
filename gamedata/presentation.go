package gamedata

import (
	"strings"

	"th10/anm"
)

// PresentationANM uses th10c.dat's traditional-Chinese texture variant when
// embedded. The original file remains available to logic and resource tests.
// The patch changes pictures, not the GUI's FUN_0043ee30 timelines.
func PresentationANM(name string) (*anm.File, error) {
	p := "anm/" + strings.TrimSuffix(name, ".anm") + ".ant"
	if data, err := ReadFile(p); err == nil {
		return anm.Load(data)
	}
	return ANM(name)
}
