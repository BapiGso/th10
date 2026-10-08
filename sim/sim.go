// Package sim drives a stage headlessly from a recorded input stream and
// emits per-frame traces in the shared trace format.
//
// It is the replica half of the differential harness: the same replay input
// stream that tools/th10probe samples from the original process is fed here
// through input.State.Restore, and the two traces are compared by
// tools/th10diff.
package sim

import (
	"fmt"
	"io"

	"th10/entity/bullet"
	"th10/game"
	"th10/input"
	"th10/replay"
	"th10/scene/stage"
	"th10/trace"
)

// Config selects what to simulate.
type Config struct {
	Stage      int // 1..6, 7=Extra
	Character  int // game.CharReimu / game.CharMarisa
	ShotType   int // game.ShotA/B/C
	Difficulty int // game.DiffEasy..DiffExtra
	Power      int // starting power in hundredths
	Seed       uint16
}

// ScriptFactory builds the stage script for a stage number. Wired by the
// caller (usually cmd) so this package does not import every stage package.
type ScriptFactory func(stageNum int) stage.Script

// Runner drives one headless stage run.
type Runner struct {
	cfg     Config
	factory ScriptFactory
	sim     *headless
}

// New builds a runner. factory must return the same script type the real game
// uses for cfg.Stage.
func New(cfg Config, factory ScriptFactory) *Runner {
	return &Runner{cfg: cfg, factory: factory}
}

// Run plays the replay chapter's input records frame by frame and writes a
// trace to out. It stops at the end of the input, when the stage finishes, or
// at maxFrames (0 = no cap).
func (r *Runner) Run(ch replay.Chapter, out io.Writer, maxFrames int) error {
	h, err := newHeadless(r.cfg, r.factory)
	if err != nil {
		return err
	}
	tw := trace.NewWriter(out)
	limit := len(ch.Records)
	if maxFrames > 0 && maxFrames < limit {
		limit = maxFrames
	}
	for i := 0; i < limit; i++ {
		rec := ch.Records[i]
		h.step(rec)
		tw.WriteFrame(h.sample(i + 1))
		if h.finished {
			break
		}
	}
	return tw.Flush()
}

// headless owns the simulated world.
type headless struct {
	st      *stage.Stage
	ctx     *stage.Context
	state   *game.GameState
	in      input.State
	gameRef *game.Game
	finished bool
}

func newHeadless(cfg Config, factory ScriptFactory) (*headless, error) {
	if factory == nil {
		return nil, fmt.Errorf("sim: no script factory")
	}
	state := game.NewState(cfg.Character, cfg.ShotType, cfg.Difficulty)
	state.Stage = cfg.Stage
	state.SingleStage = true
	state.DebugMode = true
	state.Power = cfg.Power
	if cfg.Difficulty == game.DiffExtra {
		state.Difficulty = game.DiffExtra
	}
	g := &game.Game{}
	st := stage.NewHeadless(state, factory(cfg.Stage))
	ctx := stage.ExportContext(st)
	if ctx == nil {
		return nil, fmt.Errorf("sim: stage context unavailable")
	}
	return &headless{st: st, ctx: ctx, state: state, gameRef: g}, nil
}

// step advances one frame with the given recorded input.
func (h *headless) step(rec replay.Record) {
	h.in.Restore(restoreBits(rec))
	h.st.UpdateHeadless(&h.in)
	h.finished = h.st.FinishedHeadless()
}

// sample snapshots the current world into a trace frame.
func (h *headless) sample(index int) *trace.Frame {
	f := &trace.Frame{
		Index:    index,
		PX:       h.ctx.Player.X,
		PY:       h.ctx.Player.Y,
		PLife:    h.state.Life,
		PPower:   h.state.Power,
		PFaith:   h.state.Faith,
		PScore:   h.state.Score,
		PGraze:   h.state.Graze,
		PState:   int(h.ctx.Player.State),
		PInvinc:  h.ctx.Player.InvincTimer,
		PBombTimer: h.ctx.Player.BombTimer,
		Enemies:  activeEnemies(h.ctx),
		Bullets:  h.ctx.Bullets.ActiveCount(),
		Items:    h.ctx.Items.ActiveCount(),
		PBullets: h.ctx.Player.Bullets.ActiveCount(),
	}
	for _, e := range h.ctx.Enemies {
		if len(f.E) >= trace.MaxEnemyRows {
			break
		}
		f.E = append(f.E, trace.EnemyRow{
			X: e.X, Y: e.Y, HP: e.HP, Active: e.Active, Type: int(e.Type),
		})
	}
	h.ctx.Bullets.Each(func(b *bullet.Bullet) {
		if len(f.B) >= trace.MaxBulletRows {
			return
		}
		f.B = append(f.B, trace.BulletRow{
			X: b.X, Y: b.Y, Angle: b.Angle, Speed: b.Speed,
			Type: int(b.Type), Color: b.Color,
		})
	})
	return f
}

func activeEnemies(ctx *stage.Context) int {
	n := 0
	for _, e := range ctx.Enemies {
		if e != nil && e.Active {
			n++
		}
	}
	return n
}

// restoreBits maps a replay record onto the input.State bit layout.
// input.Key order: Up, Down, Left, Right, Shot, Bomb, Focus, Pause, Ok, Cancel.
func restoreBits(rec replay.Record) uint16 {
	var bits uint16
	set := func(k input.Key, on bool) {
		if on {
			bits |= 1 << uint(k)
		}
	}
	set(input.KeyUp, rec.Up())
	set(input.KeyDown, rec.Down())
	set(input.KeyLeft, rec.Left())
	set(input.KeyRight, rec.Right())
	set(input.KeyShot, rec.Keys&replay.KeyShot != 0)
	set(input.KeyBomb, rec.Keys&replay.KeyBomb != 0)
	set(input.KeyFocus, rec.Keys&replay.KeyFocus != 0)
	// KeyOk doubles as Z for menus/dialog; the original advances dialog with the
	// shot key, so mirror it.
	set(input.KeyOk, rec.Keys&replay.KeyShot != 0)
	return bits
}
