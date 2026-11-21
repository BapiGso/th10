package stagescript

import (
	"image/color"
	"math"
	"th10/render"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

// Legacy backgrounds are still reachable through TH10_LEGACY_BG and missing
// STD resources. Keep their existing arithmetic and drawing bodies unchanged.
// The native paths are FUN_00403c80/00402850/00402ca0, not these approximations.
type legacyBackground struct {
	frame                   int
	kind                    int
	scroll                  float64
	bgX, bgY                float64
	bg1, bg2, bg3, bg4, bg5 *ebiten.Image
}

func (s *legacyBackground) init(c Config) {
	s.kind, s.scroll = c.BackgroundStage, c.BackgroundScroll
	slots := [...]**ebiten.Image{&s.bg1, &s.bg2, &s.bg3, &s.bg4, &s.bg5}
	for i, name := range c.Backgrounds {
		if name != "" {
			*slots[i] = render.LoadImage(name)
		}
	}
}

func (s *legacyBackground) advance(frame int) {
	s.frame = frame
	if s.kind == 1 {
		cam := stage01StdCamera(frame)
		s.bgX, s.bgY = cam.x, cam.y
	} else {
		s.bgY += s.scroll
	}
}

func (s *legacyBackground) draw(field *ebiten.Image) {
	switch s.kind {
	case 1:
		s.drawStage1(field)
	case 2:
		s.drawStage2(field)
	case 3:
		s.drawStage3(field)
	case 4:
		s.drawStage4(field)
	case 5:
		s.drawStage5(field)
	case 6:
		s.drawStage6(field)
	case 7:
		s.drawStage7(field)
	}
}

// Additive blend for light cloud/leaf layers (ANM ins_67(6) = additive).
var blendAdd = ebiten.Blend{
	BlendFactorSourceRGB:        ebiten.BlendFactorOne,
	BlendFactorSourceAlpha:      ebiten.BlendFactorOne,
	BlendFactorDestinationRGB:   ebiten.BlendFactorOne,
	BlendFactorDestinationAlpha: ebiten.BlendFactorOne,
	BlendOperationRGB:           ebiten.BlendOperationAdd,
	BlendOperationAlpha:         ebiten.BlendOperationAdd,
}

func (s *legacyBackground) drawStage1(field *ebiten.Image) {
	w := float32(field.Bounds().Dx())
	h := float32(field.Bounds().Dy())

	field.Fill(color.RGBA{16, 24, 48, 255})
	if s.bg1 != nil {
		// --- Ground layers (STD entries 0-3) ---
		// Entry 1 (Z=0): main ground tile, stg1bg.png
		render.TileBackground(field, s.bg1, s.bgY, 1)
		// Entry 3 (Z=-128): deeper ground, stg1bg2.png with alpha 192/255 ≈ 0.75
		// (ANM script2: ins_51(192)). Depth ratio 630/502 ≈ 1.255 → scrolls
		// slightly faster than the main ground for a subtle depth effect.
		render.TileBackground(field, s.bg2, s.bgY*1.255, 0.75)

		// --- Fog overlay (between ground and cloud layers) ---
		s.drawFog(field)

		// --- Cloud/leaf entries (STD entries 4-7) ---
		s.drawStdCloudEntries(field)
		return
	}

	// Vector-line fallback when PNGs are missing.
	offset := float32(math.Mod(s.bgY, 40))
	for y := offset - 40; y < h; y += 40 {
		vector.StrokeLine(field, 0, y, w, y, 0.5, color.RGBA{40, 50, 80, 100}, false)
	}
	vector.StrokeLine(field, w*0.2, 0, w*0.2, h, 1, color.RGBA{60, 70, 100, 150}, false)
	vector.StrokeLine(field, w*0.8, 0, w*0.8, h, 1, color.RGBA{60, 70, 100, 150}, false)
}

func (s *legacyBackground) drawStage2(field *ebiten.Image) {
	w := float32(field.Bounds().Dx())
	h := float32(field.Bounds().Dy())

	// 暗紫色阶梯参道
	field.Fill(color.RGBA{24, 16, 40, 255})

	if s.bg1 != nil {
		render.TileBackground(field, s.bg1, s.bgY, 1)
	} else {
		offset := float32(math.Mod(s.bgY, 32))
		for y := offset - 32; y < h; y += 32 {
			vector.StrokeLine(field, 0, y, w, y, 0.8, color.RGBA{50, 35, 70, 120}, false)
		}
		for y := offset - 32; y < h; y += 32 {
			vector.FillRect(field, w*0.15, y, w*0.7, 2, color.RGBA{60, 45, 80, 80}, false)
		}
	}

	render.TileBackground(field, s.bg2, s.bgY*0.5, 0.25)
}

func (s *legacyBackground) drawStage3(field *ebiten.Image) {
	w := float32(field.Bounds().Dx())
	h := float32(field.Bounds().Dy())

	// 蓝绿色山间溪流
	field.Fill(color.RGBA{12, 32, 28, 255})

	if s.bg1 != nil {
		render.TileBackground(field, s.bg1, s.bgY, 1)
	} else {
		offset := float32(math.Mod(s.bgY, 48))
		for y := offset - 48; y < h; y += 48 {
			vector.StrokeLine(field, 0, y, w, y, 0.5, color.RGBA{30, 60, 55, 100}, false)
		}
		ox := float32(math.Sin(float64(s.bgY)*0.02) * 20)
		vector.StrokeLine(field, w*0.4+ox, 0, w*0.4+ox, h, 2, color.RGBA{40, 80, 120, 80}, false)
		vector.StrokeLine(field, w*0.6-ox, 0, w*0.6-ox, h, 2, color.RGBA{40, 80, 120, 80}, false)
	}

	render.TileBackground(field, s.bg2, s.bgY*0.7, 0.3)
}

func (s *legacyBackground) drawStage4(field *ebiten.Image) {
	w := float32(field.Bounds().Dx())
	h := float32(field.Bounds().Dy())

	// 红叶瀑布
	field.Fill(color.RGBA{32, 20, 16, 255})

	if s.bg1 != nil {
		render.TileBackground(field, s.bg1, s.bgY, 1)
	} else {
		offset := float32(math.Mod(s.bgY, 36))
		for y := offset - 36; y < h; y += 36 {
			vector.StrokeLine(field, 0, y, w, y, 0.5, color.RGBA{60, 40, 30, 100}, false)
		}
		vector.FillRect(field, w*0.4, 0, w*0.2, h, color.RGBA{30, 50, 80, 40}, false)
		wy := float32(math.Mod(float64(s.bgY)*3, float64(h)))
		for y := float32(0); y < h; y += 8 {
			a := uint8(40 + 20*math.Sin(float64(y+wy)*0.1))
			vector.StrokeLine(field, w*0.45, y, w*0.55, y+4, 1, color.RGBA{60, 90, 140, a}, false)
		}
	}

	render.TileBackground(field, s.bg2, s.bgY*0.6, 0.25)
}

func (s *legacyBackground) drawStage5(field *ebiten.Image) {
	w := float32(field.Bounds().Dx())
	h := float32(field.Bounds().Dy())

	// 神社境内，深蓝+紫
	field.Fill(color.RGBA{16, 16, 40, 255})

	if s.bg1 != nil {
		render.TileBackground(field, s.bg1, s.bgY, 1)
	} else {
		offset := float32(math.Mod(s.bgY, 40))
		for y := offset - 40; y < h; y += 40 {
			vector.StrokeLine(field, 0, y, w, y, 0.4, color.RGBA{40, 35, 70, 90}, false)
		}
		vector.FillRect(field, w*0.2-3, 0, 6, h, color.RGBA{120, 30, 20, 60}, false)
		vector.FillRect(field, w*0.8-3, 0, 6, h, color.RGBA{120, 30, 20, 60}, false)
	}

	render.TileBackground(field, s.bg2, s.bgY*0.5, 0.3)
}

func (s *legacyBackground) drawStage6(field *ebiten.Image) {
	w := float32(field.Bounds().Dx())
	h := float32(field.Bounds().Dy())

	// 御柱墓场 - 暗灰+紫红
	field.Fill(color.RGBA{20, 12, 24, 255})

	if s.bg1 != nil {
		render.TileBackground(field, s.bg1, s.bgY, 1)
	} else {
		offset := float32(math.Mod(s.bgY, 50))
		for y := offset - 50; y < h; y += 50 {
			vector.StrokeLine(field, 0, y, w, y, 0.3, color.RGBA{45, 30, 50, 80}, false)
		}
		for i := 0; i < 4; i++ {
			x := w*0.15 + float32(i)*w*0.22
			vector.FillRect(field, x-4, 0, 8, h, color.RGBA{80, 50, 30, 50}, false)
		}
	}

	render.TileBackground(field, s.bg2, s.bgY*0.4, 0.25)
}

func (s *legacyBackground) drawStage7(field *ebiten.Image) {
	w := float32(field.Bounds().Dx())
	h := float32(field.Bounds().Dy())

	// 山の裏 — 暗い土色+ミシャグジ紋様
	field.Fill(color.RGBA{28, 20, 16, 255})

	if s.bg1 != nil {
		render.TileBackground(field, s.bg1, s.bgY, 1)
	} else {
		offset := float32(math.Mod(s.bgY, 60))
		for y := offset - 60; y < h; y += 60 {
			vector.StrokeLine(field, 0, y, w, y, 0.3, color.RGBA{50, 40, 30, 70}, false)
		}
		for i := 0; i < 3; i++ {
			baseX := w * (0.2 + float32(i)*0.3)
			for y := float32(0); y < h; y += 4 {
				ox := float32(math.Sin(float64(y+float32(s.bgY))*0.05+float64(i))) * 15
				vector.FillRect(field, baseX+ox-1, y, 2, 4, color.RGBA{60, 45, 25, 50}, false)
			}
		}
	}

	render.TileBackground(field, s.bg2, s.bgY*0.5, 0.3)
}

// ---------------------------------------------------------------------------
// Camera track from stage01.std SCRIPT block
// ---------------------------------------------------------------------------

type stdCam struct{ x, y float64 }

// stage01StdCamera returns the camera world position at the given frame,
// derived from the stage01.std keyframes:
//
//	ins_2/ins_3 set/interpolate position; ins_1(576,3668) loops at frame 4180.
//	Easing mode 0/1 = linear, 4 = decelerate (ease-out), 9 = smooth-step.
func stage01StdCamera(frame int) stdCam {
	f := frame
	if f >= 4180 {
		// ins_1(576, 3668): loop back to frame 3668. The camera segment
		// 3668→4180 is 512 frames (1480→2504 linear).
		f = 3668 + (f-4180)%512
	}

	// Camera X: 0 until frame 2556, then eases to -140 over 512 frames.
	x := 0.0
	if f >= 3068 {
		x = -140
	} else if f >= 2556 {
		x = lerp(0, -140, smoothStep(float64(f-2556)/512))
	}

	// Camera Y: piecewise segments matching STD keyframes.
	y := 0.0
	switch {
	case f < 700:
		// ins_3(700, 4, 0, 1024, -630) — mode 4 (ease-out decelerate)
		y = lerp(0, 1024, easeOut(float64(f)/700))
	case f < 900:
		// ins_3(200, 1, 0, 150, -630) — mode 1 (linear)
		y = lerp(0, 150, float64(f-700)/200)
	case f < 1412:
		// ins_3(512, 0, 0, 1174, -630) — mode 0 (linear)
		y = lerp(150, 1174, float64(f-900)/512)
	case f < 1924:
		y = lerp(150, 1174, float64(f-1412)/512)
	case f < 2436:
		y = lerp(150, 1174, float64(f-1924)/512)
	case f < 2556:
		// ins_3(120, 4, 0, 330, -630) — mode 4
		y = lerp(150, 330, easeOut(float64(f-2436)/120))
	case f < 3068:
		// ins_3(512, 9, -140, 530, -630) — mode 9 (smooth-step)
		y = lerp(330, 530, smoothStep(float64(f-2556)/512))
	case f < 3268:
		// ins_3(200, 1, -140, 680, -630) — mode 1
		y = lerp(530, 680, float64(f-3068)/200)
	case f < 3668:
		// ins_3(400, 0, -140, 1480, -630) — mode 0
		y = lerp(680, 1480, float64(f-3268)/400)
	default:
		// ins_3(512, 0, -140, 2504, -630) — mode 0
		y = lerp(1480, 2504, float64(f-3668)/512)
	}

	return stdCam{x: x, y: y}
}

// lerp linearly interpolates a→b, clamped to [0,1].
func lerp(a, b, t float64) float64 {
	if t < 0 {
		t = 0
	} else if t > 1 {
		t = 1
	}
	return a + (b-a)*t
}

// easeOut: 1-(1-t)², decelerating to rest. Approximation for STD mode 4.
func easeOut(t float64) float64 {
	if t < 0 {
		t = 0
	} else if t > 1 {
		t = 1
	}
	return 1 - (1-t)*(1-t)
}

// smoothStep: 3t²-2t³, smooth ease-in-out. Approximation for STD mode 9.
func smoothStep(t float64) float64 {
	if t < 0 {
		t = 0
	} else if t > 1 {
		t = 1
	}
	return t * t * (3 - 2*t)
}

// ---------------------------------------------------------------------------
// Fog overlay — derived from stage01.std ins_8/ins_9
// ---------------------------------------------------------------------------

// drawFog renders a semi-transparent colored overlay to simulate the STD fog.
//
// STD fog parameters:
//
//	Frame 0:    ins_8(#000000ff, 200, 600)  — initial black fog
//	Frame 0:    ins_9(700, 0, #70a0f0ff, 900, 2000) — transition to blue
//	Frame 3668: ins_9(100, 0, #0000c0ff, 700, 1200) — deep blue fog in loop
//
// We approximate this as: dark overlay fading out during the intro dive (0-700),
// then a deep-blue tint gradually appearing in the boss loop section (3668+).
func (s *legacyBackground) drawFog(field *ebiten.Image) {
	f := s.frame

	var r, g, b uint8
	var alpha float32

	if f < 700 {
		// Intro dive: dark fog fading out (the initial #000000 fog clearing).
		t := float32(f) / 700
		alpha = 0.4 * (1 - t*t)
	} else {
		// Boss loop section: deep blue tint (#0000c0) fading in over 100 frames.
		lf := f
		if lf >= 4180 {
			lf = 3668 + (lf-4180)%512
		}
		if lf >= 3668 {
			t := float32(lf-3668) / 100
			if t > 1 {
				t = 1
			}
			r, g, b = 0, 0, 0xc0
			alpha = 0.12 * t
		}
	}

	if alpha < 0.004 { // ≈1/255, below visible threshold
		return
	}
	a := uint8(alpha * 255)
	if a == 0 {
		return
	}
	fw := float32(field.Bounds().Dx())
	fh := float32(field.Bounds().Dy())
	vector.FillRect(field, 0, 0, fw, fh, color.RGBA{r, g, b, a}, false)
}

// ---------------------------------------------------------------------------
// Cloud/leaf entries — STD entries 4-7
// ---------------------------------------------------------------------------

type stage01StdQuad struct {
	imageID int
	x, y    float64
	w, h    float64
	light   bool // true = light cloud (additive), false = dark reflection (alpha)
}

type stage01StdCloudEntry struct {
	x, y     float64
	parallax float64
	faces    []float64
	quads    []stage01StdQuad
}

// Cloud parallax values — perspective-derived from STD Z depths.
// Camera Z=-630, ground Z=0, far clouds Z=-320, near reflections Z=-30.
// True ratio groundDepth/cloudDepth gives ~2.03 / ~1.05, but the original
// game's angled camera (FOV 0.628, focus at Z≈+300) compresses the parallax.
// We use moderate values that match the observed visual style: far clouds
// drift as a slow foreground, near reflections track close to the ground.
const cloudScale = 0.72 // approximate world-to-screen scale at ground depth

// STD entries 4-7 (cloud/leaf layers), transcribed from stage01.std.
var stage01StdCloudEntries = []stage01StdCloudEntry{
	// Entry 4: far clouds (Z=-320), additive blend, script 3/5
	{
		x: -48, y: 80, parallax: 0.50,
		faces: []float64{256, 768, 1280, 1792, 2304, 2816, 3328, 3584},
		quads: []stage01StdQuad{
			{imageID: 3, x: 64, y: 0, w: 256, h: 256, light: true},
			{imageID: 4, x: -128, y: 64, w: 320, h: 320, light: true},
			{imageID: 4, x: -32, y: 128, w: 320, h: 320, light: true},
		},
	},
	// Entry 5: far clouds (Z=-320), additive blend, script 5/3
	{
		x: 18, y: 30, parallax: 0.50,
		faces: []float64{512, 1024, 1536, 2048, 2560, 3072},
		quads: []stage01StdQuad{
			{imageID: 4, x: 128, y: 0, w: 360, h: 360, light: true},
			{imageID: 3, x: -128, y: 96, w: 288, h: 288, light: true},
			{imageID: 3, x: 32, y: 32, w: 320, h: 320, light: true},
		},
	},
	// Entry 6: near reflections (Z=-30), normal blend, script 4/6
	{
		x: -48, y: 48, parallax: 0.68,
		faces: []float64{256, 768, 1280, 1792, 2304, 2816, 3328, 3584},
		quads: []stage01StdQuad{
			{imageID: 3, x: 64, y: 0, w: 256, h: -256},
			{imageID: 4, x: -128, y: 64, w: 320, h: -320},
			{imageID: 4, x: -32, y: 32, w: 320, h: -320},
		},
	},
	// Entry 7: near reflections (Z=-30), normal blend, script 6/4
	{
		x: 18, y: 56, parallax: 0.68,
		faces: []float64{512, 1024, 1536, 2048, 2560, 3072},
		quads: []stage01StdQuad{
			{imageID: 4, x: 128, y: 32, w: 360, h: -360},
			{imageID: 3, x: -128, y: 64, w: 288, h: -288},
			{imageID: 3, x: 32, y: 32, w: 320, h: -320},
		},
	},
}

func (s *legacyBackground) drawStdCloudEntries(field *ebiten.Image) {
	sway := stage01StdLeafSway(s.frame)
	// Camera X offset applied to clouds via their own parallax.
	for _, entry := range stage01StdCloudEntries {
		xShift := -s.bgX * entry.parallax * cloudScale
		for _, faceY := range entry.faces {
			screenY := 44 + (faceY-s.bgY)*entry.parallax + entry.y*cloudScale
			if screenY < -400 || screenY > float64(field.Bounds().Dy())+400 {
				continue
			}
			for _, q := range entry.quads {
				img := s.bg3
				if q.imageID == 4 || q.imageID == 6 {
					img = s.bg4
				}
				screenX := float64(field.Bounds().Dx())/2 + (entry.x+q.x)*cloudScale + sway + xShift
				qY := screenY + q.y*cloudScale
				drawStdCloudQuad(field, img, screenX, qY, q)
			}
		}
	}
}

// stage01StdLeafSway returns the ±32px horizontal sway for cloud/leaf sprites,
// matching ANM scripts 3-6: ins_56(620,9,±32,0,0) — period 1240 frames.
func stage01StdLeafSway(frame int) float64 {
	phase := math.Mod(float64(frame), 1240)
	if phase < 620 {
		return lerp(32, -32, phase/620)
	}
	return lerp(-32, 32, (phase-620)/620)
}

// drawStdCloudQuad renders one cloud/leaf sprite.
//   - Light (q.light): additive blend (ANM ins_67(6)), warm tint (255,232,208)
//     from ANM ins_52. Brightness scaled to ~0.3 to prevent over-saturation.
//   - Dark (!q.light): normal alpha blend, black tint (ins_52(0,0,0)),
//     alpha 0.5 (ANM ins_51(128) = 128/255 ≈ 0.5).
func drawStdCloudQuad(field, img *ebiten.Image, x, y float64, q stage01StdQuad) {
	if img == nil {
		return
	}
	scaleX := q.w * cloudScale / float64(img.Bounds().Dx())
	scaleY := q.h * cloudScale / float64(img.Bounds().Dy())
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(scaleX, scaleY)
	op.GeoM.Translate(x, y)
	if q.light {
		// Additive blend with warm tint: RGB scaled by brightness factor.
		op.Blend = blendAdd
		const bright = float32(0.30)
		op.ColorScale.Scale(bright, bright*0.91, bright*0.816, bright)
	} else {
		// Normal blend, black shadow/reflection, alpha 0.5 (ANM ins_51(128)).
		op.ColorScale.Scale(0, 0, 0, 0.5)
	}
	field.DrawImage(img, op)
}
