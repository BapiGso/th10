package stage2

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

// Script2 第二关脚本 — 鍵山雛
type Script2 struct {
	frame       int
	finished    bool
	bgY         float64
	bossSpawned bool
	boss        *enemy.Enemy
	bg1, bg2    *ebiten.Image // stg2bg.png [256x256], stg2bg2.png [512x512]
}

func NewScript() *Script2 {
	s := &Script2{}
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

func (s *Script2) loadBg() {
	s.bg1 = loadImage("anm/background/stg2bg.png")
	s.bg2 = loadImage("anm/background/stg2bg2.png")
}

func (s *Script2) Init(ctx *stage.Context) {
	ctx.Audio.PlayBGM(audio.BGMStage2)
}

func (s *Script2) Update(ctx *stage.Context) {
	s.frame++
	s.bgY += 2.0

	// ---- 杂兵波次 ----

	// 第1波: 交叉飞行的妖精
	if s.frame >= 60 && s.frame <= 360 && s.frame%25 == 0 {
		side := float64(80)
		if (s.frame/25)%2 == 1 {
			side = 336
		}
		e := enemy.New(side, -16, 25, enemy.TypeFairy)
		targetX := 416 - side
		e.SetPath([]enemy.PathNode{
			{X: targetX, Y: 120, Frames: 50},
			{X: targetX, Y: 500, Frames: 60},
		})
		e.DropPower = 2
		e.DropPoint = 1
		ctx.AddEnemy(e)
	}

	// 第2波: V字编队
	if s.frame >= 420 && s.frame <= 600 && s.frame%40 == 0 {
		cx := 208.0
		for i := -2; i <= 2; i++ {
			x := cx + float64(i)*40
			y := -16.0 - math.Abs(float64(i))*20
			e := enemy.New(x, y, 30, enemy.TypeFairy)
			e.SetPath([]enemy.PathNode{
				{X: x, Y: 150, Frames: 40},
				{X: x, Y: 150, Frames: 80},
				{X: x, Y: 500, Frames: 50},
			})
			e.DropPower = 1
			ctx.AddEnemy(e)
		}
	}

	// 第3波: 密集涌入
	if s.frame >= 660 && s.frame <= 900 && s.frame%12 == 0 {
		x := 40 + float64((s.frame-660)/12%8)*45
		e := enemy.New(x, -16, 35, enemy.TypeFairy)
		e.SetPath([]enemy.PathNode{
			{X: x, Y: 100, Frames: 25},
			{X: x, Y: 100, Frames: 50},
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
		if e.Age > 25 && e.Age%18 == 0 {
			ctx.Emitter.SetPos(e.X, e.Y)
			count := 8 + ctx.State.Difficulty*2
			ctx.Emitter.Ring(count, 2.2, float64(e.Age)*0.12, bullet.TypeSmall, 1)
		}
	}

	// ---- Boss: 鍵山雛 ----

	if s.frame == 1020 {
		ctx.StartDialog([]dialog.Line{
			{Speaker: "灵梦", Text: "你就是那个转来转去的神明？", IsRight: false},
			{Speaker: "鍵山雛", Text: "我是厄神！来收集你身上的厄运吧！", IsRight: true},
			{Speaker: "灵梦", Text: "不需要，我自己的运气自己管。", IsRight: false},
		})
	}

	if s.frame > 1020 && !s.bossSpawned {
		s.spawnBoss(ctx)
		s.bossSpawned = true
	}

	// Boss 阶段切换
	if s.boss != nil && s.boss.Active && s.boss.Boss != nil && s.boss.Boss.Invincible {
		if s.boss.Boss.PhaseFrame > 2 {
			s.boss.EndInvincible()
		}
	}

	if s.bossSpawned && s.boss != nil && !s.boss.Active {
		s.finished = true
	}

	if s.frame >= 4200 {
		s.finished = true
	}
}

func (s *Script2) spawnBoss(ctx *stage.Context) {
	ctx.Audio.PlayBGM(audio.BGMStage2Boss)

	phases := []enemy.SpellCard{
		{
			Name: "", HP: 400, TimeLimit: 1800, Bonus: 0,
			Update: func(e *enemy.Enemy, frame int) {
				ctx.Emitter.SetPos(e.X, e.Y)
				if frame%30 == 0 {
					count := 14 + ctx.State.Difficulty*4
					ctx.Emitter.Ring(count, 2.5, float64(frame)*0.07, bullet.TypeMiddle, 1)
				}
				if frame%50 == 0 {
					ctx.Emitter.Aimed(ctx.Player.X, ctx.Player.Y, 4+ctx.State.Difficulty, 3.5, math.Pi/5, bullet.TypeRice, 4)
				}
			},
		},
		{
			Name: "厄符「バッドフォーチュン」", HP: 500, TimeLimit: 2400, Bonus: 600000,
			Update: func(e *enemy.Enemy, frame int) {
				ctx.Emitter.SetPos(e.X, e.Y)
				// 回転弾幕
				if frame%4 == 0 {
					angle := float64(frame) * 0.06
					ctx.Emitter.Spiral(3, 2.0, angle, bullet.TypeSmall, 2)
				}
				if frame%60 == 0 {
					count := 20 + ctx.State.Difficulty*6
					ctx.Emitter.Ring(count, 1.8, float64(frame)*0.02, bullet.TypeMiddle, 6)
				}
			},
		},
		{
			Name: "厄神「ミスフォーチュンズホイール」", HP: 600, TimeLimit: 3000, Bonus: 800000,
			Update: func(e *enemy.Enemy, frame int) {
				ctx.Emitter.SetPos(e.X, e.Y)
				// 双方向螺旋
				if frame%3 == 0 {
					a1 := float64(frame) * 0.05
					a2 := -float64(frame) * 0.05
					ctx.Emitter.Spiral(2, 2.2, a1, bullet.TypeSmall, 3)
					ctx.Emitter.Spiral(2, 2.2, a2, bullet.TypeSmall, 5)
				}
				if frame%45 == 0 {
					ctx.Emitter.Aimed(ctx.Player.X, ctx.Player.Y, 6+ctx.State.Difficulty, 3.0, math.Pi/3, bullet.TypeRice, 2)
				}
			},
		},
	}

	boss := enemy.NewBoss(208, -40, phases)
	boss.SpriteID = 2
	boss.DropPower = 30
	boss.DropPoint = 20
	boss.SetPath([]enemy.PathNode{
		{X: 208, Y: 100, Frames: 60},
	})
	ctx.AddEnemy(boss)
	s.boss = boss
}

func (s *Script2) BgDraw(field *ebiten.Image) {
	w := float32(field.Bounds().Dx())
	h := float32(field.Bounds().Dy())

	// 暗紫色阶梯参道
	field.Fill(color.RGBA{24, 16, 40, 255})

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
		offset := float32(math.Mod(s.bgY, 32))
		for y := -offset; y < h; y += 32 {
			vector.StrokeLine(field, 0, y, w, y, 0.8, color.RGBA{50, 35, 70, 120}, false)
		}
		for y := -offset; y < h; y += 32 {
			vector.FillRect(field, w*0.15, y, w*0.7, 2, color.RGBA{60, 45, 80, 80}, false)
		}
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
				op.ColorScale.ScaleAlpha(0.25)
				field.DrawImage(s.bg2, op)
			}
		}
	}
}

func (s *Script2) Finished() bool { return s.finished }

var _ stage.Script = (*Script2)(nil)
