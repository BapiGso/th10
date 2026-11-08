package ending

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"th10/game"
	"th10/gamedata"
	"th10/input"
	"th10/supervisor"
	"th10/supervisor/supervisortest"

	"golang.org/x/text/encoding/japanese"
)

func headlessGame() *game.Game {
	g := &game.Game{Scores: game.NewMemoryScoreStore()}
	g.LoadProgress()
	return g
}

func keepInput(t *testing.T) {
	previous := input.Global
	t.Cleanup(func() { input.Global = previous })
	input.Global = input.State{}
}

func TestEndingBranches(t *testing.T) {
	for character := range 2 {
		for shot := range 3 {
			for difficulty := range 4 {
				for _, continues := range []int32{0, 1, 9} {
					s := game.NewState(character, shot, difficulty)
					s.ReplayState90 = continues
					s.Continued = true // the transient host bool is not DAT_00474c90
					s.Life, s.Power, s.Score = 0, 0, 0
					got, err := EndingIndex(s)
					want := character*3 + shot
					if difficulty > 0 && continues == 0 {
						want += 6
					}
					if err != nil || got != want {
						t.Fatalf("%d/%d/%d continues=%d: %d, %v, want %d", character, shot, difficulty, continues, got, err, want)
					}
				}
			}
		}
	}
	for _, s := range []*game.GameState{nil, {Character: -1}, {Character: 2}, {ShotType: 3}, {Difficulty: game.DiffExtra}} {
		if _, err := EndingIndex(s); err == nil {
			t.Fatal("invalid/Extra ending selection accepted")
		}
	}
}

func TestEndingMSGOriginalControlAndTraditionalText(t *testing.T) {
	names := []string{"staff.msg"}
	for i := range 12 {
		names = append(names, fmt.Sprintf("e%02d.msg", i))
	}
	for _, name := range names {
		data, err := gamedata.ReadFile("msg/" + name)
		if err != nil {
			t.Fatal(err)
		}
		original, err := parseMSG(data, japanese.ShiftJIS)
		if err != nil {
			t.Fatal(err)
		}
		got, err := loadMSG(name)
		if err != nil || len(got) != len(original) {
			t.Fatalf("%s: %v", name, err)
		}
		translated := 0
		for i, in := range got {
			if in.op != original[i].op || in.time != original[i].time || !bytes.Equal(in.args, original[i].args) || in.text != original[i].text {
				t.Fatalf("%s instruction %d used translated control bytes", name, i)
			}
			if in.op == 3 && in.displayText != in.text {
				translated++
			}
		}
		if name != "staff.msg" && translated == 0 {
			t.Fatalf("%s did not use the embedded MST text", name)
		}
	}
	staff, err := gamedata.PresentationANM("staff.anm")
	if err != nil {
		t.Fatal(err)
	}
	original, err := gamedata.ANM("staff.anm")
	if err != nil || !reflect.DeepEqual(original.Scripts, staff.Scripts) {
		t.Fatal("Staff ANT replaced native animation control")
	}
}

func TestEndingMSGRejectsTruncationAndWASMOffsets(t *testing.T) {
	data, err := gamedata.ReadFile("msg/e00.msg")
	if err != nil {
		t.Fatal(err)
	}
	for _, n := range []int{0, 4, 11, 15, 30} {
		if _, err := parseMSG(data[:n], japanese.ShiftJIS); err == nil {
			t.Fatalf("truncated MSG length %d accepted", n)
		}
	}
	for _, at := range []int{0, 4} {
		bad := bytes.Clone(data)
		binary.LittleEndian.PutUint32(bad[at:], 0xffffffff)
		if _, err := parseMSG(bad, japanese.ShiftJIS); err == nil {
			t.Fatal("overflowing MSG offset/count accepted")
		}
	}
}

func waitInstruction(op byte, n int32) instruction {
	args := make([]byte, 4)
	binary.LittleEndian.PutUint32(args, uint32(n))
	return instruction{op: op, args: args}
}

