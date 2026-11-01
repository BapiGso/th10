package stage

import (
	"fmt"
	"image/color"
	"th10/anm"
	"th10/audio"
	"th10/datfs"
	"th10/entity/bullet"
	"th10/entity/effect"
	"th10/entity/enemy"
	"th10/entity/item"
	"th10/entity/laser"
	"th10/entity/player"
	"th10/entity/playerbullet"
	"th10/game"
	"th10/gamedata"
	"th10/input"
	"th10/render"
	"th10/scene/dialog"
	resultscene "th10/scene/result"
	"th10/sprite"
	"th10/std"
	"th10/zun"

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
	Bullets *bullet.Pool   // 敌弹管理器（DAT_004776f0）
	Lasers  *laser.Manager // 激光管理器（DAT_0047781c）
	Items   *item.Manager  // 道具管理器（DAT_00477818）
	Effects *effect.Pool
	Enemies []*enemy.Enemy // 活跃敌人列表
	Audio   Sound          // 音频管理器；无头模拟（sim 包）保持 nil
	// World 是 ANM 特效对象（消弹、擦弹火花等），Anm 是它们与子弹脚本共用的
	// 运行环境，BulletEnv 是子弹管理器对外的接口。见 bullets.go。
	World     *anm.World
	Anm       *anm.Env
	BulletEnv *bullet.Env
	LaserEnv  *laser.Env
	ItemEnv   *item.Env
	// RNG 是原作的游戏随机数（DAT_004918b0），ECL 与子弹自身的 ANM 脚本共用。
	RNG *zun.RNG
	// STD 是本关 .std 的相机脚本（FUN_00403c80）；ECL 270/271 的敌人读它的
	// 相机位置与每帧位移（DAT_00491d7c）。找不到 .std 时为 nil（相机视为 0）。
	STD   *std.VM
	stage *Stage // 反向引用（用于触发对话等）

	// ManagerOrder 由 ECL VM 设置：按原作敌人管理器链表顺序列出存活敌人，
	// 包括隐藏的 main 敌人（trace 采样用，与 th10probe 对齐）。
	ManagerOrder func() []*enemy.Enemy
	// Boss0 由 ECL VM 设置：0 号 Boss 槽敌人的位置（跟随 Boss 的无限激光用）。
	Boss0 func() ([3]float32, bool)
	// Spell 由 ECL VM 设置：符卡状态（DAT_004776f4），灵击读写它。nil = 无符卡。
	Spell *SpellCard
	// Bomb 是灵击对象（DAT_004776ec），见 bomb.go。
	Bomb *Bomb

	// msg 是正在播放的对话（nil = 无），msgs 是本关的 .msg，bossBGM 是 op 18
	// 切换的 Boss 曲，fadeFrames 是 op 21 的淡出帧数。cleared 表示 op 20 已执行。
	msg        *dialog.MSG
	msgs       *dialog.StageMSG
	bossBGM    int
	fadeFrames int
	cleared    bool
}

