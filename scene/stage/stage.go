package stage

import (
	"fmt"
	"image/color"
	"math"
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
	resultscene "th10/scene/result"
	"th10/sprite"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	ebitext "github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
	"github.com/yohamta/ganim8/v2"
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
	Bullets *bullet.Pool // 敌弹池
	Items   *item.Pool
	Effects *effect.Pool
	Enemies []*enemy.Enemy // 活跃敌人列表
	Emitter *danmaku.Emitter
	Audio   Sound // 音频管理器；无头模拟（sim 包）保持 nil
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

// BossHUDInfo 顶部 Boss 血条/符卡 HUD 数据。脚本可选实现 BossHUD() 提供。
type BossHUDInfo struct {
	Active    bool
	HPFrac    float64
	HPMarkers []float64
	Spell     bool
	Name      string
	TimeLeft  int
	Bonus     int64
}

// DialogActive 报告对话覆盖层是否正在显示（ECL VM 的 339 用它阻塞 main 任务）。
func (c *Context) DialogActive() bool {
	return c.stage.dialog != nil
}

// DialogQueue 为 ECL 驱动的关卡桥接对话：ECL 的每个对话事件 (339) 依次消费
// 一段预设台词。ECL 的对话 id 在各关并不一致，故按出现顺序消费而非按 id 索引。
type DialogQueue struct {
	ctx   *Context
	lines [][]dialog.Line
	next  int
}

// NewDialogQueue 按顺序提供若干段对话（如 bossPre, post）。空段会被跳过。
func NewDialogQueue(ctx *Context, segments ...[]dialog.Line) *DialogQueue {
	return &DialogQueue{ctx: ctx, lines: segments}
}

// Open 由 VM 在对话事件触发时调用，弹出下一段非空台词；无更多台词返回 false。
func (q *DialogQueue) Open(_ int32) bool {
	for q.next < len(q.lines) {
		seg := q.lines[q.next]
		q.next++
		if len(seg) > 0 {
			q.ctx.StartDialog(seg)
			return true
		}
	}
	return false
}

func (c *Context) PlayerName() string {
	if c.State != nil && c.State.Character == game.CharMarisa {
		return "魔理沙"
	}
	return "灵梦"
}

func (c *Context) PlayerLine(text string) dialog.Line {
	return dialog.Line{Speaker: c.PlayerName(), Text: text, IsRight: false}
}

// Stage 通用关卡场景，驱动一关的全部逻辑
type Stage struct {
	ctx          *Context
	script       Script
	field        *ebiten.Image // 游戏区域离屏缓冲
	frontFrame   *ebiten.Image
	hud          *hud                                                 // 状态栏（ebitenui 排版）
	dialog       *dialog.Overlay                                      // 对话覆盖层（nil=无对话）
	wasBombing   bool                                                 // 上一帧是否处于 Bomb 态（检测起爆沿）
	bossNameFace ebitext.Face                                         // 符卡名字体（系统黑体，懒加载）
	bossNumFace  ebitext.Face                                         // 符卡时限数字字体（HUD 字体，懒加载）
	NextScene    func(g *game.Game, state *game.GameState) game.Scene // 通关后下一场景
	OnGameOver   func(g *game.Game, state *game.GameState) game.Scene
}

const (
	fieldW = game.FieldRight - game.FieldLeft
	fieldH = game.FieldBottom - game.FieldTop
)

func New(g *game.Game, state *game.GameState, script Script) *Stage {
	s := newStage(state, script)
	s.ctx.Audio = g.Audio()
	script.Init(s.ctx)
	return s
}

// NewHeadless 构造不接音频的 Stage，供 sim 包的逐帧差分模拟使用。
// script.Init 在 Audio 为 nil 时运行，所有音效/BGM 调用静默。
func NewHeadless(state *game.GameState, script Script) *Stage {
	s := newStage(state, script)
	script.Init(s.ctx)
	return s
}

func newStage(state *game.GameState, script Script) *Stage {
	bulletPool := bullet.NewPool(2048)
	s := &Stage{
		script:     script,
		field:      ebiten.NewImage(fieldW, fieldH),
		frontFrame: loadFrontFrame(),
	}
	s.ctx = &Context{
		State:   state,
		Player:  player.New(state, game.FieldLeft, game.FieldTop, game.FieldRight, game.FieldBottom),
		Bullets: bulletPool,
		Items:   item.NewPool(256),
		Effects: effect.NewPool(512),
		Emitter: danmaku.NewEmitter(bulletPool),
		stage:   s,
	}
	s.hud = newHud(state)
	return s
}

func loadFrontFrame() *ebiten.Image {
	return render.LoadImage("anm/front/front00s.png")
}

// UpdateHeadless 用调用方提供的输入快照推进一帧，不读全局键盘、不刷新 HUD。
// 供 sim 包的逐帧差分模拟使用；真实游戏走 Update。
func (s *Stage) UpdateHeadless(in *input.State) {
	s.update(in)
}

// FinishedHeadless 报告脚本是否已结束（无头模拟的终止条件）。
func (s *Stage) FinishedHeadless() bool { return s.script.Finished() }

// ExportContext 暴露内部 Context 给 sim 包（无头模拟需要直接读写实体池）。
func ExportContext(s *Stage) *Context { return s.ctx }

func (s *Stage) Update(g *game.Game) error {
	in := &input.Global

	// HUD 每帧同步（含 paused/dialog 时也保持，避免暂停画面 HUD 卡在切换瞬间）
	s.hud.refresh()

	// 暂停切换
	if in.JustPressed(input.KeyPause) {
		s.ctx.State.Paused = !s.ctx.State.Paused
		s.ctx.PlaySE(audio.SEPause)
	}
	if s.ctx.State.Paused {
		return nil
	}

	if err := s.update(in); err != nil {
		return err
	}

	// 关卡结束/Game Over 时切换场景
	ctx := s.ctx
	if ctx.State.Life < 0 {
		g.MarkGameOver(ctx.State)
		if s.OnGameOver != nil {
			g.GoTo(s.OnGameOver(g, ctx.State))
		} else {
			g.GoTo(resultscene.New(ctx.State, resultscene.ModeGameOver, nil))
		}
		return nil
	}
	if s.script.Finished() {
		if s.NextScene != nil {
			g.GoTo(s.NextScene(g, ctx.State))
		} else {
			g.GoTo(resultscene.New(ctx.State, resultscene.ModeStageClear, nil))
		}
	}
	return nil
}

// update 推进一帧世界状态。不涉及场景切换，供 Update 与无头模拟共用。
func (s *Stage) update(in *input.State) error {
	ctx := s.ctx

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
	s.updatePlayer(in)

	// 1b. Bomb 起爆沿：进入 Bomb 态的第一帧播放符卡音效、信仰惩罚、屏幕全屏闪光
	bombing := ctx.Player.State == player.StateBomb
	if bombing && !s.wasBombing {
		ctx.PlaySE(audio.SEBomb)
		ctx.State.AddFaith(-15000) // 使用符卡折损信仰倍率
		ctx.Effects.Spawn(ctx.Player.X, ctx.Player.Y, effect.TypeExplosion, 40)
	}
	s.wasBombing = bombing

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

	return nil
}

// updatePlayer keeps confirmed deaths separate from the deathbomb window.
func (s *Stage) updatePlayer(in *input.State) {
	p := s.ctx.Player
	x, y, life := p.X, p.Y, p.GS.Life
	p.SetHomingCandidates(s.ctx.Enemies)
	p.UpdateInput(in)
	if p.GS.Life < life {
		// FUN_00425730 emits 4 small P + 3 large P. The original delayed
		// radial ejection and respawn animation are not yet reproduced.
		for i := 0; i < 7; i++ {
			kind := item.PowerItem
			if i%2 == 1 {
				kind = item.BigPower
			}
			s.ctx.Items.Spawn(x, y, kind)
		}
	}
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
			prevHP := e.HP
			killed := e.TakeDamage(dmg)
			if e.HP < prevHP {
				st.AddFaith(int64(dmg))
				st.AddScore(int64(dmg) * 12)
			}
			if killed {
				s.defeatEnemy(e)
			}
		}
	}

	// 敌弹 vs 自机
	if !p.IsInvincible() {
		ctx.Bullets.Each(func(b *bullet.Bullet) {
			if b.HitPlayer(px, py, pr) {
				p.Hit()
				b.Active = false
				st.AddFaith(-8000)
				ctx.Effects.Spawn(px, py, effect.TypePlayerDead, 30)
				ctx.PlaySE(audio.SEDead)
			}
		})
	}

	// 擦弹（GrazePlayer 内部保证每颗子弹只计一次）
	if p.State == player.StateNormal || p.State == player.StateRespawn {
		grazed := false
		ctx.Bullets.Each(func(b *bullet.Bullet) {
			if b.Age > 4 && b.GrazePlayer(px, py, p.GrazeRadius) {
				st.Graze++
				st.AddFaith(3)
				st.AddScore(100)
				grazed = true
			}
		})
		if grazed {
			ctx.PlaySE(audio.SEGraze)
		}
	}

	// 道具拾取
	ctx.Items.CollectCallback(px, py, func(t item.ItemType) {
		ctx.PlaySE(audio.SEItem)
		switch t {
		case item.PowerItem:
			p.AddPower(game.SmallPowerValue)
			st.AddFaith(8)
		case item.BigPower:
			p.AddPower(game.BigPowerValue)
			st.AddFaith(20)
		case item.PointItem:
			st.Point++
			st.AddFaith(15)
			st.AddScore(500)
		case item.LifeFragment:
			st.Life++
			st.AddFaith(100)
		case item.BigPoint:
			st.Point++
			st.AddScoreRaw(st.Faith)
		case item.FullPower:
			st.Power = st.MaxPower
			st.AddFaith(50)
		}
	})

	// Bomb 清屏
	if p.State == player.StateBomb {
		ctx.Bullets.Each(func(b *bullet.Bullet) {
			ctx.Effects.Spawn(b.X, b.Y, effect.TypeBulletCancel, 12)
			b.Active = false
			ctx.Items.Spawn(b.X, b.Y, item.PointItem)
		})
		s.applyBombDamage()
	}
}

