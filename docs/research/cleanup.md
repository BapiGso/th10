# 收尾统一普查（t16）

日期：2026-11-08。对象：TH10 v1.00a、第三轮合并后的工作树。阶段 A 先于源码修改完成；后续阶段的实际删除和验收记录见 `docs/handoff/codex_t16.md`。

## 1．方法与边界

完整读取共用 rules、fidelity、harness、open_items、scenes_a／scenes_b、presentation_gaps、m4_s3，以及 t1–t13 相关交付第 6／9 节。没有运行原作、probe、Ghidra或数据 junction 内的程序。

采用三种相互补充的证据，不把一次文本搜索或 linter 的结果直接当删除授权。

1．`golangci-lint` 2.15.0，构建 Go 1.27.0，提交 `25c5369a`，日期 `2026-10-16T11:03:09Z`；本机 Go 为 1.27.3，Windows／amd64。默认 linter 为 errcheck、govet、ineffassign、staticcheck、unused。
2．不包含测试的 unused 分析：`golangci-lint run --config D:\GoProjects\th10\.tmp\codex\t16\lint-baseline.yml --tests=false --enable-only=unused ./...`，18 项。包含测试后，默认截断口径为 32 项；取消同类／同文／同一行截断后为 52 项：errcheck 44、staticcheck 4、unused 4，govet／ineffassign 均为 0。完整 JSON／文本在任务根 `logs/lint-before.*`、`logs/unused-before.*`。旧交付的 32／33／50／51 不再混用。
3．等效的声明可达图：任务根 `reachability.go` 只用 Go 标准库，通过 `go list -deps -export -json` 和 `go/types` 建立声明引用，根为各入口 `main`、包 `init`、全局初始化；跟踪函数值、方法值、接口候选方法和传给外部包的本地类型。Windows 根为 `./cmd ./tools/...`，WASM 根为 `./cmd`。对应 51／43 个项目包，2696／2480 个声明，194／224 个不可达候选；两种构建共同出现的候选为 183 项。原始清单在任务根 `logs/reachability-{windows,wasm,both}-before.tsv`。

可达图有意保守，但不是完整 SSA／反射分析。接口声明、实例化对象、宿主回调、反射字段和纯初始化都要人工核对；附录列的是候选而不是自动删除清单。unused 不会完整报告未被程序入口使用的导出 API，也不能仅因测试用了一个早期近似就保留该近似。

删除依据还包括全仓 `rg` 的调用点核对，以及原断言究竟在测被删近似还是仍可达的原作行为。`//go:build ignore` 的旧文件不参加任何普通游戏／工具入口。测试、开发工具的公开诊断接口和下一任务明确约定的 hook 分别记账，不混为运行时死路径。

## 2．确认可删除的旧路径

以下是在改动前人工复核的清单；“整文件”包含表、辅助类型与其闭包中的函数。

