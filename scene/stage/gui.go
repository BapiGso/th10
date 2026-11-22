package stage

// The GUI object DAT_0047770c. Its embedded VMs draw at 0x2b; the frame,
// labels and announcements are free-standing World objects on layers 14/15.
// FUN_00413810 constructs it, FUN_00413980 loads front.anm, FUN_00413bc0
// starts a stage, FUN_00415ae0 calls Calc and FUN_00415af0 calls Draw.

import (
	"fmt"
	"math"

	"th10/anm"
	"th10/anm/anmdraw"
	"th10/game"
	"th10/gamedata"
	"th10/gfx"
	"th10/supervisor"
	"th10/zun"
)

func frontFile() *anm.File { return gamedata.MustANM(gamedata.SharedPresentationANM("front.anm")) }

// DAT_00470cb8: the 38-pixel faith-hold gauge over its 130-frame cap.
var guiHoldUnit = math.Float32frombits(0x3e95a95b)

// GUI keeps the binary's VM arrays distinct, including their iteration order.
// Offset comments are relative to DAT_0047770c, not to the Go object.
type GUI struct {
	*GUIController
	ctx   *Context
	front *anm.File // +0x9ec8, ANM slot 6
	logo  *anm.File // +0x9e80, ANM slot 0x1c

	hiScore   [10]anm.VM  // +0x0010, scripts 10..19, least significant first
	score     [10]anm.VM  // +0x24c8, scripts 20..29
	lives     [9]anm.VM   // +0x4980, scripts 30..38
	power     [4]anm.VM   // +0x6a8c, scripts 47..50 (digit, dot, digit, digit)
	timer     [2]anm.VM   // +0x793c, scripts 80..81
	faith     [7]anm.VM   // +0x8094, scripts 51..57 (label, five digits, zero)
	arrow     anm.VM      // +0x9a48, ascii.anm script 0, not front.anm
	HPMarkers [4]HPMarker // +0x9e94: ratio and ARGB, in ECL 347 slot order
	BossStars int32       // +0x9e90: ECL 360's unmodified argument

	bossName               *anm.VM
	stars                  [10]*anm.VM // +0x9e28
	hp                     float32     // +0x9e84, displayed fraction
	info                   BossHUDInfo
	bossDim                bool  // +0x9eb4 bit 3
	arrowStyle             uint8 // +0x9eb4 bits 1..2
	timeLeft, previousTime int   // +0x9ec0, +0x9ec4
	spellHUD               SpellHUD
	spellName              *anm.VM
	spellID                int32
	spellText              string
	spellFrame             int32
	manualSpell            bool
	notice                 [2]*anm.VM // +0x9e14, +0x9e18
	captureDigits          [8]*anm.VM // +0x9df4
	firstCalc              bool

	frame, labels        *anm.VM // +0x9e58, +0x9e5c
	titles               [2]*anm.VM
	difficulty, practice *anm.VM // +0x9e54, +0x9e50

	clock      zun.Timer // +0x9e60
	shownHi    int32     // display copy of DAT_00474c40
	previousHi int32     // +0x9e74
	shownScore int32     // +0x9e78, stored score (displayed x10)
	scoreStep  int32     // +0x9e7c
	lastPower  int
	lastLife   int
	faithDim   bool // +0x9eb4 bit 0
}

// NewGUI is FUN_00413810/00413980/00413a20 plus the stage-start call
// FUN_00413bc0. It uses embedded resources only; its visual World objects
// select the alternate RNG (FUN_00448d00).
func NewGUI(ctx *Context) *GUI {
	if ctx.GUIController == nil {
		ctx.GUIController = &GUIController{}
	}
	g := &GUI{
		GUIController: ctx.GUIController,
		ctx:           ctx, front: frontFile(), lastPower: -1, lastLife: -1,
		timeLeft: -1, previousTime: -1,
		firstCalc: true,
		shownHi:   scoreStored(ctx.State.HiScore), shownScore: scoreStored(ctx.State.Score),
	}
	g.clock.Reset()
	if n := ctx.State.Stage; n >= 1 && n <= 7 {
		g.logo = gamedata.MustANM(gamedata.SharedPresentationANM(fmt.Sprintf("st%02dlogo.anm", n)))
	}
	g.beginStage()
	return g
}

