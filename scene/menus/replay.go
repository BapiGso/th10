package menus

import (
	"fmt"
	"math"
	"time"

	"th10/audio"
	"th10/game"
	"th10/input"
	"th10/replay"
	selectscene "th10/scene/select"
	"th10/scene/stage"
	"th10/supervisor"
	"th10/zun"

	"github.com/hajimehoshi/ebiten/v2"
)

// Replay is TitleInf state 12, FUN_004315c0/00431ba0. It lists t5's
// native .rpy store, loads full input only for the selected entry, and keeps
// the same scene as the playback return target.
type Replay struct {
	core       nativeMenu
	back       game.Scene
	entries    []*replayMenuEntry
	loaded     bool
	state      int
	timer      zun.Timer
	selection  int
	chosen     int
	chapter    int // native stage position 0..6
	chapters   [7]int
	selected   *replay.Replay
	playing    *stage.Playback
	status     string
	returning  bool
	closed     bool
	Source     ReplayMenuSource
	Background func(*supervisor.Supervisor)
}

type replayMenuEntry struct{ slot game.ReplaySlot }

func NewReplay(back game.Scene) *Replay {
	s := &Replay{back: back}
	s.timer.Reset()
	return s
}

func (s *Replay) init(g *game.Game) error {
	if err := s.core.init(g, 0x5e, 0x65); err != nil { // 0x431626/0x431653
		return err
	}
	if s.Source == nil {
		s.Source = g
	}
	s.entries = nil
	for _, slot := range s.Source.ListReplays() {
		s.entries = append(s.entries, &replayMenuEntry{slot: slot})
	}
	s.selection = wrapSelection(s.selection, len(s.entries))
	s.loaded, s.state = true, 1
	s.selected = nil
	s.timer.Reset()
	g.BindMusicProgress(nil)
	if s.returning {
		if a := g.Audio(); a != nil {
			a.PlayBGM(audio.BGMTitle) // FUN_0042cdf0's replay-return state
		}
		s.returning = false
	}
	return nil
}

func (s *Replay) Update(g *game.Game) error {
	if s.closed {
		return nil
	}
	if !s.loaded {
		if err := s.init(g); err != nil {
			return err
		}
	}
	in := &input.Global
	switch s.state {
	case 1:
		if s.timer.Cur > 6 { // 0x43174f..0x4317ac
			s.state = 2
			s.timer.Reset()
		}
	case 2:
		before := s.selection
		if menuRepeat(in, input.KeyUp) {
			s.selection = wrapSelection(s.selection-1, len(s.entries))
		}
		if menuRepeat(in, input.KeyDown) {
			s.selection = wrapSelection(s.selection+1, len(s.entries))
		}
		if s.selection != before {
			menuSE(g, 12)
		}
		if in.JustPressed(input.KeyCancel) {
			s.state = 5
			s.timer.Reset()
			menuSE(g, 11)
		} else if in.JustPressed(input.KeyOk) {
			if err := s.openSlot(s.selection); err != nil {
				s.status = err.Error()
			} else {
				menuSE(g, 10)
			}
		}
	case 4:
		if s.timer.Cur >= 15 { // 0x4318fd: chapter input is locked during the header move
			before := s.chapter
			if menuRepeat(in, input.KeyUp) {
				s.moveChapter(-1)
			}
			if menuRepeat(in, input.KeyDown) {
				s.moveChapter(1)
			}
			if s.chapter != before {
				menuSE(g, 12)
			}
			if in.JustPressed(input.KeyCancel) {
				s.selection, s.state, s.selected = s.chosen, 2, nil
				s.timer.Reset() // 0x43196e..0x43198b, restore the slot cursor
				menuSE(g, 11)
			} else if in.JustPressed(input.KeyOk) {
				s.state = 3
				s.timer.Reset()
			}
		}
	case 3:
		if s.timer.Cur == 2 {
			s.core.fade = newMenuFade(32) // 0x4319c9..0x4319d2
		}
		if s.timer.Cur >= 32 {
			if a := g.Audio(); a != nil {
				a.FadeOutBGM(360) // 0x431a18 passes float32 6 seconds
			}
			p, err := stage.NewPlayback(g, s.selected, s.chapters[s.chapter], selectscene.ReplayScript, s)
			if err != nil {
				s.status, s.state = err.Error(), 4
				s.timer.Reset()
				s.core.fade = nil
			} else {
				s.playing = p
				s.core.close()
				s.loaded, s.returning, s.state = false, true, 0
				g.GoTo(p) // FUN_004315c0 requests scene 12 at 0x431a32
				return nil
			}
		}
	case 5:
		if s.timer.Cur >= 6 { // 0x431a9e
			s.Close()
			g.GoTo(s.back)
			return nil
		}
	}
	s.timer.Tick(1) // FUN_0042cdf0 calc tail
	s.core.calc()
	return nil
}

