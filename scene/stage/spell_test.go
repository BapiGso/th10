package stage

import (
	"math"
	"testing"

	"th10/game"
	"th10/zun"
)

func TestSpellBonusStages(t *testing.T) {
	// Hand values for stored faith 5000, 5001 and 99999. Odd stored faith
	// remains odd; the project's extra units are truncated before the product.
	for _, c := range []struct {
		stage            int
		floor, odd, full int32
	}{
		{1, 650000, 650130, 12999870},
		{2, 800000, 800160, 15999840},
		{3, 950000, 950190, 18999810},
		{4, 1100000, 1100220, 21999780},
		{5, 1250000, 1250250, 24999750},
		{6, 1400000, 1400280, 27999720},
		{7, 1550000, 1550310, 30999690},
	} {
		for _, f := range []struct {
			faith int64
			want  int32
		}{{50000, c.floor}, {50001, c.floor}, {50009, c.floor}, {50010, c.odd}, {50019, c.odd}, {999990, c.full}} {
			bonus, initial := SpellBonus(c.stage, f.faith)
			if bonus != f.want || initial != f.want {
				t.Errorf("stage %d faith %d = %d/%d, want %d", c.stage, f.faith, bonus, initial, f.want)
			}
		}
	}
}

func TestSpellBonusCap(t *testing.T) {
	// FUN_00409280 caps only +0x3794. Such values cannot be reached with
	// the original faith cap, but distinguish the two fields at the boundary.
	for _, c := range []struct {
		faith          int64
		bonus, initial int32
	}{{7692300, 99999900, 99999900}, {7692310, 100000030, 99999999}, {10000000, 130000000, 99999999}} {
		bonus, initial := SpellBonus(1, c.faith)
		if bonus != c.bonus || initial != c.initial {
			t.Errorf("faith %d = %d/%d, want %d/%d", c.faith, bonus, initial, c.bonus, c.initial)
		}
	}
}

func TestSpellDeclare(t *testing.T) {
	s := game.NewState(game.CharMarisa, game.ShotC, game.DiffLuna)
	s.Stage, s.Faith = 6, 50019
	c := &SpellCard{Survival: true, Faded: true, ID: 3, Bonus: -10}
	c.Timer.Set(500)
	c.Declare(s, 60, "\x90\x4d\x95\x84", 2700)
	if !c.Active || !c.Capture || c.Survival || !c.Faded || c.ID != 60 || c.Name != "\x90\x4d\x95\x84" || c.Duration != 2700 || c.Bonus != 1400280 || c.InitialBonus != 1400280 {
		t.Fatalf("declaration = %+v", c)
	}
	if c.Timer != (zun.Timer{Prev: -1, Cur: 0, F: 0}) {
		t.Fatalf("timer = %+v", c.Timer)
	}
	if caps, seen := s.SpellCounts(60); caps != 0 || seen != 1 {
		t.Fatalf("declaration history = %d/%d", caps, seen)
	}
}

func TestSpellDecaySequence(t *testing.T) {
	for _, c := range []struct {
		faith int64
		want  []int64
	}{
		{50000, []int64{650000, 649750, 649500, 649250, 649000, 648750}},
		{50010, []int64{650130, 649880, 649630, 649380, 649130, 648880}},
	} {
		s := game.NewState(game.CharReimu, game.ShotA, game.DiffEasy)
		s.Stage, s.Faith = 1, c.faith
		card := &SpellCard{}
		card.Declare(s, 2, "card", 2700)
		card.Timer.Set(299)
		for i, want := range c.want {
			card.Tick(1)
			if card.Bonus != want || card.Timer.Cur != int32(300+i) || card.Timer.Prev != int32(299+i) {
				t.Fatalf("faith %d step %d bonus=%d timer=%+v, want %d at %d", c.faith, i, card.Bonus, card.Timer, want, 300+i)
			}
		}
	}
}

func TestSpellDecayFailedAndSlow(t *testing.T) {
	card := &SpellCard{Active: true, Capture: false, Bonus: 0, InitialBonus: 650000, Duration: 2700}
	card.Timer.Set(300)
	for i, want := range []int64{-240, -480, -720, -960} {
		card.Tick(0.5)
		if card.Bonus != want || card.Capture || card.Timer.Cur != int32(300+(i+1)/2) || card.Timer.F != 300+float32(i+1)*0.5 {
			t.Errorf("step %d = %+v, want bonus %d", i, card, want)
		}
	}
}

