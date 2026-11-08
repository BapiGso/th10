package menus

import (
	"encoding/binary"
	"fmt"
	"time"

	"th10/game"
	"th10/replay"
)

// ReplayMenuSource has the existing t5 persistence signatures. Game is the
// default implementation; a catalog adapter can supply extra original
// th10_ud????.rpy entries without doing file I/O in a scene or changing codec.
type ReplayMenuSource interface {
	ListReplays() []game.ReplaySlot
	LoadReplay(int) (*replay.Replay, error)
}

type ReplayChapterDetail struct {
	Stage        int
	ChapterIndex int // t5 NewPlayback takes a slice index, not stage-1
	Available    bool
	Score        int32
	Continues    int32
	Text         string
}

type ReplayDetails struct {
	Slot     int
	Global   replay.GlobalHeader
	Header   string
	Chapters [7]ReplayChapterDetail
}

var replayCharacters = [...]string{"ReimuA ", "ReimuB ", "ReimuC ", "MarisaA", "MarisaB", "MarisaC"} // PE 0x4746dc
var replayDifficulties = [...]string{"Easy   ", "Normal ", "Hard   ", "Lunatic", "Extra  "}          // PE 0x4746f4
var replayStages = [...]string{"tst", "St1", "St2", "St3", "St4", "St5", "St6", "Ex ", "All"}        // PE 0x474744

func replayHeaderLine(slot int, header replay.GlobalHeader) string {
	chara := "-------"
	if header.Character < 2 && header.ShotType < 3 {
		chara = replayCharacters[header.Character*3+header.ShotType]
	}
	difficulty, stage := "-------", "---"
	if header.Difficulty < uint32(len(replayDifficulties)) {
		difficulty = replayDifficulties[header.Difficulty]
	}
	if header.Stage < uint32(len(replayStages)) {
		stage = replayStages[header.Stage]
	}
	t := time.Unix(int64(header.Timestamp), 0).Local()
	return fmt.Sprintf("No.%02d %s %02d/%02d/%02d %02d:%02d %s %s %s %2.1f%%", slot, header.User,
		t.Year()%100, t.Month(), t.Day(), t.Hour(), t.Minute(), chara, difficulty, stage, header.SlowRate) // PE 0x46efb0
}

func replaySlotLine(slot game.ReplaySlot) string {
	if slot.Error != nil {
		return fmt.Sprintf("No.%02d [invalid replay]", slot.Number)
	}
	if slot.Replay == nil {
		return fmt.Sprintf("No.%02d -------- --/--/-- --:-- ------- ------- --- ---%%", slot.Number) // PE 0x46ef74
	}
	return replayHeaderLine(slot.Number, slot.Replay.Global)
}

func globalReplayContinues(r *replay.Replay) int32 {
	if len(r.Global.Raw) >= 0x64 {
		return int32(binary.LittleEndian.Uint32(r.Global.Raw[0x60:]))
	}
	return 0 // t5 preserves this word in Global.Raw, not a new decoding field
}

func (s *Replay) Details() ReplayDetails {
	d := ReplayDetails{}
	if s.selected == nil {
		return d
	}
	d.Slot, d.Global = s.entries[s.chosen].slot.Number, s.selected.Global
	d.Header = replayHeaderLine(d.Slot, d.Global)
	for i, index := range s.chapters {
		row := ReplayChapterDetail{Stage: i + 1, ChapterIndex: index, Available: index >= 0}
		if index < 0 {
			row.Text = fmt.Sprintf("%s  ---------", dataStageNames[i+1]) // PE 0x46f008
		} else {
			row.Score, row.Continues = s.selected.Global.Score, globalReplayContinues(s.selected)
			// FUN_00431ba0 (0x431d2f..0x431d68): stages 1..5 display
			// the NEXT stage's starting score; stage 6/Extra use the final.
			if i < 5 && s.chapters[i+1] >= 0 {
				next := s.selected.Chapters[s.chapters[i+1]]
				row.Score, row.Continues = next.Score, next.State90
			}
			row.Text = fmt.Sprintf("%s  %08d%d", dataStageNames[i+1], row.Score, row.Continues) // PE 0x46ef68
		}
		d.Chapters[i] = row
	}
	return d
}
