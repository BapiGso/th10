package game

// EndingInf and Music Room read these bytes relative to ST. The native
// score object puts ST at +0x1d86c, after the seven CR sections.
const (
	endingSeenAt = 0x16 // FUN_0040b560, object +0x1d882
	staffSeenAt  = 0x22 // FUN_0040b560, object +0x1d88e
	musicSeenAt  = 0x26 // FUN_00420b10/00420b90, object +0x1d892
	MusicCount   = 18
)

func (s *ScoreFile) EndingSeen(index int) bool {
	return s != nil && index >= 0 && index < 12 && s.Stats[endingSeenAt+index] != 0
}

// MarkEndingSeen is FUN_0040b560 at 0x40b6d5. Good endings also mark Staff
// on entry, before it runs, rather than waiting for the credit roll to end.
func (s *ScoreFile) MarkEndingSeen(index int) {
	if s != nil && index >= 0 && index < 12 {
		s.Stats[endingSeenAt+index] = 1
		if index >= 6 {
			s.Stats[staffSeenAt] = 1
		}
	}
}

func (s *ScoreFile) StaffSeen() bool { return s != nil && s.Stats[staffSeenAt] != 0 }

func (s *ScoreFile) MarkStaffSeen() {
	if s != nil {
		s.Stats[staffSeenAt] = 1
	}
}

func (s *ScoreFile) MusicUnlocked(index int) bool {
	return s != nil && index >= 0 && index < MusicCount && s.Stats[musicSeenAt+index] != 0
}

// MarkMusicPlayed is the unlock write at 0x420b4c/0x420bd7. Callers supply
// the original music-room ordinal, which is not a WAV filename number.
func (s *ScoreFile) MarkMusicPlayed(index int) {
	if s != nil && index >= 0 && index < MusicCount {
		s.Stats[musicSeenAt+index] = 1
	}
}

// BindMusicProgress wires the original BGM unlock write without moving
// scene switching into audio. Playback keeps t5's shared-save isolation.
func (g *Game) BindMusicProgress(state *GameState) {
	if a := g.Audio(); a != nil {
		a.SetBGMObserver(func(index int) {
			if state == nil || state.ReplayMode != 1 {
				g.RecordMusicPlayed(index)
			}
		})
	}
}

func (g *Game) RecordMusicPlayed(index int) {
	if g.Save == nil || index < 0 || index >= MusicCount {
		return
	}
	if g.Save.Native == nil {
		g.Save.Native = DefaultScoreFile(0)
	}
	if g.Save.Native.MusicUnlocked(index) {
		return
	}
	g.Save.Native.MarkMusicPlayed(index) // 0x420b4c/0x420bd7
	if err := g.SaveProgress(); err != nil {
		g.Save.LoadError = err.Error()
	}
}
