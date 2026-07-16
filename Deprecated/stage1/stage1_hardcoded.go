package deprecatedstage1

import (
	"image/color"
	"math"
	"th10/audio"
	"th10/entity/bullet"
	"th10/entity/enemy"
	"th10/render"
	"th10/scene/dialog"
	"th10/scene/stage"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

// Script1 第一关脚本
// 中Boss: 秋静葉 (Shizuha Aki)
// Boss: 秋穣子 (Minoriko Aki)
type Script1 struct {
	frame          int
	finished       bool
	bgY            float64
	midBossSpawned bool
	midBoss        *enemy.Enemy
	bossSpawned    bool
	bossDialogDone bool
	postShown      bool
	boss           *enemy.Enemy
	bg1, bg2       *ebiten.Image // stg1bg.png [256x256], stg1bg2.png [256x256]
	bg3, bg4       *ebiten.Image // stg1bg3.png [128x128], stg1bg4.png [128x128]
}

func NewScript() stage.Script {
	return NewHandwrittenScript()
}

func NewHandwrittenScript() *Script1 {
	s := &Script1{}
	s.loadBg()
	return s
}

func (s *Script1) loadBg() {
	s.bg1 = render.LoadImage("anm/background/stg1bg.png")
	s.bg2 = render.LoadImage("anm/background/stg1bg2.png")
	s.bg3 = render.LoadImage("anm/background/stg1bg3.png")
	s.bg4 = render.LoadImage("anm/background/stg1bg4.png")
}

func (s *Script1) Init(ctx *stage.Context) {
	ctx.Audio.PlayBGM(audio.BGMStage1)
}

func (s *Script1) Update(ctx *stage.Context) {
	s.frame++
	s.bgY += 1.5

	// ---- 道中第1波: 散漫飞行的妖精 ----
	if s.frame >= 60 && s.frame <= 270 && s.frame%30 == 0 {
		x := 100 + float64((s.frame/30)%5)*60
		e := enemy.New(x, -16, 20, enemy.TypeFairy)
		e.SetPath([]enemy.PathNode{
			{X: x, Y: 100, Frames: 40},
			{X: x + 50, Y: 200, Frames: 60},
			{X: x, Y: 500, Frames: 80},
		})
		e.DropPower = 2
		e.DropPoint = 1
		ctx.AddEnemy(e)
	}

	// ---- 道中第2波: 两列对称 ----
	if s.frame >= 300 && s.frame <= 480 && s.frame%20 == 0 {
		for _, side := range []float64{80, 336} {
			e := enemy.New(side, -16, 30, enemy.TypeFairy)
			targetY := 60 + float64((s.frame-300)/20)*30
			e.SetPath([]enemy.PathNode{
				{X: side, Y: targetY, Frames: 30},
				{X: 208, Y: targetY + 80, Frames: 60},
				{X: side, Y: 500, Frames: 60},
			})
			e.DropPower = 1
			ctx.AddEnemy(e)
		}
	}

	// ---- 道中第3波: 大量涌入 ----
	if s.frame >= 540 && s.frame <= 780 && s.frame%15 == 0 {
		x := 60 + float64((s.frame-540)/15%6)*50
		e := enemy.New(x, -16, 40, enemy.TypeFairy)
		e.SetPath([]enemy.PathNode{
			{X: x, Y: 120, Frames: 30},
			{X: x, Y: 120, Frames: 60},
			{X: x, Y: 500, Frames: 50},
		})
		e.DropPower = 2
		e.DropPoint = 1
		ctx.AddEnemy(e)
	}

	// 杂兵弹幕（仅画面内的敌人发射）
	for _, e := range ctx.Enemies {
		if !e.Active || e.Type != enemy.TypeFairy {
			continue
		}
		if e.Y < 0 || e.Y > 448 {
			continue
		}
		if e.Age > 30 && e.Age%20 == 0 {
			ctx.Emitter.SetPos(e.X, e.Y)
			count := 6 + ctx.State.Difficulty*2
			baseAngle := float64(e.Age) * 0.1
			ctx.Emitter.Ring(count, 2.0, baseAngle, bullet.TypeSmall, 0)
		}
	}

	// ======== 中Boss: 秋静葉 ========

	if s.frame == 900 {
		ctx.StartDialog(dialog.MidPre("stage1"))
	}

	if s.frame > 900 && !s.midBossSpawned {
		s.spawnMidBoss(ctx)
		s.midBossSpawned = true
	}

	// 中Boss 无敌解除
	s.midBoss.MaybeEndInvincible()

	// ---- 中Boss后道中（1200帧起）----
	midBossDead := s.midBossSpawned && s.midBoss != nil && !s.midBoss.Active
	if midBossDead && s.frame >= 1200 && s.frame <= 1500 && s.frame%12 == 0 {
		x := 40 + float64((s.frame-1200)/12%8)*45
		e := enemy.New(x, -16, 30, enemy.TypeFairy)
		e.SetPath([]enemy.PathNode{
			{X: x, Y: 100, Frames: 25},
			{X: x, Y: 100, Frames: 50},
			{X: x, Y: 500, Frames: 40},
		})
		e.DropPower = 2
		e.DropPoint = 1
		ctx.AddEnemy(e)
	}

	// ======== Boss: 秋穣子 ========

	if midBossDead && s.frame >= 1560 && !s.bossSpawned {
		s.bossSpawned = true
		ctx.StartDialog(dialog.BossPre("stage1"))
	}

	// 对话结束后生成Boss（帧数足够大时）
	if s.bossSpawned && s.boss == nil && s.frame > 1560 {
		s.spawnBoss(ctx)
	}

	// Boss 无敌解除
	s.boss.MaybeEndInvincible()

	// Boss 击破后播放结尾对话
	if s.boss != nil && !s.boss.Active {
		if !s.postShown {
			s.postShown = true
			if post := dialog.Post("stage1"); len(post) > 0 {
				ctx.StartDialog(post)
			}
		} else {
			s.finished = true
		}
	}

	// 安全兜底
	if s.frame >= 4800 {
		s.finished = true
	}
}

func (s *Script1) spawnMidBoss(ctx *stage.Context) {
	phases := []enemy.SpellCard{
		{
			Name: "", HP: 300, TimeLimit: 1800, Bonus: 0,
			Update: func(e *enemy.Enemy, frame int) {
				ctx.Emitter.SetPos(e.X, e.Y)
				if frame%40 == 0 {
					count := 12 + ctx.State.Difficulty*4
					ctx.Emitter.Ring(count, 2.5, float64(frame)*0.05, bullet.TypeMiddle, 1)
				}
				if frame%60 == 0 {
					count := 3 + ctx.State.Difficulty
					ctx.Emitter.Aimed(ctx.Player.X, ctx.Player.Y, count, 3.5, math.Pi/6, bullet.TypeRice, 2)
				}
			},
		},
		{
			Name: "葉符「狂いの落葉」", HP: 400, TimeLimit: 2400, Bonus: 500000,
			Update: func(e *enemy.Enemy, frame int) {
				ctx.Emitter.SetPos(e.X, e.Y)
				if frame%3 == 0 {
					angle := float64(frame) * 0.08
					ctx.Emitter.Spiral(2, 2.0, angle, bullet.TypeSmall, 3)
				}
				if frame%80 == 0 {
					count := 16 + ctx.State.Difficulty*4
					ctx.Emitter.Ring(count, 1.5, float64(frame)*0.03, bullet.TypeMiddle, 5)
				}
			},
		},
	}

	boss := enemy.NewBoss(208, -40, phases)
	boss.SpriteID = 1
	boss.DropPower = 20
	boss.DropPoint = 10
	boss.SetPath([]enemy.PathNode{
		{X: 208, Y: 100, Frames: 60},
	})
	ctx.AddEnemy(boss)
	s.midBoss = boss
}

func (s *Script1) spawnBoss(ctx *stage.Context) {
	ctx.Audio.PlayBGM(audio.BGMStage1Boss)

	phases := []enemy.SpellCard{
		{
			Name: "", HP: 500, TimeLimit: 1800, Bonus: 0,
			Update: func(e *enemy.Enemy, frame int) {
				ctx.Emitter.SetPos(e.X, e.Y)
				if frame%30 == 0 {
					count := 14 + ctx.State.Difficulty*4
					ctx.Emitter.Ring(count, 2.8, float64(frame)*0.06, bullet.TypeMiddle, 2)
				}
				if frame%45 == 0 {
					count := 4 + ctx.State.Difficulty
					ctx.Emitter.Aimed(ctx.Player.X, ctx.Player.Y, count, 3.5, math.Pi/5, bullet.TypeRice, 4)
				}
			},
		},
		{
			Name: "秋符「オータムスカイ」", HP: 600, TimeLimit: 2400, Bonus: 500000,
			Update: func(e *enemy.Enemy, frame int) {
				ctx.Emitter.SetPos(e.X, e.Y)
				if frame%4 == 0 {
					angle := float64(frame) * 0.07
					ctx.Emitter.Spiral(3, 2.2, angle, bullet.TypeSmall, 5)
				}
				if frame%60 == 0 {
					count := 18 + ctx.State.Difficulty*4
					ctx.Emitter.Ring(count, 1.5, float64(frame)*0.03, bullet.TypeMiddle, 3)
				}
				if frame%50 == 0 && frame > 60 {
					ctx.Emitter.Aimed(ctx.Player.X, ctx.Player.Y, 5, 3.0, math.Pi/4, bullet.TypeRice, 2)
				}
			},
		},
		{
			Name: "豊符「オヲトシハーベスター」", HP: 700, TimeLimit: 3000, Bonus: 800000,
			Update: func(e *enemy.Enemy, frame int) {
				ctx.Emitter.SetPos(e.X, e.Y)
				if frame%3 == 0 {
					a := float64(frame) * 0.09
					ctx.Emitter.Spiral(4, 2.0, a, bullet.TypeSmall, 1)
					ctx.Emitter.Spiral(4, 2.0, -a, bullet.TypeSmall, 6)
				}
				if frame%40 == 0 {
					ctx.Emitter.Aimed(ctx.Player.X, ctx.Player.Y, 6+ctx.State.Difficulty, 3.5, math.Pi/3, bullet.TypeRice, 4)
				}
			},
		},
	}

	boss := enemy.NewBoss(208, -40, phases)
	boss.SpriteID = 1
	boss.DropPower = 30
	boss.DropPoint = 20
	boss.SetPath([]enemy.PathNode{
		{X: 208, Y: 100, Frames: 60},
	})
	ctx.AddEnemy(boss)
	s.boss = boss
}

func (s *Script1) BgDraw(field *ebiten.Image) {
	w := float32(field.Bounds().Dx())
	h := float32(field.Bounds().Dy())

	field.Fill(color.RGBA{16, 24, 48, 255})

	if s.bg1 != nil {
		render.TileBackground(field, s.bg1, s.bgY, 1)
	} else {
		offset := float32(math.Mod(s.bgY, 40))
		for y := offset - 40; y < h; y += 40 {
			vector.StrokeLine(field, 0, y, w, y, 0.5, color.RGBA{40, 50, 80, 100}, false)
		}
		vector.StrokeLine(field, w*0.2, 0, w*0.2, h, 1, color.RGBA{60, 70, 100, 150}, false)
		vector.StrokeLine(field, w*0.8, 0, w*0.8, h, 1, color.RGBA{60, 70, 100, 150}, false)
	}

	render.TileBackground(field, s.bg2, s.bgY*0.6, 0.3)
}

func (s *Script1) Finished() bool { return s.finished }

var _ stage.Script = (*Script1)(nil)
