package stage

import (
	"encoding/binary"
	"fmt"
	"th10/anm"
	"th10/datfs"
	"th10/entity/bullet"
	"th10/entity/enemy"
	"th10/entity/item"
	"th10/entity/laser"
	"th10/entity/player"
	"th10/game"
	"th10/gamedata"
	"th10/input"
	"th10/scene/dialog"
	"th10/std"
	"th10/std/stddraw"
	"th10/supervisor"
	"th10/zun"

	"github.com/hajimehoshi/ebiten/v2"
)

// Script 关卡脚本接口，每关实现一个
type Script interface {
	Init(ctx *Context)
	Update(ctx *Context)
	Finished() bool
}

// Context 关卡运行时上下文，脚本通过它操作所有实体
type Context struct {
	State   *game.GameState
	Player  *player.Player
	Bullets *bullet.Pool   // 敌弹管理器（DAT_004776f0）
	Lasers  *laser.Manager // 激光管理器（DAT_0047781c）
	Items   *item.Manager  // 道具管理器（DAT_00477818）
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
	// FUN_0042a450's live player history/option block. NewContext connects
	// these to Player; Raw retains only fields outside the capture's range.
	CaptureReplayPlayer func(header []byte)
	RestoreReplayPlayer func(header []byte)
	ReplaySlowSample    byte            // current performance sample, FUN_004297d0
	Menu                *MenuController // DAT_00477830, whose +4 gates the player's faith rectangles
	// GUI is DAT_0047770c; its arrays draw through anmdraw at node 0x2b.
	GUI                *GUI
	GUIController      *GUIController                                     // the same GUI control fields, also available to logic-only contexts
	RequestShake       func(kind, frames, x, y, z int32)                  // FUN_0043c8b0; includes fade kinds as well as shakes
	DrawScreenEffects  func(sv *supervisor.Supervisor, first, last int32) // FUN_0043c970's dynamic draw nodes
	ClearScreenEffects func()                                             // FUN_0043cb00: release both nodes at stage exit
	StageFlags         uint32                                             // DAT_00477810+0x58; screen-effect gates read this raw controller word
	StageExiting       bool                                               // DAT_004918a4 != 0; set when this stage is released
	// BossSection is DAT_00474c84: MSG entry id+1 while a message is up,
	// reset by MSG op 0, or the raw section number written by ECL 344.
	BossSection int32
	// CalcShake is FUN_0043c550, installed by the ECL wrapper. It runs
	// at priority 0x0e, not inside the enemy update or during Draw.
	CalcShake func()
	// Popups is the independent DAT_00477840 manager (calc 0x0f, draw 0x23).
	Popups *ScorePopups

	// msg 是正在播放的对话（nil = 无），msgs 是本关的 .msg，bossBGM 是 op 18
	// 切换的 Boss 曲，fadeFrames 是 op 21 的淡出帧数。
	msg        *dialog.MSG
	msgs       *dialog.StageMSG
	bossBGM    int
	fadeFrames int
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
	// The GUI reads these fields directly in FUN_00414900. HPMarkers above
	// is the old diagnostic view; Markers preserves all four ordered colours.
	HP, MaxHP   int32
	PhaseHP     int32
	Position    [2]float32
	Flags       uint32
	Midboss     bool
	HasTimer    bool
	Stars       int32
	Markers     [4]HPMarker
	DisplayName string // th10c.dat presentation only; Name keeps the ECL text
}

// DialogActive 报告是否有对话（MSG 对象存在）：对话期间自机不能射击和放灵击，
// 敌弹不判定，rank 不增长。
func (c *Context) DialogActive() bool {
	return c.msg != nil
}

// MSG 返回正在播放的对话，nil = 无。
func (c *Context) MSG() *dialog.MSG { return c.msg }

// SetBossSection is the DAT_00474c84 write in FUN_00412ed0 (ECL 344),
// FUN_00415db0 (MSG open) and FUN_00415e90 (op 0). DAT_00474c8c's
// separate section clock belongs to the game/replay controller.
func (c *Context) SetBossSection(n int32) { c.BossSection = n }

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
		Clear:   c.clearStage,
		FadeBGM: func() { c.FadeBGM(c.fadeFrames) },
	})
	if m == nil {
		return
	}
	// FUN_00415b00 binds both MSG ZunTimers to DAT_00476f78, including
	// stage 6's quarter-speed post-dialog; input still comes from this calc.
	m.GameSpeed = func() float32 { return c.Anm.GameSpeed }
	if c.GUI != nil {
		c.GUI.attachMSG(m, c.msgs.Stage, c.State.Character&1)
	}
	c.msg = m
	c.SetBossSection(id + 1) // FUN_00415db0, 0x415e01-0x415e0c
	c.Bullets.CancelAll(c.BulletEnv)
}

