package stage5

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

// Script5 第五关脚本 — 東風谷早苗
type Script5 struct {
	frame       int
	finished    bool
	bgY         float64
	bossSpawned bool
	boss        *enemy.Enemy
	bg1, bg2    *ebiten.Image // stg5bg.png [256x256], stg5bg2.png [256x256]
}

func NewScript() *Script5 {
	s := &Script5{}
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

func (s *Script5) loadBg() {
	s.bg1 = loadImage("anm/background/stg5bg.png")
	s.bg2 = loadImage("anm/background/stg5bg2.png")
}

func (s *Script5) Init(ctx *stage.Context) {
	ctx.Audio.PlayBGM(audio.BGMStage5)
}

func (s *Script5) Update(ctx *stage.Context) {
	s.frame++
	s.bgY += 1.5

	// ---- 杂兵 ----
	if s.frame >= 60 && s.frame <= 480 && s.frame%16 == 0 {
		x := 40 + float64((s.frame/16)%9)*38
		e := enemy.New(x, -16, 35, enemy.TypeFairy)
		e.SetPath([]enemy.PathNode{
			{X: x, Y: 100, Frames: 25},
			{X: x + 20, Y: 160, Frames: 40},
			{X: x, Y: 500, Frames: 50},
		})
		e.DropPower = 2
		e.DropPoint = 1
		ctx.AddEnemy(e)
	}

	if s.frame >= 540 && s.frame <= 840 && s.frame%10 == 0 {
		x := 30 + float64((s.frame-540)/10%11)*34
		e := enemy.New(x, -16, 45, enemy.TypeFairy)
		e.SetPath([]enemy.PathNode{
			{X: x, Y: 120, Frames: 20},
			{X: x, Y: 120, Frames: 80},
			{X: x, Y: 500, Frames: 35},
		})
		e.DropPower = 2
		e.DropPoint = 1
		ctx.AddEnemy(e)
	}

	if s.frame >= 900 && s.frame <= 1200 && s.frame%8 == 0 {
		for _, side := range []float64{40, 376} {
			e := enemy.New(side, -16, 50, enemy.TypeFairy)
			e.SetPath([]enemy.PathNode{
				{X: 208, Y: 80, Frames: 30},
				{X: 208, Y: 80, Frames: 50},
				{X: side, Y: 500, Frames: 35},
			})
			e.DropPower = 1
			ctx.AddEnemy(e)
		}
	}

	// 杂兵弹幕
	for _, e := range ctx.Enemies {
		if !e.Active || e.Type != enemy.TypeFairy {
			continue
		}
		if e.Age > 20 && e.Age%12 == 0 {
			ctx.Emitter.SetPos(e.X, e.Y)
			count := 10 + ctx.State.Difficulty*3
			ctx.Emitter.Ring(count, 3.0, float64(e.Age)*0.14, bullet.TypeSmall, 4)
		}
	}

	// ---- 后半道中 ----
	if s.frame >= 1200 && s.frame <= 1380 && s.frame%8 == 0 {
		for _, side := range []float64{40, 376} {
			e := enemy.New(side, -16, 50, enemy.TypeFairy)
			e.SetPath([]enemy.PathNode{
				{X: 208, Y: 80, Frames: 30},
				{X: 208, Y: 80, Frames: 50},
				{X: side, Y: 500, Frames: 35},
			})
			e.DropPower = 1
			ctx.AddEnemy(e)
		}
	}

	// ---- Boss: 東風谷早苗 ----
	if s.frame == 1380 {
		ctx.StartDialog([]dialog.Line{
			{Speaker: "灵梦", Text: "你是……巫女？和我一样的？", IsRight: false},
			{Speaker: "東風谷早苗", Text: "我是守矢神社的风祝，東風谷早苗！", IsRight: true},
			{Speaker: "東風谷早苗", Text: "为了信仰，我不能让你通过！", IsRight: true},
			{Speaker: "灵梦", Text: "巫女对巫女……有点意思。", IsRight: false},
		})
	}

	if s.frame > 1380 && !s.bossSpawned {
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

	if s.frame >= 6000 {
		s.finished = true
	}
}

func (s *Script5) spawnBoss(ctx *stage.Context) {
	ctx.Audio.PlayBGM(audio.BGMStage5Boss)

	phases := []enemy.SpellCard{
		{
			Name: "", HP: 700, TimeLimit: 1800, Bonus: 0,
			Update: func(e *enemy.Enemy, frame int) {
				ctx.Emitter.SetPos(e.X, e.Y)
				if frame%18 == 0 {
					ctx.Emitter.Ring(20+ctx.State.Difficulty*5, 3.0, float64(frame)*0.08, bullet.TypeMiddle, 2)
				}
				if frame%30 == 0 {
					ctx.Emitter.Aimed(ctx.Player.X, ctx.Player.Y, 6+ctx.State.Difficulty*2, 4.0, math.Pi/5, bullet.TypeRice, 1)
				}
			},
		},
		{
			Name: "奇跡「白昼の客星」", HP: 800, TimeLimit: 2400, Bonus: 1000000,
			Update: func(e *enemy.Enemy, frame int) {
				ctx.Emitter.SetPos(e.X, e.Y)
				if frame%3 == 0 {
					a := float64(frame) * 0.08
					ctx.Emitter.Spiral(4, 2.5, a, bullet.TypeSmall, 6)
				}
				if frame%60 == 0 {
					ctx.Emitter.Ring(28+ctx.State.Difficulty*6, 1.5, float64(frame)*0.03, bullet.TypeMiddle, 3)
				}
			},
		},
		{
			Name: "開海「モーゼの奇跡」", HP: 900, TimeLimit: 3000, Bonus: 1500000,
			Update: func(e *enemy.Enemy, frame int) {
				ctx.Emitter.SetPos(e.X, e.Y)
				if frame%2 == 0 {
					a := float64(frame) * 0.06
					ctx.Emitter.Spiral(5, 2.8, a, bullet.TypeSmall, 4)
					ctx.Emitter.Spiral(5, 2.8, -a+math.Pi, bullet.TypeSmall, 7)
				}
				if frame%45 == 0 {
					ctx.Emitter.Aimed(ctx.Player.X, ctx.Player.Y, 10+ctx.State.Difficulty*2, 3.5, math.Pi/3, bullet.TypeRice, 2)
				}
			},
		},
		{
			Name: "秘術「グレイソーマタージ」", HP: 1000, TimeLimit: 3600, Bonus: 2000000,
			Update: func(e *enemy.Enemy, frame int) {
				ctx.Emitter.SetPos(e.X, e.Y)
				if frame%2 == 0 {
					a := float64(frame) * 0.1
					ctx.Emitter.Spiral(6, 2.5, a, bullet.TypeSmall, 5)
				}
				if frame%35 == 0 {
					ctx.Emitter.Ring(32+ctx.State.Difficulty*8, 2.0, float64(frame)*0.05, bullet.TypeMiddle, 1)
				}
				if frame%50 == 0 {
					ctx.Emitter.Aimed(ctx.Player.X, ctx.Player.Y, 8, 4.0, math.Pi/4, bullet.TypeRice, 3)
				}
			},
		},
	}

	boss := enemy.NewBoss(208, -40, phases)
	boss.SpriteID = 5
	boss.DropPower = 40
	boss.DropPoint = 30
	boss.SetPath([]enemy.PathNode{
		{X: 208, Y: 90, Frames: 50},
	})
	ctx.AddEnemy(boss)
	s.boss = boss
}

func (s *Script5) BgDraw(field *ebiten.Image) {
	w := float32(field.Bounds().Dx())
	h := float32(field.Bounds().Dy())

	// 神社境内，深蓝+紫
	field.Fill(color.RGBA{16, 16, 40, 255})

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
		offset := float32(math.Mod(s.bgY, 40))
		for y := -offset; y < h; y += 40 {
			vector.StrokeLine(field, 0, y, w, y, 0.4, color.RGBA{40, 35, 70, 90}, false)
		}
		vector.FillRect(field, w*0.2-3, 0, 6, h, color.RGBA{120, 30, 20, 60}, false)
		vector.FillRect(field, w*0.8-3, 0, 6, h, color.RGBA{120, 30, 20, 60}, false)
	}

	// Overlay second texture with alpha
	if s.bg2 != nil {
		tw := float64(s.bg2.Bounds().Dx())
		th := float64(s.bg2.Bounds().Dy())
		offY := math.Mod(s.bgY*0.5, th)
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

func (s *Script5) Finished() bool { return s.finished }

var _ stage.Script = (*Script5)(nil)