| 位置 | 不可达对象 | 依据与计划 |
|---|---|---|
| `Deprecated/stage1/stage1_hardcoded.go`，整文件 | `Script1`、`NewScript`／`NewHandwrittenScript`、背景、道中波次、两组 Boss 阶段及其回调 | `go:build ignore`；普通构建没有该包，旧实体字段也已经不匹配。删除只供参考的近似，不移植或启用它。 |
| `scene/dialog/scripts.go`，整文件 | `stageScript`、`scripts`、`r`／`l`、人物名常量、`cloneLines`、`MidPre`／`BossPre`／`Post` | 仅上述 ignored 文件调用公开取台词函数；原 MSG 没有调用。表初始化只产生这些未读取的字面量和纯 `Line` 值。 |
| `scene/dialog/dialog.go`，整文件 | `Line` | 只服务上述手写台词表；原 MSG 使用 `TextLine`／`Presentation`，不使用 `Line`。 |
| `scene/dialog/msg.go` | `msgFaceExpression` | 无运行时读者；原立绘直接使用 ANM face id，不再使用 PNG 表情后缀。只有 `TestMSGFaceExpressionMapping` 测该旧映射。 |
| `scene/ui/menu.go`，整文件 | `MenuItem`／`MenuStyle`／`MenuList`，`Update`／`move`／`Draw`／`colorFor`／`drawOutlinedText` | 全仓无构造或调用；t10 已由 TitleController／Selector／原 ANM 替换。没有相关测试。 |
| `scene/ui/ui.go` | `NewPanel`／`Button`，`Scene` 及 `SetCursor`／`Cursor`／`Update`／`activateFocused`／`Draw` | 无构造／调用；保留仍被 Result 回退使用的 `Theme`／`DefaultTheme`／`NewRoot`／`Label`，不删除其字体加载调用。 |
| `scene/menus/option.go`，整文件 | `Option`、`NewOption` 及所有 Scene 方法 | 没有游戏／工具／测试构造它；实际 Option 在同一个 TitleController 的页 4／5。t10 的页状态机和测试保留。 |
| `scene/select/select.go` | `Select`、`New`／`NewExtra`／`NewPractice` 及其 Scene 方法 | 旧独立选人场景已经没有调用者；主标题直接调用 `Launch`。保留关卡工厂、`ReplayScript`、`Launch`、两个 debug 构造入口。 |
| `scene/ui/title_controller.go` | `RestartChild` | 无调用者；现行 Option／Key Config 使用 `SetChildSprite`，不能将精灵切换错误地改成重启脚本。 |
| `scene/menus/replay.go` | `replayEntryLabel` | 只有 `TestReplayEntryLabels`；实际列表／详情走原小字库和 `ReplayDetails`。存储、列表、选章、返回测试保留。 |
| `scene/stage/ecl/disasm.go`，整文件 | `Program.Disassemble`／`Sub.Disassemble`／`Instr.argString`／`Param.String`／`quoteList`／`varName`／`rankString`／`opName`，`opNames` | 没有游戏、工具或测试调用；带参数的 `Param.String(int32)` 也不是 `fmt.Stringer`。未读取的名称表初始化没有外部副作用。保留二进制 loader／IR。 |
| `scene/stage/ecl/game_ops.go`、`scene/stage/ecl/vm.go` | `eclUnset`、`VM.aimAngleAt` | 包含测试的 unused 也报告这两项；现行分支直接用 float32 哨兵或原瞄准路径。 |
| `scene/stage/ecl/opcodes.go` | `opSetChapter`／`opStars` | 全仓只有定义，是 344／360 的旧别名；当前执行分支不引用它们。 |
| `input/input.go` | `State.Snapshot` | 无游戏、工具或测试读者；回放写入的是 `ReplayRecord` 的三字，菜单读取 `MenuInput`，不是抽象 Key 的快照。 |
| `entity/enemy/enemy.go` | `NewBoss`／`SetPath`／`EndInvincible`／`MaybeEndInvincible`／`CurrentPhaseName`／`PhaseProgress`／`RemainingPhases`／`InBounds` | 只有 ignored 旧关卡的调用或完全没有调用。保留 `New`／`Update`／`HitRect`，以及仍用于验证原 ECL flags／hurt 的 `Shootable`／`TakeDamage`。不改数据字段或逐敌更新。 |
| `collision/collision.go` | `CircleHit`／`OrthoCircleHit`／`InBounds` | 前两项只有相互调用，第三项只有上述旧 Enemy.InBounds 调用；没有测试。保留真正使用的 `RectHit`。 |

允许删除的测试仅两项：`TestMSGFaceExpressionMapping` 和 `TestReplayEntryLabels`。它们分别只验证被删的后缀映射和旧列表字符串，不能代表当前 MSG 立绘或原回放列表。其余测试的断言、期望值、容差和 golden 均保持原样。

## 3．不可直接删除的候选及其他任务范围