// beginStage is FUN_00413bc0, 0x413bed..0x413d80: two World roots, then
// the six arrays. VM.Start is FUN_0043e710 and runs frame zero immediately.
func (g *GUI) beginStage() {
	w, env := g.ctx.World, g.ctx.Anm
	g.frame = w.SpawnTail(g.front, 0, 15, env)
	g.labels = w.SpawnTail(g.front, 1, 15, env)
	for i := range g.score {
		g.hiScore[i].Start(g.front, i+10, env)
		g.score[i].Start(g.front, i+20, env)
	}
	for i := range g.lives {
		g.lives[i].Start(g.front, i+30, env)
	}
	for i := range g.power {
		g.power[i].Start(g.front, i+47, env)
	}
	for i := range g.timer {
		g.timer[i].Start(g.front, i+80, env)
	}
	for i := range g.faith {
		g.faith[i].Start(g.front, i+51, env)
	}
	g.arrow.Start(asciiFile(), 0, env) // FUN_00413bc0 0x41402b..0x41403f
	g.SetLives(g.ctx.State.Life)
	g.syncPower()
	// FUN_00413bc0 0x413e3e..0x413fff: stage title and stage music title;
	// title-idle demos instead use front script 0x71, not stage practice.
	if g.ctx.State.RunFlags&game.RunFlagDemo != 0 {
		g.titles[0] = w.SpawnTail(g.front, 0x71, 15, env)
	} else if g.logo != nil {
		g.titles[0] = w.SpawnTail(g.logo, 0, 15, env)
		g.titles[1] = w.SpawnTail(g.logo, 1, 15, env)
	}
	// FUN_00413bc0 0x414044: first-stage point-of-collection hint.
	// FUN_00413bc0 0x414044..0x414068 also requires zero continues.
	if g.ctx.State.Stage == 1 && g.ctx.State.StageMode == 0 && g.ctx.State.ReplayState90 == 0 {
		w.SpawnTail(g.front, 0x4f, 15, env)
	}
	// FUN_00413bc0 0x414113/0x41424c: fresh-run indicator and difficulty.
	diff := min(max(g.ctx.State.Difficulty, 0), 4)
	if g.ctx.State.FreshRun {
		g.practice = w.SpawnTail(g.front, 0x66+diff, 15, env)
		w.Interrupt(g.practice, 3)
	}
	g.difficulty = w.SpawnTail(g.front, 0x6b+diff, 15, env)
	if g.practice != nil {
		w.Interrupt(g.practice, 3) // 0x414336 tests +0x9e50 again, not +0x9e54
	}
}

// SetLives is FUN_00413790: only FlagDraw changes; all nine VMs keep running.
func (g *GUI) SetLives(n int) {
	n = min(max(n, 0), len(g.lives))
	for i := range g.lives {
		if i < n {
			g.lives[i].Flags |= anm.FlagDraw
		} else {
			g.lives[i].Flags &^= anm.FlagDraw
		}
	}
	g.lastLife = n
}

// SetPower is FUN_004054b0: the caller supplies the integer and hundredths,
// and only the three digit sprites are replaced. The dot VM is untouched.
func (g *GUI) SetPower(whole, percent int) {
	g.power[0].SetSprite(whole + 8)
	g.power[2].SetSprite(percent/10 + 8)
	g.power[3].SetSprite(percent%10 + 8)
}

// powerDigits is the conversion at FUN_00413bc0 0x413dd1 and the pickup,
// bomb and death callers of FUN_004054b0. Project power stores original x5.
func powerDigits(power int) (whole, percent int) {
	p := min(max(power/5, 0), 100)
	return p / 20, (p % 20) * 100 / 20
}

func (g *GUI) syncPower() {
	if p := g.ctx.State.Power; p != g.lastPower {
		if g.lastPower >= 0 && g.lastPower < 500 && p >= 500 {
			g.ShowNotice(2, 0) // FUN_00418930 0x418975..0x418991
		}
		g.SetPower(powerDigits(p))
		g.lastPower = p
	}
}

// Calc is FUN_00415ae0 -> FUN_00414900, before the World calc (0x1a).
// The score arrays are advanced separately by Numbers (FUN_00417040).
func (g *GUI) Calc() {
	g.TickClear(g.ctx.systemFlags()) // FUN_00414900, before MSG execution
	env := g.ctx.Anm
	if g.firstCalc {
		// FUN_00413a20 runs after replay restoration in the binary. sim.Run
		// restores the chapter after constructing the Stage, so take that
		// initial score before rolling, without writing State.
		if g.ctx.State.StageMode == 1 {
			g.shownScore = scoreStored(g.ctx.State.Score)
			g.shownHi = max(g.shownHi, g.shownScore)
		}
		g.firstCalc = false
	}
	if g.lastLife != g.ctx.State.Life {
		if g.lastLife >= 0 && g.ctx.State.Life > g.lastLife {
			g.ShowNotice(4, 0) // FUN_004188a0 0x4188ef..0x41891c
		}
		g.SetLives(g.ctx.State.Life)
	}
	g.syncPower()
	for i := range g.lives {
		g.lives[i].Run(env)
	}
	for i := range g.power {
		g.power[i].Run(env)
	}
	for i := range g.timer {
		g.timer[i].Run(env)
	}
	g.calcFaith()
	g.calcBoss()
	g.calcSpellFallback()
}

// calcFaith is FUN_00414900 0x41499f..0x414aca. The dimming test has
// hysteresis, and addresses 0x470d00/0x470d10 are doubles (-112/-128).
func (g *GUI) calcFaith() {
	p := g.ctx.Player
	x, y := float32(p.X-gameX), float32(p.Y-gameY)
	if !g.faithDim && y > 416 && x < -128 {
		for i := range g.faith {
			g.faith[i].Interrupt(3)
		}
		g.faithDim = true
	} else if g.faithDim && (y < 400 || x > -112) {
		for i := range g.faith {
			g.faith[i].Interrupt(2)
		}
		g.faithDim = false
	}
	env := g.ctx.Anm
	g.faith[0].Run(env)
	n, place := min(max(g.ctx.State.Faith/10, 0), 99999), int64(10000)
	for i := 1; i <= 5; i++ {
		g.faith[i].SetSprite(int(n/place) + 30)
		n %= place
		place /= 10
		g.faith[i].Run(env)
	}
	g.faith[6].Run(env)
}

