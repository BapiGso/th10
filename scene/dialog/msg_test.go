package dialog

import "testing"

// texts lists an entry's text ops with the side speaking each (op 7/8).
func texts(code []Instr) (lines []TextLine) {
	side := 0
	for _, in := range code {
		switch in.Op {
		case msgSpeakerPlayer, msgSpeakerBoss:
			side = int(in.Op - msgSpeakerPlayer)
		case msgLine1, msgLine2, msgText:
			lines = append(lines, TextLine{in.Text, side})
		}
	}
	return lines
}

func countOp(code []Instr, op uint8) (n int) {
	for _, in := range code {
		if in.Op == op {
			n++
		}
	}
	return n
}

// TestParseMSGStage2Reimu verifies the official .msg loader decodes real TH10
// stage-2 (Reimu) dialog: entry count, speakers, faces and Shift-JIS text.
func TestParseMSGStage2Reimu(t *testing.T) {
	sm := LoadStage("stage2", 0)
	if sm == nil {
		t.Fatal("LoadStage(stage2, Reimu) = nil")
	}
	if len(sm.Entries) != 2 {
		t.Fatalf("stage2 entries = %d, want 2 (bossPre, post)", len(sm.Entries))
	}
	if sm.PlayerName != "博麗霊夢" || sm.BossNames[0] != "鍵山雛" {
		t.Errorf("speakers %q / %q, want 博麗霊夢 / 鍵山雛", sm.PlayerName, sm.BossNames[0])
	}
	pre := texts(sm.Entries[0])
	if len(pre) < 4 || pre[0] != (TextLine{"しょっぱなから気持ち悪いなぁ", 0}) ||
		pre[3] != (TextLine{"あらあらまだ居たの？", 1}) {
		t.Fatalf("bossPre lines %+v, want the player's しょっぱなから気持ち悪いなぁ first and 雛's あらあらまだ居たの？ fourth", pre)
	}
	// The first face is op 12 with id 6, before the first line.
	for _, in := range sm.Entries[0] {
		if in.Op == msgPlayerFace {
			if in.Arg != 6 {
				t.Errorf("first player face id %d, want 6", in.Arg)
			}
			break
		}
	}
}

func TestMSGFaceExpressionMapping(t *testing.T) {
	tests := map[int]string{
		0: "no",
		1: "an",
		2: "hp",
		3: "an",
		4: "an",
		5: "pr",
		6: "n2",
		7: "sp",
		8: "dp",
	}
	for id, want := range tests {
		if got := msgFaceExpression(id); got != want {
			t.Errorf("msgFaceExpression(%d) = %q, want %q", id, got, want)
		}
	}
}

// TestStageEntries: every stage's entries load for both characters and end
// in op 0. Only the last entry clears the stage (op 20), and the boss's
// pre-fight entry starts the boss theme (op 18); midboss entries do not.
func TestStageEntries(t *testing.T) {
	for _, tc := range []struct {
		stage   string
		ch      int
		entries int
		bossPre int
	}{
		{"stage1", 0, 2, 0}, {"stage1", 1, 2, 0},
		{"stage2", 0, 2, 0}, {"stage3", 0, 4, 2},
		{"stage4", 0, 2, 0}, {"stage4", 1, 2, 0},
		{"stage5", 0, 2, 0}, {"stage6", 0, 2, 0},
		{"extra", 0, 3, 1}, {"extra", 1, 3, 1},
	} {
		sm := LoadStage(tc.stage, tc.ch)
		if sm == nil || len(sm.Entries) != tc.entries {
			t.Errorf("%s/%d: %v, want %d entries", tc.stage, tc.ch, sm, tc.entries)
			continue
		}
		last := len(sm.Entries) - 1
		for i, code := range sm.Entries {
			if len(code) == 0 || code[len(code)-1].Op != msgEnd {
				t.Errorf("%s/%d entry %d does not end in op 0", tc.stage, tc.ch, i)
			}
			if clears := countOp(code, msgStageClear) == 1; clears != (i == last) {
				t.Errorf("%s/%d entry %d: clears %v, want %v", tc.stage, tc.ch, i, clears, i == last)
			}
			if bgm := countOp(code, msgBossBGM) == 1; bgm != (i == tc.bossPre) {
				t.Errorf("%s/%d entry %d: boss theme %v, want %v", tc.stage, tc.ch, i, bgm, i == tc.bossPre)
			}
		}
	}
	// Stage 6 has no post-boss dialog: its last entry only clears the stage.
	if sm := LoadStage("stage6", 0); len(texts(sm.Entries[1])) != 0 {
		t.Errorf("stage6 entry 1 has %d lines, want none", len(texts(sm.Entries[1])))
	}
}

func TestOfficialMSGExtraSegments(t *testing.T) {
	sm := LoadStage("extra", 0)
	if sm == nil {
		t.Fatal("LoadStage(extra, Reimu) = nil")
	}
	for i, want := range []string{"八坂神奈子", "洩矢諏訪子", "洩矢諏訪子"} {
		m := sm.Open(int32(i), Hooks{})
		if m == nil || m.View.BossName != want {
			t.Fatalf("extra entry %d: %v, want boss %s", i, m, want)
		}
	}
	if sm.Open(3, Hooks{}) != nil || sm.Open(-1, Hooks{}) != nil {
		t.Fatal("opened a missing entry")
	}
}
