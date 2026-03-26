package extra

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

// ScriptExtra Extra面脚本
// 中Boss: 八坂神奈子
// Boss: 洩矢諏訪子
type ScriptExtra struct {
	frame          int
	finished       bool
	bgY            float64
	midBossSpawned bool
	midBoss        *enemy.Enemy
	bossSpawned    bool
	boss           *enemy.Enemy
	bg1, bg2       *ebiten.Image // stg7bg.png [256x256], stg7bg2.png [256x256]
	bg3            *ebiten.Image // stg7bg3.png [32x256]
}

func NewScript() *ScriptExtra {
	s := &ScriptExtra{}
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

func (s *ScriptExtra) loadBg() {
	s.bg1 = loadImage("anm/background/stg7bg.png")
	s.bg2 = loadImage("anm/background/stg7bg2.png")
	s.bg3 = loadImage("anm/background/stg7bg3.png")
}

func (s *ScriptExtra) Init(ctx *stage.Context) {
	ctx.Audio.PlayBGM(audio.BGMExtra)
}

func (s *ScriptExtra) Update(ctx *stage.Context) {
	s.frame++
	s.bgY += 2.0

	// ---- 道中第1波: 高速妖精群 ----
	if s.frame >= 60 && s.frame <= 360 && s.frame%10 == 0 {
		x := 40 + float64((s.frame/10)%10)*34
		e := enemy.New(x, -16, 60, enemy.TypeFairy)
		e.SetPath([]enemy.PathNode{
			{X: x, Y: 80, Frames: 15},
			{X: x + 30, Y: 160, Frames: 30},
			{X: x, Y: 500, Frames: 40},
		})
		e.DropPower = 2
		e.DropPoint = 1
		ctx.AddEnemy(e)
	}

	// ---- 道中第2波: 左右包夹 ----
	if s.frame >= 420 && s.frame <= 720 && s.frame%8 == 0 {
		for _, side := range []float64{20, 396} {
			e := enemy.New(side, -16, 55, enemy.TypeFairy)
			e.SetPath([]enemy.PathNode{
				{X: 208, Y: 100, Frames: 20},
				{X: 208, Y: 100, Frames: 40},
				{X: side, Y: 500, Frames: 30},
			})
			e.DropPower = 2
			ctx.AddEnemy(e)
		}
	}

	// ---- 道中第3波: 大量密集 ----
	if s.frame >= 780 && s.frame <= 1140 && s.frame%6 == 0 {
		x := 30 + float64((s.frame-780)/6%12)*30
		e := enemy.New(x, -16, 70, enemy.TypeFairy)
		e.SetPath([]enemy.PathNode{
			{X: x, Y: 110, Frames: 18},
			{X: x, Y: 110, Frames: 60},
			{X: x, Y: 500, Frames: 30},
		})
		e.DropPower = 2
		e.DropPoint = 1
		ctx.AddEnemy(e)
	}

	// 杂兵弹幕（Extra级密度）
	for _, e := range ctx.Enemies {
		if !e.Active || e.Type != enemy.TypeFairy {
			continue
		}
		if e.Age > 15 && e.Age%8 == 0 {
			ctx.Emitter.SetPos(e.X, e.Y)
			count := 14 + ctx.State.Difficulty*4
			ctx.Emitter.Ring(count, 3.5, float64(e.Age)*0.18, bullet.TypeSmall, 5)
		}
	}

	// ======== 中Boss: 東風谷早苗 ========

	if s.frame == 1200 {
		ctx.StartDialog([]dialog.Line{
			{Speaker: "灵梦", Text: "又来了？你不是已经输了吗。", IsRight: false},
			{Speaker: "八坂神奈子", Text: "哼，那只是表面上的让步。", IsRight: true},
			{Speaker: "八坂神奈子", Text: "让你见识一下真正的神力！", IsRight: true},
			{Speaker: "灵梦", Text: "好吧，速战速决。", IsRight: false},
		})
	}

	if s.frame > 1200 && !s.midBossSpawned {
		s.spawnMidBoss(ctx)
		s.midBossSpawned = true
	}

	if s.midBoss != nil && s.midBoss.Active && s.midBoss.Boss != nil && s.midBoss.Boss.Invincible {
		if s.midBoss.Boss.PhaseFrame > 2 {
			s.midBoss.EndInvincible()
		}
	}

	// ---- 中Boss后道中 ----
	midBossDead := s.midBossSpawned && s.midBoss != nil && !s.midBoss.Active
	if midBossDead && s.frame >= 1800 && s.frame <= 2400 && s.frame%6 == 0 {
		x := 30 + float64((s.frame-1800)/6%12)*30
		e := enemy.New(x, -16, 65, enemy.TypeFairy)
		e.SetPath([]enemy.PathNode{
			{X: x, Y: 100, Frames: 18},
			{X: x, Y: 100, Frames: 50},
			{X: x, Y: 500, Frames: 30},
		})
		e.DropPower = 2
		e.DropPoint = 1
		ctx.AddEnemy(e)
	}

	// ======== Boss: 洩矢諏訪子 ========

	if midBossDead && s.frame >= 2520 && !s.bossSpawned {
		ctx.StartDialog([]dialog.Line{
			{Speaker: "洩矢諏訪子", Text: "呀哈！你终于来了！", IsRight: true},
			{Speaker: "灵梦", Text: "你是……土著神？", IsRight: false},
			{Speaker: "洩矢諏訪子", Text: "我是洩矢諏訪子，这座山最古老的神明！", IsRight: true},
			{Speaker: "洩矢諏訪子", Text: "让你见识一下诹访大战的力量！", IsRight: true},
			{Speaker: "灵梦", Text: "不管多古老的神，挡路就得让开！", IsRight: false},
		})
		s.bossSpawned = true
	}

	if s.bossSpawned && s.boss == nil && s.frame > 2520 {
		s.spawnBoss(ctx)
	}

	if s.boss != nil && s.boss.Active && s.boss.Boss != nil && s.boss.Boss.Invincible {
		if s.boss.Boss.PhaseFrame > 2 {
			s.boss.EndInvincible()
		}
	}

	if s.boss != nil && !s.boss.Active {
		s.finished = true
	}

	if s.frame >= 9000 {
		s.finished = true
	}
}

func (s *ScriptExtra) spawnMidBoss(ctx *stage.Context) {
	phases := []enemy.SpellCard{
		{
			Name: "", HP: 600, TimeLimit: 1800, Bonus: 0,
			Update: func(e *enemy.Enemy, frame int) {
				ctx.Emitter.SetPos(e.X, e.Y)
				if frame%20 == 0 {
					count := 20 + ctx.State.Difficulty*6
					ctx.Emitter.Ring(count, 3.0, float64(frame)*0.07, bullet.TypeMiddle, 2)
				}
				if frame%35 == 0 {
					ctx.Emitter.Aimed(ctx.Player.X, ctx.Player.Y, 8+ctx.State.Difficulty*2, 4.0, math.Pi/4, bullet.TypeRice, 4)
				}
			},
		},
		{
			Name: "神符「水眼の如き美しき源泉」", HP: 800, TimeLimit: 2400, Bonus: 1000000,
			Update: func(e *enemy.Enemy, frame int) {
				ctx.Emitter.SetPos(e.X, e.Y)
				if frame%3 == 0 {
					a := float64(frame) * 0.09
					ctx.Emitter.Spiral(5, 2.5, a, bullet.TypeSmall, 6)
					ctx.Emitter.Spiral(5, 2.5, -a, bullet.TypeSmall, 3)
				}
				if frame%50 == 0 {
					ctx.Emitter.Ring(24+ctx.State.Difficulty*6, 1.8, float64(frame)*0.04, bullet.TypeMiddle, 7)
				}
			},
		},
	}

	boss := enemy.NewBoss(208, -40, phases)
	boss.SpriteID = 7
	boss.DropPower = 30
	boss.DropPoint = 20
	boss.SetPath([]enemy.PathNode{
		{X: 208, Y: 100, Frames: 60},
	})
	ctx.AddEnemy(boss)
	s.midBoss = boss
}

func (s *ScriptExtra) spawnBoss(ctx *stage.Context) {
	ctx.Audio.PlayBGM(audio.BGMExtraBoss)

	phases := []enemy.SpellCard{
		// 通常攻撃1
		{
			Name: "", HP: 1000, TimeLimit: 1800, Bonus: 0,
			Update: func(e *enemy.Enemy, frame int) {
				ctx.Emitter.SetPos(e.X, e.Y)
				if frame%12 == 0 {
					ctx.Emitter.Ring(24+ctx.State.Difficulty*6, 3.5, float64(frame)*0.09, bullet.TypeMiddle, 1)
				}
				if frame%20 == 0 {
					ctx.Emitter.Aimed(ctx.Player.X, ctx.Player.Y, 8+ctx.State.Difficulty*2, 4.5, math.Pi/4, bullet.TypeRice, 4)
				}
			},
		},
		// 符卡1: 開宴「二拝二拍一拝」
		{
			Name: "開宴「二拝二拍一拝」", HP: 1200, TimeLimit: 2400, Bonus: 2000000,
			Update: func(e *enemy.Enemy, frame int) {
				ctx.Emitter.SetPos(e.X, e.Y)
				if frame%3 == 0 {
					a := float64(frame) * 0.08
					ctx.Emitter.Spiral(6, 2.8, a, bullet.TypeSmall, 5)
				}
				if frame%40 == 0 {
					ctx.Emitter.Ring(30+ctx.State.Difficulty*8, 2.0, float64(frame)*0.03, bullet.TypeMiddle, 2)
				}
				if frame%55 == 0 {
					ctx.Emitter.Aimed(ctx.Player.X, ctx.Player.Y, 10, 3.5, math.Pi/3, bullet.TypeRice, 7)
				}
			},
		},
		// 通常攻撃2
		{
			Name: "", HP: 1000, TimeLimit: 1800, Bonus: 0,
			Update: func(e *enemy.Enemy, frame int) {
				ctx.Emitter.SetPos(e.X, e.Y)
				if frame%2 == 0 {
					a := float64(frame) * 0.11
					ctx.Emitter.Spiral(4, 3.0, a, bullet.TypeSmall, 3)
					ctx.Emitter.Spiral(4, 3.0, -a, bullet.TypeSmall, 6)
				}
				if frame%30 == 0 {
					ctx.Emitter.Aimed(ctx.Player.X, ctx.Player.Y, 10+ctx.State.Difficulty*2, 4.0, math.Pi/3, bullet.TypeRice, 1)
				}
			},
		},
		// 符卡2: 蛙狩「蛙は口ゆえ蛇に呑まるる」
		{
			Name: "蛙狩「蛙は口ゆえ蛇に呑まるる」", HP: 1500, TimeLimit: 3000, Bonus: 3000000,
			Update: func(e *enemy.Enemy, frame int) {
				ctx.Emitter.SetPos(e.X, e.Y)
				if frame%2 == 0 {
					a := float64(frame) * 0.1
					ctx.Emitter.Spiral(8, 2.5, a, bullet.TypeSmall, 5)
				}
				if frame%25 == 0 {
					ctx.Emitter.Ring(36+ctx.State.Difficulty*10, 2.0, float64(frame)*0.04, bullet.TypeMiddle, 7)
				}
				if frame%40 == 0 {
					ctx.Emitter.Aimed(ctx.Player.X, ctx.Player.Y, 12, 4.5, math.Pi/3, bullet.TypeRice, 2)
				}
			},
		},
		// 通常攻撃3
		{
			Name: "", HP: 1200, TimeLimit: 1800, Bonus: 0,
			Update: func(e *enemy.Enemy, frame int) {
				ctx.Emitter.SetPos(e.X, e.Y)
				if frame%2 == 0 {
					a := float64(frame) * 0.13
					ctx.Emitter.Spiral(6, 3.2, a, bullet.TypeSmall, 1)
				}
				if frame%15 == 0 {
					ctx.Emitter.Ring(28+ctx.State.Difficulty*8, 2.5, float64(frame)*0.06, bullet.TypeMiddle, 4)
				}
			},
		},
		// 符卡3: 「諏訪大戦 ～ 土着神話 vs 中央神話」
		{
			Name: "「諏訪大戦 ～ 土着神話 vs 中央神話」", HP: 2000, TimeLimit: 3600, Bonus: 5000000,
			Update: func(e *enemy.Enemy, frame int) {
				ctx.Emitter.SetPos(e.X, e.Y)
				// 最终符: 极高密度多层弹幕
				if frame%2 == 0 {
					a := float64(frame) * 0.12
					ctx.Emitter.Spiral(10, 2.8, a, bullet.TypeSmall, 7)
					ctx.Emitter.Spiral(10, 2.8, -a+math.Pi/5, bullet.TypeSmall, 3)
				}
				if frame%20 == 0 {
					ctx.Emitter.Ring(40+ctx.State.Difficulty*12, 2.0, float64(frame)*0.05, bullet.TypeMiddle, 5)
				}
				if frame%30 == 0 {
					ctx.Emitter.Aimed(ctx.Player.X, ctx.Player.Y, 14, 4.5, math.Pi/2, bullet.TypeRice, 2)
				}
			},
		},
	}

	boss := enemy.NewBoss(208, -40, phases)
	boss.SpriteID = 7
	boss.DropPower = 60
	boss.DropPoint = 50
	boss.SetPath([]enemy.PathNode{
		{X: 208, Y: 80, Frames: 60},
	})
	ctx.AddEnemy(boss)
	s.boss = boss
}

func (s *ScriptExtra) BgDraw(field *ebiten.Image) {
	w := float32(field.Bounds().Dx())
	h := float32(field.Bounds().Dy())

	// 山の裏 — 暗い土色+ミシャグジ紋様
	field.Fill(color.RGBA{28, 20, 16, 255})

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
		offset := float32(math.Mod(s.bgY, 60))
		for y := -offset; y < h; y += 60 {
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

func (s *ScriptExtra) Finished() bool { return s.finished }

var _ stage.Script = (*ScriptExtra)(nil)
