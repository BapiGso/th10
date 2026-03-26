package stage3

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

// Script3 第三关脚本 — 河城にとり
type Script3 struct {
	frame          int
	finished       bool
	bgY            float64
	midBossDone    bool
	bossSpawned    bool
	boss           *enemy.Enemy
	bg1, bg2       *ebiten.Image // stg3bg.png [256x256], stg3bg2.png [128x128]
	bg3, bg4       *ebiten.Image // stg3bg3.png [512x512], stg3bg4.png [256x512]
}

func NewScript() *Script3 {
	s := &Script3{}
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

func (s *Script3) loadBg() {
	s.bg1 = loadImage("anm/background/stg3bg.png")
	s.bg2 = loadImage("anm/background/stg3bg2.png")
	s.bg3 = loadImage("anm/background/stg3bg3.png")
	s.bg4 = loadImage("anm/background/stg3bg4.png")
}

func (s *Script3) Init(ctx *stage.Context) {
	ctx.Audio.PlayBGM(audio.BGMStage3)
}

func (s *Script3) Update(ctx *stage.Context) {
	s.frame++
	s.bgY += 1.8

	// ---- 杂兵波次 ----

	if s.frame >= 60 && s.frame <= 300 && s.frame%20 == 0 {
		x := 60 + float64((s.frame/20)%7)*40
		e := enemy.New(x, -16, 25, enemy.TypeFairy)
		e.SetPath([]enemy.PathNode{
			{X: x, Y: 80 + float64((s.frame/20)%3)*40, Frames: 30},
			{X: x + 30, Y: 200, Frames: 50},
			{X: x, Y: 500, Frames: 60},
		})
		e.DropPower = 2
		ctx.AddEnemy(e)
	}

	if s.frame >= 360 && s.frame <= 600 && s.frame%15 == 0 {
		for _, side := range []float64{60, 356} {
			e := enemy.New(side, -16, 30, enemy.TypeFairy)
			e.SetPath([]enemy.PathNode{
				{X: 208, Y: 100, Frames: 40},
				{X: side, Y: 250, Frames: 50},
				{X: side, Y: 500, Frames: 40},
			})
			e.DropPower = 1
			e.DropPoint = 1
			ctx.AddEnemy(e)
		}
	}

	if s.frame >= 660 && s.frame <= 960 && s.frame%10 == 0 {
		x := 40 + float64((s.frame-660)/10%10)*35
		e := enemy.New(x, -16, 40, enemy.TypeFairy)
		e.SetPath([]enemy.PathNode{
			{X: x, Y: 130, Frames: 20},
			{X: x, Y: 130, Frames: 60},
			{X: x, Y: 500, Frames: 40},
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
		if e.Age > 20 && e.Age%16 == 0 {
			ctx.Emitter.SetPos(e.X, e.Y)
			count := 8 + ctx.State.Difficulty*3
			ctx.Emitter.Ring(count, 2.5, float64(e.Age)*0.15, bullet.TypeSmall, 2)
		}
	}

	// ---- 中Boss: にとり (短期) ----
	if s.frame == 720 && !s.midBossDone {
		midBoss := enemy.New(208, -40, 300, enemy.TypeMidBoss)
		midBoss.SpriteID = 3
		midBoss.HitWidth = 40
		midBoss.HitHeight = 40
		midBoss.DropPower = 15
		midBoss.SetPath([]enemy.PathNode{
			{X: 208, Y: 100, Frames: 50},
			{X: 208, Y: 100, Frames: 480},
			{X: 208, Y: -60, Frames: 50},
		})
		ctx.AddEnemy(midBoss)
	}

	// 中Boss弹幕
	for _, e := range ctx.Enemies {
		if !e.Active || e.Type != enemy.TypeMidBoss {
			continue
		}
		if e.Age%35 == 0 {
			ctx.Emitter.SetPos(e.X, e.Y)
			ctx.Emitter.Ring(16+ctx.State.Difficulty*4, 2.5, float64(e.Age)*0.06, bullet.TypeMiddle, 4)
		}
		if !e.Active {
			s.midBossDone = true
		}
	}

	// ---- Boss: にとり ----

	if s.frame == 1200 {
		ctx.StartDialog([]dialog.Line{
			{Speaker: "灵梦", Text: "又是你？刚才不是打跑你了吗？", IsRight: false},
			{Speaker: "河城にとり", Text: "那只是打个招呼！来见识一下河童的技术吧！", IsRight: true},
		})
	}

	if s.frame > 1200 && !s.bossSpawned {
		s.spawnBoss(ctx)
		s.bossSpawned = true
	}

	if s.boss != nil && s.boss.Active && s.boss.Boss != nil && s.boss.Boss.Invincible {
		if s.boss.Boss.PhaseFrame > 2 {
			s.boss.EndInvincible()
		}
	}

	if s.bossSpawned && s.boss != nil && !s.boss.Active {
		s.finished = true
	}

	if s.frame >= 4800 {
		s.finished = true
	}
}

func (s *Script3) spawnBoss(ctx *stage.Context) {
	ctx.Audio.PlayBGM(audio.BGMStage3Boss)

	phases := []enemy.SpellCard{
		{
			Name: "", HP: 500, TimeLimit: 1800, Bonus: 0,
			Update: func(e *enemy.Enemy, frame int) {
				ctx.Emitter.SetPos(e.X, e.Y)
				if frame%25 == 0 {
					ctx.Emitter.Ring(16+ctx.State.Difficulty*4, 2.8, float64(frame)*0.08, bullet.TypeMiddle, 4)
				}
				if frame%40 == 0 {
					ctx.Emitter.Aimed(ctx.Player.X, ctx.Player.Y, 5+ctx.State.Difficulty, 3.5, math.Pi/5, bullet.TypeRice, 1)
				}
			},
		},
		{
			Name: "河童「のびーるアーム」", HP: 600, TimeLimit: 2400, Bonus: 700000,
			Update: func(e *enemy.Enemy, frame int) {
				ctx.Emitter.SetPos(e.X, e.Y)
				if frame%5 == 0 {
					angle := float64(frame) * 0.1
					ctx.Emitter.Spiral(4, 2.0, angle, bullet.TypeSmall, 4)
				}
				if frame%70 == 0 {
					ctx.Emitter.Ring(24+ctx.State.Difficulty*6, 1.5, 0, bullet.TypeMiddle, 6)
				}
			},
		},
		{
			Name: "河童「お化けキューカンバー」", HP: 700, TimeLimit: 3000, Bonus: 1000000,
			Update: func(e *enemy.Enemy, frame int) {
				ctx.Emitter.SetPos(e.X, e.Y)
				if frame%3 == 0 {
					a := float64(frame) * 0.07
					ctx.Emitter.Spiral(3, 2.5, a, bullet.TypeSmall, 2)
					ctx.Emitter.Spiral(3, 2.5, -a, bullet.TypeSmall, 5)
				}
				if frame%50 == 0 {
					ctx.Emitter.Aimed(ctx.Player.X, ctx.Player.Y, 8, 3.0, math.Pi/3, bullet.TypeRice, 3)
				}
			},
		},
	}

	boss := enemy.NewBoss(208, -40, phases)
	boss.SpriteID = 3
	boss.DropPower = 30
	boss.DropPoint = 20
	boss.SetPath([]enemy.PathNode{
		{X: 208, Y: 100, Frames: 60},
	})
	ctx.AddEnemy(boss)
	s.boss = boss
}

func (s *Script3) BgDraw(field *ebiten.Image) {
	w := float32(field.Bounds().Dx())
	h := float32(field.Bounds().Dy())

	// 蓝绿色山间溪流
	field.Fill(color.RGBA{12, 32, 28, 255})

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
		offset := float32(math.Mod(s.bgY, 48))
		for y := -offset; y < h; y += 48 {
			vector.StrokeLine(field, 0, y, w, y, 0.5, color.RGBA{30, 60, 55, 100}, false)
		}
		ox := float32(math.Sin(float64(s.bgY)*0.02) * 20)
		vector.StrokeLine(field, w*0.4+ox, 0, w*0.4+ox, h, 2, color.RGBA{40, 80, 120, 80}, false)
		vector.StrokeLine(field, w*0.6-ox, 0, w*0.6-ox, h, 2, color.RGBA{40, 80, 120, 80}, false)
	}

	// Overlay second texture with alpha
	if s.bg2 != nil {
		tw := float64(s.bg2.Bounds().Dx())
		th := float64(s.bg2.Bounds().Dy())
		offY := math.Mod(s.bgY*0.7, th)
		for y := -offY; y < float64(h); y += th {
			for x := 0.0; x < float64(w); x += tw {
				op := &ebiten.DrawImageOptions{}
				op.GeoM.Translate(x, y)
				op.ColorScale.ScaleAlpha(0.3)
				field.DrawImage(s.bg2, op)
			}
		}
	}
}

func (s *Script3) Finished() bool { return s.finished }

var _ stage.Script = (*Script3)(nil)
