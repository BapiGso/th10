package replay

import (
	"encoding/binary"
	"testing"
)

func TestChapterCountsCannotOverflowWASMInt(t *testing.T) {
	b := chapterBlock(1, 1, 1)
	binary.LittleEndian.PutUint32(b[4:], ^uint32(0))
	binary.LittleEndian.PutUint32(b[8:], ^uint32(0))
	ch := parseChapters(b, 1)
	if len(ch) != 1 || len(ch[0].Records) != 1 {
		t.Fatal("unsigned lengths not capped before int conversion")
	}
}

func TestValidateUnsafeGameplayIndices(t *testing.T) {
	r := &Replay{Global: GlobalHeader{Stage: 1}, Chapters: []Chapter{{Index: 1, Records: []Record{{}}}}}
	if err := r.ValidatePlayback(); err != nil {
		t.Fatal(err)
	}
	r.Global.Character = ^uint32(0)
	if r.ValidatePlayback() == nil {
		t.Fatal("unsafe character accepted")
	}
	r.Global.Character, r.Global.Stage = 0, ^uint32(0)
	if r.ValidatePlayback() == nil {
		t.Fatal("unsafe final stage accepted")
	}
}
