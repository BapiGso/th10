package main

import (
	"bytes"
	"reflect"
	"testing"

	"th10/entity/item"
	"th10/entity/laser"
	"th10/entity/player"
	"th10/game"
	"th10/gamedata"
	"th10/replay"
	"th10/scene/stage"
	"th10/sim"
	"th10/trace"
	"th10/zun"
)

func TestSimExtensionReadsPortFieldsWithoutRecomputing(t *testing.T) {
	s := &game.GameState{HiScore: 123450, HiScoreContinues: 3, FaithHold: 71, FaithRate: 9, Rank: -12}
	p := &player.Player{IX: -1855, IY: 42000, StateTimer: zun.Timer{Cur: 14, F: 14.25}, Invinc: zun.Timer{Cur: 8, F: 8.75}}
	c := &stage.Context{State: s, Player: p, Bomb: &stage.Bomb{Timer: zun.Timer{Cur: 17, F: 17.5}}, Items: item.NewManager(), Lasers: laser.NewManager(), RNG: &zun.RNG{State: 123, Count: 5}}
	c.Items.Live = 71 // not derived from the two active rows
	c.Items.Items[0] = item.Item{Kind: 2, State: 1, Pos: [3]float32{10, 20}, T: zun.Timer{Cur: 9, F: 9.75}}
	c.Items.Items[2197] = item.Item{Kind: 8, State: 5, Pos: [3]float32{30, 40}}
	env := &laser.Env{}
	id := c.Lasers.FireLine(&laser.LineTemplate{Type: 31, Colour: 2, Pos: [3]float32{10, 20}, Angle: .5, Init: 80, Width: 16}, env)
	c.Lasers.FireInfinite(&laser.InfiniteTemplate{Type: 6, Colour: 7, Pos: [3]float32{30, 40}, Angle: 1, Init: 448, Width: 16}, env)
	c.Lasers.Find(id).Base().State = laser.StateDead
	beforeState, beforeRNG := *s, *c.RNG
	f := &trace.Frame{Index: 1, Items: 2}
	snapshotExtended(c, f)
	x := f.Ext
	if x.HiScore != 123450 || x.HiScoreContinues != 3 || x.FaithHold != 71 || x.FaithRate != 9 || x.Rank != -12 ||
		x.PlayerIX != -1855 || x.PlayerIY != 42000 || x.StateClock != 14 || x.StateClockF != 14.25 || x.InvincF != 8.75 || x.BombClock != 17 || x.BombClockF != 17.5 || x.ItemCalc != 71 {
		t.Fatalf("derived instead of read: %+v", x)
	}
	if len(f.I) != 2 || f.I[1].Slot != 2197 || f.I[0].ClockF != 9.75 || x.Lasers != 2 || len(f.L) != 2 || f.L[0].State != int(laser.StateDead) || f.L[1].Kind != 1 {
		t.Fatalf("full lists/pending dead: items=%+v lasers=%+v", f.I, f.L)
	}
	if !reflect.DeepEqual(*s, beforeState) || *c.RNG != beforeRNG {
		t.Fatal("extension modified port state/RNG")
	}
}

func TestSimOptInExtensionPreservesOrdinaryFrames(t *testing.T) {
	raw, err := gamedata.ReadFile("misc/demo0.rpy")
	if err != nil {
		t.Fatal(err)
	}
	rp, err := replay.Parse(raw)
	if err != nil {
		t.Fatal(err)
	}
	cfg, err := sim.ConfigFor(rp, 0)
	if err != nil {
		t.Fatal(err)
	}
	var plain, ext bytes.Buffer
	r := sim.New(cfg, scriptFactory)
	if err := r.Run(rp.Chapters[0], &plain, 3); err != nil {
		t.Fatal(err)
	}
	r = sim.New(cfg, scriptFactory)
	r.TraceFrame = extendedFrame
	if err := r.Run(rp.Chapters[0], &ext, 3); err != nil {
		t.Fatal(err)
	}
	a, b := trace.NewReader(&plain), trace.NewReader(&ext)
	for {
		x, err := a.Next()
		if err != nil {
			t.Fatal(err)
		}
		y, err := b.Next()
		if err != nil {
			t.Fatal(err)
		}
		if x == nil {
			if y != nil {
				t.Fatal("extension changed run length")
			}
			break
		}
		if y == nil || y.Ext == nil {
			t.Fatal("missing opt-in extension")
		}
		y.Ext, y.I, y.L = nil, nil, nil
		// Compare the serialized v2 portion; this also excludes reader bookkeeping.
		var xText, yText bytes.Buffer
		xw, yw := trace.NewWriter(&xText), trace.NewWriter(&yText)
		xw.WriteFrame(x)
		yw.WriteFrame(y)
		if err := xw.Flush(); err != nil {
			t.Fatal(err)
		}
		if err := yw.Flush(); err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(xText.Bytes(), yText.Bytes()) {
			t.Fatal("opt-in fields changed ordinary trace values")
		}
	}
}