func TestEndingWaitClockAndNegativeStaffWait(t *testing.T) {
	e := &Ending{firstEnding: true}
	m := &msgVM{owner: e}
	m.reset("wait", []instruction{waitInstruction(5, 3), {time: 1, op: 0}}, false)
	for i := range 3 {
		done, err := m.step(&game.Game{}, false, false)
		if err != nil || done {
			t.Fatalf("wait tick %d: %v, done=%v", i, err, done)
		}
		if i < 2 && (m.clock.Cur != 0 || m.wait.Cur != int32(2-i)) {
			t.Fatal("wait advanced the script clock or decremented twice")
		}
	}
	if m.clock.Cur != 1 || m.wait.Cur != 0 {
		t.Fatal("timeout did not release the wait once")
	}
	done, err := m.step(&game.Game{}, false, false)
	if err != nil || !done {
		t.Fatal("next-time terminator not consumed")
	}
	m.reset("staff", []instruction{waitInstruction(5, -1), {time: 1, op: 0}}, true)
	e.firstEnding = false
	for range 2000 {
		done, err := m.step(&game.Game{}, false, true)
		if err != nil || done || m.wait.Cur != 999 || m.clock.Cur != 0 {
			t.Fatal("negative Staff wait ended by elapsed time/Ctrl")
		}
	}
	if _, err := m.step(&game.Game{}, true, false); err != nil || m.clock.Cur != 1 {
		t.Fatal("Z did not release Staff's explicit terminal wait")
	}
}

func TestEndingSeenFlagsAndReplayIsolation(t *testing.T) {
	keepInput(t)
	g := headlessGame()
	s := game.NewState(1, 2, game.DiffHard)
	e := New(s, false, nil)
	if err := e.Update(g); err != nil {
		t.Fatal(err)
	}
	if !e.firstEnding || !e.firstStaff || !g.Save.Native.EndingSeen(11) || !g.Save.Native.StaffSeen() {
		t.Fatal("first-view gates were read after writing the ST flags")
	}
	again := New(s, false, nil)
	if err := again.Update(g); err != nil || again.firstEnding || again.firstStaff {
		t.Fatal("seen ending/Staff did not enable native skip gates")
	}
	before, err := game.EncodeScoreFile(g.Save.Native)
	if err != nil {
		t.Fatal(err)
	}
	s.ReplayMode, s.ShotType = 1, 0
	readOnly := New(s, false, nil)
	for range 3 {
		if err := readOnly.Update(g); err != nil {
			t.Fatal(err)
		}
	}
	after, err := game.EncodeScoreFile(g.Save.Native)
	if err != nil || !bytes.Equal(before, after) {
		t.Fatal("replay mutated shared ending/music progress")
	}
}

func TestEndingAndStaffHeadlessDraw(t *testing.T) {
	keepInput(t)
	for index := range 12 {
		t.Run(fmt.Sprintf("e%02d", index), func(t *testing.T) {
			input.Global = input.State{}
			difficulty := game.DiffEasy
			if index >= 6 {
				difficulty = game.DiffNormal
			}
			s := game.NewState((index%6)/3, index%3, difficulty)
			e := New(s, false, nil)
			calls := 0
			e.After = func(*game.Game) game.Scene { calls++; return nil }
			g, dev := headlessGame(), &supervisortest.Rec{}
			sv := supervisor.New(dev, nil)
			draws := 0
			for frame := 0; frame < 350 && !e.closed; frame++ {
				bits := uint16(0)
				if frame%2 == 0 {
					bits = 1 << input.KeyOk
				}
				input.Global.Restore(bits)
				if err := e.Update(g); err != nil {
					t.Fatal(err)
				}
				e.DrawPorted(sv)
				for _, c := range dev.Take() {
					if strings.HasPrefix(c, "draw ") {
						draws++
					}
				}
			}
			if draws == 0 {
				t.Fatal("scene did not draw native ANM images/text")
			}
			if index < 6 {
				if calls != 1 || e.IsStaff() || e.RequestedScene != 15 || e.world.Len() != 0 {
					t.Fatal("Bad ending did not go straight to the retained completion menu")
				}
			} else if calls != 0 || !e.IsStaff() || e.ScriptName() != "staff.msg" {
				t.Fatal("Good ending bypassed Staff or called the completion menu too early")
			}
			e.Close()
		})
	}
}