// AddEnemy 脚本用：添加敌人
func (c *Context) AddEnemy(e *enemy.Enemy) {
	c.Enemies = append(c.Enemies, e)
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

// DialogActive 报告是否有对话（MSG 对象存在）：对话期间自机不能射击和放灵击，
// 敌弹不判定，rank 不增长。
func (c *Context) DialogActive() bool {
	return c.msg != nil
}

// MSG 返回正在播放的对话，nil = 无。
func (c *Context) MSG() *dialog.MSG { return c.msg }

// LoadMSG loads the stage's .msg ("stage4", "extra") for the player's
// character; op 18 starts bossBGM. Op 21 fades the music out over 2 s, 8 s on
// stage 6 (FUN_00420c30).
func (c *Context) LoadMSG(stage string, bossBGM int) {
	c.msgs = dialog.LoadStage(stage, c.State.Character)
	c.bossBGM = bossBGM
	c.fadeFrames = 120
	if stage == "stage6" {
		c.fadeFrames = 480
	}
}

// OpenMSG starts entry id for ECL 338 (FUN_00415db0). The MSG's constructor
// (FUN_00415b00) cancels every bullet (the lasers are cleared by ECL 338
// itself); a missing entry opens nothing.
func (c *Context) OpenMSG(id int32) {
	m := c.msgs.Open(id, dialog.Hooks{
		SE:      c.PlaySEID,
		BossBGM: func() { c.PlayBGM(c.bossBGM) },
		Clear:   func() { c.cleared = true },
		FadeBGM: func() { c.FadeBGM(c.fadeFrames) },
	})
	if m == nil {
		return
	}
	c.msg = m
	c.Bullets.CancelAll(c.BulletEnv)
}

// Cleared reports whether a dialog's op 20 has cleared the stage.
func (c *Context) Cleared() bool { return c.cleared }

// MSGWaiting is ECL 339's condition: a dialog is up and has not signalled
// (op 11) this frame.
func (c *Context) MSGWaiting() bool {
	return c.msg != nil && c.msg.Signal == 0
}

// UpdateMSG runs the dialog for a frame, late in the calc chain: advance is
// the shot key's press edge, skip whether Ctrl is held.
func (c *Context) UpdateMSG(advance, skip bool) {
	if c.msg != nil && !c.msg.Update(advance, skip) {
		c.msg = nil
	}
}

// Stage 通用关卡场景，驱动一关的全部逻辑
type Stage struct {
	ctx          *Context
	script       Script
	field        *ebiten.Image // 游戏区域离屏缓冲
	frontFrame   *ebiten.Image
	hud          *hud                                                 // 状态栏（ebitenui 排版）
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
	s := &Stage{
		script:     script,
		field:      ebiten.NewImage(fieldW, fieldH),
		frontFrame: loadFrontFrame(),
	}
	s.ctx = NewContext(state)
	s.ctx.stage = s
	s.ctx.STD = loadSTD(state.Stage)
	s.hud = newHud(state)
	return s
}

// loadSTD starts stage n's embedded .std (n 7 = Extra), nil if missing.
func loadSTD(n int) *std.VM {
	data, err := gamedata.ReadFile(datfs.EmbedPath(fmt.Sprintf("stage%02d.std", n)))
	if err != nil {
		return nil
	}
	f, err := std.Load(data)
	if err != nil {
		return nil
	}
	return std.New(f)
}

func loadFrontFrame() *ebiten.Image {
	return render.LoadImage("anm/front/front00s.png")
}

// UpdateHeadless 用调用方提供的输入快照推进一帧，不读全局键盘、不刷新 HUD。
// 供 sim 包的逐帧差分模拟使用；真实游戏走 Update。
func (s *Stage) UpdateHeadless(in *input.State) {
	s.update(in)
}

// FinishedHeadless 报告关卡是否已结束（无头模拟的终止条件）。
func (s *Stage) FinishedHeadless() bool { return s.finished() }

// finished reports the stage's end: the MSG's op 20, else the script's own
// condition once no dialog is up. Stages 4-6 and Extra loop in their main sub
// forever after the boss, so only op 20 ends them.
func (s *Stage) finished() bool {
	return s.ctx.cleared || (s.ctx.msg == nil && s.script.Finished())
}

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
	if s.finished() {
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

	ctx.State.Frame++

	// 0. STD 相机脚本（FUN_00402720），先于敌人管理器
	if ctx.STD != nil {
		ctx.STD.Update(ctx.Anm.GameSpeed)
	}

	// 信仰值自然衰减
	ctx.State.DecayFaith()

	// 1. 玩家（对话期间不能射击和放灵击，FUN_004281d0 与灵击检查都要求没有 MSG）
	ctx.Player.NoAttack = ctx.msg != nil
	s.updatePlayer(in)
	ctx.SyncPlayer() // the script aims at this frame's position

	// 1b. 灵击（FUN_00405750，calc 0x11）：随世界中的 ANM 对象上升、取半径并消弹
	ctx.Bomb.Update()

	// 2. 脚本驱动（敌人/弹幕生成）
	s.script.Update(ctx)

	// 3. 敌人
	for _, e := range ctx.Enemies {
		e.Update()
	}

	// 4. 敌弹与激光（子弹自身的命中、擦弹判定在这里完成）
	ctx.UpdateBullets()

	// 5. 道具（移动与拾取都在 FUN_0041afd0 里）
	ctx.UpdateItems(in.IsPressed(input.KeyFocus))

	// 6. 特效
	ctx.Effects.Update()
	ctx.UpdateWorld()

	// 7. 脚本的伤害后处理：ECL 的阶段中断（超时）与计时器，见 FUN_0040dc80
	if lu, ok := s.script.(interface{ LateUpdate(*Context) }); ok {
		lu.LateUpdate(ctx)
	}

	// 8. 对话（calc chain 末尾的 GUI/MSG）
	ctx.UpdateMSG(in.JustPressed(input.KeyOk), in.IsPressed(input.KeySkip))

	// 9. 清理
	s.cleanEnemies()

	return nil
}

// updatePlayer runs the player (FUN_00425730); the death drops come through
// Player.SpawnItem.
func (s *Stage) updatePlayer(in *input.State) {
	p := s.ctx.Player
	p.SetHomingCandidates(s.ctx.Enemies)
	p.UpdateInput(in)
}

// cleanEnemies drops deleted enemies. Leaving the field is the ECL's call
// (the per-enemy offscreen test in FUN_0040dc80, scene/stage/ecl/motion.go).
func (s *Stage) cleanEnemies() {
	n := 0
	for _, e := range s.ctx.Enemies {
		if e.Active {
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
	s.drawWorld(s.field)

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

	// 3b. 对话（立绘与对话框都在 Field 范围内）
	if ctx.msg != nil {
		dialog.Draw(screen, &ctx.msg.View)
	}

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
}

// 以下绘制全部用几何占位，后续替换为贴图

// drawItems draws the items by FUN_0041ba30.
func (s *Stage) drawItems(field *ebiten.Image) {
	fl, ft := float64(game.FieldLeft), float64(game.FieldTop)
	s.ctx.Items.Draw(func(vm *anm.VM) {
		sprite.DrawVM(field, vm, -fl, -ft)
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
		if !e.Active || e.Hidden {
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

// drawBullets draws the bullets layer by layer as the manager sorted them,
// then the lasers, each beam under its emitter.
func (s *Stage) drawBullets(field *ebiten.Image) {
	fl, ft := float64(game.FieldLeft), float64(game.FieldTop)
	for l := range bullet.Layers {
		s.ctx.Bullets.EachLayer(l, func(b *bullet.Bullet) {
			sprite.DrawVM(field, &b.VM, -fl, -ft)
		})
	}
	s.ctx.Lasers.Each(s.ctx.LaserEnv, func(beam, emitter *anm.VM) {
		sprite.DrawVM(field, beam, -fl, -ft)
		if emitter != nil {
			sprite.DrawVM(field, emitter, -fl, -ft)
		}
	})
}

// drawWorld draws the ANM effect objects by layer.
func (s *Stage) drawWorld(field *ebiten.Image) {
	fl, ft := float64(game.FieldLeft), float64(game.FieldTop)
	for l := range anm.Layers {
		s.ctx.World.Each(l, func(vm *anm.VM) {
			sprite.DrawVM(field, vm, -fl, -ft)
		})
	}
}

func (s *Stage) drawPlayer(field *ebiten.Image) {
	p := s.ctx.Player
	if p.State == player.StateDying {
		return
	}
	fl, ft := float64(game.FieldLeft), float64(game.FieldTop)
	px := p.X - fl
	py := p.Y - ft

	// 无敌闪烁：原作是 ANM 标志 0x8000 配颜色 0xff0000ff，这里先用半透明代替。
	alpha := float32(1.0)
	if p.Flash {
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

	nameY := int(barY) + barH + 4
	if info.Spell {
		if s.bossNameFace == nil {
			s.bossNameFace = render.LoadSystemFace(15)
		}
		// 符卡名（右对齐于条下方），描边提升可读性。
		nameW, _ := render.MeasureText(s.bossNameFace, info.Name)
		render.DrawTextShadow(field, info.Name, s.bossNameFace,
			int(w)-int(barPad)-nameW, nameY,
			color.RGBA{255, 255, 255, 255}, color.RGBA{0, 0, 0, 200}, 1, 1)
	}

	// 时限倒数（左上角大号数字）：非符阶段同样有时限（ECL 334 的时间槽）。
	if info.TimeLeft > 0 {
		if s.bossNumFace == nil {
			s.bossNumFace = render.LoadHUDFace(18)
		}
		render.DrawTextShadow(field, fmt.Sprintf("%d", info.TimeLeft), s.bossNumFace,
			int(barPad), nameY-2,
			color.RGBA{255, 255, 255, 255}, color.RGBA{0, 0, 0, 200}, 1, 1)
	}
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
