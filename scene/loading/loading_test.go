package loading

import (
	"strings"
	"testing"

	"th10/game"
	"th10/supervisor"
	"th10/supervisor/supervisortest"

	"github.com/hajimehoshi/ebiten/v2"
)

type testTitle struct{ calc int }

func (t *testTitle) Update(*game.Game) error { t.calc++; return nil }
func (*testTitle) Draw(*ebiten.Image)        {}

func TestLoadingGateCountsDrawsNotUpdates(t *testing.T) {
	g := &game.Game{}
	l := New()
	title := &testTitle{}
	l.Next = func(*game.Game) (game.Scene, error) { return title, nil }
	g.SetScene(l)
	if err := g.CalcScene(); err != nil {
		t.Fatal(err)
	}
	if err := g.CalcScene(); err != nil {
		t.Fatal(err)
	}
	if g.CurrentSceneID() != game.SceneTitle {
		t.Fatal("ready loading did not request scene 4")
	}
	for range 400 {
		if err := g.CalcScene(); err != nil {
			t.Fatal(err)
		}
	}
	if title.calc != 0 || l.DrawCount != 0 {
		t.Fatal("Update advanced loading draw clock")
	}
	d := &supervisortest.Rec{}
	sv := supervisor.New(d, nil)
	for frame := 0; frame < 299; frame++ {
		l.DrawPorted(sv)
		if err := g.CalcScene(); err != nil {
			t.Fatal(err)
		}
		if title.calc != 0 {
			t.Fatalf("title enabled on draw %d", frame+1)
		}
	}
	l.DrawPorted(sv)
	if err := g.CalcScene(); err != nil {
		t.Fatal(err)
	}
	if title.calc != 1 || l.DrawCount != 300 || l.world.Len() != 0 {
		t.Fatal("native 300-draw release/enable boundary differs")
	}
	if !strings.Contains(strings.Join(d.Calls, "\n"), "draw ") {
		t.Fatal("sig.anm made no headless draws")
	}
}

func TestLoadingGateSystemSkip(t *testing.T) {
	g := &game.Game{SystemFlags: 0x80}
	l := New()
	title := &testTitle{}
	l.Next = func(*game.Game) (game.Scene, error) { return title, nil }
	g.SetScene(l)
	if err := g.CalcScene(); err != nil {
		t.Fatal(err)
	}
	if err := g.CalcScene(); err != nil {
		t.Fatal(err)
	}
	if title.calc != 1 || l.DrawCount != 0 {
		t.Fatal("system 0x80 did not bypass minimum draws")
	}
}
