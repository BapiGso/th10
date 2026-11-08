package menus

import (
	"fmt"
	"math"
	"slices"
	"strings"

	"th10/anm"
	"th10/audio"
	"th10/game"
	"th10/input"
	"th10/render"
	"th10/supervisor"
	"th10/zun"

	"github.com/hajimehoshi/ebiten/v2"
)

// MusicRoom is TitleInf state 14, FUN_00433ef0. Tracks and comments are
// original data; the timelines are title.anm/text.anm World VMs.
type MusicRoom struct {
	core           nativeMenu
	back           game.Scene
	save           *game.SaveData
	tracks         []MusicTrack
	selection      int
	first          int // +0x58d8: ten-row window
	state          int
	timer          zun.Timer
	current        int // playing BGM ordinal, -1 after the warning stops it
	commentTrack   int // +0x690: comments follow the last confirmed song
	commentAt      int
	commentContent [8]string
	warning        bool // +0x694: second confirmation permits an unseen track
	labels         []*anm.VM
	comments       [8]*anm.VM
	initialized    bool
	closed         bool
	Background     func(*supervisor.Supervisor) // optional t10 persistent title backdrop
}

func NewMusicRoom(back game.Scene) *MusicRoom {
	s := &MusicRoom{back: back, current: audio.BGMTitle}
	s.timer.Set(1) // FUN_0042c5c0's entry has already passed the calc-tail tick
	return s
}

func (s *MusicRoom) init(g *game.Game) error {
	if err := s.core.init(g, 0x5f, 0x67); err != nil { // 0x433f55/0x433f83
		return err
	}
	var err error
	s.tracks, err = musicFiles()
	if err != nil {
		return err
	}
	s.save = g.Save
	g.BindMusicProgress(nil)
	s.labels = make([]*anm.VM, len(s.tracks))
	for i := range s.comments {
		s.comments[i] = s.core.spawnText(0x27 + i) // 0x4340b0, text.anm
	}
	s.initialized = true
	return nil
}

func (s *MusicRoom) Update(g *game.Game) error {
	if s.closed {
		return nil
	}
	if !s.initialized {
		if err := s.init(g); err != nil {
			return err
		}
	}
	switch s.state {
	case 0: // 0x434103..0x434343: two labels per frame, then state 1
		if s.timer.Cur < 10 {
			for i := int(s.timer.Cur)*2 - 2; i < int(s.timer.Cur)*2 && i < len(s.tracks); i++ {
				if i < 0 {
					continue
				}
				v := s.core.spawnTitle(0xad + i) // the labels are in title.anm, not text.anm
				s.labels[i] = v
				label := s.tracks[i].Title
				if !s.TrackUnlocked(i) {
					label = fmt.Sprintf("No.%d  %s", i+1, strings.Repeat("\uff1f", 11)) // PE 0x46ecf4
				}
				render.WriteVMText(v, label, render.TextStyle{Size: 16, Color: 0xffffff})
				s.moveLabel(i, v)
			}
		} else {
			s.state, s.commentAt = 1, 0
			s.timer.Reset()
		}
	case 1:
		s.revealComment()
		if s.timer.Cur > 4 {
			s.state = 2 // 0x4343f6..0x43440a
			s.timer.Reset()
		}
	case 2:
		s.revealComment()
		previous := s.selection
		if menuRepeat(&input.Global, input.KeyUp) {
			s.selection = wrapSelection(s.selection-1, len(s.tracks))
		}
		if menuRepeat(&input.Global, input.KeyDown) {
			s.selection = wrapSelection(s.selection+1, len(s.tracks))
		}
		if previous != s.selection {
			menuSE(g, 12)
			s.warning = false // 0x434667; the playing song/comments remain unchanged
			if s.selection < s.first {
				s.first = s.selection
			} else if s.selection >= s.first+10 {
				s.first = s.selection - 9
			}
			s.moveLabels()
		}
		if input.Global.JustPressed(input.KeyOk) {
			if err := s.Play(g, s.selection); err != nil {
				return err
			}
		}
		if input.Global.JustPressed(input.KeyCancel) {
			for _, v := range s.labels {
				s.core.interrupt(v, 1)
			}
			for _, v := range s.comments {
				s.core.interrupt(v, 1)
			}
			s.state = 3
			s.timer.Reset()
			menuSE(g, 11)
		}
	case 3:
		if s.timer.Cur >= 10 { // 0x43494e
			s.Close()
			if a := g.Audio(); a != nil {
				a.PlayBGM(audio.BGMTitle) // 0x4349a1..0x4349b1
			}
			g.RecordMusicPlayed(0)
			g.GoTo(s.back)
			return nil
		}
	}
	s.timer.Tick(1) // FUN_0042cdf0's calc tail
	s.core.calc()
	return nil
}

