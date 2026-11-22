package gamedata

import (
	"sync"

	"th10/anm"
	"th10/datfs"
)

// sharedKey names one process-wide parsed file.
type sharedKey struct {
	name                  string
	presentation, ordinal bool
}

type sharedFile struct {
	once sync.Once
	f    *anm.File
	err  error
}

var sharedFiles sync.Map // sharedKey -> *sharedFile

// SharedANM is ANM(name) parsed once per process: every caller gets the same
// *anm.File, so renderers that key GPU sheets by file upload its textures
// once. The file and its textures must be treated as read-only; ANM still
// returns a private copy for code that mutates one.
func SharedANM(name string) (*anm.File, error) {
	return shared(sharedKey{name: name})
}

// SharedPresentationANM is PresentationANM(name) parsed once per process,
// with SharedANM's read-only contract. Without a th10c.dat variant it is
// SharedANM(name) itself.
func SharedPresentationANM(name string) (*anm.File, error) {
	return shared(sharedKey{name: name, presentation: true})
}

// SharedOrdinalANM is OrdinalScripts of SharedPresentationANM(name) when
// presentation is set, else of SharedANM(name), made once per process:
// FUN_00447470's runtime script table as text.anm's callers address it.
func SharedOrdinalANM(name string, presentation bool) (*anm.File, error) {
	return shared(sharedKey{name: name, presentation: presentation, ordinal: true})
}

// MustANM returns f, panicking on err; for files every build embeds, as in
// MustANM(SharedANM("bullet.anm")).
func MustANM(f *anm.File, err error) *anm.File {
	if err != nil {
		panic("gamedata: embedded ANM: " + err.Error())
	}
	return f
}

func shared(k sharedKey) (*anm.File, error) {
	v, _ := sharedFiles.LoadOrStore(k, &sharedFile{})
	s := v.(*sharedFile)
	s.once.Do(func() {
		switch {
		case k.ordinal:
			base := k
			base.ordinal = false
			var f *anm.File
			if f, s.err = shared(base); s.err == nil {
				s.f = f.OrdinalScripts()
			}
		case k.presentation:
			if data, err := readANM(presentationPath(k.name)); err == nil {
				s.f, s.err = anm.Load(data) // readANM's bytes are never handed out mutable
				return
			}
			s.f, s.err = shared(sharedKey{name: k.name})
		default:
			var data []byte
			if data, s.err = readANM(datfs.EmbedPath(k.name)); s.err == nil {
				s.f, s.err = anm.Load(data)
			}
		}
	})
	return s.f, s.err
}
