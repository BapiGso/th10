package stage

// ShownScore reads the retained GUI+0x9e78 field used by FUN_00417040.
// The return value is in native stored units, not the displayed x10 units.
func (g *GUI) ShownScore() int32 { return g.shownScore }