// Cleared reports whether a dialog's op 20 has cleared the stage.
func (c *Context) Cleared() bool { return c.GUIController != nil && c.GUIController.completed() }

// MSGWaiting is ECL 339's condition: a dialog is up and has not signalled
// (op 11) this frame.
func (c *Context) MSGWaiting() bool {
	return c.msg != nil && c.msg.Signal == 0
}

// UpdateMSG runs the dialog for a frame, late in the calc chain: advance is
// the shot key's press edge, skip whether Ctrl is held.
func (c *Context) UpdateMSG(advance, skip bool) {
	if c.msg != nil && !c.msg.Update(advance, skip) {
		c.SetBossSection(0) // FUN_00415e90 op 0, 0x416f3c-0x416f50
		c.msg = nil
	}
}

// Stage 通用关卡场景，驱动一关的全部逻辑
type Stage struct {
	sceneFlags uint32 // DAT_00477810+0x58, menu freeze is bit 0x10
	ctx        *Context
	script     Script
	// The scene callbacks are required for a stage driven by Update (see
	// selector.buildStage); headless and playback stages never call them.
	NextScene      func(g *game.Game, state *game.GameState) game.Scene // 通关后下一场景
	OnGameOver     func(g *game.Game, state *game.GameState) game.Scene
	OnContinue     func(g *game.Game, state *game.GameState) game.Scene // DAT_00491fb8 = 13 dispatcher hook
	OnRetry        func(g *game.Game, state *game.GameState) game.Scene // DAT_00491fb8 = 10, return to initial stage
	replaySeed     uint16
	replayTemplate []byte
	liveGame       *game.Game // nil for sim; original direct menu calls need the existing scene/audio host

	// drawn is set by the first Draw, which sets the clear colour.
	drawn bool

	// bg is the STD drawer (nodes 0x07 and 0x0a, see nodeBackground and
	// nodeSTDFront); nil when the stage has no embedded STD or its ANM.
	// Every stage's STD uses only render modes anmdraw draws: stages 2, 3,
	// 4, 6 and 7 use modes 0, 1, 2, 3 and 8; stages 1 and 5 add mode 6. See
	// docs/research/m4_s3.md.
	bg *stddraw.Drawer
}

func New(g *game.Game, state *game.GameState, script Script) *Stage {
	g.BindState(state)
	if state.StageMode == 0 {
		copy(state.ReplayConfig[:], g.Config.NativeBytes())
		state.ConfigFlags = binary.LittleEndian.Uint32(state.ReplayConfig[0x30:])
	}
	s := newStage(state, script)
	s.liveGame = g
	if a := g.Audio(); a != nil {
		s.ctx.Audio = a
	}
	script.Init(s.ctx)
	s.beginRecording()
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
	state.BindScoreController()
	s := &Stage{script: script}
	s.ctx = NewContext(state)
	s.ctx.stage = s
	s.ctx.Menu = NewMenuController(s)
	s.replaySeed = s.ctx.RNG.State
	s.ctx.ReplaySlowSample = 60
	state.BeginStageFaith() // FUN_00417870 sets the decay rate at stage start
	s.ctx.STD = loadSTD(state.Stage, s.ctx.Anm)
	// A stage whose STD has run its Enable (Active set) draws its
	// background with stddraw.
	if s.ctx.STD != nil && s.ctx.STD.Active != nil {
		s.bg = &stddraw.Drawer{STD: s.ctx.STD}
	}
	s.ctx.GUI = NewGUI(s.ctx)
	s.ctx.Popups = NewScorePopups()
	s.ctx.ItemEnv.Popup = s.ctx.AddScorePopup
	// FUN_00417870 (0x417c34): successful initialization consumes the
	// manager/replay reuse and continue flags, after their setup effects.
	state.RunFlags &^= game.RunFlagReuseManagers | game.RunFlagKeepReplay | game.RunFlagContinued
	return s
}

