package menus

import (
	"bytes"
	"fmt"
	"strings"
	"time"

	"th10/game"

	"golang.org/x/text/encoding/japanese"
)

type PlayerStatistics struct {
	Runs, Frames, Clears int32
	PlayTime             string
	Rankings             [10]game.ScoreEntry
}

type SpellHistoryRow struct {
	Number             int
	Name, Text         string
	Known              bool
	Captures, Attempts int32
	Colour             uint32
}

// spellDifficulty is the 110-byte PE table at 0x4743c0. Player Data uses
// this table (FUN_004088c0/00432690), not a guessed stage/card-name group.
func spellDifficulty(id int) int {
	if id < 50 {
		return [...]int{2, 3, 0, 1}[id%4]
	}
	if id < 97 {
		return [...]int{1, 2, 3, 0}[(id-50)%4]
	}
	return 4
}

func spellIDs(difficulty int) []int {
	var ids []int
	for id := range game.SpellCount {
		if spellDifficulty(id) == difficulty {
			ids = append(ids, id)
		}
	}
	return ids
}

func (s *PlayerData) pageCount() int {
	return (len(spellIDs(s.difficulty))+9)/10 + 1 // FUN_00431ee0, 0x431f60..0x431f81
}

func (s *PlayerData) Statistics() PlayerStatistics {
	if s.save == nil {
		return PlayerStatistics{PlayTime: "  0:00:00"}
	}
	p := &s.save.Progress[s.shot]
	// FUN_004329f0 reads the selected CR, not SaveData's aggregate totals.
	return PlayerStatistics{
		Runs: p.Runs, Frames: p.Frames, Clears: p.Clears[s.difficulty],
		PlayTime: fmt.Sprintf("%3d:%02d:%02d", p.Frames/216000, p.Frames/3600%60, p.Frames/60%60),
		Rankings: s.save.Rankings[s.shot][s.difficulty],
	}
}

// SpellRows is FUN_00432690. The total record controls name visibility;
// the selected shot supplies both counts. GUI's 99/99 cap does not apply.
func (s *PlayerData) SpellRows() []SpellHistoryRow {
	if s.page == 0 {
		return nil
	}
	ids := spellIDs(s.difficulty)
	start := (s.page - 1) * 10
	if start >= len(ids) {
		return nil
	}
	var rows []SpellHistoryRow
	for _, id := range ids[start:min(start+10, len(ids))] {
		row := SpellHistoryRow{Number: id + 1, Name: strings.Repeat("\uff1f", 21), Colour: 0x808080}
		nameBytes := 42
		if s.history != nil {
			counts, total := &s.history.Shots[s.shot][id], &s.history.Total[id]
			row.Captures, row.Attempts = counts.Captures, counts.Attempts
			row.Known = total.Attempts != 0 // 0x432778, not the shot's attempts
			if row.Known {
				raw := total.Name[:]
				if at := bytes.IndexByte(raw, 0); at >= 0 {
					raw = raw[:at]
				}
				nameBytes = len(raw)
				decoded, _ := japanese.ShiftJIS.NewDecoder().Bytes(raw)
				row.Name, row.Colour = string(decoded), 0xefefef
				if row.Captures != 0 {
					row.Colour = 0xffff80 // 0x432877..0x43288b
				}
			}
		}
		padding := strings.Repeat(" ", max(42-nameBytes, 0)) // original strlen is SJIS bytes
		row.Text = fmt.Sprintf("No.%3d %s%s %4d/%4d", row.Number, row.Name, padding, row.Captures, row.Attempts)
		rows = append(rows, row)
	}
	return rows
}

var dataStageNames = [...]string{"test   ", "Stage 1", "Stage 2", "Stage 3", "Stage 4", "Stage 5", "Stage 6", "Extra  ", "AllClear"} // PE 0x47471c

func rankingLine(rank int, entry game.ScoreEntry) string {
	if entry.Timestamp == 0 {
		return fmt.Sprintf("%2d  %s  %9d%d  ----/--/-- --:--  Stage -  ---%%", rank, entry.Name, entry.Score, int8(entry.Continues)) // PE 0x46ee50
	}
	t := time.Unix(int64(entry.Timestamp), 0).Local() // original localtime, FUN_004329f0
	stage := "Stage -"
	if int(entry.Stage) < len(dataStageNames) {
		stage = dataStageNames[entry.Stage]
	}
	return fmt.Sprintf("%2d  %s  %9d%d  %04d/%02d/%02d %02d:%02d  %s  %2.1f%%", rank, entry.Name,
		entry.Score, int8(entry.Continues), t.Year(), t.Month(), t.Day(), t.Hour(), t.Minute(), stage, entry.SlowRate) // PE 0x46ee84
}
