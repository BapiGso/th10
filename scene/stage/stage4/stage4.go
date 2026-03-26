package stage4

import (
	"bytes"
	"image"
	"image/color"
	_ "image/png"
	"math"
	"th10/assets"
	"th10/audio"
	"th10/entity/bullet"
	"th10/entity/enemy"
	"th10/scene/dialog"
	"th10/scene/stage"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

// Script4 第四关脚本 — 中Boss: 犬走椛 / Boss: 射命丸文
type Script4 struct {
	frame          int
	finished       bool
	bgY            float64
	midBossSpawned bool
	midBoss        *enemy.Enemy
	bossSpawned    bool
	boss           *enemy.Enemy
	bg1            *ebiten.Image // stg4bg.png [512x512]
	bg2            *ebiten.Image // stg4bg3.png [512x256]
	bg3            *ebiten.Image // stg4bg7.png [384x448]
}

func NewScript() *Script4 {
	s := &Script4{}
	s.loadBg()
	return s
}

func loadImage(path string) *ebiten.Image {
	data, err := assets.Assets.ReadFile(path)
	if err != nil {
		return nil
	}
	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil
	}
	return ebiten.NewImageFromImage(img)
}

func (s *Script4) loadBg() {
	s.bg1 = loadImage("anm/background/stg4bg.png")
	s.bg2 = loadImage("anm/background/stg4bg3.png")
	s.bg3 = loadImage("anm/background/stg4bg7.png")
}

func (s *Script4) Init(ctx *stage.Context) {
	ctx.Audio.PlayBGM(audio.BGMStage4)
}

