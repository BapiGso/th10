package item

import (
	"testing"

	"th10/game"
)

func TestCollectionPopupValues(t *testing.T) {
	for _, tc := range []struct {
		name    string
		kind    int32
		power   int
		faith   int64
		y       float32
		diff    int
		value   int32
		color   uint32
		noPopup bool
	}{
		{"small power", KindPower, 0, 50000, 300, 1, 0, 0xffff4040, false},
		{"power level", KindPower, 95, 50000, 300, 1, -1, 0xffffff40, false},
		{"big power", KindBigPower, 0, 50000, 300, 1, -1, 0xffffff40, false},
		{"full power", KindBigPower, 500, 50000, 300, 1, 50, 0xffff4040, false},
		{"point above", KindPoint, 0, 50000, 100, 1, 50000, 0xffffff00, false},
		{"point below", KindPoint, 0, 100000, 300, 1, 50000, 0xffffffff, false},
		{"full point", KindFullPoint, 0, 50000, 300, 1, 50000, 0xffffff00, false},
		{"big faith", KindBigFaith, 0, 50000, 300, 2, 8000, 0xff00ff00, false},
		{"faith", KindFaith, 0, 50000, 300, 1, 100, 0xff00ff00, false},
		{"cancel", KindCancel, 0, 50000, 300, 1, 0, 0, true},
		{"life", KindLife, 0, 50000, 300, 1, 0, 0, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			st := game.NewState(0, 0, tc.diff)
			st.Power, st.Faith = tc.power, tc.faith
			pos := [3]float32{-20, 200, 3}
			calls := 0
			env := &Env{State: st, Player: Player{Pos: [2]float32{0, tc.y}}, Popup: func(p [3]float32, value int32, color uint32) {
				calls++
				if p != pos || value != tc.value || color != tc.color {
					t.Errorf("popup %v / %d / %#x, want %v / %d / %#x", p, value, color, pos, tc.value, tc.color)
				}
			}}
			m := NewManager()
			m.collect(env, &Item{Kind: tc.kind, Pos: pos})
			wantCalls := 1
			if tc.noPopup {
				wantCalls = 0
			}
			if calls != wantCalls {
				t.Errorf("%d popup calls, want %d", calls, wantCalls)
			}
		})
	}
}