func TestStaffTimelineDifficultyAndFinish(t *testing.T) {
	keepInput(t)
	for difficulty := game.DiffNormal; difficulty <= game.DiffLuna; difficulty++ {
		t.Run(fmt.Sprint(difficulty), func(t *testing.T) {
			input.Global = input.State{}
			g := headlessGame()
			s := NewStaff(game.NewState(0, 0, difficulty), nil)
			calls := 0
			s.After = func(*game.Game) game.Scene { calls++; return nil }
			dev := &supervisortest.Rec{}
			sv := supervisor.New(dev, nil)
			for range 4843 {
				if err := s.Update(g); err != nil {
					t.Fatal(err)
				}
				s.DrawPorted(sv)
				dev.Take()
			}
			if s.vm.clock.Cur != 4840 || s.vm.wait.Cur != 999 || calls != 0 {
				t.Fatal("Staff did not stop at the exact 4840 wait")
			}
			if v := s.vm.images[6]; v == nil || v.ScriptID != 31+difficulty {
				t.Fatal("Staff did not choose the original difficulty-specific final image")
			}
			if (s.vm.images[7] != nil) != (difficulty == game.DiffLuna) {
				t.Fatal("Lunatic's second final image gate is wrong")
			}
			input.Global.Restore(1 << input.KeyOk)
			if err := s.Update(g); err != nil {
				t.Fatal(err)
			}
			input.Global.Restore(0)
			for range 120 {
				if err := s.Update(g); err != nil {
					t.Fatal(err)
				}
			}
			if calls != 1 || s.RequestedScene != 15 || !s.closed || s.world.Len() != 0 {
				t.Fatal("Staff's 4960 terminator did not finish exactly once")
			}
		})
	}
}

func TestStaffReturnSixRepeatsCalcNotDraw(t *testing.T) {
	keepInput(t)
	for _, seen := range []bool{false, true} {
		g := headlessGame()
		if seen {
			g.Save.Native.MarkStaffSeen()
		}
		s := NewStaff(game.NewState(0, 0, game.DiffNormal), nil)
		input.Global.Restore(1 << input.KeySkip)
		for range 2 {
			if err := s.Update(g); err != nil {
				t.Fatal(err)
			}
		}
		before := s.vm.clock.Cur
		if err := s.Update(g); err != nil {
			t.Fatal(err)
		}
		want := int32(1)
		if seen {
			want = 12
		}
		if s.vm.clock.Cur-before != want {
			t.Fatalf("seen=%v: clock advanced %d, want %d", seen, s.vm.clock.Cur-before, want)
		}
		s.Close()
	}
}

func TestEndingSystemSceneOverride(t *testing.T) {
	keepInput(t)
	e := New(game.NewState(0, 0, game.DiffEasy), false, nil)
	e.vm.instructions, e.vm.pc = []instruction{{op: 0}}, 0
	g := headlessGame()
	g.SystemFlags = 0x1000
	got := int32(0)
	e.ResolveScene = func(_ *game.Game, scene int32) game.Scene { got = scene; return nil }
	if err := e.Update(g); err != nil || got != 2 || e.RequestedScene != 2 {
		t.Fatal("EndingInf ignored the live supervisor's scene-2 override")
	}
}

func TestEndingFadeTiming(t *testing.T) {
	f := newScreenFade(5, 60)
	for frame := range 62 {
		if f.calc() {
			t.Fatalf("fade expired at %d", frame)
		}
		want := int32(frame * 255 / 60)
		if frame >= 60 {
			want = 255
		}
		if f.alpha != want {
			t.Fatalf("frame %d alpha %d, want %d", frame, f.alpha, want)
		}
	}
	if !f.calc() {
		t.Fatal("fade did not expire after two full-alpha frames")
	}
}
