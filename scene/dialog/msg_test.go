package dialog

import "testing"

// TestParseMSGStage2Reimu verifies the official .msg loader decodes real TH10
// stage-2 (Reimu) dialog: correct segment count, speakers, and Shift-JIS text.
func TestParseMSGStage2Reimu(t *testing.T) {
	sd := loadStageDialog("stage2", 0)
	if sd == nil {
		t.Fatal("loadStageDialog(stage2, Reimu) = nil")
	}
	if len(sd.segments) != 2 {
		t.Fatalf("stage2 segments = %d, want 2 (bossPre, post)", len(sd.segments))
	}
	pre := sd.segments[0]
	if len(pre) == 0 {
		t.Fatal("bossPre segment empty")
	}
	// First line is the player (灵梦) per the decoded stream (op 12;6 then 7).
	if pre[0].IsRight {
		t.Errorf("first line IsRight=true, want player (left)")
	}
	if pre[0].Expression != "n2" {
		t.Errorf("first line expression = %q, want n2 from face id 6", pre[0].Expression)
	}
	// The known first boss line (雛) is "しょっぱなから気持ち悪いなぁ".
	var foundBoss bool
	var foundAngryBoss bool
	for _, l := range pre {
		if l.IsRight && l.Speaker != "鍵山雛" {
			t.Errorf("right speaker = %q, want 鍵山雛", l.Speaker)
		}
		if l.Text == "しょっぱなから気持ち悪いなぁ" {
			foundBoss = true
		}
		if l.IsRight && l.Expression == "an" {
			foundAngryBoss = true
		}
	}
	if !foundBoss {
		t.Errorf("did not find expected boss line; got: %+v", pre)
	}
	if !foundAngryBoss {
		t.Errorf("did not find any boss line mapped to angry expression; got: %+v", pre)
	}
	post := sd.segments[1]
	if len(post) == 0 || post[0].Expression != "dp" {
		t.Fatalf("first post expression = %q, want dp from face id 8", post[0].Expression)
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

// TestBossPreCFallsBack ensures the character-aware accessor returns non-empty
// for every stage (official .msg or hand-written fallback) and never panics.
func TestBossPreCAllStages(t *testing.T) {
	for _, st := range []string{"stage1", "stage2", "stage3", "stage4", "stage5", "stage6", "extra"} {
		for _, ch := range []int{0, 1} {
			if got := BossPreC(st, ch); len(got) == 0 {
				t.Errorf("BossPreC(%s, %d) empty", st, ch)
			}
		}
	}
}

func TestOfficialMSGExtraSegments(t *testing.T) {
	sd := loadStageDialog("extra", 0)
	if sd == nil {
		t.Fatal("loadStageDialog(extra, Reimu) = nil")
	}
	if len(sd.segments) != 3 {
		t.Fatalf("extra segments = %d, want 3 (midPre, bossPre, post)", len(sd.segments))
	}
	if got := sd.segments[0][0].Speaker; got != "八坂神奈子" {
		t.Fatalf("extra segment 0 speaker = %q, want 八坂神奈子", got)
	}
	if !segmentHasRightSpeaker(sd.segments[1], "洩矢諏訪子") {
		t.Fatalf("extra segment 1 has no right speaker 洩矢諏訪子: %+v", sd.segments[1])
	}
	if !segmentHasRightSpeaker(sd.segments[2], "洩矢諏訪子") {
		t.Fatalf("extra segment 2 has no right speaker 洩矢諏訪子: %+v", sd.segments[2])
	}
}

func segmentHasRightSpeaker(lines []Line, speaker string) bool {
	for _, l := range lines {
		if l.IsRight && l.Speaker == speaker {
			return true
		}
	}
	return false
}
