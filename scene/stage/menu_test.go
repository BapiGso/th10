package stage

import (
	"strings"
	"testing"

	"th10/game"
	"th10/input"
	"th10/replay"
	"th10/supervisor"
	"th10/supervisor/supervisortest"
)

func menuFixture(t *testing.T) (*Stage, *game.Game, func(uint16) MenuAction) {
	t.Helper()
	t.Setenv("TH10_DATA_DIR", t.TempDir())
	state := game.NewState(0, 0, 1)
	state.Stage, state.SingleStage = 1, true
	state.RunFlags = game.RunFlagPractice
	s := NewHeadless(state, &recordScript{})
	s.beginRecording()
	var in input.State
	in.Restore(1 << input.KeyRight)
	s.UpdateHeadless(&in)
	g := &game.Game{Save: game.DefaultSaveData(), Replays: game.NewMemoryReplayStore()}
	var menuIn input.State
	tick := func(bits uint16) MenuAction {
		menuIn.Restore(bits)
		action := s.ctx.Menu.Calc(g, &menuIn)
		s.ctx.Menu.world.Run(s.ctx.Menu.env)
		return action
	}
	return s, g, tick
}

func menuWait(tick func(uint16) MenuAction, n int) {
	for range n {
		tick(0)
	}
}

func TestMenuNameAndReplaySave(t *testing.T) {
	s, g, tick := menuFixture(t)
	m := s.ctx.Menu
	if m.State != MenuDormant || m.Flags != 2 || m.Timer.Cur != -1 {
		t.Fatal("constructor differs")
	}
	m.OpenGameOver(g)
	tick(1 << input.KeyOk)
	if m.State != MenuPrompt {
		t.Fatal("prompt accepted before ten frames")
	}
	menuWait(tick, 12)
	tick(1 << input.KeyOk)
	menuWait(tick, 12)
	if m.State != MenuChoices || m.Selection != 2 {
		t.Fatalf("choices: %+v", m)
	}
	tick(1 << input.KeyUp)
	tick(0)
	if m.Selection != 1 {
		t.Fatal("save choice did not move")
	}
	tick(1 << input.KeyOk)
	menuWait(tick, 12)
	if m.State != MenuSlots || len(m.slots) != 25 {
		t.Fatal("slots did not open")
	}
	tick(1 << input.KeyOk)
	menuWait(tick, 12)
	if m.State != MenuReplayName {
		t.Fatal("name entry did not open")
	}
	for i := 0; i < 8; i++ {
		m.Character = i
		tick(1 << input.KeyOk)
		tick(0)
	}
	if string(m.Name[:]) != "ABCDEFGH" || m.NameAt != 8 || m.Character != len(menuCharacters)-1 {
		t.Fatalf("eight-character input: %q / %d / %d", m.Name, m.NameAt, m.Character)
	}
	tick(1 << input.KeyCancel)
	tick(0)
	if m.NameAt != 7 || m.Name[7] != ' ' {
		t.Fatal("cancel did not backspace")
	}
	m.Character = 7
	tick(1 << input.KeyOk)
	tick(0)
	tick(1 << input.KeyOk) // finish cell
	if m.State != MenuSlots || m.Error != "" {
		t.Fatalf("save failed: state %d error %s", m.State, m.Error)
	}
	r, err := g.LoadReplay(1)
	if err != nil {
		t.Fatal(err)
	}
	if r.Global.User != "ABCDEFGH" || len(r.Chapters[0].Records) != 1 || r.Chapters[0].Records[0].Keys != replay.KeyRight {
		t.Fatalf("saved replay differs: %+v", r.Global)
	}
	if g.Save.LastReplayName != "ABCDEFGH" {
		t.Fatal("last name not retained")
	}
}

func TestMenuNameGridWrapping(t *testing.T) {
	s, _, _ := menuFixture(t)
	m := s.ctx.Menu
	var in input.State
	in.Restore(1 << input.KeyLeft)
	m.moveName(&in)
	if m.Character != 12 {
		t.Fatal("left did not wrap within the row")
	}
	in.Restore(0)
	in.Restore(1 << input.KeyRight)
	m.moveName(&in)
	if m.Character != 0 {
		t.Fatal("right did not wrap within the row")
	}
	in.Restore(0)
	in.Restore(1 << input.KeyUp)
	m.moveName(&in)
	if m.Character != len(menuCharacters)-13 {
		t.Fatal("up did not wrap the grid")
	}
}

func TestMenuRankingAndExit(t *testing.T) {
	s, g, tick := menuFixture(t)
	s.ctx.State.SingleStage = false
	s.ctx.State.RunFlags &^= game.RunFlagPractice
	s.ctx.State.Score = 1234560
	m := s.ctx.Menu
	m.OpenClear(g)
	menuWait(tick, 12)
	tick(1 << input.KeyOk)
	menuWait(tick, 12)
	if m.State != MenuRanking || m.rank != 0 || m.NoRank {
		t.Fatal("ranking did not open")
	}
	menuWait(tick, 12)
	m.Character = len(menuCharacters) - 1
	tick(1 << input.KeyOk)
	tick(0)
	if m.State != MenuChoices || g.ScoreTable(s.ctx.State)[0].Score != 123456 {
		t.Fatal("ranking did not settle")
	}
	tick(1 << input.KeyCancel)
	tick(0)
	tick(1 << input.KeyOk)
	if m.State != MenuLeaving {
		t.Fatal("exit did not enter animation delay")
	}
	action := MenuStay
	for range 14 {
		if a := tick(0); a != MenuStay {
			action = a
		}
	}
	if action != MenuTitle || m.State != MenuFinished {
		t.Fatal("exit action differs")
	}
}

func TestMenuDrawUsesANMWithoutBackBuffer(t *testing.T) {
	s, g, tick := menuFixture(t)
	m := s.ctx.Menu
	m.OpenGameOver(g)
	menuWait(tick, 12)
	tick(1 << input.KeyOk)
	menuWait(tick, 12)
	tick(1 << input.KeyUp)
	tick(0)
	tick(1 << input.KeyOk)
	menuWait(tick, 12)
	d := &supervisortest.Rec{}
	sv := supervisor.New(d, nil)
	sv.BeginFrame()
	sv.Enter2D()
	s.nodeScreen15(sv)
	s.nodeScreen16(sv)
	s.nodeMenu(sv)
	sv.EndFrame()
	draws := 0
	for _, call := range d.Take() {
		if strings.HasPrefix(call, "draw ") {
			draws++
		}
	}
	if draws == 0 {
		t.Fatal("menu did not reach anmdraw")
	}
	for _, node := range drawChain {
		if node.prio == 0x2e && node.legacy {
			t.Fatal("node 0x2e still legacy")
		}
	}
}