func (s *Stage) defeatEnemy(e *enemy.Enemy) {
	ctx := s.ctx
	st := ctx.State
	totalDrops := e.DropPower + e.DropPoint + e.DropLife + e.DropBigPower + e.DropBigPoint
	dropIndex := 0
	spawnDrop := func(t item.ItemType) {
		x, y := defeatDropPosition(e, dropIndex, totalDrops)
		ctx.Items.Spawn(x, y, t)
		dropIndex++
	}
	for i := 0; i < e.DropPower; i++ {
		drop := item.PowerItem
		if st.Power >= st.MaxPower {
			drop = item.PointItem
		}
		spawnDrop(drop)
	}
	for i := 0; i < e.DropPoint; i++ {
		spawnDrop(item.PointItem)
	}
	for i := 0; i < e.DropLife; i++ {
		spawnDrop(item.LifeFragment)
	}
	for i := 0; i < e.DropBigPower; i++ {
		spawnDrop(item.BigPower)
	}
	for i := 0; i < e.DropBigPoint; i++ {
		spawnDrop(item.BigPoint)
	}
	if e.Type != enemy.TypeFairy {
		ctx.Items.Spawn(e.X, e.Y, item.BigPower)
	}
	ctx.Effects.Spawn(e.X, e.Y, effect.TypeExplosion, 20)
	st.AddFaith(int64(50 + e.DropPower*12 + e.DropPoint*8))
	if e.Type != enemy.TypeFairy {
		st.AddFaith(200)
	}
	st.AddScore(1000 + int64(e.DropPower+e.DropPoint)*100)
}

