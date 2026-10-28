package audio

import (
	"testing"

	"th10/datfs"
	"th10/gamedata"
)

func TestSETable(t *testing.T) {
	for id, want := range map[int]string{7: "se_tan00.wav", 24: "se_kira00.wav", 28: "se_graze.wav"} {
		if got, _ := SEName(id); got != want {
			t.Errorf("SEName(%d) = %q, want %q", id, got, want)
		}
	}
	if _, ok := SEName(len(seTable)); ok {
		t.Errorf("id past the table resolved")
	}
	for i, name := range seWav {
		if _, err := gamedata.ReadFile(datfs.EmbedPath(name)); err != nil {
			t.Errorf("wav %d %s: %v", i, name, err)
		}
	}
}