// loadSTD starts stage n's embedded .std (n 7 = Extra), nil if missing. Its
// objects run on the stage's .anm with env (FUN_00404450); without that .anm
// they stay idle.
func loadSTD(n int, env *anm.Env) *std.VM {
	data, err := gamedata.ReadFile(datfs.EmbedPath(fmt.Sprintf("stage%02d.std", n)))
	if err != nil {
		return nil
	}
	f, err := std.Load(data)
	if err != nil {
		return nil
	}
	v := std.New(f)
	if a, err := gamedata.SharedANM(f.ANM); err == nil {
		v.Enable(a, env)
	}
	return v
}

// UpdateHeadless 用调用方提供的输入快照推进一帧，不读全局键盘。
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
	return s.ctx.Cleared() || (s.ctx.msg == nil && s.script.Finished())
}

// ExportContext 暴露内部 Context 给 sim 包（无头模拟需要直接读写实体池）。
func ExportContext(s *Stage) *Context { return s.ctx }

func (s *Stage) Update(g *game.Game) error {
	s.liveGame = g
	in := &input.Global
	s.ctx.ReplaySlowSample = g.ReplaySample()

	action, blocked := s.UpdateMenu(g, in)
	if blocked {
		switch action {
		case MenuNext:
			g.GoTo(s.NextScene(g, s.ctx.State))
		case MenuContinue:
			next := s.OnContinue(g, s.ctx.State) // preserve t5 callback timing/signature
			g.GoToFactory(game.SceneContinue, func(*game.Game) (game.Scene, error) { return next, nil })
		case MenuTitle:
			s.goGameOver(g)
		case MenuRetry:
			g.GoToFactory(game.SceneRetry, func(g *game.Game) (game.Scene, error) { return s.OnRetry(g, s.ctx.State), nil })
		}
		return nil
	}

	in.ApplyReplayOptions(s.ctx.State.ConfigFlags)

	s.update(in)

	// 关卡结束/Game Over 时切换场景
	ctx := s.ctx
	if ctx.Menu.Active() { // MSG may have opened FUN_00423370 in this very calc chain
		return nil
	}
	if ctx.GUIController.SceneRequest != 0 {
		s.deliverSceneRequest(g)
		return nil
	}
	if ctx.Cleared() { // normal final clear waits for GUI+0x9ecc, not script completion
		return nil
	}
	if ctx.State.Life < 0 {
		// FUN_00425730 calls FUN_004231d0 in state 2 at timer >= 30.
		// The player tail has already incremented it in update, so use > 30.
		if ctx.Player.State == player.StateDying && ctx.Player.StateTimer.Cur > 30 {
			s.finishReplayMetadata(false)
			if !ctx.State.DebugMode {
				g.MarkGameOver(ctx.State)
			}
			ctx.Menu.OpenGameOver(g)
		}
		return nil
	}
	if s.finished() { // legacy scripts without MSG retain their explicit completion API
		s.finishReplayMetadata(ctx.State.Stage >= game.StageCount)
		if ctx.State.SingleStage || ctx.State.Stage == game.StageExtra {
			if !ctx.State.DebugMode {
				g.MarkClear(ctx.State)
			}
			ctx.Menu.OpenClear(g) // MSG 0x14 -> FUN_00423370 (practice / Extra)
		} else {
			g.GoTo(s.NextScene(g, ctx.State))
		}
	}
	return nil
}

// OpenCompletion connects the post-ending flow to the same ranking/replay
// controller. The ending/staff visuals remain with their existing scene.
func (s *Stage) OpenCompletion(g *game.Game) { s.ctx.Menu.OpenClear(g) }

func (s *Stage) goGameOver(g *game.Game) { g.GoTo(s.OnGameOver(g, s.ctx.State)) }

