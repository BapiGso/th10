package stage

import (
	"math"
	"strings"
	"sync"

	"th10/anm"
	"th10/anm/anmdraw"
	"th10/gamedata"
	"th10/gfx"
	"th10/render"

	"golang.org/x/text/encoding/japanese"
)

var asciiFile = sync.OnceValue(func() *anm.File {
	f, err := gamedata.ANM("ascii.anm")
	if err != nil {
		panic("stage: embedded ascii.anm: " + err.Error())
	}
	return f
})

var textFile = sync.OnceValue(func() *anm.File {
	f, err := gamedata.ANM("text.anm")
	if err != nil {
		panic("stage: embedded text.anm: " + err.Error())
	}
	return f.OrdinalScripts() // FUN_00447470's table; stored IDs are -73..-1
})

var (
	guiHPRise     = math.Float32frombits(0x3ccccccd) // DAT_00470cf8: 0.025
	guiArrowAlpha = math.Float32frombits(0xc03f0000) // DAT_00470ce8: -2.984375
)

// HPMarker is one of GUI+0x9e94's four 8-byte records. The colour is an
// unmodified ARGB dword, not a vertical tick or an RGB colour with added alpha.
type HPMarker struct {
	Fraction float32
	Color    uint32
}

// SetHPMarker is the write in ECL 347 (FUN_0040e770 0x40fc85..0x40fc96).
// The ECL side has already divided HP by the enemy's initial maximum HP.
func (g *GUI) SetHPMarker(index int, fraction float32, colour uint32) {
	g.HPMarkers[index] = HPMarker{fraction, colour}
}

// ClearHPMarkers is ECL 331's GUI write (0x40ff3c..0x40ff65).
func (g *GUI) ClearHPMarkers() { clear(g.HPMarkers[:]) }

// SetBossStars is ECL 360's write, FUN_0040e770 0x411b8d..0x411b93.
// Keep the raw value; only the fixed ten-handle rendering walk is bounded.
func (g *GUI) SetBossStars(n int32) { g.BossStars = n }

// SetBossSection supplies the original DAT_00474c84, shared by MSG,
// ECL 344, FUN_00414900's boss name and FUN_00409280's Extra branches.
func (g *GUI) SetBossSection(n int32) { g.ctx.SetBossSection(n) }

// SetBossTimer is the GUI field written by FUN_004127a0 (0x4128b0).
func (g *GUI) SetBossTimer(seconds int) { g.timeLeft = min(seconds, 99) }

// calcBoss is FUN_00414900 0x414acf..0x41513c, before it executes MSG.
func (g *GUI) calcBoss() {
	info := BossHUDInfo{}
	if s := g.ctx.stage; s != nil {
		if h, ok := s.script.(interface{ BossHUD() BossHUDInfo }); ok {
			info = h.BossHUD()
		}
	}
	g.info = info
	w := g.ctx.World
	if !info.Active || g.ctx.DialogActive() {
		w.Interrupt(g.bossName, 1)
		g.bossName = nil
		g.hp = 0
		for i := range g.HPMarkers {
			g.HPMarkers[i].Fraction = 0 // 0x41510a leaves the colour dwords alone
		}
		return
	}
	target := float32(info.HPFrac)
	if info.MaxHP != 0 {
		target = float32(info.HP) / float32(info.MaxHP)
	}
	if g.hp < target {
		g.hp += guiHPRise // 0x414b22..0x414b2e: gradual fill, immediate loss
	}
	if target < g.hp {
		g.hp = target
	}
	px, py := float32(g.ctx.Player.X-gameX), float32(g.ctx.Player.Y-gameY)
	if !g.bossDim && py < 64 && px < -16 {
		g.bossInterrupt(3)
		g.bossDim = true
	} else if g.bossDim && (py >= 96 || px > 0) {
		g.bossInterrupt(2)
		g.bossDim = false
	}
	if g.bossName == nil {
		mid := g.ctx.BossSection < 0x18 // FUN_00414900 0x414e14/24/38/4c/71
		if id := bossNameScript(g.ctx.State.Stage, mid); id >= 0 {
			g.bossName = w.SpawnTail(g.front, id, 15, g.ctx.Anm)
		}
	}
	// FUN_00414900 0x414f33..0x415084: ten fixed handles. The count is
	// retained exactly as written by ECL 360; shipped counts are 0..10.
	for i := range g.stars {
		if int32(i) < g.BossStars {
			if g.stars[i] == nil {
				g.stars[i] = w.SpawnTail(g.front, i+0x5b, 15, g.ctx.Anm)
			}
		} else {
			w.Interrupt(g.stars[i], 1)
			g.stars[i] = nil
		}
	}
}

// bossNameScript is FUN_00414900's switch at 0x414e14..0x414e82. Stages
// 2/3 do not show a midboss name; Extra's midboss is Kanako, not Suwako.
func bossNameScript(stage int, mid bool) int {
	switch stage {
	case 1:
		if mid {
			return 0x85
		}
		return 0x86
	case 2, 3:
		if mid {
			return -1
		}
		return 0x85 + stage
	case 4:
		if mid {
			return 0x89
		}
		return 0x8a
	case 5:
		return 0x8b
	case 6:
		return 0x8c
	case 7:
		if mid {
			return 0x8c
		}
		return 0x8d
	}
	return -1
}