| 位置 | 对象／原因 | 处理 |
|---|---|---|
| `scene/stage/chain_legacy.go` | `nodeBackground` 始终在 draw 0x07 注册；STD 缺失、非转换关或 `TH10_LEGACY_BG=1` 时真实调用 `Script.BgDraw`。`runChain` 的无头跳过不能证明交互不可达。 | 保留整个路径及环境开关。原正常路径为 `FUN_00403060 → FUN_00402850`；不能借“收尾”删除用户仍可运行的对比模式。 |
| `scene/stage/stage.go`、`scene/result/result.go` | 没有 NextScene／OnGameOver 时仍有 Result 回退；Script 是开放接口，不只接受七关默认工厂。 | 未证明这些回退在所有调用约定下不可达，保留。 |
| `entity/enemy/enemy.go` | `Stage.update` 每帧真实调用 `Enemy.Update`，其 `Age++`／旧 path 与 Boss nil 检查仍执行。 | 不删执行路径，不以“当前 path 为空”代替可达性证明；相关原逻辑在 `FUN_0040dc80`。 |
| `scene/stage/ecl/motion.go` | `ease` 只被测试调用，但转发的是仍可达的 `zun.Ease`／`FUN_0044c350`，原精度测试验证真实行为。 | 保留，不删除或改写该测试。 |
| `scene/stage` 的 native 诊断接口 | `VM.BossActive`、`GUI.SetBossSection`、`GUI.BeginSpellPresentation`、`SpellCard.Tick`、`Stage.SceneFlags`，Playback 的 `Stage`／`Chapter`／`UpdateHeadless`／`DrawPorted` 等 | 原状态、回放和演出规格仍通过这些接口验证，保留。 |
| `scene/title`／`scene/loading`／`scene/ui` | `DrawPorted`、`Title.SetPageHooks`、`RegisterTitlePage`、`BeginExtra` 等 | 原场景／无头绘制规格或 scenes_a 第 9 节明确的共享接口，保留；不把接口当前未被主入口直接调用当作早期近似。 |
| `scene/menus` | `Tracks`／`Slots` 等只读视图 | t11 原数据／回放测试及交接接口需要，保留。 |
| `scene/select/select.go` | `NewDebugStage` 没被 cmd.main 直接用，但 `cmd/debug_stage_test.go` 验证真实关卡构造路径。 | 保留，cmd 与其测试属于 t15，不修改。 |
| `sprite/anm.go`／`sprite/vm.go`／`sprite/loaders.go`／`sprite/sheet.go`，`anm/resolve.go` | 旧 `AnmSheet`／`DrawVM`／`SheetFor`／`ScriptSprite`／`Frame`／`Image`、静态 `SpriteAt` 等入口候选 | ANM 贴图加载／解码属于 t14；只列出，不删或改。默认 World／自机／STD 已使用 anmdraw。 |
| `render/text.go` | `LoadGameFace`／`LineHeightText` 等候选 | 字体加载属于 t14；保留。 |
| `tools/th10extract/main.go` | unused 的旧 `run` 包装函数，只被工具测试用 | t14，只列出。 |
| `trace/trace.go` | unused 的 `Reader.err` 字段；`FormatVersion` 没有主入口直接读者 | t17，只列出。 |
| `game/persistence*.go`／`game/native*.go` | 内存存储、decode／测试桥接、旧迁移入口等候选 | 存储／WASM 接口涉及 t15；不改。 |
| `gfx/shader.go` | unused 非测试模式下的 `alpha8`／`depthColor`／`fragIn`／`fragment`／`texel`／`texelAt`／`addressIndex`／`lerp4`／`shade` | CPU 参考实现与 shader 测试还在验证绘制公式；不能删除仍可达 GPU 行为的规格。非 t16 范围。 |
| `replay/replay.go`、`entity/playerbullet`、`datfs`、`std`、`sht` 等 | `parseChapters`／`normAngle` 和静态诊断／测试辅助，完整候选见附录 | 不扩张到回放、射击或原数值移植；只列出。 |

## 4．stageN 手写行为清单

七个包都不是纯配置。共有 `frame`／`finished`、VM 和背景图片字段，构造时载 ECL／PNG，Init 绑定 context、播放 BGM、加载 MSG，Update 更新 VM 与背景，并提供 BossHUD／ShakeOffset／LateUpdate 转发。这些是明确可达的。

