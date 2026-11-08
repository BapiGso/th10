package stage

import (
	"encoding/binary"
	"fmt"
	"th10/game"
	"th10/input"
	"th10/replay"
	sceneui "th10/scene/ui"
	"th10/supervisor"

	"github.com/hajimehoshi/ebiten/v2"
)

// Playback drives interactive stage scenes with the same ConfigFor, Stream,
// RestoreReplay and chapter-state restore used by sim. It owns no user-file
// I/O; the caller obtains the decoded replay through game persistence.
type Playback struct {
	source    *replay.Replay
	chapter   int
	factory   func(int) Script
	stage     *Stage
	stream    *replay.Stream
	in        input.State
	back      game.Scene
	done      bool
	modeFlags uint32
	demoFade  *sceneui.Fade
	OnFrame   func(*Stage) // inspection hook, world frames only, like sim.Runner
}

func NewPlayback(g *game.Game, source *replay.Replay, first int, factory func(int) Script, back game.Scene) (*Playback, error) {
	return newPlayback(g, source, first, factory, back, 0)
}

// NewDemoPlayback is the title's native request 12. Set bit 0x20 before
// stage/GUI/resource construction, while keeping t5's ordinary entry intact.
func NewDemoPlayback(g *game.Game, source *replay.Replay, first int, factory func(int) Script, back game.Scene) (*Playback, error) {
	return newPlayback(g, source, first, factory, back, game.RunFlagDemo)
}

func newPlayback(g *game.Game, source *replay.Replay, first int, factory func(int) Script, back game.Scene, flags uint32) (*Playback, error) {
	if err := source.ValidatePlayback(); err != nil {
		return nil, err
	}
	if first < 0 || first >= len(source.Chapters) || factory == nil {
		return nil, fmt.Errorf("stage: invalid playback chapter or factory")
	}
	p := &Playback{source: source, chapter: first, factory: factory, back: back, modeFlags: flags}
	if err := p.startChapter(g); err != nil {
		return nil, err
	}
	return p, nil
}

func (p *Playback) startChapter(g *game.Game) error {
	cfg, err := replay.ConfigFor(p.source, p.chapter)
	if err != nil {
		return err
	}
	// Set playback mode before New binds shared state; even binding must
	// leave an injected empty save view unchanged (t5's read-only contract).
	state := game.NewState(cfg.Character, cfg.ShotType, cfg.Difficulty)
	state.Stage, state.StageMode, state.ReplayMode, state.FreshRun = cfg.Stage, 1, 1, cfg.FreshRun
	state.RunFlags = p.modeFlags
	state.Power, state.Life, state.Rank = cfg.Power, cfg.Lives, cfg.Rank
	state.RNGSeed, state.HasSeed = cfg.Seed, true
	state.ReplayConfig = cfg.Config
	state.ConfigFlags = binary.LittleEndian.Uint32(cfg.Config[0x30:])
	if g.Save != nil {
		top := g.ScoreTable(state)[0]
		state.HiScore, state.HiScoreContinues = int64(top.Score)*10, int32(int8(top.Continues))
	}
	p.stage = New(g, state, p.factory(cfg.Stage))
	p.stage.RestoreReplayChapter(p.source.Chapters[p.chapter])
	p.stream = replay.NewStream(p.source.Chapters[p.chapter])
	p.in = input.State{}
	p.in.RestoreReplay(p.stream.Prime())
	return nil
}

func (p *Playback) Stage() *Stage  { return p.stage }
func (p *Playback) Chapter() int   { return p.chapter }
func (p *Playback) Finished() bool { return p.done }

// step consumes transition-tail records without advancing an already
// finished world, exactly as sim's recording observer does. Each following
// chapter uses its stored seed and state, never the previous run's outcome.
func (p *Playback) step(g *game.Game) error {
	if p.done {
		return nil
	}
	if p.modeFlags&game.RunFlagDemo != 0 {
		// FUN_00418190, 0x4186a7..0x4186d2: fade at 2940, leave at 3000.
		if p.stage.ctx.State.Frame >= 3000 || p.stage.FinishedHeadless() {
			p.done = true
			g.GoTo(p.back)
			return nil
		}
		if p.demoFade != nil {
			p.demoFade.Calc(1, false, 0)
		}
	}
	rec, rate, ok := p.stream.Next()
	if ok && !p.stage.FinishedHeadless() {
		p.in.RestoreReplay(rec)
		p.stage.ctx.ReplaySlowSample = rate
		p.stage.UpdateHeadless(&p.in)
		if p.modeFlags&game.RunFlagDemo != 0 && p.stage.ctx.State.Frame == 2940 {
			p.demoFade = sceneui.NewFade(5, 60, 0)
		}
		if p.OnFrame != nil {
			p.OnFrame(p.stage)
		}
	}
	if !ok || p.stream.Done() {
		if p.chapter+1 < len(p.source.Chapters) {
			p.chapter++
			return p.startChapter(g)
		}
		p.done = true
		g.GoTo(p.back)
	}
	return nil
}

// UpdateHeadless exercises the very same playback flow without physical
// menu keys; used for saved-replay and sim-equivalence tests.
func (p *Playback) UpdateHeadless(g *game.Game) error { return p.step(g) }

func (p *Playback) Update(g *game.Game) error {
	if p.modeFlags&game.RunFlagDemo != 0 && input.Global.Menu().Current&input.MenuIdleReset != 0 {
		p.done = true
		g.GoTo(p.back)
		return nil // FUN_00418190, 0x418679..0x4186a1
	}
	// Esc uses the native two-choice playback pause menu. Its input is
	// physical and separate from the recorded input/stream cursor.
	if action, blocked := p.stage.UpdateMenu(g, &input.Global); blocked {
		if action == MenuTitle || action == MenuRetry {
			p.done = true
			g.GoTo(p.back)
		}
		return nil
	}
	// Physical X can stop playback; recorded bombs remain in p.in and
	// cannot accidentally trigger a menu. Neither path writes progress.
	if input.Global.Menu().Pressed&input.MenuBomb != 0 {
		p.done = true
		g.GoTo(p.back)
		return nil
	}
	return p.step(g)
}

func (p *Playback) Draw(screen *ebiten.Image) {
	p.stage.Draw(screen)
	if p.demoFade != nil && p.demoFade.Active {
		sv := supervisor.Get()
		p.demoFade.Draw(sv)
		sv.EndFrame()
		if sv.Frame != nil {
			screen.DrawImage(sv.Frame, nil)
		}
	}
}

func (p *Playback) DrawPorted(sv *supervisor.Supervisor) {
	p.stage.runChain(sv, false)
	if p.demoFade != nil {
		p.demoFade.Draw(sv)
		sv.EndFrame()
	}
}