func defeatDropPosition(e *enemy.Enemy, index, total int) (float64, float64) {
	if e == nil {
		return 0, 0
	}
	if total <= 1 || (e.DropAreaW == 0 && e.DropAreaH == 0) {
		return e.X, e.Y
	}
	const goldenAngle = 2.399963229728653
	r := math.Sqrt((float64(index) + 0.5) / float64(total))
	a := float64(index) * goldenAngle
	return e.X + math.Cos(a)*e.DropAreaW*0.5*r, e.Y + math.Sin(a)*e.DropAreaH*0.5*r
}

func (s *Stage) applyBombDamage() {
	ctx := s.ctx
	st := ctx.State
	p := ctx.Player
	if p.BombTimer%10 != 0 {
		return
	}
	damage := 40 + st.Power/20
	for _, e := range ctx.Enemies {
		if !e.Active {
			continue
		}
		prevHP := e.HP
		killed := e.TakeDamage(damage)
		if e.HP < prevHP {
			st.AddScore(int64(damage) * 8)
			ctx.Effects.Spawn(e.X, e.Y, effect.TypeBulletCancel, 10)
		}
		if killed {
			s.defeatEnemy(e)
		}
	}
}

func (s *Stage) cleanEnemies() {
	n := 0
	for _, e := range s.ctx.Enemies {
		if e.Active && e.InBounds(game.FieldLeft, game.FieldTop, game.FieldRight, game.FieldBottom) {
			s.ctx.Enemies[n] = e
			n++
		} else {
			e.Active = false // release sticky homing targets when an enemy leaves the stage
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
	screen.Fill(color.Black)
	s.field.Clear()

	// 1. 背景
	s.script.BgDraw(s.field)

	// 2. 实体（用几何占位图形）
	s.drawItems(s.field)
	s.drawPlayerBullets(s.field)
	s.drawEnemies(s.field)
	s.drawBullets(s.field)
	s.drawPlayerOptions(s.field)
	s.drawPlayer(s.field)
	s.drawEffects(s.field)

	// Bomb 全屏闪光：起爆后渐弱的白色叠层（叠在 field 上，跟随震动）
	s.drawBombFlash(s.field)

	// Boss 血条/符卡 HUD（顶部横条），叠在 field 顶部
	s.drawBossHUD(s.field)

	// 3. Field → Screen（ECL ins_337 屏幕震动：脚本可选实现 ShakeOffset）
	op := &ebiten.DrawImageOptions{}
	sx, sy := 0.0, 0.0
	if sh, ok := s.script.(interface{ ShakeOffset() (float64, float64) }); ok {
		sx, sy = sh.ShakeOffset()
	}
	op.GeoM.Translate(game.FieldLeft+sx, game.FieldTop+sy)
	screen.DrawImage(s.field, op)

	// 4. 游戏框体/HUD 底板
	if s.frontFrame != nil {
		screen.DrawImage(s.frontFrame, nil)
	}

	// 5. HUD 文本
	s.hud.draw(screen)

	// 6. 暂停遮罩
	if ctx.State.Paused {
		vector.FillRect(screen, 0, 0, game.ScreenWidth, game.ScreenHeight,
			color.RGBA{0, 0, 0, 128}, false)
		ebitenutil.DebugPrintAt(screen, "PAUSED  (Esc to resume)", game.ScreenWidth/2-80, game.ScreenHeight/2)
	}

	// 7. 对话覆盖层
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
		if b.ScriptID >= 0 && sprite.Global != nil && sprite.Global.ReimuAnm != nil {
			if img := sprite.Global.ReimuAnm.ScriptSprite(b.ScriptID, max(b.Age-1, 0)); img != nil {
				op := &ebiten.DrawImageOptions{}
				op.GeoM.Translate(-float64(img.Bounds().Dx())/2, -float64(img.Bounds().Dy())/2)
				op.GeoM.Rotate(b.Angle)
				op.GeoM.Translate(float64(x), float64(y))
				op.ColorScale.ScaleAlpha(128.0 / 255) // pl00.anm scripts 5/7
				field.DrawImage(img, op)
				return
			}
		}
		vector.FillRect(field, x-3, y-6, 6, 12, color.RGBA{255, 255, 255, 230}, false)
	})
}

