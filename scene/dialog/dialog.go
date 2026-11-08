// Package dialog parses MSG timelines and applies their original ANM effects.
// msgvm.go owns timing, presentation.go the World VMs. Text is rasterized
// from embedded fonts; th10c.dat overlays only traditional-Chinese text.
package dialog

// Line is retained for scripts.go's legacy hand-written text API. Runtime
// MSG presentation no longer uses expression suffixes or these portrait keys.
type Line struct {
	Speaker    string
	Expression string
	Text       string
	IsRight    bool
}
