package stage6

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

// Script6 第六关脚本 — 八坂神奈子
type Script6 struct {
	frame       int
	finished    bool
	bgY         float64
	bossSpawned bool
	boss        *enemy.Enemy
	bg1, bg2    *ebiten.Image // stg6bg.png [512x512], stg6bg2.png [256x256]
	bg3         *ebiten.Image // stg6bg3.png [32x256]
	bg4, bg5    *ebiten.Image // stg6bg5.png [256x256], stg6bg6.png [32x128]
}

func NewScript() *Script6 {
	s := &Script6{}
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

func (s *Script6) loadBg() {
	s.bg1 = loadImage("anm/background/stg6bg.png")
	s.bg2 = loadImage("anm/background/stg6bg2.png")
	s.bg3 = loadImage("anm/background/stg6bg3.png")
	s.bg4 = loadImage("anm/background/stg6bg5.png")
	s.bg5 = loadImage("anm/background/stg6bg6.png")
}

func (s *Script6) Init(ctx *stage.Context) {
	ctx.Audio.PlayBGM(audio.BGMStage6)
}

func (s *Script6) Update(ctx *stage.Context) {
	s.frame++
	s.bgY += 1.2

	// ---- 杂兵 ----
	if s.frame >= 60 && s.frame <= 600 && s.frame%14 == 0 {
		x := 30 + float64((s.frame/14)%10)*36
		e := enemy.New(x, -16, 50, enemy.TypeFairy)
		e.SetPath([]enemy.PathNode{
			{X: x, Y: 100, Frames: 22},
			{X: x, Y: 100, Frames: 60},
			{X: x, Y: 500, Frames: 40},
		})
		e.DropPower = 2
		e.DropPoint = 1
		ctx.AddEnemy(e)
	}

	if s.frame >= 660 && s.frame <= 1020 && s.frame%8 == 0 {
		for _, side := range []float64{30, 386} {
			e := enemy.New(side, -16, 55, enemy.TypeFairy)
			e.SetPath([]enemy.PathNode{
				{X: 208, Y: 80, Frames: 25},
				{X: 208, Y: 80, Frames: 40},
				{X: side, Y: 500, Frames: 30},
			})
			e.DropPower = 2
			ctx.AddEnemy(e)
		}
	}

	// 杂兵弹幕
	for _, e := range ctx.Enemies {
		if !e.Active || e.Type != enemy.TypeFairy {
			continue
		}
		if e.Age > 20 && e.Age%10 == 0 {
			ctx.Emitter.SetPos(e.X, e.Y)
			count := 12 + ctx.State.Difficulty*4
			ctx.Emitter.Ring(count, 3.2, float64(e.Age)*0.15, bullet.TypeSmall, 5)
		}
	}

	// ---- Boss: 八坂神奈子 ----
	if s.frame == 1500 {
		ctx.StartDialog([]dialog.Line{
			{Speaker: "灵梦", Text: "终于找到你了……你就是山上的神明？", IsRight: false},
			{Speaker: "八坂神奈子", Text: "我是八坂神奈子，是掌管风雨的神！", IsRight: true},
			{Speaker: "八坂神奈子", Text: "你想阻止我收集信仰吗？", IsRight: true},
			{Speaker: "灵梦", Text: "你在幻想乡擅自搞事，当然要管！", IsRight: false},
			{Speaker: "八坂神奈子", Text: "那就让你见识一下真正的神之力！", IsRight: true},
		})
	}

	if s.frame > 1500 && !s.bossSpawned {
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

	if s.frame >= 7200 {
		s.finished = true
	}
}

func (s *Script6) spawnBoss(ctx *stage.Context) {
	ctx.Audio.PlayBGM(audio.BGMStage6Boss)

	phases := []enemy.SpellCard{
		{
			Name: "", HP: 800, TimeLimit: 1800, Bonus: 0,
			Update: func(e *enemy.Enemy, frame int) {
				ctx.Emitter.SetPos(e.X, e.Y)
				if frame%15 == 0 {
					ctx.Emitter.Ring(22+ctx.State.Difficulty*6, 3.2, float64(frame)*0.08, bullet.TypeMiddle, 7)
				}
				if frame%25 == 0 {
					ctx.Emitter.Aimed(ctx.Player.X, ctx.Player.Y, 8+ctx.State.Difficulty*2, 4.0, math.Pi/4, bullet.TypeRice, 2)
				}
			},
		},
		{
			Name: "神祭「エクスパンデッドオンバシラ」", HP: 1000, TimeLimit: 2400, Bonus: 1500000,
			Update: func(e *enemy.Enemy, frame int) {
				ctx.Emitter.SetPos(e.X, e.Y)
				if frame%3 == 0 {
					a := float64(frame) * 0.07
					ctx.Emitter.Spiral(4, 2.8, a, bullet.TypeSmall, 5)
				}
				if frame%50 == 0 {
					ctx.Emitter.Ring(30+ctx.State.Difficulty*8, 1.8, float64(frame)*0.03, bullet.TypeMiddle, 1)
				}
			},
		},
		{
			Name: "「風神様の神徳」", HP: 1200, TimeLimit: 3000, Bonus: 2000000,
			Update: func(e *enemy.Enemy, frame int) {
				ctx.Emitter.SetPos(e.X, e.Y)
				if frame%2 == 0 {
					a := float64(frame) * 0.09
					ctx.Emitter.Spiral(6, 2.5, a, bullet.TypeSmall, 3)
					ctx.Emitter.Spiral(6, 2.5, -a, bullet.TypeSmall, 6)
				}
				if frame%40 == 0 {
					ctx.Emitter.Aimed(ctx.Player.X, ctx.Player.Y, 10+ctx.State.Difficulty*3, 3.5, math.Pi/3, bullet.TypeRice, 4)
				}
			},
		},
		{
			Name: "「マウンテン・オブ・フェイス」", HP: 1500, TimeLimit: 3600, Bonus: 3000000,
			Update: func(e *enemy.Enemy, frame int) {
				ctx.Emitter.SetPos(e.X, e.Y)
				// 最终符: 高密度多层弹幕
				if frame%2 == 0 {
					a := float64(frame) * 0.12
					ctx.Emitter.Spiral(8, 2.8, a, bullet.TypeSmall, 7)
				}
				if frame%30 == 0 {
					ctx.Emitter.Ring(36+ctx.State.Difficulty*10, 2.0, float64(frame)*0.04, bullet.TypeMiddle, 2)
				}
				if frame%35 == 0 {
					ctx.Emitter.Aimed(ctx.Player.X, ctx.Player.Y, 12, 4.0, math.Pi/3, bullet.TypeRice, 5)
				}
			},
		},
	}

	boss := enemy.NewBoss(208, -40, phases)
	boss.SpriteID = 6
	boss.DropPower = 50
	boss.DropPoint = 40
	boss.SetPath([]enemy.PathNode{
		{X: 208, Y: 80, Frames: 60},
	})
	ctx.AddEnemy(boss)
	s.boss = boss
}

func (s *Script6) BgDraw(field *ebiten.Image) {
	w := float32(field.Bounds().Dx())
	h := float32(field.Bounds().Dy())

	// 御柱墓场 - 暗灰+紫红
	field.Fill(color.RGBA{20, 12, 24, 255})

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
		offset := float32(math.Mod(s.bgY, 50))
		for y := -offset; y < h; y += 50 {
			vector.StrokeLine(field, 0, y, w, y, 0.3, color.RGBA{45, 30, 50, 80}, false)
		}
		for i := 0; i < 4; i++ {
			x := w*0.15 + float32(i)*w*0.22
			vector.FillRect(field, x-4, 0, 8, h, color.RGBA{80, 50, 30, 50}, false)
		}
	}

	// Overlay second texture with alpha
	if s.bg2 != nil {
		tw := float64(s.bg2.Bounds().Dx())
		th := float64(s.bg2.Bounds().Dy())
		offY := math.Mod(s.bgY*0.4, th)
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

func (s *Script6) Finished() bool { return s.finished }

var _ stage.Script = (*Script6)(nil)