func (s *MusicRoom) TrackUnlocked(index int) bool {
	return index >= 0 && index < len(s.tracks) && s.save != nil && s.save.Native.MusicUnlocked(index)
}

func (s *MusicRoom) Tracks() []MusicTrack { return slices.Clone(s.tracks) }

func (s *MusicRoom) Play(g *game.Game, index int) error {
	if index < 0 || index >= len(s.tracks) {
		return fmt.Errorf("music room: track %d out of range", index)
	}
	if index != s.commentTrack {
		s.warning = false
	}
	s.commentTrack, s.commentAt = index, 0
	s.timer.Reset()
	for _, v := range s.comments {
		s.core.interrupt(v, 3) // 0x434690..0x4346d4
	}
	if !s.TrackUnlocked(index) && !s.warning {
		s.warning = true // 0x434745..0x43478f
		s.Stop(g)        // original sound command 3, not a permanently disabled entry
		return nil
	}
	s.current = s.tracks[index].BGMIndex
	if a := g.Audio(); a != nil {
		a.PlayBGMPreview(s.current) // 0x43479b; do not unlock the previewed song
	}
	g.RecordMusicPlayed(0) // 0x4347e4 writes only the title-song byte
	s.warning = false
	return nil
}

func (s *MusicRoom) Stop(g *game.Game) {
	s.current = -1
	if a := g.Audio(); a != nil {
		a.StopBGM() // FUN_0043e460 command 3 at 0x43477e
	}
}

func (s *MusicRoom) revealComment() {
	if s.timer.Cur%2 != 0 || s.commentAt >= len(s.comments) {
		return
	}
	i := s.commentAt
	text, colour := s.tracks[s.commentTrack].Comments[i], uint32(0xffffff)
	if !s.TrackUnlocked(s.commentTrack) && s.warning {
		text, colour = musicWarning[i], 0x8080ff // 0x434471..0x43448b
	}
	render.WriteVMText(s.comments[i], text, render.TextStyle{Size: 16, Color: colour})
	s.commentContent[i] = text
	s.core.interrupt(s.comments[i], 2)
	s.commentAt++
}

func (s *MusicRoom) moveLabels() {
	// 0x434547..0x434661: Y=96+(row-first)*20, selected X=64-8.
	for i, v := range s.labels {
		s.moveLabel(i, v)
	}
}

func (s *MusicRoom) moveLabel(i int, v *anm.VM) {
	if v == nil {
		return
	}
	if i >= s.first && i < s.first+10 {
		v.Flags |= anm.FlagDraw
	} else {
		v.Flags &^= anm.FlagDraw
	}
	target := [3]float32{64, float32(96 + (i-s.first)*20), 0}
	label := int16(3)
	if i == s.selection {
		target[0] -= 8 // double at 0x470c98 is exactly 8
		label = 2
	}
	if math.Abs(float64(v.Pos[1]-target[1])) < 40 { // PE 0x470c64
		s.core.move(v, target)
	} else {
		v.Pos = target
		delete(s.core.positions, v)
	}
	s.core.interrupt(v, label)
}

func (s *MusicRoom) Close() { s.closed = true; s.core.close() }

func (s *MusicRoom) Draw(screen *ebiten.Image) {
	sv := supervisor.Get()
	s.DrawPorted(sv)
	if screen != nil && sv.Frame != nil {
		screen.DrawImage(sv.Frame, nil)
	}
}

func (s *MusicRoom) DrawPorted(sv *supervisor.Supervisor) {
	s.core.draw(sv, s.Background, nil) // Music Room has no extra native draw callback
}