func (g *GUI) bossInterrupt(label int16) {
	w := g.ctx.World
	for i := range g.stars {
		if int32(i) < g.BossStars {
			w.Interrupt(g.stars[i], label)
		}
	}
	w.Interrupt(g.bossName, label)
}

// AfterMSG is FUN_00414900 0x41549a..0x4157d4. The timer's interrupt is
// queued after its frame's Run; sprite replacement is immediate. The arrow
// keeps running even while MSG suppresses the health bar and timer.
func (g *GUI) AfterMSG() {
	if g.info.Active && !g.ctx.DialogActive() && g.timeLeft >= 0 {
		n := g.timeLeft
		if n < g.previousTime {
			label := int16(0)
			if n < 6 {
				label = 9
			} else if n < 11 {
				label = 8
			}
			if label != 0 {
				g.timer[0].Interrupt(label)
				g.timer[1].Interrupt(label)
				g.ctx.PlaySEID(0) // 0x4154e3 / 0x41550c
			}
		} else if g.previousTime < n {
			g.timer[0].Interrupt(7)
			g.timer[1].Interrupt(7)
		}
		if n != g.previousTime {
			g.timer[0].SetSprite(n/10 + 8)
			g.timer[1].SetSprite(n%10 + 8)
		}
		g.previousTime = n
	}
	if g.info.Active && g.info.Flags&0x11 == 0 {
		g.calcArrow()
	}
	g.clock.Tick(g.ctx.Anm.GameSpeed)
}

// calcArrow is FUN_00414900 0x415574..0x415764. It changes at most one
// warning state per frame; the non-spell and spell thresholds differ.
func (g *GUI) calcArrow() {
	hp := g.info.PhaseHP
	threshold := [3]int32{700, 400, 200}
	if g.info.Spell {
		threshold = [3]int32{2000, 1000, 400}
	}
	if g.arrowStyle < 3 {
		if hp < threshold[g.arrowStyle] {
			g.arrow.Interrupt(int16(g.arrowStyle) + 7)
			g.arrowStyle++
		}
	} else if hp > threshold[2] {
		g.arrow.Interrupt(10)
		g.arrowStyle = 0
	}
	g.arrow.Run(g.ctx.Anm)
	x := g.info.Position[0]
	g.arrow.Pos = [3]float32{x + 224, 480, 0}
	d := float32(math.Abs(float64(x - float32(g.ctx.Player.X-gameX))))
	if d < 64 { // DAT_00470cf0 is a double, 0x4050000000000000
		g.arrow.Alpha = uint8(64 - int32(float32(d*guiArrowAlpha)))
	} else {
		g.arrow.Alpha = 255
	}
	if x < -192 || x > 192 {
		g.arrow.Alpha = 0
	}
}

// drawBoss is FUN_00415800 0x41593c..0x415ac6. Coloured markers are
// overlapping filled segments in slot order, not sorted white tick marks.
func (g *GUI) drawBoss(r *anmdraw.Renderer, rect func([4]float32, gfx.Color)) {
	if g.hp > 0 {
		black, white := hpRects(g.hp)
		rect(black, 0xff000000)
		rect(white, 0xffffffff)
		for _, m := range g.HPMarkers {
			if m.Fraction != 0 {
				end := float32(min(m.Fraction, g.hp)*342) + 40
				rect([4]float32{40, 22, end, 24}, gfx.Color(m.Color))
			}
		}
	}
	if g.info.Active && g.info.Flags&0x11 == 0 {
		r.Draw(&g.arrow) // 0x415a76
	}
	if g.info.Active && g.timeLeft >= 0 && !g.ctx.DialogActive() {
		for i := range g.timer {
			r.Draw(&g.timer[i]) // 0x415ab8
		}
	}
}

func hpRects(hp float32) (black, white [4]float32) {
	end := float32(hp*342) + 41
	return [4]float32{41, 23, end, 25}, [4]float32{40, 22, end - 1, 24}
}

// SetSpellName is the GUI/font half of FUN_00409280 (0x409681 calls
// FUN_00447ae0). t2 owns the VM handle, its declaration and its deletion;
// it can pass that VM here without spawning a second name/cut-in.
func (g *GUI) SetSpellName(vm *anm.VM, rawName string) {
	g.spellName, g.spellText = vm, rawName
	// FUN_00447ae0 takes a NUL-terminated Shift-JIS format string. Decode
	// only at the portable font boundary, never into SpellCard/history.
	if at := strings.IndexByte(rawName, 0); at >= 0 {
		rawName = rawName[:at]
	}
	name, _ := japanese.ShiftJIS.NewDecoder().String(rawName)
	render.WriteVMText(vm, name, render.TextStyle{Right: true, Color: 0xffffff})
}