func (s *Script4) Update(ctx *stage.Context) {
	s.frame++
	s.bgY += 2.5

	// ---- 杂兵 ----
	if s.frame >= 60 && s.frame <= 400 && s.frame%18 == 0 {
		x := 50 + float64((s.frame/18)%8)*42
		e := enemy.New(x, -16, 30, enemy.TypeFairy)
		e.SetPath([]enemy.PathNode{
			{X: x, Y: 90, Frames: 25},
			{X: x + 40, Y: 180, Frames: 45},
			{X: x, Y: 500, Frames: 50},
		})
		e.DropPower = 2
		e.DropPoint = 1
		ctx.AddEnemy(e)
	}

	if s.frame >= 460 && s.frame <= 720 && s.frame%12 == 0 {
		for _, side := range []float64{50, 366} {
			e := enemy.New(side, -16, 35, enemy.TypeFairy)
			e.SetPath([]enemy.PathNode{
				{X: 208, Y: 100, Frames: 35},
				{X: 208, Y: 100, Frames: 40},
				{X: side, Y: 500, Frames: 40},
			})
			e.DropPower = 1
			ctx.AddEnemy(e)
		}
	}

	if s.frame >= 780 && s.frame <= 1080 && s.frame%10 == 0 {
		x := 30 + float64((s.frame-780)/10%10)*36
		e := enemy.New(x, -16, 45, enemy.TypeFairy)
		e.SetPath([]enemy.PathNode{
			{X: x, Y: 110, Frames: 20},
			{X: x, Y: 110, Frames: 70},
			{X: x, Y: 500, Frames: 35},
		})
		e.DropPower = 2
		e.DropPoint = 1
		ctx.AddEnemy(e)
	}

	// 杂兵弹幕
	for _, e := range ctx.Enemies {
		if !e.Active || e.Type != enemy.TypeFairy {
			continue
		}
		if e.Age > 20 && e.Age%14 == 0 {
			ctx.Emitter.SetPos(e.X, e.Y)
			count := 10 + ctx.State.Difficulty*3
			ctx.Emitter.Ring(count, 2.8, float64(e.Age)*0.13, bullet.TypeSmall, 3)
		}
	}

	// ---- 中Boss: 犬走椛（白狼天狗警戒线） ----
	if s.frame == 840 && !s.midBossSpawned {
		midBoss := enemy.New(208, -40, 400, enemy.TypeMidBoss)
		midBoss.SpriteID = 4
		midBoss.HitWidth = 40
		midBoss.HitHeight = 40
		midBoss.DropPower = 20
		midBoss.DropPoint = 10
		midBoss.SetPath([]enemy.PathNode{
			{X: 208, Y: 90, Frames: 40},
			{X: 208, Y: 90, Frames: 480},
			{X: 208, Y: -60, Frames: 40},
		})
		ctx.AddEnemy(midBoss)
		s.midBoss = midBoss
		s.midBossSpawned = true
		ctx.StartDialog([]dialog.Line{
			{Speaker: "灵梦", Text: "前面就是山顶了吧。", IsRight: false},
			{Speaker: "犬走椛", Text: "这里开始是妖怪之山的警戒线，闲人禁止通行。", IsRight: true},
			{Speaker: "犬走椛", Text: "我是白狼天狗犬走椛，会在这里把你拦下。", IsRight: true},
		})
	}

	for _, e := range ctx.Enemies {
		if !e.Active || e.Type != enemy.TypeMidBoss {
			continue
		}
		if e.Age%25 == 0 {
			ctx.Emitter.SetPos(e.X, e.Y)
			ctx.Emitter.Ring(18+ctx.State.Difficulty*4, 3.0, float64(e.Age)*0.08, bullet.TypeMiddle, 5)
		}
		if e.Age%40 == 0 {
			ctx.Emitter.SetPos(e.X, e.Y)
			ctx.Emitter.Aimed(ctx.Player.X, ctx.Player.Y, 5+ctx.State.Difficulty, 4.0, math.Pi/5, bullet.TypeRice, 1)
		}
	}

	// ---- 中Boss后道中 ----
	midBossDead := s.midBossSpawned && s.midBoss != nil && !s.midBoss.Active
	if midBossDead && s.frame >= 1080 && s.frame <= 1200 && s.frame%10 == 0 {
		x := 40 + float64((s.frame-1080)/10%8)*42
		e := enemy.New(x, -16, 35, enemy.TypeFairy)
		e.SetPath([]enemy.PathNode{
			{X: x, Y: 100, Frames: 20},
			{X: x, Y: 100, Frames: 40},
			{X: x, Y: 500, Frames: 35},
		})
		e.DropPower = 2
		e.DropPoint = 1
		ctx.AddEnemy(e)
	}

	// ---- Boss: 射命丸文 ----
	if midBossDead && s.frame >= 1200 && !s.bossSpawned {
		ctx.StartDialog([]dialog.Line{
			{Speaker: "灵梦", Text: "你那速度……是天狗吗？", IsRight: false},
			{Speaker: "射命丸文", Text: "我是文々。新聞的射命丸文！来采访你一下！", IsRight: true},
			{Speaker: "灵梦", Text: "不接受采访！", IsRight: false},
		})
		s.bossSpawned = true
	}

	if s.bossSpawned && s.boss == nil && s.frame > 1200 {
		s.spawnBoss(ctx)
	}

	if s.boss != nil && s.boss.Active && s.boss.Boss != nil && s.boss.Boss.Invincible {
		if s.boss.Boss.PhaseFrame > 2 {
			s.boss.EndInvincible()
		}
	}

	if s.bossSpawned && s.boss != nil && !s.boss.Active {
		s.finished = true
	}

	if s.frame >= 5400 {
		s.finished = true
	}
}