func (s *Replay) openSlot(index int) error {
	if index < 0 || index >= len(s.entries) {
		return fmt.Errorf("replay menu: slot out of range")
	}
	slot := s.entries[index].slot
	if slot.Error != nil {
		return slot.Error
	}
	if slot.Replay == nil {
		return fmt.Errorf("replay menu: empty slot")
	}
	r, err := s.Source.LoadReplay(slot.Number)
	if err != nil {
		return err
	}
	if err := r.ValidatePlayback(); err != nil {
		return err
	}
	for i := range s.chapters {
		s.chapters[i] = -1
	}
	for i, ch := range r.Chapters {
		s.chapters[int(ch.Index)-1] = i // FUN_0042a200's stage-indexed pointers
	}
	s.chapter = 0
	for s.chapter < len(s.chapters) && s.chapters[s.chapter] < 0 {
		s.chapter++
	}
	s.selected, s.chosen, s.state, s.status = r, index, 4, ""
	s.timer.Reset()
	return nil
}

func (s *Replay) moveChapter(delta int) {
	for range len(s.chapters) {
		s.chapter = wrapSelection(s.chapter+delta, len(s.chapters))
		if s.chapters[s.chapter] >= 0 {
			return // 0x4318b0..0x4318f3, skip absent stages
		}
	}
}

func (s *Replay) Slots() []game.ReplaySlot {
	slots := make([]game.ReplaySlot, len(s.entries))
	for i, e := range s.entries {
		slots[i] = e.slot
	}
	return slots
}

func (s *Replay) Close() {
	s.closed = true
	s.core.close()
	s.selected = nil
	s.entries = nil
}

func replayEntryLabel(slot game.ReplaySlot) string {
	if slot.Error != nil {
		return fmt.Sprintf("No.%02d  [invalid replay]", slot.Number)
	}
	if slot.Replay == nil {
		return fmt.Sprintf("No.%02d  -- empty --", slot.Number)
	}
	g := slot.Replay.Global
	chara, diff := "Unknown", "Unknown"
	if g.Character < 2 {
		chara = []string{"Reimu", "Marisa"}[g.Character]
	}
	if g.Difficulty < 5 {
		diff = []string{"Easy", "Normal", "Hard", "Lunatic", "Extra"}[g.Difficulty]
	}
	return fmt.Sprintf("No.%02d  %-8s %s  %s%c / %s  St%d", slot.Number, g.User,
		time.Unix(int64(g.Timestamp), 0).Format("06/01/02"), chara, 'A'+g.ShotType, diff, g.Stage)
}

func (s *Replay) Draw(screen *ebiten.Image) {
	sv := supervisor.Get()
	s.DrawPorted(sv)
	if screen != nil && sv.Frame != nil {
		screen.DrawImage(sv.Frame, nil)
	}
}

func (s *Replay) DrawPorted(sv *supervisor.Supervisor) {
	s.core.draw(sv, s.Background, func() {
		switch s.state {
		case 2:
			first := s.selection / 25 * 25 // fixed t5 slots; optional catalog pages stay bounded
			for i := first; i < min(first+25, len(s.entries)); i++ {
				colour := uint32(0xff808080)
				if i == s.selection {
					colour = 0xffffff00
				}
				s.core.smallText(sv, replaySlotLine(s.entries[i].slot), 58, float32(80+(i-first)*15), colour)
			}
		case 4:
			d := s.Details()
			y := float32(80)
			if s.timer.Cur < 10 {
				// 0x431bf7..0x431c1a: selected list row -> Y 80 in ten frames.
				x := float32(10 - s.timer.F)
				x = float32(x * float32((s.chosen%25)*15))
				x = float32(x * math.Float32frombits(0x3dcccccd))
				y = float32(x + 80)
			}
			s.core.smallText(sv, d.Header, 80, y, 0xffffffff)
			if s.timer.Cur >= 10 {
				for i, row := range d.Chapters {
					colour := uint32(0xff808080)
					if i == s.chapter {
						colour = 0xffffff00
					}
					s.core.smallText(sv, row.Text, 220, float32(128+i*18), colour)
				}
			}
		}
		if s.status != "" {
			s.core.smallText(sv, s.status[:min(len(s.status), 60)], 48, 448, 0xffffffff)
		}
	}) // FUN_00431ba0, dispatched at 0x42d275
}