// update 推进一帧世界状态。不涉及场景切换，供 Update 与无头模拟共用。
func (s *Stage) update(in *input.State) {
	ctx := s.ctx
	if ctx.State.Recording != nil {
		ctx.State.Recording.Append(in.ReplayRecord(), ctx.ReplaySlowSample)
	}

	ctx.State.Frame++

	// 0. STD 相机脚本（FUN_00402720），先于敌人管理器
	if ctx.STD != nil {
		ctx.STD.Update(ctx.Anm.GameSpeed)
	}
	if ctx.CalcShake != nil {
		ctx.CalcShake() // FUN_0043c970: calc 0x0e, before 0x0f/0x10
	}
	if ctx.Popups != nil {
		ctx.Popups.Calc(ctx.Anm.GameSpeed) // FUN_0042b9a0, calc priority 0x0f
	}

	// 1. 玩家（对话期间不能射击和放灵击，FUN_004281d0 与灵击检查都要求没有 MSG）
	ctx.Player.NoAttack = ctx.msg != nil
	s.updatePlayer(in)
	ctx.SyncPlayer() // the script aims at this frame's position

	// 1b. 灵击（FUN_00405750，calc 0x11）：随世界中的 ANM 对象上升、取半径并消弹
	ctx.Bomb.Update()

	// 2. 脚本驱动（敌人/弹幕生成）
	s.script.Update(ctx)

	// 4. 敌弹与激光（子弹自身的命中、擦弹判定在这里完成）
	ctx.UpdateBullets()

	// 5. 道具（移动与拾取都在 FUN_0041afd0 里）
	ctx.UpdateItems(in.IsPressed(input.KeyFocus))

	// 6. 符卡状态（calc 0x16），先于 GUI（0x18）和 World（0x1a）。
	if lu, ok := s.script.(interface{ LateUpdate(*Context) }); ok {
		lu.LateUpdate(ctx)
	}
	if ctx.GUI != nil {
		ctx.GUI.Calc()
	}
	// FUN_00414900 runs MSG before its timer/arrow tail. New MSG World
	// objects then get the same frame's 0x1a step, as in the original chain.
	ctx.UpdateMSG(in.JustPressed(input.KeyOk), in.IsPressed(input.KeySkip))
	if ctx.GUI != nil {
		ctx.GUI.AfterMSG()
	}
	ctx.UpdateWorld()
	s.updateScoreController() // FUN_00418190 -> FUN_00417040 after the calc chain
	// FUN_00418190 returns after Numbers when stage+0x58 has the menu's
	// 0x10 freeze bit. The play-time counter and faith tail are skipped.
	if ctx.Menu != nil && ctx.Menu.Active() {
		s.cleanEnemies()
		return
	}
	ctx.State.RecordPlayFrame() // FUN_00418190, original score-object play-time counter

	// 8b. 信仰保持计时与衰减：FUN_00418190 在 calc chain 之后、帧计数之前调用
	// FUN_00418a00（见 faithGate）
	if s.faithGate() {
		ctx.State.TickFaith()
	}

	// 9. 清理
	s.cleanEnemies()
}

// faithGate is the test FUN_00418190 makes before FUN_00418a00 (0x418751..
// 0x418764): no boss in slot 0 (DAT_00477704+0x10 == 0), no dialogue open
// (DAT_0047770c+0x9eb8 == 0), and the stage clock at least 90 (+0x14 >= 0x5a).
// Frame already counts this update, and the original reads +0x14 before its
// own frame clock DAT_00474c88 increments at the end of the frame; the
// zero-based frame index is the value that test sees (inferred, see
// docs/research/faith.md), so the gate is Frame-1 >= 90, that is Frame > 90.
func (s *Stage) faithGate() bool {
	if s.ctx.DialogActive() || s.ctx.State.Frame <= 90 {
		return false
	}
	if b, ok := s.script.(interface{ BossSlot0Busy() bool }); ok && b.BossSlot0Busy() {
		return false
	}
	return true
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

// Draw runs one frame of the draw chain (drawFrame) and copies the frame to
// screen. Node 1 clears the back buffer, so there is no fill here.
func (s *Stage) Draw(screen *ebiten.Image) {
	sv := supervisor.Get()
	if !s.drawn {
		// FUN_00417c80 sets the clear colour (DAT_004923a8) at stage start;
		// done at the first Draw so headless stages never make the device.
		sv.ClearColor = 0xff000000
		s.drawn = true
	}
	// FUN_0043c5ca sampled these offsets at calc 0x0e. No logical
	// coordinates or viewport positions move; ANM's emit adds Field.Shift.
	sx, sy := 0.0, 0.0
	if sh, ok := s.script.(interface{ ShakeOffset() (float64, float64) }); ok {
		sx, sy = sh.ShakeOffset()
	}
	sv.Field.Shift = [2]float32{float32(sx), float32(sy)}
	s.drawFrame(sv)
	screen.DrawImage(sv.Frame, nil)
}

// 确保 Stage 实现 game.Scene
var _ game.Scene = (*Stage)(nil)