| 行为／原位置 | 执行情况 | 原作地址与统一方案 |
|---|---|---|
| 七关 `NewScript`／`Init` | 游戏与 `tools/th10sim` 都调用；PNG 加载即使默认使用 STD 也照常发生。 | ECL 管理器／关卡初始化 `FUN_00417870`；BGM `FUN_00420b10／00420b90`；MSG 入口 `FUN_00415db0／00415b00`。移至共用 Script 配置工厂，绑定先于首个 calc 的顺序不变。 |
| 七关 `Update` 的 `frame++`、`MainDone`／60000 帧安全结束 | 每次关内 calc 都执行；主要通关仍由 MSG op 20 和 GUI 控制器决定。 | 原控制器 `FUN_00418190`、敌机／ECL `FUN_0040d750／0040dc80`、最终 MSG `FUN_00415e90`。60000 的人为 cap 没有对应原作常量，不能写成移植；原样留在共用 Script，交付第 6 节列明。 |
| stage1 `stage01StdCamera`／`stdCam`／`lerp`／`easeOut`／`smoothStep` | Update 每帧运行，结果只服务旧背景；不是不可达。 | 原 STD 指令运行 `FUN_00403c80`，相机插值 `FUN_00412ac0／00404610`。此处 float64 关键帧／近似缓动非逐函数移植，原样移动，不删、不替换。 |
| stage1 `BgDraw`／`drawFog`／云表类型与常量／`drawStdCloudEntries`／`stage01StdLeafSway`／`drawStdCloudQuad`／`blendAdd` | Legacy 回退才画；地面、色块雾、经验视差和云亮度仍可运行。 | STD draw `FUN_00402850`、雾指令 `FUN_00403c80／00405040`、ANM `FUN_0043ee30`、模式 6 `FUN_00443fb0`。0.72／0.50／0.68／0.30 等是早期经验值，没有独立 PE 真值地址；原样移到共用 legacy 背景。 |
| stage2／3／4／5／6／Extra 的 `bgY += 2.0／1.8／2.5／1.5／1.2／2.0` | Update 每帧运行，累加顺序和 float64 精度保留。 | 原 STD `FUN_00403c80`，但这些恒速不是原脚本的逐函数移植；移到共用背景，按配置选关。 |
| stage2–6／Extra `BgDraw` 的底色、PNG 平铺与 vector fallback | Legacy 回退可执行；默认 STD 不调用。 | 正常原路径 `FUN_00402850／00402ca0`；手画线／色块没有逐项原地址。仍保留 PNG 缺失 fallback，不借清理改画面。 |
| stage3 bg3／bg4、stage4 bg3、stage6 bg3／bg4／bg5、Extra bg3 | 加载调用实际执行，但绘制不读取这些字段。 | 属旧背景加载残余。不能将有初始化调用的字段误报为完全不可达；调用和路径原样移动，后续资源工作由 t14 处理。 |
| 七关 `BossHUD`／`ShakeOffset`／`LateUpdate` | Stage 的可选接口和 GUI／绘制路径真实调用。 | Boss／中断 `FUN_0040dc80／00412720／004127a0`、震屏 `FUN_0043c550／0043c5ca`、符卡 calc `FUN_00408d60`。合并为共用转发，不重复执行 VM、不移动 calc 优先级。 |

阶段 C 的目标是声明级纯配置：ECL／MSG 文件名、BGM 号、背景类型、滚动量、帧上限。`NewScript`／`NewECLScript` 采用绑定配置的方法值，旧具体类型以类型别名兼容；包内没有函数体或运行时状态。这样不用改 `tools/`、`sim/` 或已有测试调用点。行为统一到新共用包，不把“移出关卡包”宣传成“近似已移植或已删除”。

## 5．t10／t11 重复实现对照

