package player

import (
	"math"
	"sync"

	"th10/anm"
	"th10/gamedata"
	"th10/gfx"
)

// FUN_004247f0, 0x4247f8-0x42481c: the animation depends only on the
// character, not the shot type. Each embedded file is parsed once per process.
var playerFiles = [...]func() *anm.File{
	sync.OnceValue(func() *anm.File { return loadPlayerAnm("pl00.anm") }),
	sync.OnceValue(func() *anm.File { return loadPlayerAnm("pl01.anm") }),
}

func loadPlayerAnm(name string) *anm.File {
	f, err := gamedata.ANM(name)
	if err != nil {
		panic("player: embedded " + name + ": " + err.Error())
	}
	return f
}

// AttachAnm connects the embedded VM and world handles. A nil world or
// environment disables animation without affecting headless movement tests.
// FUN_004247f0 (0x4248ab-0x4248b1) starts script 0 through FUN_00404f30.
func (p *Player) AttachAnm(w *anm.World, env *anm.Env, bullet *anm.File) {
	p.world, p.env, p.bulletAnm = w, env, bullet
	if p.anmAttached() {
		p.VM.Offset = [3]float32{}
		p.VM.Start(p.Anm, 0, env)
		p.Bullets.AttachAnm(w, env, p.Anm)
		p.RefreshOptions() // FUN_004247f0, 0x424d3d
	}
}

func (p *Player) anmAttached() bool { return p.world != nil && p.env != nil }

// Draw is FUN_00426360: state 2 draws nothing; the player's embedded VM
// is drawn directly in node 0x16, followed by the optional faith-hold bars.
// bar combines the four external gates at 0x4263d7-0x42642e (m4_r11.md).
func (p *Player) Draw(draw func(*anm.VM), rect func([4]float32, gfx.Color), hold int32, bar bool) {
	if p.State == StateDying || !p.anmAttached() {
		return
	}
	x, y := p.gamePos()
	p.VM.Offset = [3]float32{x + 224, y + 16, 0} // 0x426373-0x4263a5
	draw(&p.VM)
	if !bar || hold == 0 { // 0x426434-0x42643b
		return
	}
	// 0x426441-0x4264a4, x87 24-bit operand order. FUN_00427c50 is
	// floor; DAT_00470cb8 is 0x3e95a95b, not a rounded decimal literal.
	x1 := float32(math.Floor(float64(float32(float32(x+225) - 16))))
	y1 := float32(math.Floor(float64(float32(float32(y+17) - 24))))
	x2 := float32(float32(hold)*math.Float32frombits(0x3e95a95b)) + x1
	y2 := y1 + 2
	rect([4]float32{x1, y1, x2, y2}, 0x80000000)                 // 0x426495-0x4264a8
	rect([4]float32{x1 - 1, y1 - 1, x2 - 1, y2 - 1}, 0xffffffff) // 0x4264ad-0x4264e8
}
