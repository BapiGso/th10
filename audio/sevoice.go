package audio

// seOutput is where FUN_0043ddf0's voices sound: one fixed voice per
// sound-table id, like the original's DirectSound secondary buffer per id.
// play stops and rewinds that id's voice and starts it again from the first
// sample with the given per-channel gains (table volume times the
// DirectSound pan law); nothing is allocated per play. master is the
// independently adjustable SE volume (Option / SetSEVolume).
//
// Desktop mixes the voices in Go into one Ebiten player (semixer.go); the
// browser hands them to Web Audio (seweb_js.go), which mixes on its own audio
// thread instead of WASM's single main thread.
type seOutput interface {
	play(id int, pcm []byte, left, right float64)
	setMaster(v float64)
}