| 项目 | t10 | t11 | 差异及不改行为的统一方案 |
|---|---|---|---|
| 标题背景／根句柄 | `scene/ui/title_controller.go` 的 Spawn／Root／LeaveScript／Interrupt／Child；`title_pages.go` 的 0x58／0x5a／0x5b／0x5e 等根 | `scene/menus/native.go` 的 `nativeMenu.init`／`spawnTitle`／`interrupt`，分别生成 0x5e／0x5f 和页头 | 同是 `FUN_0042c670／0042c6d0／0042c770`，移出 t10 根操作为 `ui.TitleAnimations`，两边调用。背景脚本、创建帧、World 生命周期、raw ANM 与 PresentationANM 文件选择不互换；不新建持久全局 World 或把整个标题画到菜单后面。 |
| ASCII script 6／8 | ShowLoading 生成 script 6；难度页生成 script 8，但只跑通用 World；未找到私有 ANM-91 补发实现 | `nativeMenu.calcLoading` 生成 script 7 前插粒子，第一次 16 个，以后每三帧 1 个 | script 6 是不同的 `FUN_0040c540`，不与 91 混为一谈。91 子集（`FUN_0043ee30` 的 `0x43ef55／0x43f00d`）移入共用 `ui.LoadingEmitter`，World.Run／出生后补发只保留一份。t10 仍不启用此前缺失的补发，防止额外 VM／备用 RNG／像素改变；缺口交付第 6 节明确列出。 |
| 淡入／淡出 | `scene/ui/fade.go`，kind 0／2／3／5；参数包含 speed、exiting、stageFlags；先算 alpha 再 Tick | `scene/ending/fade.go` 的 screenFade，kind 0／5；`native.go` 的 menuFade，kind 5 | 原同属 `FUN_0043c8b0／0043c970／0043bd40／0043c230`，以 `ui.Fade` 为唯一计算实现。调用点仍控制 draw 0x2b／0x31 和 text／World 前后位置。保留 t11 的薄测试观察适配，不复制计算公式。 |
| 淡出提交状态 | ui.Fade.Draw 选 Screen、关闭 Fog、提交 Rect | ending 只选 Screen；replay 已在当前 Screen 上提交 Rect | 共用 Rect 提交与完整 Draw 分开，调用者保留原 Use／Fog 状态调用，不能用合并之名调整层序或强插 Fog。 |
| 菜单重复 | `input.MenuInput.Update／Move`：按位 26／8 计数；t10 只读快照 | `menus.menuRepeat` 与 `stage.menuRepeat`：抽象 Key 的 HoldFrames，首按、26／34／42… | 当前方向键序列等价。两个私有重复函数删除，所有方向读同一 `MenuInput.Move`；确认／取消仍用原 JustPressed 或 Pressed，不改变 Enter／Z、Esc／X／Q／R 和相反方向优先级。 |
| 小字库 | `ui.GlyphText` | `nativeMenu.smallText` | 同是 `FUN_00401760`，精灵 98 起、步长 9。统一绘制函数，文件由各自现有入口传入，不动字体加载和光栅化。 |
| Music Room 的四帧位置补间 | 无对应私有实现 | `nativeMenu.move`／positions | 独立原函数 `FUN_00434a80`，不是标题背景或原 ANM-91；保留，继续在 World.Run／补发之后执行。 |

原结局 MSG VM 为 `FUN_0040bd80`，关内 MSG VM 为 `FUN_00415e90`；它们本来就是不同函数，不合并状态机、wait、时钟或台词执行。t10／t11 已有的场景、暂停、Staff／Music／Data／Replay 测试原样保留。

## 6．lint 唯一口径

阶段 E 加入仓库根 `.golangci.yml`，默认 standard linter 加显式 unused；包含测试、不按 linter／同文／行截断，不启用 modernize，不新增依赖。

唯一常规运行命令：`golangci-lint run`。普查的 `--tests=false --enable-only=unused` 只是可达性取证，不替代常规 lint；捕获 JSON／文本的输出路径也不改变统计规则。剩余告警须按包和实际任务范围逐项交付，不通过修改其他任务的源码或测试来清零。

本任务测试授权不允许修改仍可达行为的测试。`Stage.update` 的两个 errcheck 告警不能用改测试解决；该内部函数当前所有路径都只返回 nil，可去掉无意义的 error 返回和生产调用者的空错误分支，原测试保持原样。ECL 的空分支保留 `execStackMath` 调用，只去掉无操作的条件分支。MSG 的 QF 建议不作为批量 modernize 授权。

## 7．候选附录

以下附录由两种构建共有的声明候选合并而来。它补充范围外／测试辅助的完整名字，删除决策仍以第 2／3 节人工依据为准；字段 `Reader.err` 另由 unused 给出。

