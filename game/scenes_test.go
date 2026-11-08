package game

import (
	"errors"
	"fmt"
	"reflect"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
)

type numberedTestScene struct {
	id    SceneID
	name  string
	calls *[]string
	calc  func(*Game)
}

func (s *numberedTestScene) SceneMode() SceneID { return s.id }
func (s *numberedTestScene) SceneEnter(*Game) error {
	*s.calls = append(*s.calls, s.name+" enter")
	return nil
}
func (s *numberedTestScene) SceneLeave(_ *Game, next SceneID) {
	*s.calls = append(*s.calls, fmt.Sprintf("%s leave %d", s.name, next))
}
func (s *numberedTestScene) Update(g *Game) error {
	*s.calls = append(*s.calls, s.name+" calc")
	if s.calc != nil {
		s.calc(g)
	}
	return nil
}
func (*numberedTestScene) Draw(*ebiten.Image) {}

func TestSceneRequestDelayedLifecycleAndNormalization(t *testing.T) {
	for _, request := range []SceneID{SceneBoot, SceneStage, SceneRetry, SceneNextStage, ScenePlayback, SceneContinue, SceneEnding, ScenePlayerData} {
		t.Run(fmt.Sprint(request), func(t *testing.T) {
			g := &Game{}
			var calls []string
			old := &numberedTestScene{SceneTitle, "old", &calls, nil}
			g.SetScene(old)
			calls = nil
			g.GoToFactory(request, func(*Game) (Scene, error) {
				calls = append(calls, "construct")
				return &numberedTestScene{normalizedScene(request), "new", &calls, nil}, nil
			})
			if len(calls) != 0 || g.CurrentSceneID() != SceneTitle || g.RequestedSceneID() != request {
				t.Fatal("constructor ran in the request frame")
			}
			if err := g.CalcScene(); err != nil {
				t.Fatal(err)
			}
			want := []string{fmt.Sprintf("old leave %d", request), "construct", "new enter", "new calc"}
			if !reflect.DeepEqual(calls, want) {
				t.Fatalf("calls=%v want %v", calls, want)
			}
			if g.CurrentSceneID() != normalizedScene(request) || g.PreviousSceneID() != SceneTitle || g.RequestedSceneID() != g.CurrentSceneID() {
				t.Fatal("native mode words differ")
			}
			if request == ScenePlayerData && g.TitleEntry != 3 {
				t.Fatal("Player Data return lost title entry reason 3")
			}
		})
	}
}

func TestSceneEqualRequestAndGoToNil(t *testing.T) {
	g := &Game{}
	var calls []string
	s := &numberedTestScene{SceneTitle, "scene", &calls, nil}
	g.SetScene(s)
	calls = nil
	g.RequestScene(SceneTitle)
	if err := g.CalcScene(); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(calls, []string{"scene calc"}) {
		t.Fatal("equal mode reconstructed the scene")
	}
	g.GoTo(nil)
	if err := g.CalcScene(); err != nil {
		t.Fatal(err)
	}
	if g.current != nil {
		t.Fatal("legacy nil scene request ignored")
	}
}

func TestSceneRequestFromCalcAndFactoryFailure(t *testing.T) {
	g := &Game{}
	var calls []string
	g.RegisterScene(SceneStage, func(*Game) (Scene, error) {
		calls = append(calls, "construct stage")
		return &numberedTestScene{SceneStage, "stage", &calls, nil}, nil
	})
	g.SetScene(&numberedTestScene{SceneTitle, "title", &calls, func(g *Game) { g.RequestScene(ScenePlayback) }})
	calls = nil
	if err := g.CalcScene(); err != nil {
		t.Fatal(err)
	}
	if g.CurrentSceneID() != SceneTitle {
		t.Fatal("request switched within requesting calc")
	}
	if err := g.CalcScene(); err != nil {
		t.Fatal(err)
	}
	want := []string{"title calc", "title leave 12", "construct stage", "stage enter", "stage calc"}
	if !reflect.DeepEqual(calls, want) {
		t.Fatalf("calls %v", calls)
	}
	bad := errors.New("test constructor failure")
	g.GoToFactory(SceneEnding, func(*Game) (Scene, error) { return nil, bad })
	if !errors.Is(g.CalcScene(), bad) || g.current != nil {
		t.Fatal("failed construction left active old scene")
	}
}

func TestSceneExitSystemOverride(t *testing.T) {
	g := &Game{SystemFlags: 0x1000}
	var calls []string
	g.SetScene(&numberedTestScene{SceneTitle, "title", &calls, nil})
	g.RequestScene(SceneStage)
	if g.RequestedSceneID() != SceneExitSystem {
		t.Fatal("FUN_0040ac90 override not applied")
	}
	if !errors.Is(g.CalcScene(), ebiten.Termination) || g.current != nil {
		t.Fatal("system exit did not release scene")
	}
}
