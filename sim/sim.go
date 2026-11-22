// Package sim drives a stage headlessly from a recorded input stream and
// emits per-frame traces in the shared trace format.
//
// It is the replica half of the differential harness: the same replay input
// stream that tools/th10probe samples from the original process is fed here
// through input.State.Restore, and the two traces are compared by
// tools/th10diff.
package sim

import (
	"encoding/binary"
	"fmt"
	"io"
	"math"
	"os"
	"runtime"
	"strconv"
	"strings"

	"th10/entity/bullet"
	"th10/game"
	"th10/input"
	"th10/replay"
	"th10/scene/stage"
	"th10/scene/stage/ecl"
	"th10/trace"
	"th10/zun"
)

// Config selects what to simulate.
type Config = replay.RunConfig

// ScriptFactory builds the stage script for a stage number. Wired by the
// caller (usually cmd) so this package does not import every stage package.
type ScriptFactory func(stageNum int) stage.Script

// Runner drives one headless stage run.
type Runner struct {
	cfg     Config
	factory ScriptFactory
	sim     *headless
	// OnFrame, if set, is called after each simulated frame (after the step,
	// before the frame's trace is written) with the headless stage.
	OnFrame func(st *stage.Stage)
	// Recorder observes the same input cursor, including transition input
	// after MSG op 20. Those tail records do not advance or lengthen trace.
	Recorder *replay.Recorder
	// ConfigureState is an optional tool-only initial-condition override,
	// called before stage/GUI construction. Nil preserves the default run.
	ConfigureState func(*game.GameState)
	// TraceFrame adds optional diagnostic fields after the ordinary snapshot.
	// It must not modify the stage or consume RNG.
	TraceFrame func(*stage.Stage, *trace.Frame)
	// Writer is optionally supplied by tools for atomic capture/session output.
	Writer *trace.Writer
}

// New builds a runner. factory must return the same script type the real game
// uses for cfg.Stage.
func New(cfg Config, factory ScriptFactory) *Runner {
	return &Runner{cfg: cfg, factory: factory}
}

// ConfigFor builds the Config that replays chapter chapter (0-based) of rp,
// with the character, shot, difficulty and start state the original ran with.
func ConfigFor(rp *replay.Replay, chapter int) (Config, error) {
	return replay.ConfigFor(rp, chapter)
}

// Run plays the replay chapter's input records frame by frame and writes a
// trace to out. It stops at the end of the input, when the stage finishes, or
// at maxFrames (0 = no cap).
func (r *Runner) Run(ch replay.Chapter, out io.Writer, maxFrames int) error {
	h, err := newHeadless(r.cfg, r.factory, 0, r.ConfigureState)
	if err != nil {
		return err
	}
	r.sim = h
	h.st.RestoreReplayChapter(ch)
	if r.Recorder != nil {
		if err := h.st.StartRecording(r.Recorder); err != nil {
			return err
		}
	}
	tw := r.Writer
	if tw == nil {
		tw = trace.NewWriter(out)
	}
	limit := len(ch.Records)
	if maxFrames > 0 && maxFrames < limit {
		limit = maxFrames
	}
	stream := replay.NewStream(ch)
	h.in.RestoreReplay(stream.Prime())
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
	// TH10SIM_RNGLOG=first-last logs every draw from the game RNG in those
	// frames to stderr, with the calling functions.
	if rng0, rng1 := frameRange(os.Getenv("TH10SIM_RNGLOG")); rng0 >= 0 {
		zun.RNGHook = func(g *zun.RNG, v uint32) {
			if frame < rng0 || frame > rng1 || g != h.ctx.RNG {
				return
			}
			pc := make([]uintptr, 32)
			n := runtime.Callers(3, pc)
			fr := runtime.CallersFrames(pc[:n])
			fmt.Fprintf(os.Stderr, "%d #%d %08x", frame, g.Count/2, v)
			for {
				f, more := fr.Next()
				name := f.Function
				if k := strings.LastIndexByte(name, '/'); k >= 0 {
					name = name[k+1:]
				}
				fmt.Fprintf(os.Stderr, " < %s", name)
				if !more {
					break
				}
			}
			fmt.Fprintln(os.Stderr)
		}
		defer func() { zun.RNGHook = nil }()
	}
	for i := 0; i < limit; i++ {
		frame = i + 1
		rec, rate, _ := stream.Next()
		h.ctx.ReplaySlowSample = rate
		if h.finished {
			// FUN_004297d0 still consumes the transition/menu tail. The
			// headless world and its trace end at the original clear event.
			r.Recorder.Append(rec, rate)
			continue
		}
		h.step(rec)
		if r.OnFrame != nil {
			r.OnFrame(h.st)
		}
		f := h.sample(i + 1)
		if r.TraceFrame != nil {
			r.TraceFrame(h.st, f)
		}
		tw.WriteFrame(f)
		if i+1 >= dump0 && i+1 <= dump1 {
			h.dumpBullets(os.Stderr, i+1)
		}
		if h.finished && r.Recorder == nil {
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

// newHeadlessWithAlternateSeed is newHeadless without a state override
// (kept for sim's alternate-RNG tests).
func newHeadlessWithAlternateSeed(cfg Config, factory ScriptFactory, seed uint16) (*headless, error) {
	return newHeadless(cfg, factory, seed, nil)
}

// newHeadless builds the world for cfg. seed is the alternate generator
// DAT_004918a8, injected before STD/ANM initialization; replay seeds only
// restore the main generator (FUN_00428f60 / FUN_0042a6a0). configure, when
// set, overrides the initial state before stage/GUI construction.
func newHeadless(cfg Config, factory ScriptFactory, seed uint16, configure func(*game.GameState)) (*headless, error) {
	if factory == nil {
		return nil, fmt.Errorf("sim: no script factory")
	}
	g := &game.Game{}
	g.AlternateRNG().State = seed
	state := g.NewState(cfg.Character, cfg.ShotType, cfg.Difficulty)
	state.Stage = cfg.Stage
	state.SingleStage = true
	state.DebugMode = true
	state.StageMode, state.ReplayMode, state.FreshRun = 1, 1, cfg.FreshRun
	state.Power = cfg.Power
	state.Life = cfg.Lives
	state.RNGSeed, state.HasSeed = cfg.Seed, true
	state.ReplayConfig = cfg.Config
	state.ConfigFlags = binary.LittleEndian.Uint32(cfg.Config[0x30:])
	state.Rank = cfg.Rank
	if cfg.Difficulty == game.DiffExtra {
		state.Difficulty = game.DiffExtra
	}
	if configure != nil {
		configure(state)
	}
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
	h.in.RestoreReplay(rec)
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
	// The enemy manager's list order, hidden main first (set by the ECL VM).
	enemies := h.ctx.ManagerOrder()
	bomb := 0
	if b := h.ctx.Bomb; b.Active() {
		bomb = int(b.Timer.Cur)
	}
	f := &trace.Frame{
		Index:      index,
		PX:         h.ctx.Player.X - gameOriginX,
		PY:         h.ctx.Player.Y - gameOriginY,
		PLife:      h.state.Life,
		PPower:     h.state.Power,
		PFaith:     h.state.Faith,
		PScore:     h.state.Score,
		PGraze:     0, // TH10 keeps no graze count (FUN_00406240 / FUN_0041d3d0 only set bullet flag 4, spark, sound)
		PState:     int(h.ctx.Player.State),
		PInvinc:    int(h.ctx.Player.Invinc.Cur),
		PBombTimer: bomb,
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
