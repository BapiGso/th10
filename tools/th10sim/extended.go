package main

import (
	"th10/entity/laser"
	"th10/scene/stage"
	"th10/trace"
)

func extendedFrame(st *stage.Stage, f *trace.Frame) {
	snapshotExtended(stage.ExportContext(st), f)
}

// All values come from the port's corresponding native fields; this is an
// observer, not a second scoring, lifetime, rank or entity-count algorithm.
func snapshotExtended(c *stage.Context, f *trace.Frame) {
	s, p := c.State, c.Player
	x := &trace.Extended{HiScore: s.HiScore, HiScoreContinues: int(s.HiScoreContinues),
		FaithHold: s.FaithHold, FaithRate: s.FaithRate, Rank: s.Rank,
		PlayerIX: p.IX, PlayerIY: p.IY, StateClock: int(p.StateTimer.Cur), StateClockF: float64(p.StateTimer.F),
		InvincF: float64(p.Invinc.F), Focus: int(p.ReplayFocus()),
		BombActive: c.Bomb.Active(), BombClock: int(c.Bomb.Timer.Cur), BombClockF: float64(c.Bomb.Timer.F),
		ItemCalc: int(c.Items.Live), Lasers: c.Lasers.Count()}
	if c.GUI != nil {
		x.ShownScore = int64(c.GUI.ShownScore()) * 10
	}
	if card := c.Spell; card != nil {
		x.Spell = &trace.SpellRow{ID: int(card.ID), Clock: int(card.Timer.Cur), ClockF: float64(card.Timer.F),
			Bonus: card.Bonus, Initial: int(card.InitialBonus), Duration: int(card.Duration),
			Active: card.Active, Capture: card.Capture, Survival: card.Survival}
	}
	for slot := range c.Items.Items {
		it := &c.Items.Items[slot]
		if it.State != 0 {
			f.I = append(f.I, trace.ItemRow{Slot: slot, Type: int(it.Kind), State: int(it.State),
				X: float64(it.Pos[0]), Y: float64(it.Pos[1]), Clock: int(it.T.Cur), ClockF: float64(it.T.F)})
		}
	}
	c.Lasers.Inspect(func(l laser.Laser) {
		b := l.Base()
		row := trace.LaserRow{ID: int(b.ID), State: int(b.State), X: float64(b.Pos[0]), Y: float64(b.Pos[1]),
			Angle: float64(b.Angle), Length: float64(b.Length), Width: float64(b.Width), Clock: int(b.T.Cur), ClockF: float64(b.T.F)}
		switch l := l.(type) {
		case *laser.Line:
			row.Kind, row.Type, row.Color = 0, int(l.Tpl.Type), int(l.Tpl.Colour)
		case *laser.Infinite:
			row.Kind, row.Type, row.Color = 1, int(l.Tpl.Type), int(l.Tpl.Colour)
		}
		f.L = append(f.L, row)
	})
	f.Ext = x
}
