package loading

import (
	"github.com/hajimehoshi/ebiten/v2"
	"th10/anm"
	"th10/game"
	"th10/gamedata"
	"th10/scene/title"
	"th10/supervisor"
)

// Loading is FUN_0041f850/0041fd00's DAT_00477820. Embedded setup is
// synchronous; TitleGate models title initialization waiting for 300 draws.
type Loading struct {
	DrawCount        int // +0x3ec, advanced by Draw rather than Update
	Next             game.SceneFactory
	world            *anm.World
	env              *anm.Env
	sig, ascii       *anm.File
	logo, marker     *anm.VM
	ready, handedOff bool
}

func New() *Loading {
	return &Loading{Next: func(*game.Game) (game.Scene, error) { return title.New(), nil }}
}
func (*Loading) SceneMode() game.SceneID { return game.SceneLoading }

// SceneEnter is FUN_0041fac0 -> FUN_0041f990. All setup reads embedded
// resources, never install/dat paths or platform-specific thread APIs.
func (l *Loading) SceneEnter(g *game.Game) error {
	if l.ready {
		return nil
	}
	l.sig = gamedata.MustANM(gamedata.SharedANM("sig.anm"))
	l.ascii = gamedata.MustANM(gamedata.SharedANM("ascii.anm"))
	l.world = &anm.World{}
	l.env = &anm.Env{GameSpeed: 1, Alt: g.AlternateRNG(), Spawn: l.world.Add}
	l.ready = true
	return nil
}

// Update is 0x41feb0: resource completion requests title 4, but its calc
// remains disabled while the retained loading draw object is visible.
func (l *Loading) Update(g *game.Game) error {
	if err := l.SceneEnter(g); err != nil {
		return err
	}
	l.world.Run(l.env)
	if !l.handedOff {
		l.handedOff = true
		g.SystemFlags &^= 0x1000 // 0x41fed2
		g.GoToFactory(game.SceneTitle, func(g *game.Game) (game.Scene, error) {
			inner, err := l.Next(g)
			if err != nil {
				return nil, err
			}
			return &TitleGate{Loading: l, Title: inner}, nil
		})
	}
	return nil
}

// DrawPorted is FUN_0041fdd0, priority 2. StartKeep spawns sig 0 and
// ascii 6 here; the following calc populates the world's draw lists.
func (l *Loading) DrawPorted(sv *supervisor.Supervisor) {
	sv.BeginFrame()
	sv.Start()
	sv.Use(&sv.Screen)
	if l.ready {
		if l.logo == nil {
			l.logo = l.world.SpawnTail(l.sig, 0, 15, l.env)
		}
		if l.marker == nil {
			l.marker = l.world.SpawnTail(l.ascii, 6, 15, l.env)
			l.world.SetOffset(l.marker, [3]float32{480, 392, 0}) // 0x41fe56..0x41fe82
		}
		for layer := 0; layer < anm.Layers; layer++ {
			sv.ANM.DrawLayer(l.world, layer)
		}
	}
	l.DrawCount++ // 0x41fe91..0x41fe99
	sv.Finish()
	sv.EndFrame()
}
func (l *Loading) Draw(screen *ebiten.Image) {
	sv := supervisor.Get()
	l.DrawPorted(sv)
	if sv.Frame != nil {
		screen.DrawImage(sv.Frame, nil)
	}
}
func (l *Loading) release() {
	if l.world != nil {
		l.world.Clear()
	}
	l.logo, l.marker = nil, nil
}

// SceneLeave is FUN_0041fb50. The scene-4 gate temporarily retains the
// loading draw callback until FUN_0042c9f0's minimum display is satisfied.
func (l *Loading) SceneLeave(_ *game.Game, next game.SceneID) {
	if !l.handedOff || next != game.SceneTitle {
		l.release()
	}
}

type TitleGate struct {
	Loading *Loading
	Title   game.Scene
	enabled bool
}

func (*TitleGate) SceneMode() game.SceneID { return game.SceneTitle }
func (t *TitleGate) SceneEnter(g *game.Game) error {
	if life, ok := t.Title.(game.SceneLifecycle); ok {
		return life.SceneEnter(g)
	}
	return nil
}
func (t *TitleGate) SceneLeave(g *game.Game, next game.SceneID) {
	t.Loading.release()
	if life, ok := t.Title.(game.SceneLifecycle); ok {
		life.SceneLeave(g, next)
	}
}
func (t *TitleGate) Update(g *game.Game) error {
	if !t.enabled {
		// FUN_0042c9f0, 0x42ca2a/0x42ca40: 300 draws or system bit 0x80.
		if t.Loading.DrawCount < 300 && g.SystemFlags&0x80 == 0 {
			t.Loading.world.Run(t.Loading.env)
			return nil
		}
		t.Loading.release()
		t.enabled = true
	}
	return t.Title.Update(g)
}
func (t *TitleGate) Draw(screen *ebiten.Image) {
	if t.enabled {
		t.Title.Draw(screen)
	} else {
		t.Loading.Draw(screen)
	}
}
func (t *TitleGate) DrawPorted(sv *supervisor.Supervisor) {
	if !t.enabled {
		t.Loading.DrawPorted(sv)
		return
	}
	if draw, ok := t.Title.(interface{ DrawPorted(*supervisor.Supervisor) }); ok {
		draw.DrawPorted(sv)
	}
}
