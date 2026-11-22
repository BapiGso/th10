//go:build th10nostory

package dialog

import (
	"fmt"
	"testing"

	"th10/gamedata"
)

// Without story every stage script runs to its end with no input, keeping
// its signals, boss BGM and stage clear, and loads no text.
func TestNoStoryRunsScriptsWithoutInput(t *testing.T) {
	for stage := 1; stage <= 7; stage++ {
		for chara := 0; chara < 2; chara++ {
			data, err := gamedata.ReadFile(fmt.Sprintf("msg/st%02d_%02d.msg", stage, chara))
			if err != nil {
				t.Fatal(err)
			}
			for id, code := range ParseMSG(data) {
				want := map[uint8]int{}
				for _, in := range code {
					if in.Text != "" {
						t.Fatalf("st%02d_%02d entry %d: text %q decoded", stage, chara, id, in.Text)
					}
					want[in.Op]++
				}
				got := map[string]int{}
				m := NewMSG(int32(id), code, Hooks{
					SE:      func(int) { got["se"]++ },
					BossBGM: func() { got["bgm"]++ },
					Clear:   func() { got["clear"]++ },
					FadeBGM: func() { got["fade"]++ },
				})
				signals, frames := 0, 0
				for ; m.Update(false, false); frames++ {
					if m.Signal != 0 {
						signals++
					}
					if frames > 60*60 {
						t.Fatalf("st%02d_%02d entry %d did not finish without input", stage, chara, id)
					}
				}
				if got["bgm"] != want[msgBossBGM] || got["clear"] != want[msgStageClear] || got["fade"] != want[msgFadeBGM] ||
					(want[msgSignal] > 0) != (signals > 0) {
					t.Fatalf("st%02d_%02d entry %d: effects %v, signals %d, ops %v", stage, chara, id, got, signals, want)
				}
			}
		}
	}
}