func (s *Script4) spawnBoss(ctx *stage.Context) {
	ctx.Audio.PlayBGM(audio.BGMStage4Boss)

	phases := []enemy.SpellCard{
		{
			Name: "", HP: 600, TimeLimit: 1800, Bonus: 0,
			Update: func(e *enemy.Enemy, frame int) {
				ctx.Emitter.SetPos(e.X, e.Y)
				if frame%20 == 0 {
					ctx.Emitter.Ring(18+ctx.State.Difficulty*4, 3.0, float64(frame)*0.09, bullet.TypeMiddle, 5)
				}
				if frame%35 == 0 {
					ctx.Emitter.Aimed(ctx.Player.X, ctx.Player.Y, 6+ctx.State.Difficulty, 4.0, math.Pi/4, bullet.TypeRice, 1)
				}
			},
		},
		{
			Name: "風神「風神木の葉隠れ」", HP: 700, TimeLimit: 2400, Bonus: 800000,
			Update: func(e *enemy.Enemy, frame int) {
				ctx.Emitter.SetPos(e.X, e.Y)
				if frame%4 == 0 {
					a := float64(frame) * 0.09
					ctx.Emitter.Spiral(5, 2.2, a, bullet.TypeSmall, 3)
				}
				if frame%55 == 0 {
					ctx.Emitter.Ring(24+ctx.State.Difficulty*6, 1.8, float64(frame)*0.04, bullet.TypeMiddle, 7)
				}
			},
		},
		{
			Name: "「天狗のマクロバースト」", HP: 800, TimeLimit: 3000, Bonus: 1200000,
			Update: func(e *enemy.Enemy, frame int) {
				ctx.Emitter.SetPos(e.X, e.Y)
				// 高速放射
				if frame%2 == 0 {
					a := float64(frame) * 0.11
					ctx.Emitter.Spiral(6, 2.5, a, bullet.TypeSmall, 5)
				}
				if frame%40 == 0 {
					ctx.Emitter.Aimed(ctx.Player.X, ctx.Player.Y, 8+ctx.State.Difficulty*2, 3.5, math.Pi/3, bullet.TypeRice, 2)
				}
				if frame%90 == 0 {
					ctx.Emitter.Ring(30+ctx.State.Difficulty*8, 1.5, 0, bullet.TypeMiddle, 6)
				}
			},
		},
	}

	boss := enemy.NewBoss(208, -40, phases)
	boss.SpriteID = 4
	boss.DropPower = 30
	boss.DropPoint = 25
	boss.SetPath([]enemy.PathNode{
		{X: 208, Y: 90, Frames: 40},
	})
	ctx.AddEnemy(boss)
	s.boss = boss
}

func (s *Script4) BgDraw(field *ebiten.Image) {
	w := float32(field.Bounds().Dx())
	h := float32(field.Bounds().Dy())

	// 红叶瀑布
	field.Fill(color.RGBA{32, 20, 16, 255})

	// Tile main background texture
	if s.bg1 != nil {
		tw := float64(s.bg1.Bounds().Dx())
		th := float64(s.bg1.Bounds().Dy())
		offY := math.Mod(s.bgY, th)
		for y := -offY; y < float64(h); y += th {
			for x := 0.0; x < float64(w); x += tw {
				op := &ebiten.DrawImageOptions{}
				op.GeoM.Translate(x, y)
				field.DrawImage(s.bg1, op)
			}
		}
	} else {
		offset := float32(math.Mod(s.bgY, 36))
		for y := -offset; y < h; y += 36 {
			vector.StrokeLine(field, 0, y, w, y, 0.5, color.RGBA{60, 40, 30, 100}, false)
		}
		vector.FillRect(field, w*0.4, 0, w*0.2, h, color.RGBA{30, 50, 80, 40}, false)
		wy := float32(math.Mod(float64(s.bgY)*3, float64(h)))
		for y := float32(0); y < h; y += 8 {
			a := uint8(40 + 20*math.Sin(float64(y+wy)*0.1))
			vector.StrokeLine(field, w*0.45, y, w*0.55, y+4, 1, color.RGBA{60, 90, 140, a}, false)
		}
	}

	// Overlay second texture with alpha
	if s.bg2 != nil {
		tw := float64(s.bg2.Bounds().Dx())
		th := float64(s.bg2.Bounds().Dy())
		offY := math.Mod(s.bgY*0.6, th)
		for y := -offY; y < float64(h); y += th {
			for x := 0.0; x < float64(w); x += tw {
				op := &ebiten.DrawImageOptions{}
				op.GeoM.Translate(x, y)
				op.ColorScale.ScaleAlpha(0.25)
				field.DrawImage(s.bg2, op)
			}
		}
	}
}

func (s *Script4) Finished() bool { return s.finished }

var _ stage.Script = (*Script4)(nil)