| 文件 | 声明候选 |
|---|---|
| `anm/resolve.go` | `const th10/anm.opStop`、`const th10/anm.opSetSprite`、`const th10/anm.opGoto`、`const th10/anm.opLoop`、`func *th10/anm.Script.SpriteAt`、`func *th10/anm.Script.offsetIndex` |
| `audio/audio.go` | `const th10/audio.BGMEnding`、`const th10/audio.BGMStaff`、`const th10/audio.SEShoot`、`const th10/audio.SEEnemy`、`const th10/audio.SEDamage`、`const th10/audio.SEGraze`、`const th10/audio.SEItem`、`const th10/audio.SEExtend`、`const th10/audio.SEBomb`、`const th10/audio.SEOk`、`const th10/audio.SECancel`、`const th10/audio.SESelect`、`const th10/audio.SEPause`、`const th10/audio.SEPowerUp` |
| `collision/collision.go` | `func th10/collision.CircleHit`、`func th10/collision.OrthoCircleHit`、`func th10/collision.InBounds` |
| `datfs/archive.go` | `func *th10/datfs.Archive.Has`、`func *th10/datfs.Archive.Open`、`func *th10/datfs.Archive.ReadDir`、`func *th10/datfs.Archive.dirEntries`、`func th10/datfs.fileInfo.Sys`、`func th10/datfs.fileInfo.Mode`、`type th10/datfs.file`、`func *th10/datfs.file.Stat`、`func *th10/datfs.file.Close`、`type th10/datfs.dirFile`、`func *th10/datfs.dirFile.Stat`、`func *th10/datfs.dirFile.Read`、`func *th10/datfs.dirFile.Close`、`func *th10/datfs.dirFile.ReadDir` |
| `entity/bullet/bullet.go` | `func *th10/entity/bullet.Bullet.Live`、`func *th10/entity/bullet.Pool.Reset` |
| `entity/enemy/enemy.go` | `func th10/entity/enemy.NewBoss`、`func *th10/entity/enemy.Enemy.SetPath`、`func *th10/entity/enemy.Enemy.Shootable`、`func *th10/entity/enemy.Enemy.TakeDamage`、`func *th10/entity/enemy.Enemy.EndInvincible`、`func *th10/entity/enemy.Enemy.MaybeEndInvincible`、`func *th10/entity/enemy.Enemy.CurrentPhaseName`、`func *th10/entity/enemy.Enemy.PhaseProgress`、`func *th10/entity/enemy.Enemy.RemainingPhases`、`func *th10/entity/enemy.Enemy.InBounds` |
| `entity/item/item.go` | `func *th10/entity/item.Manager.Each` |
| `entity/laser/laser.go` | `func *th10/entity/laser.Manager.Reset`、`func *th10/entity/laser.Manager.Count` |
| `entity/player/options.go` | `func *th10/entity/player.Player.Options` |
| `entity/player/player.go` | `func *th10/entity/player.Player.SetPos`、`func *th10/entity/player.Player.AddPower` |
| `entity/playerbullet/playerbullet.go` | `func *th10/entity/playerbullet.Pool.Fire`、`func th10/entity/playerbullet.normAngle`、`func *th10/entity/playerbullet.Pool.Each` |
| `game/native_config.go` | `func *th10/game.NativeConfig.Flags` |
| `game/native_score.go` | `func th10/game.DecodeScoreFile` |
| `game/persistence_score.go` | `func *th10/game.MemoryScoreStore.SetLegacyJSON`、`func th10/game.LoadSaveData` |
| `game/scenes.go` | `func *th10/game.numberedScene.DrawPorted`、`interface method interface{DrawPorted(*th10/supervisor.Supervisor)}.DrawPorted` |
| `game/stages.go` | `func th10/game.CharacterTitle`、`func th10/game.ShotDescription` |
| `game/state.go` | `const th10/game.ShotB`、`const th10/game.SmallPowerValue`、`const th10/game.BigPowerValue`、`func *th10/game.GameState.AddScore`、`func *th10/game.GameState.AddScoreRaw`、`func *th10/game.GameState.AddFaith`、`func *th10/game.GameState.FaithMultiplier`、`func *th10/game.GameState.IsExtraUnlocked` |
| `gfx/gfx.go` | `const th10/gfx.FogTableLinear`、`const th10/gfx.AddressClamp` |
| `gfx/shader.go` | `func th10/gfx.alpha8`、`func th10/gfx.depthColor`、`type th10/gfx.fragIn`、`func th10/gfx.fragment`、`func th10/gfx.texel`、`func th10/gfx.texelAt`、`func th10/gfx.addressIndex`、`func th10/gfx.lerp4`、`func th10/gfx.shade` |
| `input/input.go` | `func *th10/input.State.Snapshot` |
| `input/menu.go` | `func *th10/input.State.RestoreMenu`、`func *th10/input.State.SetMenuAux` |
| `render/text.go` | `func th10/render.LoadGameFace`、`func th10/render.MeasureText`、`func th10/render.LineHeightText`、`func th10/render.DrawText`、`func th10/render.DrawTextShadow` |
| `replay/record.go` | `func *th10/replay.Stream.Position` |
| `replay/replay.go` | `func th10/replay.parseChapters` |
| `scene/dialog/msg.go` | `interface method interface{Bytes([]byte) ([]byte, error)}.Bytes`、`func th10/scene/dialog.msgFaceExpression` |
| `scene/dialog/msgvm.go` | `func *th10/scene/dialog.MSG.Presentation` |
| `scene/dialog/scripts.go` | `func th10/scene/dialog.cloneLines`、`func th10/scene/dialog.MidPre`、`func th10/scene/dialog.BossPre`、`func th10/scene/dialog.Post` |
| `scene/ending/ending.go` | `func *th10/scene/ending.Ending.ScriptName` |
| `scene/loading/loading.go` | `func *th10/scene/loading.TitleGate.DrawPorted`、`interface method interface{DrawPorted(*th10/supervisor.Supervisor)}.DrawPorted` |
| `scene/menus/musicroom.go` | `func *th10/scene/menus.MusicRoom.Tracks` |
| `scene/menus/option.go` | `type th10/scene/menus.Option`、`func th10/scene/menus.NewOption`、`func *th10/scene/menus.Option.SceneMode`、`func *th10/scene/menus.Option.SceneEnter`、`func *th10/scene/menus.Option.SceneLeave`、`func *th10/scene/menus.Option.Update`、`func *th10/scene/menus.Option.Draw`、`func *th10/scene/menus.Option.DrawPorted` |
| `scene/menus/replay.go` | `func *th10/scene/menus.Replay.Slots`、`func th10/scene/menus.replayEntryLabel` |
| `scene/select/select.go` | `type th10/scene/select.Select`、`func th10/scene/select.New`、`func th10/scene/select.NewExtra`、`func th10/scene/select.NewPractice`、`func *th10/scene/select.Select.SceneMode`、`func *th10/scene/select.Select.SceneEnter`、`func *th10/scene/select.Select.SceneLeave`、`func *th10/scene/select.Select.Update`、`func *th10/scene/select.Select.Draw`、`func *th10/scene/select.Select.DrawPorted`、`func th10/scene/select.NewDebugStage` |
| `scene/stage/ecl/disasm.go` | `func *th10/scene/stage/ecl.Program.Disassemble`、`func *th10/scene/stage/ecl.Sub.Disassemble`、`func th10/scene/stage/ecl.Instr.argString`、`func th10/scene/stage/ecl.Param.String`、`func th10/scene/stage/ecl.quoteList`、`func th10/scene/stage/ecl.varName`、`func th10/scene/stage/ecl.rankString`、`func th10/scene/stage/ecl.opName` |
| `scene/stage/ecl/game_ops.go` | `func th10/scene/stage/ecl.eclUnset` |
| `scene/stage/ecl/motion.go` | `func th10/scene/stage/ecl.ease` |
| `scene/stage/ecl/opcodes.go` | `const th10/scene/stage/ecl.opSetChapter`、`const th10/scene/stage/ecl.opStars` |
| `scene/stage/ecl/shake.go` | `func *th10/scene/stage/ecl.ScreenEffect.SetPhase`、`func *th10/scene/stage/ecl.ScreenEffect.SetTimer` |
| `scene/stage/ecl/vm.go` | `func *th10/scene/stage/ecl.VM.BossActive`、`func *th10/scene/stage/ecl.VM.aimAngleAt` |
| `scene/stage/gui_boss.go` | `func *th10/scene/stage.GUI.SetBossSection` |
| `scene/stage/gui_spell.go` | `func *th10/scene/stage.GUI.BeginSpellPresentation` |
| `scene/stage/playback.go` | `func *th10/scene/stage.Playback.Stage`、`func *th10/scene/stage.Playback.Chapter`、`func *th10/scene/stage.Playback.UpdateHeadless`、`func *th10/scene/stage.Playback.DrawPorted` |
| `scene/stage/scene_mode.go` | `func *th10/scene/stage.Stage.SceneFlags` |
| `scene/stage/spell.go` | `func *th10/scene/stage.SpellCard.Tick` |
| `scene/title/title.go` | `func *th10/scene/title.Title.SetPageHooks`、`func *th10/scene/title.Title.DrawPorted` |
| `scene/ui/menu.go` | `type th10/scene/ui.MenuItem`、`type th10/scene/ui.MenuStyle`、`type th10/scene/ui.MenuList`、`func *th10/scene/ui.MenuList.Update`、`func *th10/scene/ui.MenuList.move`、`func *th10/scene/ui.MenuList.Draw`、`func *th10/scene/ui.MenuList.colorFor`、`func th10/scene/ui.drawOutlinedText` |
| `scene/ui/title_controller.go` | `func th10/scene/ui.RegisterTitlePage`、`func *th10/scene/ui.TitleController.RestartChild` |
| `scene/ui/title_pages.go` | `func *th10/scene/ui.TitleController.BeginExtra` |
| `scene/ui/ui.go` | `func th10/scene/ui.NewPanel`、`func th10/scene/ui.Button`、`type th10/scene/ui.Scene`、`func *th10/scene/ui.Scene.SetCursor`、`func *th10/scene/ui.Scene.Cursor`、`func *th10/scene/ui.Scene.Update`、`func *th10/scene/ui.Scene.activateFocused`、`func *th10/scene/ui.Scene.Draw` |
| `sht/shot.go` | `func *th10/sht.File.ValidateReimuA` |
| `sprite/anm.go` | `type th10/sprite.AnmSheet`、`func th10/sprite.LoadAnmSheet`、`func th10/sprite.NewAnmSheet`、`func *th10/sprite.AnmSheet.Sprite`、`func *th10/sprite.AnmSheet.ScriptSprite` |
| `sprite/loaders.go` | `func *th10/sprite.BulletSprite.Frame` |
| `sprite/sheet.go` | `func *th10/sprite.Sheet.Image` |
| `sprite/vm.go` | `func th10/sprite.SheetFor`、`func th10/sprite.DrawVM` |
| `std/project.go` | `func *th10/std.Camera.Matrices` |