// scoreStored and scoreDigits use the score limit of FUN_00409d90. The
// original has nine stored digits plus the trailing digit DAT_00474c90/94.
func scoreStored(n int64) int32 { return int32(min(max(n/10, 0), 999999999)) }

func scoreDigits(n int64) (out [10]int) {
	n = min(max(n, 0), 9999999990)
	for i := range out {
		out[i] = int(n % 10)
		n /= 10
	}
	return out
}

// Numbers is the display-only part of FUN_00417040, called after the calc
// chain by FUN_00418190 (0x4186dd). Score rolls toward the real score with
// a retained step. The gameplay high-score/extend side effects stay with
// GameState; this function must not award a life or change the trace.
func (g *GUI) Numbers() {
	target := scoreStored(g.ctx.State.Score)
	old := g.shownScore
	if old != target {
		step := (target - old) / 32
		if step >= 0x8d55e { // 0x41706c, 578910 stored points per frame
			step = 0x8d55e
		} else if step == 0 {
			step = 1
		}
		g.scoreStep = max(g.scoreStep, step)
		g.scoreStep = min(g.scoreStep, target-g.shownScore)
		g.shownScore += g.scoreStep
		if g.shownScore >= target {
			g.scoreStep = 0
		}
		if old < 100000000 && g.shownScore >= 100000000 {
			g.score[9].Interrupt(4) // 0x4170eb: reveal the tenth digit
		}
	}
	g.shownHi = max(g.shownHi, g.shownScore)
	if g.clock.Cur >= 20 {
		if g.previousHi < 100000000 && g.shownHi >= 100000000 {
			g.hiScore[9].Interrupt(4) // 0x417174
		}
		g.previousHi = g.shownHi
	}
	hi := scoreDigits(int64(g.shownHi) * 10)
	score := scoreDigits(int64(g.shownScore) * 10)
	// FUN_00417040 0x417189..0x417283: digits 1..9 interleaved, then 0.
	for i := 1; i < 10; i++ {
		g.hiScore[i].SetSprite(hi[i] + 8)
		g.score[i].SetSprite(score[i] + 8)
		g.hiScore[i].Run(g.ctx.Anm)
		g.score[i].Run(g.ctx.Anm)
	}
	g.hiScore[0].SetSprite(int(g.ctx.State.HiScoreContinues) + 8) // DAT_00474c94, 0x417247
	g.score[0].SetSprite(int(g.ctx.State.ReplayState90) + 8)      // DAT_00474c90, 0x417260
	g.hiScore[0].Run(g.ctx.Anm)
	g.score[0].Run(g.ctx.Anm)
}

// Draw is FUN_00415800's unconditional VM arrays, then the faith-hold
// rectangles. The boss tail is added by the HUD part of the GUI.
func (g *GUI) Draw(r *anmdraw.Renderer) {
	g.draw(r, r.Rect)
}

// draw takes the flat primitive as a callback so tests can record GUI call
// sites independently of t1's shared Rect implementation.
func (g *GUI) draw(r *anmdraw.Renderer, rect func([4]float32, gfx.Color)) {
	for i := range g.score {
		r.Draw(&g.hiScore[i]) // 0x41582c
		r.Draw(&g.score[i])   // 0x415839
	}
	for i := range g.lives {
		r.Draw(&g.lives[i]) // 0x41585a
	}
	for i := range g.power {
		r.Draw(&g.power[i]) // 0x41587b
	}
	for i := range g.faith {
		r.Draw(&g.faith[i]) // 0x4158a8
	}
	if n := g.ctx.State.FaithHold; n != 0 {
		// FUN_00415800 0x4158c0..0x415937. Shared Rect is owned by t1.
		black, white, c := faithRects(n, g.faith[0].Alpha)
		rect(black, c)
		rect(white, c|0xffffff)
	}
	g.drawBoss(r, rect)
}

// faithRects is the call-site arithmetic of FUN_00415800, not Rect's
// primitive implementation. It intentionally does not clamp or snap.
func faithRects(hold int, alpha uint8) (black, white [4]float32, c gfx.Color) {
	end := float32(float32(hold)*guiHoldUnit) + 51
	return [4]float32{51, 461, end, 463}, [4]float32{50, 460, end - 1, 462}, gfx.Color(alpha) << 24
}

// nodeGUI is node 0x2b (FUN_00415af0). Nil GUI is useful for the existing
// minimal World-only fixtures, which construct a Stage without newStage.
func (s *Stage) nodeGUI(sv *supervisor.Supervisor) {
	if s.ctx.GUI != nil {
		s.ctx.GUI.Draw(sv.ANM)
	}
}
