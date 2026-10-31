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
	"math"
	"os"
	"strconv"
	"strings"

	"th10/entity/bullet"
	"th10/entity/enemy"
	"th10/game"
	"th10/input"
	"th10/replay"
	"th10/scene/stage"
	"th10/scene/stage/ecl"
	"th10/trace"
)

// Config selects what to simulate.
type Config struct {
	Stage      int // 1..6, 7=Extra
	Character  int // game.CharReimu / game.CharMarisa
	ShotType   int // game.ShotA/B/C
	Difficulty int // game.DiffEasy..DiffExtra
	Power      int // starting power in hundredths
	Lives      int // lives in stock at the chapter's start
	Seed       uint16
	Rank       int // dynamic rank at the chapter's start
	// PlayerX, PlayerY are the player's fixed-point start position from the
	// chapter header (FUN_00428e10).
	PlayerX, PlayerY int32
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
	if limit > 0 {
		// The chapter can start with keys held from before it: Aux1 is the
		// press edge (DAT_00474e62), so seed the previous frame with the
		// held keys that do not newly go down on frame 1.
		r0 := ch.Records[0]
		h.in.Restore(restoreBits(replay.Record{Keys: r0.Keys &^ r0.Aux1}))
	}
	// TH10SIM_BDUMP=frame or first-last dumps bullets to stderr.
	dump0, dump1 := frameRange(os.Getenv("TH10SIM_BDUMP"))
	// TH10SIM_ETRACE=first-last logs every ECL instruction run in those
	// frames to stderr.
	etrace0, etrace1 := frameRange(os.Getenv("TH10SIM_ETRACE"))
	frame := 0
	if etrace0 >= 0 {
		ecl.Trace = func(sub string, in *ecl.Instr) {
			if frame < etrace0 || frame > etrace1 {
				return
			}
			fmt.Fprintf(os.Stderr, "%d %s@%d t%d op%d", frame, sub, in.Offset, in.Time, in.Opcode)
			for _, p := range in.Params {
				switch {
				case p.IsVar:
					fmt.Fprintf(os.Stderr, " $%d", int32(p.Raw))
				case p.Type == 'f':
					fmt.Fprintf(os.Stderr, " %gf", math.Float32frombits(p.Raw))
				default:
					fmt.Fprintf(os.Stderr, " %d", int32(p.Raw))
				}
			}
			fmt.Fprintln(os.Stderr)
		}
		defer func() { ecl.Trace = nil }()
	}
	for i := 0; i < limit; i++ {
		frame = i + 1
		rec := ch.Records[i]
		h.step(rec)
		tw.WriteFrame(h.sample(i + 1))
		if i+1 >= dump0 && i+1 <= dump1 {
			h.dumpBullets(os.Stderr, i+1)
		}
		if h.finished {
			break
		}
	}
	return tw.Flush()
}

// headless owns the simulated world.
type headless struct {
	st       *stage.Stage
	ctx      *stage.Context
	state    *game.GameState
	in       input.State
	gameRef  *game.Game
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
	state.Life = cfg.Lives
	state.RNGSeed, state.HasSeed = cfg.Seed, true
	state.Rank = cfg.Rank
	if cfg.Difficulty == game.DiffExtra {
		state.Difficulty = game.DiffExtra
	}
	g := &game.Game{}
	st := stage.NewHeadless(state, factory(cfg.Stage))
	ctx := stage.ExportContext(st)
	if ctx == nil {
		return nil, fmt.Errorf("sim: stage context unavailable")
	}
	ctx.Player.SetFixed(cfg.PlayerX, cfg.PlayerY)
	return &headless{st: st, ctx: ctx, state: state, gameRef: g}, nil
}

// step advances one frame with the given recorded input.
func (h *headless) step(rec replay.Record) {
	h.in.Restore(restoreBits(rec))
	h.st.UpdateHeadless(&h.in)
	h.finished = h.st.FinishedHeadless()
}

// Trace positions are in the original's game space (package trace): the
// playfield's top-centre is the origin.
const (
	gameOriginX = float64(game.FieldLeft+game.FieldRight) / 2
	gameOriginY = float64(game.FieldTop)
)