## 8．执行后的核对

A–E 已按本文顺序实施，逐阶段全量 build／vet／test／WASM／gofmt／完整 smoke 均通过。六章每章五次 cmp 均逐字节相同，motion 一致帧数始终为 15853／17733／29763／29469／39343／40099；生成的 trace 已删除。

确认删除的 6 个整文件共 925 行，包含局部助手／导入／两项专用旧测试的纯删除为 1302 行。七个关卡包都通过“无函数体、函数字面量、调用表达式或非别名类型”的声明级配置测试；15 个原背景函数体逐文本不变，加载顺序／旧滚动和安全 cap 保留。原 `_test.go` 的 980 个保留函数源码不变，只有本文授权的两项删除，新增 9 项测试。

共用根操作、ANM-91 补发、Fade 与小字库位于 `scene/ui/presentation.go`／`scene/ui/fade.go`；两个场景保留自己的 World、文件选择、提交状态和生命周期。原本不同的 MSG 与 Music 四帧补间继续独立。t10 的 ANM-91 no-op 未补造，t11 opt-in 的出生 pass 不变。八组演出的状态、备用 RNG、设备调用、绘制顶点摘要在 D 前、D 后、最终 E 后相同。

`.golangci.yml` 已加入，唯一常规命令 `golangci-lint run`，包含测试、标准 linter＋unused、无同文／同一行／linter 截断。版本 2.15.0，配置验证通过；统一总数 52→46，errcheck 44→42、staticcheck 4→2、unused 4→2，govet／ineffassign 均为 0。t16 范围为 0，46 项范围外按包与任务归属列在交付第 4 节；没有新增 nolint、排除规则、依赖或批量 modernize，不称全仓 lint 全绿。

`chain_legacy.go`、60000 帧安全结束、未绘制图片的加载、原 Enemy.Update、t10 缺失补发及持久生命周期等边界仍按本文第 3–5 节保留，并在交付第 6 节给出原函数／无原地址的理由。没有将移动近似误报为移植，也没有改 `tools/`／`trace/`／资源／音频／字体／浏览器存储或 golden。完整文件、测试、函数地址、接口和验收记录见 `docs/handoff/codex_t16.md`。
