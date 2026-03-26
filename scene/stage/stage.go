package stage

import (
	"fmt"
	"image/color"
	"th10/audio"
	"th10/danmaku"
	"th10/entity/bullet"
	"th10/entity/effect"
	"th10/entity/enemy"
	"th10/entity/item"
	"th10/entity/player"
	"th10/entity/playerbullet"
	"th10/game"
	"th10/input"
	"th10/render"
	"th10/scene/dialog"
	"th10/sprite"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

// Script 关卡脚本接口，每关实现一个
type Script interface {
	Init(ctx *Context)
	Update(ctx *Context)
	BgDraw(field *ebiten.Image)
	Finished() bool
}

// Context 关卡运行时上下文，脚本通过它操作所有实体
type Context struct {
	State   *game.GameState
	Player  *player.Player
	Bullets *bullet.Pool  // 敌弹池
	Items   *item.Pool
	Effects *effect.Pool
	Enemies []*enemy.Enemy // 活跃敌人列表
	Emitter *danmaku.Emitter
	Audio   *audio.Manager // 音频管理器
	stage   *Stage // 反向引用（用于触发对话等）
}

// AddEnemy 脚本用：添加敌人
func (c *Context) AddEnemy(e *enemy.Enemy) {
	c.Enemies = append(c.Enemies, e)
}

// StartDialog 脚本用：触发对话覆盖层
func (c *Context) StartDialog(lines []dialog.Line) {
	c.stage.dialog = dialog.NewOverlay(lines)
}

// Stage 通用关卡场景，驱动一关的全部逻辑
type Stage struct {
	ctx      *Context
	script   Script
	renderer *render.Renderer
	field    *ebiten.Image      // 游戏区域离屏缓冲
	dialog   *dialog.Overlay    // 对话覆盖层（nil=无对话）
	NextScene func(g *game.Game, state *game.GameState) game.Scene // 通关后下一场景
}

const (
	fieldW = game.FieldRight - game.FieldLeft
	fieldH = game.FieldBottom - game.FieldTop
)

func New(g *game.Game, state *game.GameState, script Script) *Stage {
	bulletPool := bullet.NewPool(2048)
	s := &Stage{
		script:   script,
		renderer: render.New(),
		field:    ebiten.NewImage(fieldW, fieldH),
	}
	s.ctx = &Context{
		State:   state,
		Player:  player.New(state, game.FieldLeft, game.FieldTop, game.FieldRight, game.FieldBottom),
		Bullets: bulletPool,
		Items:   item.NewPool(256),
		Effects: effect.NewPool(512),
		Emitter: danmaku.NewEmitter(bulletPool),
		Audio:   g.Audio(),
		stage:   s,
	}
	script.Init(s.ctx)
	return s
}

func (s *Stage) Update(g *game.Game) error {
	in := &input.Global
	ctx := s.ctx

	// 暂停切换
	if in.JustPressed(input.KeyPause) {
		ctx.State.Paused = !ctx.State.Paused
		ctx.Audio.PlaySE(audio.SEPause)
	}
	if ctx.State.Paused {
		return nil
	}

	// 对话覆盖层：暂停实体更新，只推进对话
	if s.dialog != nil {
		if !s.dialog.Update(in.JustPressed(input.KeyOk)) {
			s.dialog = nil
		}
		return nil
	}

	ctx.State.Frame++

	// 信仰值自然衰减
	ctx.State.DecayFaith()

	// 1. 玩家
	ctx.Player.Update()

	// 2. 脚本驱动（敌人/弹幕生成）
	s.script.Update(ctx)

	// 3. 敌人
	for _, e := range ctx.Enemies {
		e.Update()
	}

	// 4. 敌弹
	ctx.Bullets.Update(game.FieldLeft, game.FieldTop, game.FieldRight, game.FieldBottom)

	// 5. 道具
	focused := in.IsPressed(input.KeyFocus)
	ctx.Items.Update(ctx.Player.X, ctx.Player.Y, focused, game.FieldBottom)

	// 6. 特效
	ctx.Effects.Update()

	// 7. 碰撞
	s.checkCollisions()

	// 8. 清理
	s.cleanEnemies()

	// 9. Game Over
	if ctx.State.Life < 0 {
		g.GoTo(nil) // TODO: → Result scene
	}
	if s.script.Finished() {
		if s.NextScene != nil {
			g.GoTo(s.NextScene(g, ctx.State))
		} else {
			g.GoTo(nil) // TODO: → Result/Ending
		}
	}

	return nil
}

// ---------- 碰撞检测 ----------

func (s *Stage) checkCollisions() {
	ctx := s.ctx
	p := ctx.Player
	st := ctx.State
	px, py, pr := p.X, p.Y, p.HitboxRadius

	// 自机子弹 vs 敌人
	for _, e := range ctx.Enemies {
		if !e.Active {
			continue
		}
		ex, ey, ew, eh := e.HitRect()
		dmg := p.Bullets.HitEnemy(ex, ey, ew, eh)
		if dmg > 0 {
			killed := e.TakeDamage(dmg)
			if killed {
				for i := 0; i < e.DropPower; i++ {
					ctx.Items.Spawn(e.X, e.Y, item.PowerItem)
				}
				for i := 0; i < e.DropPoint; i++ {
					ctx.Items.Spawn(e.X, e.Y, item.PointItem)
				}
				ctx.Effects.Spawn(e.X, e.Y, effect.TypeExplosion, 20)
				st.AddScore(1000)
			}
		}
	}

	// 敌弹 vs 自机
	if !p.IsInvincible() {
		ctx.Bullets.Each(func(b *bullet.Bullet) {
			if b.HitPlayer(px, py, pr) {
				p.Hit()
				b.Active = false
				ctx.Effects.Spawn(px, py, effect.TypePlayerDead, 30)
				ctx.Audio.PlaySE(audio.SEDead)
			}
		})
	}

	// 擦弹（GrazePlayer 内部保证每颗子弹只计一次）
	if p.State == player.StateNormal || p.State == player.StateRespawn {
		grazed := false
		ctx.Bullets.Each(func(b *bullet.Bullet) {
			if b.Age > 4 && b.GrazePlayer(px, py, p.GrazeRadius) {
				st.Graze++
				st.AddScore(100)
				grazed = true
			}
		})
		if grazed {
			ctx.Audio.PlaySE(audio.SEGraze)
		}
	}

	// 道具拾取
	ctx.Items.CollectCallback(px, py, func(t item.ItemType) {
		ctx.Audio.PlaySE(audio.SEItem)
		switch t {
		case item.PowerItem:
			p.AddPower(1)
		case item.BigPower:
			p.AddPower(8)
		case item.PointItem:
			st.Point++
			st.AddScore(500)
		case item.LifeFragment:
			st.Life++
		case item.BombFragment:
			st.Bomb++
		case item.FullPower:
			st.Power = 500
		}
	})

	// Bomb 清屏
	if p.State == player.StateBomb {
		ctx.Bullets.Each(func(b *bullet.Bullet) {
			ctx.Effects.Spawn(b.X, b.Y, effect.TypeBulletCancel, 12)
			b.Active = false
			ctx.Items.Spawn(b.X, b.Y, item.PointItem)
		})
	}
}

func (s *Stage) cleanEnemies() {
	n := 0
	for _, e := range s.ctx.Enemies {
		if e.Active && e.InBounds(game.FieldLeft, game.FieldTop, game.FieldRight, game.FieldBottom) {
			s.ctx.Enemies[n] = e
			n++
		}
	}
	for i := n; i < len(s.ctx.Enemies); i++ {
		s.ctx.Enemies[i] = nil
	}
	s.ctx.Enemies = s.ctx.Enemies[:n]
}

// ---------- 绘制 ----------

func (s *Stage) Draw(screen *ebiten.Image) {
	ctx := s.ctx
	s.field.Clear()

	// 1. 背景
	s.script.BgDraw(s.field)

	// 2. 实体（用几何占位图形）
	s.drawItems(s.field)
	s.drawPlayerBullets(s.field)
	s.drawEnemies(s.field)
	s.drawBullets(s.field)
	s.drawPlayer(s.field)
	s.drawEffects(s.field)

	// 3. Field → Screen
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(game.FieldLeft, game.FieldTop)
	screen.DrawImage(s.field, op)

	// 4. HUD
	s.drawHUD(screen)

	// 5. 暂停遮罩
	if ctx.State.Paused {
		vector.FillRect(screen, 0, 0, game.ScreenWidth, game.ScreenHeight,
			color.RGBA{0, 0, 0, 128}, false)
		ebitenutil.DebugPrintAt(screen, "PAUSED  (Esc to resume)", game.ScreenWidth/2-80, game.ScreenHeight/2)
	}

	// 6. 对话覆盖层
	if s.dialog != nil {
		s.dialog.Draw(screen)
	}
}

// 以下绘制全部用几何占位，后续替换为贴图

func (s *Stage) drawItems(field *ebiten.Image) {
	fl, ft := float64(game.FieldLeft), float64(game.FieldTop)
	s.ctx.Items.Each(func(it *item.Item) {
		x := float32(it.X - fl)
		y := float32(it.Y - ft)
		var c color.RGBA
		switch it.Type {
		case item.PowerItem:
			c = color.RGBA{255, 64, 64, 255}
		case item.PointItem:
			c = color.RGBA{64, 128, 255, 255}
		default:
			c = color.RGBA{255, 255, 64, 255}
		}
		vector.FillRect(field, x-4, y-4, 8, 8, c, false)
	})
}

func (s *Stage) drawPlayerBullets(field *ebiten.Image) {
	fl, ft := float64(game.FieldLeft), float64(game.FieldTop)
	s.ctx.Player.Bullets.Each(func(b *playerbullet.Bullet) {
		x := float32(b.X - fl)
		y := float32(b.Y - ft)
		vector.FillRect(field, x-3, y-6, 6, 12, color.RGBA{255, 255, 255, 230}, false)
	})
}

func (s *Stage) drawEnemies(field *ebiten.Image) {
	fl, ft := float64(game.FieldLeft), float64(game.FieldTop)
	res := sprite.Global
	for _, e := range s.ctx.Enemies {
		if !e.Active {
			continue
		}
		x := e.X - fl
		y := e.Y - ft

		// sprite 绘制
		if res != nil && e.Type == enemy.TypeFairy {
			// 用 Age 做简单动画 (正面前4帧循环)
			frame := res.Enemies.FairyFrame(0, (e.Age/8)%4)
			if frame != nil {
				fw := float64(frame.Bounds().Dx())
				fh := float64(frame.Bounds().Dy())
				op := &ebiten.DrawImageOptions{}
				op.GeoM.Translate(-fw/2, -fh/2)
				op.GeoM.Translate(x, y)
				field.DrawImage(frame, op)
				continue
			}
		}

		// fallback / boss
		if res != nil && (e.Type == enemy.TypeBoss || e.Type == enemy.TypeMidBoss) && e.SpriteID > 0 {
			if bossFrame, ok := res.Bosses[e.SpriteID]; ok && bossFrame != nil {
				op := &ebiten.DrawImageOptions{}
				op.GeoM.Translate(-32, -32) // center 64x64 sprite
				op.GeoM.Translate(x, y)
				field.DrawImage(bossFrame, op)
				continue
			}
		}

		// geometry fallback (red rect)
		hw := float32(e.HitWidth / 2)
		hh := float32(e.HitHeight / 2)
		vector.FillRect(field, float32(x)-hw, float32(y)-hh, hw*2, hh*2, color.RGBA{255, 0, 0, 200}, false)
	}
}

func (s *Stage) drawBullets(field *ebiten.Image) {
	fl, ft := float64(game.FieldLeft), float64(game.FieldTop)
	res := sprite.Global
	s.ctx.Bullets.Each(func(b *bullet.Bullet) {
		x := b.X - fl
		y := b.Y - ft

		// sprite 绘制
		if res != nil {
			frame := res.Bullets.Frame(int(b.Type), b.Color)
			if frame != nil {
				fw := float64(frame.Bounds().Dx())
				fh := float64(frame.Bounds().Dy())
				op := &ebiten.DrawImageOptions{}
				op.GeoM.Translate(-fw/2, -fh/2)
				op.GeoM.Rotate(b.Angle + 1.5707963) // 弹头朝运动方向（+π/2 修正）
				op.GeoM.Translate(x, y)
				if b.Additive {
					op.Blend = ebiten.BlendLighter
				}
				field.DrawImage(frame, op)
				return
			}
		}

		// fallback
		r := float32(b.Radius)
		if r < 2 {
			r = 2
		}
		vector.FillCircle(field, float32(x), float32(y), r*2, color.RGBA{255, 255, 0, 220}, false)
	})
}

func (s *Stage) drawPlayer(field *ebiten.Image) {
	p := s.ctx.Player
	if p.State == player.StateDead {
		return
	}
	fl, ft := float64(game.FieldLeft), float64(game.FieldTop)
	px := p.X - fl
	py := p.Y - ft

	alpha := float32(1.0)
	if p.State == player.StateRespawn && p.InvincTimer%6 < 3 {
		alpha = 0.3
	}

	// sprite 绘制
	res := sprite.Global
	if res != nil {
		ps := res.Player[s.ctx.State.Character]
		// 映射 AnimState → sprite state
		var state int
		switch p.AnimState {
		case player.AnimFront:
			state = 0
		case player.AnimToSide:
			if p.FaceRight {
				state = 3
			} else {
				state = 1
			}
		case player.AnimSide:
			if p.FaceRight {
				state = 4
			} else {
				state = 2
			}
		}
		frame := ps.Frame(state, p.AnimFrame)
		if frame != nil {
			fw := float64(frame.Bounds().Dx())
			fh := float64(frame.Bounds().Dy())
			op := &ebiten.DrawImageOptions{}
			op.GeoM.Translate(-fw/2, -fh/2)
			op.GeoM.Translate(px, py)
			op.ColorScale.ScaleAlpha(alpha)
			field.DrawImage(frame, op)
			// 判定点（集中时显示）
			vector.FillCircle(field, float32(px), float32(py), 3, color.RGBA{255, 255, 255, uint8(alpha * 255)}, false)
			return
		}
	}

	// fallback: 几何占位
	vector.FillRect(field, float32(px)-8, float32(py)-12, 16, 24, color.RGBA{0, 220, 0, uint8(alpha * 255)}, false)
	vector.FillCircle(field, float32(px), float32(py), 3, color.RGBA{255, 255, 255, 255}, false)
}

func (s *Stage) drawEffects(field *ebiten.Image) {
	fl, ft := float64(game.FieldLeft), float64(game.FieldTop)
	s.ctx.Effects.Each(func(e *effect.Effect) {
		x := float32(e.X - fl)
		y := float32(e.Y - ft)
		r := float32(e.Scale * 8)
		a := uint8(e.Alpha * 255)
		vector.FillCircle(field, x, y, r, color.RGBA{255, 200, 100, a}, false)
	})
}

func (s *Stage) drawHUD(screen *ebiten.Image) {
	st := s.ctx.State

	// 侧边栏背景
	vector.FillRect(screen, game.FieldRight+4, 0,
		game.ScreenWidth-game.FieldRight-4, game.ScreenHeight,
		color.RGBA{20, 20, 40, 255}, false)
	vector.FillRect(screen, 0, 0, game.FieldLeft, game.ScreenHeight,
		color.RGBA{20, 20, 40, 255}, false)
	// 边框
	vector.StrokeRect(screen,
		game.FieldLeft-1, game.FieldTop-1,
		fieldW+2, fieldH+2,
		1, color.RGBA{128, 128, 200, 255}, false)

	x := game.FieldRight + 16
	ebitenutil.DebugPrintAt(screen, fmt.Sprintf("HiScore %012d", st.HiScore), x, 20)
	ebitenutil.DebugPrintAt(screen, fmt.Sprintf("Score   %012d", st.Score), x, 40)

	lifeStr := ""
	for i := 0; i < st.Life; i++ {
		lifeStr += "★"
	}
	ebitenutil.DebugPrintAt(screen, "Life  "+lifeStr, x, 70)

	bombStr := ""
	for i := 0; i < st.Bomb; i++ {
		bombStr += "★"
	}
	ebitenutil.DebugPrintAt(screen, "Bomb  "+bombStr, x, 90)

	ebitenutil.DebugPrintAt(screen, fmt.Sprintf("Power %d.%02d/%d.%02d", st.Power/100, st.Power%100, st.MaxPower/100, st.MaxPower%100), x, 120)
	ebitenutil.DebugPrintAt(screen, fmt.Sprintf("Graze %d", st.Graze), x, 140)
	ebitenutil.DebugPrintAt(screen, fmt.Sprintf("Point %d", st.Point), x, 160)
	ebitenutil.DebugPrintAt(screen, fmt.Sprintf("Faith %d", st.Faith), x, 180)

	ebitenutil.DebugPrintAt(screen, fmt.Sprintf("%.0f FPS", ebiten.ActualFPS()), x, 220)
	ebitenutil.DebugPrintAt(screen, fmt.Sprintf("Bullets: %d", s.ctx.Bullets.ActiveCount()), x, 240)
	ebitenutil.DebugPrintAt(screen, fmt.Sprintf("Enemies: %d", len(s.ctx.Enemies)), x, 260)
}

// 确保 Stage 实现 game.Scene
var _ game.Scene = (*Stage)(nil)