func TestSpellTickKeepsTimerBits(t *testing.T) {
	for _, speed := range []float32{1, 0.991, 1.009, 0.99, 1.01, 0.5} {
		card := &SpellCard{Active: true, Survival: true}
		card.Timer.Set(27)
		want := card.Timer
		for range 32 {
			want.Tick(speed)
			card.Tick(speed)
			if card.Timer.Prev != want.Prev || card.Timer.Cur != want.Cur || math.Float32bits(card.Timer.F) != math.Float32bits(want.F) {
				t.Fatalf("speed %v timer=%+v, want %+v", speed, card.Timer, want)
			}
		}
	}
	card := &SpellCard{Bonus: 100}
	card.Timer.Set(301)
	want := *card
	card.Tick(1)
	if card.Timer != want.Timer || card.Bonus != want.Bonus || card.Active != want.Active {
		t.Fatal("inactive card ticked")
	}
}

func TestSpellSurvivalBonus(t *testing.T) {
	card := &SpellCard{Active: true, Capture: true, Survival: true, Bonus: 650000, InitialBonus: 650000, Duration: 300}
	card.Timer.Set(300)
	for range 1000 {
		card.Tick(1) // no division by zero: survival bypasses the decay branch
	}
	if card.Bonus != 650000 || !card.Capture || card.Timer.Cur != 1300 {
		t.Fatalf("survival = %+v", card)
	}
}

func TestSpellFinishPaths(t *testing.T) {
	for _, c := range []struct {
		name                      string
		capture, survival, replay bool
		bonus, score              int64
		caps                      int32
	}{
		{"capture", true, false, false, 649757, 649750, 1},
		{"zero capture", true, false, false, 0, 0, 1},
		{"failure", false, false, false, 649750, 0, 0},
		{"survival capture", true, true, false, 650000, 650000, 1},
		{"survival failure", false, true, false, 650000, 0, 0},
		{"replay capture", true, false, true, 650000, 650000, 0},
	} {
		t.Run(c.name, func(t *testing.T) {
			s := game.NewState(game.CharReimu, game.ShotA, game.DiffEasy)
			if c.replay {
				s.ReplayMode = 1
			}
			s.RecordSpellDeclaration(2, "card")
			card := &SpellCard{Active: true, Capture: c.capture, Survival: c.survival, ID: 2, Bonus: c.bonus}
			card.Timer.Set(600)
			timer := card.Timer
			ctx := &Context{State: s}
			if got := card.Finish(ctx); got != c.capture || card.Active || card.Capture != c.capture || card.Timer != timer || card.ID != 2 || card.Bonus != c.bonus {
				t.Fatalf("finish %v, card=%+v", got, card)
			}
			if s.Score != c.score || s.HiScore != c.score {
				t.Fatalf("score = %d/%d, want %d", s.Score, s.HiScore, c.score)
			}
			if caps, _ := s.SpellCounts(2); caps != c.caps {
				t.Fatalf("captures = %d, want %d", caps, c.caps)
			}
			if card.Finish(ctx) || s.Score != c.score {
				t.Fatal("card settled twice")
			}
		})
	}
	s := game.NewState(game.CharReimu, game.ShotA, game.DiffEasy)
	s.Score = 9999999980
	card := &SpellCard{Active: true, Capture: true, Bonus: 650000}
	card.Finish(&Context{State: s})
	if s.Score != 9999999990 {
		t.Fatalf("score cap = %d", s.Score)
	}
}

func TestSpellPlayerHitFailure(t *testing.T) {
	for _, timer := range []int32{59, 60} {
		c := NewContext(game.NewState(game.CharReimu, game.ShotA, game.DiffEasy))
		c.Spell = &SpellCard{Active: true, Capture: true, Bonus: 650000}
		c.Spell.Timer.Set(timer)
		c.hitPlayer() // FUN_00426cf0 loses the capture at >59, not at 59
		want := timer == 59
		if c.Spell.Capture != want || (c.Spell.Bonus != 0) != want || c.Spell.Timer.Cur != timer {
			t.Errorf("hit at %d: %+v", timer, c.Spell)
		}
	}
}
