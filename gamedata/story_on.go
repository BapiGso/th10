//go:build !th10nostory

package gamedata

// Story reports whether stage dialogue and the endings are built in. The
// th10nostory tag removes both: the original's fan-work guidelines ask that
// its endings not be published. Stage logic still runs every MSG script
// (signals, boss BGM, stage clear), only silently; see scene/dialog.
const Story = true
