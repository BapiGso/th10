package stage

import (
	"fmt"
	"image"
	"maps"
	"math"
	"slices"
	"strings"
	"time"

	"th10/anm"
	"th10/game"
	"th10/supervisor"

	"github.com/hajimehoshi/ebiten/v2"
)

var menuChara = [...]string{"ReimuA ", "ReimuB ", "ReimuC ", "MarisaA", "MarisaB", "MarisaC"}                                      // 0x4746dc
var menuDifficulty = [...]string{"E", "N", "H", "L", "X"}                                                                          // 0x474708
var menuStage = [...]string{"tst", "St1", "St2", "St3", "St4", "St5", "St6", "Ex ", "All"}                                         // 0x474744
var rankingStage = [...]string{"test   ", "Stage 1", "Stage 2", "Stage 3", "Stage 4", "Stage 5", "Stage 6", "Extra  ", "AllClear"} // 0x47471c

// glyphText is the small-font subset of FUN_00401760 used by
// FUN_00422660 (+0x8988 = 1). Sprites start at 98 for character 0x20;
// glyphs are 8x8, advance is 9, and both anchors are top/left (0x40176f).
// It does not use system fonts or the legacy back buffer.
func glyphText(sv *supervisor.Supervisor, text string, x, y float32, colour uint32) {
	vm := anm.New()
	vm.File = menuASCIIFile()
	vm.Flags = vm.Flags&0xffd7ffff | 0x140001
	vm.Color = [3]uint8{uint8(colour >> 16), uint8(colour >> 8), uint8(colour)}
	vm.Alpha = uint8(colour >> 24)
	start := x
	for i := 0; i < len(text); i++ {
		c := text[i]
		if c == '\n' {
			x, y = start, y+16
			continue
		}
		if c < 0x20 || c > 0x81 {
			c = '?'
		}
		vm.SetSprite(int(c) + 66)
		vm.Pos = [3]float32{x, y, 0}
		sv.ANM.Draw(vm)
		x += math.Float32frombits(0x41100000) // 9, 0x470c10
	}
}

func menuHighlight(selected bool) uint32 {
	if selected {
		return 0xffffff00
	}
	return 0xff808080 // FUN_00422660: -1 & 0x7f7e80 plus 0xff808080
}

func replayMenuLine(slot game.ReplaySlot) string {
	if slot.Error != nil {
		return fmt.Sprintf("No.%02d [invalid replay]", slot.Number)
	}
	if slot.Replay == nil {
		return fmt.Sprintf("No.%02d -------- --/--/-- ------- - St-", slot.Number)
	}
	g := slot.Replay.Global
	date := time.Unix(int64(g.Timestamp), 0).Format("06/01/02")
	chara := min(int(g.Character*3+g.ShotType), len(menuChara)-1)
	diff := min(int(g.Difficulty), len(menuDifficulty)-1)
	stage := min(int(g.Stage), len(menuStage)-1)
	return fmt.Sprintf("No.%02d %-8s %s %s %s %s", slot.Number, g.User, date, menuChara[chara], menuDifficulty[diff], menuStage[stage])
}

// Draw is FUN_00422aa0 -> FUN_00422660, node 0x2e. Constants and layout
// are from r11/func_422660.asm and FUN_004224c0, not an Ebiten pause mask.
func (m *MenuController) Draw(sv *supervisor.Supervisor) {
	const white = 0xffffffff
	switch m.State {
	case MenuSlots:
		for i, slot := range m.slots {
			glyphText(sv, replayMenuLine(slot), 48, float32(64+i*15), menuHighlight(i == m.Selection))
		}
	case MenuReplayName:
		// 0x4227a8..0x4227d0: selected row Y -> 224 in ten frames.
		y := float32(224)
		if m.Timer.Cur < 10 {
			start := float32(m.Selection*15 + 64)
			y = start + float32(float32(float32(224-start)*float32(m.Timer.Cur))*math.Float32frombits(0x3dcccccd))
		}
		m.drawName(sv, 102, y)
		state := m.stage.ctx.State
		r := state.Recording.Replay()
		date := time.Unix(int64(r.Global.Timestamp), 0).Format("06/01/02")
		line := fmt.Sprintf("No.%02d          %s %s %s %s", m.Selection+1, date,
			menuChara[min(state.Character*3+state.ShotType, 5)], menuDifficulty[min(state.Difficulty, 4)], menuStage[min(int(r.Global.Stage), 8)])
		glyphText(sv, line, 48, y, white)
	case MenuRanking:
		glyphText(sv, "            Score Ranking!!", 48, 64, white)
		if !m.NoRank {
			m.drawName(sv, 150, float32(m.rank*18+96))
		}
		if table := m.ranking; table != nil {
			for i, entry := range table {
				date, stage := "--/--/--", "Stage -"
				if entry.Timestamp != 0 {
					date = time.Unix(int64(entry.Timestamp), 0).Format("06/01/02")
					stage = rankingStage[min(int(entry.Stage), len(rankingStage)-1)]
				}
				name := entry.Name
				if i == m.rank && !m.NoRank {
					name = "        "
				}
				line := fmt.Sprintf("%2d %-8s %09d%d %s %s", i+1, name, entry.Score, entry.Continues, date, stage)
				glyphText(sv, line, 48, float32(96+i*18), menuHighlight(i == m.rank && !m.NoRank))
			}
		}
	}
	if m.Error != "" {
		glyphText(sv, m.Error[:min(len(m.Error), 40)], 48, 448, white)
	}
}