func (s *Stage) drawPlayerOptions(field *ebiten.Image) {
	for _, option := range s.ctx.Player.Options() {
		x, y := option.X-game.FieldLeft, option.Y-game.FieldTop
		if sprite.Global != nil && sprite.Global.ReimuAnm != nil {
			// Script 17's sprite; its spawn/interrupt transforms await the ANM VM.
			if img := sprite.Global.ReimuAnm.ScriptSprite(17, 0); img != nil {
				op := &ebiten.DrawImageOptions{}
				op.GeoM.Translate(x-float64(img.Bounds().Dx())/2, y-float64(img.Bounds().Dy())/2)
				field.DrawImage(img, op)
				continue
			}
		}
		vector.FillCircle(field, float32(x), float32(y), 5, color.RGBA{255, 128, 160, 255}, false)
	}
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
			// 优先用真实 enemy.anm 脚本（ECL anmSetMain 设定）解析当前帧，
			// 失败再回退到旧的 FairyKind 网格。
			var frame *ebiten.Image
			if e.MainScript >= 0 && res.EnemyAnm != nil {
				frame = res.EnemyAnm.ScriptSprite(e.MainScript, e.Age)
			}
			if frame == nil {
				frame = res.Enemies.FairyFrame(e.FairyKind, (e.Age/8)%4)
			}
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

		// 中/大 boss：优先用真实 stgenm01.anm 脚本（ECL anmSetMain 设定），
		// 失败再回退到旧的 Bosses 贴图表。
		if res != nil && (e.Type == enemy.TypeBoss || e.Type == enemy.TypeMidBoss) {
			if e.MainScript >= 0 && res.StageEnmAnm != nil {
				if frame := res.StageEnmAnm.ScriptSprite(e.MainScript, e.Age); frame != nil {
					fw := float64(frame.Bounds().Dx())
					fh := float64(frame.Bounds().Dy())
					op := &ebiten.DrawImageOptions{}
					op.GeoM.Translate(-fw/2, -fh/2)
					op.GeoM.Translate(x, y)
					field.DrawImage(frame, op)
					continue
				}
			}
			if e.SpriteID > 0 {
				if bossFrame, ok := res.Bosses[e.SpriteID]; ok && bossFrame != nil {
					op := &ebiten.DrawImageOptions{}
					op.GeoM.Translate(-32, -32) // center 64x64 sprite
					op.GeoM.Translate(x, y)
					field.DrawImage(bossFrame, op)
					continue
				}
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
		if b.Type == bullet.TypeLaser {
			drawLaserBullet(field, b, x, y)
			return
		}

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

func drawLaserBullet(field *ebiten.Image, b *bullet.Bullet, x, y float64) {
	length := b.Length
	if length <= 0 {
		length = 256
	}
	width := b.Width
	if width <= 0 {
		width = 12
	}
	x2 := x + math.Cos(b.Angle)*length
	y2 := y + math.Sin(b.Angle)*length
	c := laserColor(b.Color, 120)
	vector.StrokeLine(field, float32(x), float32(y), float32(x2), float32(y2), float32(width+6), color.RGBA{255, 255, 255, 48}, false)
	vector.StrokeLine(field, float32(x), float32(y), float32(x2), float32(y2), float32(width), c, false)
	vector.StrokeLine(field, float32(x), float32(y), float32(x2), float32(y2), float32(math.Max(2, width*0.28)), color.RGBA{255, 255, 255, 190}, false)
}

func laserColor(idx int, alpha uint8) color.RGBA {
	palette := []color.RGBA{
		{255, 80, 80, alpha},
		{255, 150, 64, alpha},
		{255, 230, 80, alpha},
		{100, 230, 120, alpha},
		{80, 190, 255, alpha},
		{110, 120, 255, alpha},
		{220, 110, 255, alpha},
		{255, 120, 190, alpha},
	}
	if len(palette) == 0 {
		return color.RGBA{255, 255, 255, alpha}
	}
	idx %= len(palette)
	if idx < 0 {
		idx += len(palette)
	}
	return palette[idx]
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

	res := sprite.Global
	if res != nil {
		ps := res.Player[s.ctx.State.Character]
		var kind sprite.AnimKind
		switch p.AnimState {
		case player.AnimFront:
			kind = sprite.AnimFront
		case player.AnimToSide:
			if p.FaceRight {
				kind = sprite.AnimToRight
			} else {
				kind = sprite.AnimToLeft
			}
		case player.AnimSide:
			if p.FaceRight {
				kind = sprite.AnimRight
			} else {
				kind = sprite.AnimLeft
			}
		}
		anim := ps.Anim(kind)
		if anim != nil {
			anim.Update()
			opts := ganim8.DrawOpts(px, py, 0, 1, 1, 0.5, 0.5)
			opts.ColorM.Scale(1, 1, 1, float64(alpha))
			anim.Draw(field, opts)
			vector.FillCircle(field, float32(px), float32(py), 3, color.RGBA{255, 255, 255, uint8(alpha * 255)}, false)
			return
		}
	}

	// fallback: 几何占位
	vector.FillRect(field, float32(px)-8, float32(py)-12, 16, 24, color.RGBA{0, 220, 0, uint8(alpha * 255)}, false)
	vector.FillCircle(field, float32(px), float32(py), 3, color.RGBA{255, 255, 255, 255}, false)
}

// drawBossHUD 在 Field 顶部画 Boss 血条（横条）+ 符卡名 + 时限倒数。
// 数据来自脚本可选实现的 BossHUD()（ECL VM 的 331/347/357/359 解析结果）。
func (s *Stage) drawBossHUD(field *ebiten.Image) {
	bh, ok := s.script.(interface{ BossHUD() BossHUDInfo })
	if !ok {
		return
	}
	info := bh.BossHUD()
	if !info.Active {
		return
	}
	w := float32(field.Bounds().Dx())

	// 血条：顶部内嵌 12px 处一条横条，红色底 + 黄色当前血量。
	const barY, barH, barPad = 10, 4, 24
	barW := w - barPad*2
	vector.FillRect(field, barPad, barY, barW, barH, color.RGBA{60, 12, 12, 200}, false)
	vector.FillRect(field, barPad, barY, barW*float32(info.HPFrac), barH,
		color.RGBA{255, 210, 90, 230}, false)
	for _, marker := range info.HPMarkers {
		if marker <= 0 || marker >= 1 {
			continue
		}
		x := barPad + barW*float32(marker)
		vector.FillRect(field, x-1, barY-2, 2, barH+4, color.RGBA{255, 255, 255, 210}, false)
	}

	if !info.Spell {
		return
	}
	if s.bossNameFace == nil {
		s.bossNameFace = render.LoadSystemFace(15)
	}
	if s.bossNumFace == nil {
		s.bossNumFace = render.LoadHUDFace(18)
	}
	// 符卡名（右对齐于条下方），描边提升可读性。
	nameY := int(barY) + barH + 4
	nameW, _ := render.MeasureText(s.bossNameFace, info.Name)
	render.DrawTextShadow(field, info.Name, s.bossNameFace,
		int(w)-int(barPad)-nameW, nameY,
		color.RGBA{255, 255, 255, 255}, color.RGBA{0, 0, 0, 200}, 1, 1)

	// 时限倒数（左上角大号数字）。
	if info.TimeLeft > 0 {
		render.DrawTextShadow(field, fmt.Sprintf("%d", info.TimeLeft), s.bossNumFace,
			int(barPad), nameY-2,
			color.RGBA{255, 255, 255, 255}, color.RGBA{0, 0, 0, 200}, 1, 1)
	}
}

// drawBombFlash 在使用符卡期间绘制全屏白色闪光：起爆瞬间最亮，随符卡剩余时间衰减。
func (s *Stage) drawBombFlash(field *ebiten.Image) {
	p := s.ctx.Player
	if p.State != player.StateBomb || p.BombTimer <= 0 {
		return
	}
	// BombTimer 从 bombDuration 递减到 0：起爆头一道强闪，之后保留淡幕。
	frac := float64(p.BombTimer) / float64(player.BombDuration())
	var alpha float64
	if frac > 0.85 {
		alpha = (frac - 0.85) / 0.15 * 0.6
	} else {
		alpha = frac * 0.18
	}
	if alpha <= 0.004 {
		return
	}
	w := float32(field.Bounds().Dx())
	h := float32(field.Bounds().Dy())
	vector.FillRect(field, 0, 0, w, h, color.RGBA{255, 255, 255, uint8(alpha * 255)}, false)
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

// 确保 Stage 实现 game.Scene
var _ game.Scene = (*Stage)(nil)