// sample snapshots the current world into a trace frame.
func (h *headless) sample(index int) *trace.Frame {
	enemies := liveEnemies(h.ctx)
	f := &trace.Frame{
		Index:      index,
		PX:         h.ctx.Player.X - gameOriginX,
		PY:         h.ctx.Player.Y - gameOriginY,
		PLife:      h.state.Life,
		PPower:     h.state.Power,
		PFaith:     h.state.Faith,
		PScore:     h.state.Score,
		PGraze:     h.state.Graze,
		PState:     int(h.ctx.Player.State),
		PInvinc:    h.ctx.Player.InvincTimer,
		PBombTimer: h.ctx.Player.BombTimer,
		Enemies:    len(enemies),
		Bullets:    h.ctx.Bullets.Live(),
		Items:      h.ctx.Items.ActiveCount(),
		PBullets:   h.ctx.Player.Bullets.ActiveCount(),
	}
	if m := h.ctx.MSG(); m != nil {
		clock, wait, at, op := m.Pos()
		f.MSG = &trace.MSGRow{ID: int(m.ID), Clock: clock, Wait: wait, Signal: m.Signal, At: at, Op: op}
	}
	for _, e := range enemies {
		if len(f.E) >= trace.MaxEnemyRows {
			break
		}
		// The probe's Active is "not flagged 0x20000": an enemy swept by
		// FUN_0040e6a0 stays listed until the manager's next walk deletes it.
		f.E = append(f.E, trace.EnemyRow{
			X: e.X - gameOriginX, Y: e.Y - gameOriginY, HP: e.HP, Active: e.ECLFlags&0x20000 == 0, Type: int(e.Type),
		})
	}
	h.ctx.Bullets.Each(func(b *bullet.Bullet) {
		if len(f.B) >= trace.MaxBulletRows {
			return
		}
		f.B = append(f.B, trace.BulletRow{
			X: float64(b.Pos[0]), Y: float64(b.Pos[1]), Angle: float64(b.Angle), Speed: float64(b.Speed),
			Type: int(b.Type), Color: int(b.Colour),
		})
	})
	return f
}

// frameRange parses "n" or "first-last"; -1, -1 when empty or malformed.
func frameRange(v string) (int, int) {
	a, b, ok := strings.Cut(v, "-")
	first, err := strconv.Atoi(a)
	if err != nil {
		return -1, -1
	}
	if !ok {
		return first, first
	}
	last, err := strconv.Atoi(b)
	if err != nil {
		return -1, -1
	}
	return first, last
}

// dumpBullets writes every live bullet's motion fields at full precision,
// with their float32 bits, for chasing sub-ulp divergences
// (TH10SIM_BDUMP=frame).
func (h *headless) dumpBullets(w io.Writer, frame int) {
	n := 0
	h.ctx.Bullets.Each(func(b *bullet.Bullet) {
		f := func(v float32) string { return fmt.Sprintf("%.9g/%08x", v, math.Float32bits(v)) }
		fmt.Fprintf(w, "%d B%d pos %s %s ang %s spd %s vel %s %s ex %#x state %d\n", frame, n,
			f(b.Pos[0]), f(b.Pos[1]), f(b.Angle), f(b.Speed), f(b.Vel[0]), f(b.Vel[1]), b.ExFlags, b.State)
		n++
	})
}

// liveEnemies is the enemy list in the original manager's order (hidden main
// first) when the stage runs an ECL VM, else the stage's own list.
func liveEnemies(ctx *stage.Context) []*enemy.Enemy {
	if ctx.ManagerOrder != nil {
		return ctx.ManagerOrder()
	}
	out := make([]*enemy.Enemy, 0, len(ctx.Enemies))
	for _, e := range ctx.Enemies {
		if e != nil && e.Active {
			out = append(out, e)
		}
	}
	return out
}

// restoreBits maps a replay record onto the input.State bit layout.
// input.Key order: Up, Down, Left, Right, Shot, Bomb, Focus, Pause, Ok, Cancel, Skip.
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
	set(input.KeySkip, rec.Keys&replay.KeySkip != 0)
	return bits
}
