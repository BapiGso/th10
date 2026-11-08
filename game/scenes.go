package game

import (
	"fmt"
	"th10/supervisor"

	"github.com/hajimehoshi/ebiten/v2"
)

// SceneID is supervisor +0x38c/+0x390, not TitleInf's menu-page state.
type SceneID int32

const (
	SceneUninitialized SceneID = -2 // FUN_00420470
	SceneUnnumbered    SceneID = -1 // compatibility for a Scene without SceneMode
	SceneBoot          SceneID = 0
	SceneLoading       SceneID = 1
	SceneExitSystem    SceneID = 2
	SceneExit          SceneID = 3
	SceneTitle         SceneID = 4
	SceneStage         SceneID = 7
	SceneRetry         SceneID = 10
	SceneNextStage     SceneID = 11
	ScenePlayback      SceneID = 12
	SceneContinue      SceneID = 13
	SceneEnding        SceneID = 14
	ScenePlayerData    SceneID = 15
)

// SceneFactory runs only when FUN_004218d0 consumes the request. This
// keeps resource construction out of the requesting scene's calc.
type SceneFactory func(*Game) (Scene, error)

// NumberedScene and SceneLifecycle extend, but do not replace, Scene.
// The hooks correspond to the original object's registration/destruction.
type NumberedScene interface{ SceneMode() SceneID }
type SceneLifecycle interface {
	SceneEnter(*Game) error
	SceneLeave(*Game, SceneID)
}

type sceneSwitcher struct {
	initialized                bool
	clearPending               bool
	current, request, previous SceneID
	factories                  map[SceneID]SceneFactory
	pending                    SceneFactory
	err                        error
}

func (g *Game) initScenes() {
	if !g.scenes.initialized {
		g.scenes.initialized = true
		g.scenes.current, g.scenes.request = SceneUninitialized, SceneBoot // FUN_00420470
	}
}

func sceneMode(s Scene) SceneID {
	if n, ok := s.(NumberedScene); ok {
		return n.SceneMode()
	}
	return SceneUnnumbered
}

func (g *Game) CurrentSceneID() SceneID   { g.initScenes(); return g.scenes.current }
func (g *Game) RequestedSceneID() SceneID { g.initScenes(); return g.scenes.request }
func (g *Game) PreviousSceneID() SceneID  { g.initScenes(); return g.scenes.previous }

func (g *Game) CurrentScene() Scene { return g.current }

// RegisterScene supplies the constructor owned by another scene package.
// In particular ending/staff/replay retain their t5/t8 entry points.
func (g *Game) RegisterScene(id SceneID, factory SceneFactory) {
	g.initScenes()
	if g.scenes.factories == nil {
		g.scenes.factories = make(map[SceneID]SceneFactory)
	}
	g.scenes.factories[id] = factory
}

// RequestScene is FUN_0040ac90. Its system-exit override is deliberately
// separate from the direct writer FUN_0041ff30 (GoToFactory).
func (g *Game) RequestScene(id SceneID) {
	if g.SystemFlags&0x1000 != 0 { // 0x40ac96..0x40acaf
		id = SceneExitSystem
	}
	g.initScenes()
	g.scenes.request, g.scenes.pending = id, nil
	g.next, g.nextSet = nil, true
}

// GoToFactory is a direct request with a delayed constructor. A new
// instance of the same numbered mode is allowed, as in requests 10..13.
func (g *Game) GoToFactory(id SceneID, factory SceneFactory) {
	g.initScenes()
	g.scenes.request, g.scenes.pending = id, factory
	g.next, g.nextSet = nil, true
}

// normalizedScene is FUN_004218d0's request switch (0x421924..0x421ae8).
func normalizedScene(id SceneID) SceneID {
	switch id {
	case SceneBoot:
		return SceneLoading
	case SceneRetry, SceneNextStage, ScenePlayback, SceneContinue:
		return SceneStage
	case ScenePlayerData:
		return SceneTitle
	default:
		return id
	}
}

// CalcScene is FUN_0041ff80 -> FUN_004218d0 followed by the current calc
// callbacks. Tests can drive it without a wall clock or keyboard device.
func (g *Game) CalcScene() error {
	g.initScenes()
	if g.scenes.err != nil {
		return g.scenes.err
	}
	if g.nextSet || g.scenes.request != g.scenes.current {
		requested := g.scenes.request
		factory, target := g.scenes.pending, g.next
		g.nextSet, g.next, g.scenes.pending = false, nil, nil
		if factory == nil && target == nil && requested == g.scenes.current {
			if g.current != nil {
				return g.current.Update(g) // FUN_004218d0 equal-mode early return
			}
			return nil
		}
		if requested == SceneExit || requested == SceneExitSystem {
			// FUN_004218d0 case 3 -> FUN_004203f0; host termination replaces
			// the supervisor calc return code 4 and Win32 exit machinery.
			if life, ok := g.current.(SceneLifecycle); ok {
				life.SceneLeave(g, requested)
			}
			g.scenes.previous, g.scenes.current = g.scenes.current, requested
			g.current = nil
			return ebiten.Termination
		}
		if factory == nil && target == nil && requested != SceneUnnumbered {
			factory = g.scenes.factories[requested]
			if factory == nil {
				factory = g.scenes.factories[normalizedScene(requested)]
			}
			if factory == nil {
				return fmt.Errorf("game: no constructor for scene %d", requested)
			}
		}
		g.scenes.previous = g.scenes.current // supervisor +0x394
		if life, ok := g.current.(SceneLifecycle); ok {
			life.SceneLeave(g, requested)
		}
		if requested == ScenePlayerData { // 0x4219d6..0x4219ee
			g.TitleEntry = 3
		}
		if factory != nil {
			var err error
			target, err = factory(g)
			if err != nil {
				g.current, g.scenes.err = nil, err
				return err
			}
		}
		g.current = target
		g.scenes.clearPending = true
		g.scenes.current = normalizedScene(requested)
		g.scenes.request = g.scenes.current
		if life, ok := target.(SceneLifecycle); ok {
			if err := life.SceneEnter(g); err != nil {
				g.scenes.err = err
				return err
			}
		}
	}
	if g.current != nil {
		return g.current.Update(g)
	}
	return nil
}

// NumberScene annotates an existing t5/t8 Scene without changing its
// concrete ending/staff APIs (notably Ending.After).
func NumberScene(id SceneID, s Scene) Scene {
	if s == nil {
		return nil
	}
	return &numberedScene{id: id, Scene: s}
}

type numberedScene struct {
	Scene
	id SceneID
}

func (s *numberedScene) SceneMode() SceneID { return s.id }
func (s *numberedScene) SceneEnter(g *Game) error {
	if life, ok := s.Scene.(SceneLifecycle); ok {
		return life.SceneEnter(g)
	}
	return nil
}
func (s *numberedScene) SceneLeave(g *Game, next SceneID) {
	if life, ok := s.Scene.(SceneLifecycle); ok {
		life.SceneLeave(g, next)
	}
}

func (s *numberedScene) DrawPorted(sv *supervisor.Supervisor) {
	if draw, ok := s.Scene.(interface{ DrawPorted(*supervisor.Supervisor) }); ok {
		draw.DrawPorted(sv)
	}
}

// MenuSelection retains the native application-level selection globals;
// the run's modes/resources still live solely in t8's GameState.
type MenuSelection struct {
	Character, Shot, Difficulty int // DAT_00474c68/6c/74
	PracticeStage               int // DAT_00491c08, zero-based last practice cursor
	DemoIndex                   int // DAT_00474ca8, demo1/2/3/0 rotation
	Initialized                 bool
}