// drawName is FUN_004224c0: name/cursor then a 13-column character grid.
func (m *MenuController) drawName(sv *supervisor.Supervisor, x, y float32) {
	glyphText(sv, string(m.Name[:]), x, y, 0xffffffff)
	glyphText(sv, "_", x+float32(min(m.NameAt, 7)*9), y, 0xffffff00)
	for i := range len(menuCharacters) {
		c := menuCharacters[i]
		switch i {
		case len(menuCharacters) - 3:
			c = 0x81
		case len(menuCharacters) - 2:
			c = 0x7f
		case len(menuCharacters) - 1:
			c = 0x80
		}
		glyphText(sv, string([]byte{c}), float32(112+(i%13)*18), float32(320+(i/13)*16), menuHighlight(i == m.Character))
	}
}

// capture adapts FUN_00424480's queued back-buffer-to-texture copy. Ebiten
// has no D3D StretchRect, so copy opaque RGBA pixels into a BGRA THTX block,
// then draw that sprite through sv.ANM like every other object.
func (m *MenuController) capture(sv *supervisor.Supervisor) {
	if !m.capturePending || m.backdrop == nil {
		return
	}
	m.capturePending = false
	const w, h = 384, 448 // FUN_004231d0 / FUN_00423370, 32,16,384,448
	pixels := make([]byte, 512*512*4)
	if sv.Frame != nil {
		rgba := make([]byte, w*h*4)
		sv.Frame.SubImage(image.Rect(32, 16, 416, 464)).(*ebiten.Image).ReadPixels(rgba)
		for y := 0; y < h; y++ {
			for x := 0; x < w; x++ {
				src, dst := (y*w+x)*4, (y*512+x)*4
				pixels[dst], pixels[dst+1], pixels[dst+2], pixels[dst+3] = rgba[src+2], rgba[src+1], rgba[src], 255
			}
		}
	}
	f := *menuASCIIFile()
	f.Entries = slices.Clone(f.Entries)
	f.Sprites = maps.Clone(f.Sprites)
	f.Entries[0] = anm.Entry{Name: "@menu-capture", Width: 512, Height: 512, Format: anm.FormatBGRA8888,
		Texture: &anm.Texture{Format: anm.FormatBGRA8888, W: 512, H: 512, Pixels: pixels}}
	f.Sprites[231] = anm.Sprite{ID: 231, Entry: 0, W: w, H: h}
	m.backdrop.File = &f
	m.backdrop.SetSprite(231)
	m.backdrop.Pos = [3]float32{224, 240, 0}
	m.backdrop.Flags |= anm.FlagVisible | anm.FlagDraw
}

// Layer 15 prepares the capture before menu layer 16. This is a graphics
// adapter for the existing node, not a new draw-chain priority.
func (s *Stage) nodeScreen15(sv *supervisor.Supervisor) {
	if s.ctx.Menu != nil {
		s.ctx.Menu.capture(sv)
	}
	layers(15)(s, sv)
	if s.ctx.Menu != nil {
		sv.ANM.DrawLayer(s.ctx.Menu.world, 15)
	}
}

func (s *Stage) nodeScreen16(sv *supervisor.Supervisor) {
	layers(16)(s, sv)
	if s.ctx.Menu != nil {
		sv.ANM.DrawLayer(s.ctx.Menu.world, 16)
	}
}

func (s *Stage) nodeMenu(sv *supervisor.Supervisor) {
	if s.ctx.Menu != nil {
		s.ctx.Menu.Draw(sv)
	}
}

func (s *Stage) finishReplayMetadata(clear bool) {
	state := s.ctx.State
	state.RNGSeed, state.HasSeed = s.ctx.RNG.State, true
	if state.Recording != nil {
		g := &state.Recording.Replay().Global
		g.Score, g.Stage = int32(state.Score/10), uint32(state.Stage)
		if clear && state.RunFlags&game.RunFlagPractice == 0 {
			g.Stage = 8
		}
		g.User = strings.TrimRight(g.User, " ")
	}
}
