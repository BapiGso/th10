package stage

import "th10/supervisor"

// DrawPorted is one frame of the draw chain without the legacy nodes
// (runChain with legacy false), for the external tests. s must have the
// STD drawer (HasSTDDrawer).
func DrawPorted(s *Stage, sv *supervisor.Supervisor) { s.runChain(sv, false) }

// HasSTDDrawer reports whether s draws its background with stddraw.
func HasSTDDrawer(s *Stage) bool { return s.bg != nil }
