# t8：模式、存档与控制器

## 1．摘要

工作树：`D:/GoProjects/th10/.tmp/codex/t8/tree`。A → E 均完成，各阶段及最终 rules 第 6 节验收全绿。只在本任务工作树修改源码，主树只写本交接副本与任务自己的日志／脚本；未运行原作、probe、转换器或 Ghidra，未修改 golden。交付前发现正确文件名下已有真实存档，17,655 B 全文件与类型视图字节往返已验证；由原作读回复刻文件与最终震屏／结局视觉仍需用户验证或 t7 接线。

| 阶段 | 状态 |
|---|---|
| A．备用 RNG 种子策略 | 完成，完整验收全绿，六章 trace 字节相同 |
| B．模式字段 | 完成，完整验收全绿，六章 trace 字节相同 |
| C．score.dat | 完成，容器／历史／排行／练习字段持久化，完整验收全绿 |
| D．extend 与高分 | 完成，滚动分数驱动资源更新，完整验收全绿 |
| E．通关与最终场景 | 完成，原奖金／GUI 120 帧场景请求，完整验收全绿 |

## 2．改动文件清单

### 新增

- `docs/handoff/codex_t8.md`：阶段与最终交接记录。
- `game/modes.go`：原模式位、练习入口与跨关状态消费。
- `game/modes_test.go`：原练习／重开字段与非 1 历史模式测试。
- `game/native_score.go`：原 score 头、CR／ST、初始化、校验、压缩／加密及字节保留。
- `game/native_score_progress.go`：原计数、练习八字节记录、场景与最终通关标志。
- `game/native_score_test.go`：原默认值、crypto 向量、边界、历史／排行原字节，以及真实 17,655 B 文件／类型视图往返测试。
- `game/persistence_score.go`：共用存储接口、内存后端、视图与一次 JSON 导入。
- `game/persistence_progress.go`：从两平台重复代码统一进度流程与原计数接线。
- `game/persistence_score_test.go`：迁移优先级、内存不别名与损坏输入测试。
- `game/persistence_score_desktop_test.go`：用户目录、原字节与原子写失败测试。
- `game/score_controller.go`：原高分资源、自然 extend 门槛／游标／上限。
- `game/score_controller_test.go`：各难度门槛、每帧一次、上限、门控与排行行高分测试。
- `game/clear_bonus.go`：MSG 最终加分的 int32 调用顺序。
- `game/clear_bonus_test.go`：正常／Extra／练习奖金与每次加分上限测试。
- `replay/container_codec.go`：复用 t5 原 LZSS／XOR 的共用包装，没有另写近似压缩器。
- `scene/stage/controller.go`：GUI 控制字段、MSG／场景请求、计分调用点与外部震屏 hook。
- `scene/stage/controller_test.go`：真实滚动加残帧、音效／通知／数字同帧写入测试。
- `scene/stage/gui_modes_test.go`：GUI 原模式门控，不用 SingleStage／HasSeed 替代。
- `scene/stage/clear_controller_test.go`：原通关路由、119／120 帧、继续接口与存档隔离测试。

### 修改

- `game/game.go`：应用 RNG、共享历史／进度绑定、系统请求字与新局入口。
- `game/state.go`：原模式／高分／游标字段，备用 RNG 改为引用，实际 Stage 计分分离。
- `game/alternate_rng_test.go`：授权更新为应用隔离与同应用跨局保持。
- `game/persistence.go`：桌面原 score 字节安全读写，删除旧进度 JSON 写出。
- `game/persistence_wasm.go`：原页面会话后端保存同样的 score 字节。
- `game/persistence_data.go`：原记录 Raw 与原存档视图，24 B 搬移保留 padding／NUL 尾部。
- `game/replay_progress_test.go`：仅把 bool 回放赋值适配为原 mode 1。
- `game/restart.go`：原位驱动 rank／练习资源／fresh-run 与 mode 13 重开。
- `game/spell_history.go`：历史只按 int32 mode 是否等于 1 门控。
- `game/spell_history_test.go`：仅做回放字段类型适配，断言与期望不变。
- `replay/config.go`：正常配置路径携带原 fresh-run 位。
- `scene/select/select.go`：应用级状态创建、原练习入口、跨关消费。
- `scene/stage/stage.go`：绑定共享状态、原控制器调用、MSG 清场／请求与冻结尾部，删除旧 cleared 字段。
- `scene/stage/bullets.go`：Context 引用应用 RNG，并保留 logic-only GUI 控制字段。
- `scene/stage/alternate_rng_test.go`：授权更新跨局／跨关与主 RNG 隔离前提。
- `scene/stage/gui.go`：只改控制器有关的初始化、计时、继续数字与门控，没有重排符卡函数。
- `scene/stage/gui_msg.go`：原练习分支不显示普通 Stage Clear banner。
- `scene/stage/menu.go`：原练习排行／分数／回放门控与实际分数提交。
- `scene/stage/menu_draw.go`：回放 metadata 用原练习位，不猜 SingleStage。
- `scene/stage/menu_test.go`：只补原练习位初始化／退出初始化，旧断言不变。
- `scene/stage/playback.go`：先设原 mode 再绑定，避免绑定本身修改存档。
- `scene/stage/playback_test.go`：仅做回放字段类型适配。
- `scene/stage/record.go`：原 fresh-run 与 mode 字段接快照／恢复／录制。
- `scene/stage/record_test.go`：仅做回放字段类型适配。
- `scene/stage/spell_test.go`：仅把 bool 赋值改为 mode 1，没有改符卡断言。
- `sim/sim.go`：固定种子应用、构造前备用种子注入及原回放模式。
- `sim/alternate_rng_test.go`：授权重写为 newHeadless／ReplayConfig／RestoreReplayChapter 路径。

### 删除

删除文件：无。删除旧 `Context.cleared` 字段与不再使用的 `savePath` helper。
## 3．移植的函数地址

| 原作 | Go |
|---|---|
| `FUN_004201b0`，`0x4201d0…0x4201e1` | `game.New` 的两个 RNG 启动种子、`Game.AlternateRNG` |
| `FUN_00417870`／`FUN_00428f60`／`FUN_0042a6a0` 对备用 RNG 不写入 | `Game.BindState`、`Game.NewState`、`GameState.AlternateRNG` |
| `FUN_0042d420`，`0x42d7ce`；`FUN_00417870`，`0x41794f…0x4179e2` | `GameState.Practice`、`practiceLife`、原 bit 8 的 rank 初始化 |
| `FUN_004180e0`，`0x41811d…0x418121`；`FUN_004218d0`，`0x421a03…0x421ae8` | `StageMode`、练习／回放／跨关／重开入口 |
| `FUN_00413bc0`，`0x413e4b`、`0x414044…0x414068`、`0x414113` | GUI 标题 demo 位、回收线提示、fresh-run 提示 |
| `FUN_00428f60`；`FUN_00409280`／`FUN_00409c00` 的 `mode != 1` | int32 `ReplayMode` 与历史／进度／录制门控 |
| `FUN_0042ae60`／`FUN_0042acb0`／`FUN_0042add0` | `DefaultScoreFile`、`scoreStartupSeed`、原 CR/ST 初始化 |
| `FUN_0042b030`／`FUN_0042b1e0`／`FUN_0042ac90` | `DecodeScoreFile`／`EncodeScoreFile`／`scoreChecksum` |
| `FUN_004359b0`／`FUN_00435dc0`／`FUN_0044b220`／`FUN_0044b0d0` | t5 原 codec 的共用导出包装，score 使用增量 0x35／block 0x10／初值 0xac |
| `FUN_00417870`／`FUN_00418190`；`FUN_004236f0` case 7 | `BeginRun` 的原计数、`RecordPlayFrame`、`RecordPracticeScore` |
| `FUN_00417040`，`0x417118`、`0x417288…0x4172d5` | `GameState.UpdateScoreController`、`Stage.updateScoreController` |
| `FUN_004188a0`，`0x4188a0…0x418926` | natural extend 的残机上限、SE 0x2c、同帧 front 0x4b 通知与 `GUI.SetLives` |
| `FUN_00417870`／`FUN_00421f60` | 当前排行行的高分／继续次数初始化、菜单提交实际分数 |
| `FUN_00415e90` case 0x14，`0x416c08…0x416e8a` | `Context.clearStage`、`GameState.AwardClearBonus`、原练习标志／最终通关计数 |
| `FUN_00414900`，`0x414906…0x414937` | `GUIController.TickClear`，GUI＋0x9ecc 的 120 帧场景请求 |
| `FUN_0040ac90`／`FUN_004218d0` 的请求接口 | `GUIController.request`／`Stage.deliverSceneRequest`，仍通过 `Game.GoTo` 与 t5 callbacks |

## 4．测试

阶段 A 的相关测试实际通过：`go test -count=1 ./game ./sim ./scene/stage -run AlternateRNG`。`sim` 耗时 50.652 秒，逐章校验完整编码 trace 与主 RNG state／count；注入发生在 STD／ANM 初始化之前，章节恢复与回放配置走正常路径。

完整验收脚本：`.tmp/codex/t8/accept.ps1`。阶段输出保存在 `.tmp/codex/t8/checks_A.log`、`trace_A.log`、`results_A.json`。阶段 A：build／vet／`go test -count=1 ./...`／WASM／完整 smoke 全部 exit 0；`gofmt -l .` 输出为空。完整 smoke 耗时 126.590 秒。

阶段 B：新增练习／重开字段、历史非回放 mode 值、GUI 五类原门控测试。build／vet／全仓 test／WASM／完整 smoke 全部 exit 0，gofmt 空；完整 smoke 137.434 秒。记录在 `checks_B.log`、`trace_B.log`、`results_B.json`。

阶段 C：新增 9 项测试，覆盖完整文件与类型视图逐字节往返、原 XOR 手算向量／unsigned checksum、原默认排行与符卡保留字表、512 个单步 ST 随机字、独立坏节跳过／坏版本／越界长度／越界机体、跨局／重载符卡历史、24 B 排行搬移保留 NUL 尾部与 padding、练习记录／帧计数饱和、JSON 一次迁移、内存不别名、桌面原子写失败与用户目录。相关测试首次发现回放入口绑定空存档也会修改历史指针，已修复为先设置 mode 再绑定；原 `TestPlaybackStoredReplayRestoresEachChapterWithoutProgress` 重新通过，未改其断言。最终 build／vet／全仓 test／WASM／完整 smoke 全部 exit 0，gofmt 空；完整 smoke 123.107 秒。日志为 `checks_C.log`、`trace_C.log`、`results_C.json`。

阶段 D：新增 4 项测试，覆盖全部难度门槛、每帧单次 extend、上限不通知但游标推进、原最终通关门控、非法回放游标、防止 aggregate 高分替代机体排行行，以及真实 Stage 的第 31／32 帧滚动门槛／音效／通知／数字／残机同帧写入。保留 `TestGUIScoreRollIsVisualOnly` 与直接 `Numbers` 不加残校验。build／vet／全仓 test／WASM／完整 smoke 全部 exit 0，gofmt 空；完整 smoke 116.716 秒。日志为 `checks_D.log`、`trace_D.log`、`results_D.json`。

阶段 E：新增 4 项测试，覆盖全部正常／Extra／练习奖金与每次 AddPoints 上限、GUI 119／120 帧真实 Scene callback、通关计数不重复、最终加分不 extend、t5 post-ending menu、练习立即菜单与帧尾冻结、Extra 立即菜单／计数持久化、回放立即退出且存档字节不变、普通关间请求 0xb 与系统 0x1000 请求覆盖。build／vet／全仓 test／WASM／完整 smoke 全部 exit 0，gofmt 空；完整 smoke 120.127 秒。日志为 `checks_E.log`、`trace_E.log`、`results_E.json`。额外 lint 首次有缓存写入沙盒外被拒绝与 36 条报告，随后已用任务内缓存重跑并修掉本任务引入项；按 rules 第 6 节 lint 不是阶段验收项。

收尾轮 `checks_final.log`／`results_final.json`：`go build ./...`、`go vet ./...`、`go test -count=1 ./...`、WASM build、完整 smoke 全部 exit 0，`gofmt -l .` 输出为空；full smoke 为 116.466 秒。本任务的新 lint 报告已修，使用任务内 `lintcache` 重跑 `golangci-lint run`：exit 1，默认上限显示 33 条（errcheck 26／staticcheck 2／unused 5）。只读 `.tmp/codex/base`、`GOFLAGS=-mod=readonly` 与任务缓存下实际跑 baseline lint，首次得到相同 33 条位置与内容，见 `lint_base.log`。默认 max-same-issues 会改变截取位置，故另关闭报告上限实跑两树：都为 51 条（errcheck 44／staticcheck 2／unused 5），去除因新增行导致的行号差异后完整报告无差异，见 `lint_full_base.log`、`lint_full_tree.log`、`lint_full_comparison.txt`。本任务没有新增 lint 报告，未修改无关代码替基线擦尾；rules 第 6 节不把 lint 列为验收门槛。

交付前新增真实文件测试实际通过：`go test -count=1 -v ./game -run TestOriginalScoreFileRoundtrip`，输出 `original scoreth10.dat: 17655 bytes identical, all CR/ST sections accepted`，无 skip。原文件 Decode→Encode 与 saveView→nativeBytes 全字节相同。新增测试后的实际最后一轮 `checks_delivered.log`／`results_delivered.json`：build／vet／全仓 test／WASM／full smoke 全部 exit 0，gofmt 空，full smoke＝131.690 秒；默认 lint 仍为同类 33 条基线报告，无新增。七轮规定验收均通过。

## 5．Trace 结果

阶段 A 六章与 `base_cN` 的二进制 cmp 全部为 0。字节数为 94186573／144171327／271969413／319443819／430006225／510140938；motion 一致帧数为 15853／17733／29763／29469／39343／40099，无下降。逐章生成、比较后只删除本任务输出，保留日志，不保留大 trace。

阶段 B 六章 cmp 全部为 0，字节数与阶段 A 相同；motion 覆盖仍为 15853／17733／29763／29469／39343／40099。GUI 原模式修正只改变演出，不改变逻辑。

阶段 C 六章 cmp 全部为 0，字节数与阶段 A 相同；motion 覆盖仍为 15853／17733／29763／29469／39343／40099。用户存档不进入 sim 的固定初始状态。

阶段 D 六章 cmp 仍全部为 0，没有第一分歧帧；motion 覆盖仍为 15853／17733／29763／29469／39343／40099。新增资源分支没有改变本验收回放的 trace；32 帧滚动门槛与全部独立 extend 边界由新增测试实际覆盖，不靠更新 golden。

阶段 E：c0～c4 cmp＝0；c5 cmp＝1，首个且位于该章最后一帧的分歧为 **50288**，score 从基线 1655940 变为 9055940。原作依据：MSG 0x14 的 `0x416c69…0x416cc0`，Easy、life＝0、power＝120、faith＝50000，对应项目加分 `(50000/10)*1000 + 0*20000000 + (120/5)*100000 = 7400000`。其他状态与运动不变；六章 motion 仍为 15853／17733／29763／29469／39343／40099，原作六面采样到 45000，最终加分帧在采样之外。trace 大文件均已按规则删除。

最后一轮再次得到相同结果，`trace_final.log` 保留完整 diff：c0～c4 全文件 cmp＝0，c5 只有最后帧 50288 的 score＋7400000，输出总字节数与 A 相同。六章 motion 一致帧数为 **15853／17733／29763／29469／39343／40099**，没有下降。所有任务生成的 cN 大文件已删；原作／基线／golden 未动。

增加真实存档测试后又完整跑七轮中的最终 `delivered`：`trace_delivered.log` 与上一轮结论相同，c0～c4 cmp＝0，c5 仅 50288 帧增加 7400000；motion 为 15853／17733／29763／29469／39343／40099，无下降。

## 6．待决与推测

- 全阶段实现与规定验收已通过；额外 lint 的默认 33 条／完整 51 条为已实际复现的同类基线报告，无新增，不在 rules 第 6 节验收门槛中。
- 启动用可跨平台的 `time.Now().UnixMilli()`，遵循架构师定下的墙钟毫秒策略；它不是 Windows 特有的 `timeGetTime` 系统计时来源。无头应用与测试显式使用种子 0。
- 独立 `game.NewState` 保持旧主 RNG 默认路径，备用 RNG 来自一个零种子的无头应用；交互入口使用 `Game.NewState`，初始主 RNG 来自启动种子，回放随后覆盖主 RNG。
- 原字段取证勘误：0x10 是关卡练习；0x20 是标题待机 demo，不是练习。bit 8 的写者是后续关卡重开／续关路径，它也决定新局 rank＝−512；当前没有单独 Boss 练习 UI，原位已供入口使用，不用普通练习模拟 bit 8。GUI 原 `0x66+难度` 提示读 `DAT_00491fc4`，不是练习位。
- `DAT_00474ca0` 全部直接引用已核对：bit 0 复用关卡管理器，bit 1 保留跨关回放管理器，bit 2 新高分，bit 3 后续关卡重开／续关，bit 4 关卡练习，bit 5 待机 demo；bit 6 只在进入 demo 时被清除，未找到其设置者或读者。未给 bit 6 虚构含义。复用位在 Go 现有 Scene 生命周期中表达／消费，不越界重写整个原监督者与管理器生命周期。
- 指定 dump 的地址／文件名有误：`FUN_00420870` 是 cfg 读取与校验；`FUN_00439ff0` 是启动互斥锁与快捷方式检查。原 score 容器实际由 `FUN_0042b030`／`FUN_0042b1e0` 读写，PE 字符串 `0x46e848` 为 `scoreth10.dat`。复刻按任务要求在现有用户目录写 `score.dat`，字节格式相同。依据为完整 native asm，摘录在 `.tmp/codex/t8/evidence/score_native.asm`；未请求／运行 Ghidra。
- 容器头 0x18 B：`TH10`、＋4 文件长度、＋8 u16 版本 3、＋0xc 默认 0x100、＋0x10 压缩长度、＋0x14 解压长度。有效负载为 7×0x437c 的 CR 与 0x448 的 ST（0x1dcac B）。节头＋0 tag／＋2 version 0／＋4 unsigned 字节和／＋8 节长；CR＋0xc 为机体索引。校验从节＋8 到节尾，不是全文件 checksum。
- 所有 CR／ST 原字节保留，包括第七汇总块、排行 24 B 的 padding／名字 NUL 后尾部、符卡保留字、音乐解锁字节、ST 512 个随机字。原初始化会消耗 512 个主 RNG 单步，新应用使用启动种子生成默认 ST，主 RNG 的新局初值包括这一步；备用 RNG 不受影响。
- 回放的原 MSG 0x14 无条件写练习场景标志；为保留 t5“回放不改存档”的接口，回放绑定原存档的私有进度视图，字段仍按原作变动，但不写回共享存档。符卡历史仍共享，只由原 `mode !=1` 门控禁止回放写入。
- 迁移是项目侧映射：旧 JSON 全局统计分配给其最后选择的机体／难度；不能精确恢复旧格式没有的各机体明细。游戏结束总数／上次机体选择没有已确认的原存储槽，只保留会话数据，不编造节或占用原保留字。WASM 仍沿用原后端的页面会话内存，不声称刷新后持久。
- 文件名勘误进一步确认：原目录虽没有 score.dat，却已有 `scoreth10.dat`（17,655 B）。已只读验证这份真实文件的全文件与类型视图往返相同，全部节通过校验；SHA-256＝`5A2A4C3F934FB7BAE22529EA8F2BD76C66DD21C98A941758D035B85F528B245D`。未运行同目录的 ScoreConverter.exe 或原作。
- 仍请用户提供受控的 TH10 v1.00a 原生 `scoreth10.dat`：一份新生成的默认存档，以及一份注明机体／难度／练习面、排行名字与符卡尝试／收取次数的已使用存档，最好含 Normal＋／Extra 通关，以校核字段语义；还需由用户操作原作读回复刻生成文件。本任务已有一份真实字节验证，不把它误说成双向运行时互通已验证。
- 高分资源写入／extend 只在 stage 更新链的 `FUN_00418190 → FUN_00417040` 处发生，不在绘制或 `GUI.Numbers` 中发生。普通门槛为存储分数 2000000／4000000／8000000／15000000，Extra 为 3000000／10000000，最后 1000000000 是永不可达哨兵，不是重复加残。门槛游标就是 `ReplayState9C`（章头＋0x1b8），继续次数为 `ReplayState90`（＋0x1b4）。自然 extend 不加 rank，残机上限 9 的尝试不发声音／通知。
- 项目兼容边界：没有 Stage 控制器的独立 GameState／logic-only 测试仍保留旧即时 HiScore API；所有实际 Stage 在构造前 `BindScoreController`，`AddPoints` 不再绕过原滚动高分更新。这样既有直接符卡／道具测试的断言未被削弱，实际游戏／sim／回放路径仍由原控制器更新。
- 最终通关的 GUI 0x20 在奖金前置位；普通六面非回放再置 0x10／计时 0，第 120 个后续 GUI calc 请求 scene 14，系统 `DAT_00491ff4 & 0x1000` 时请求 2。Extra／练习走原直接 clear menu，回放最终请求 4，不等 120 帧。关间 0xb 的实际 stage 递增仍放在现有 Scene callback 中，保留旧无头 trace 的通关帧终止边界，不运行过场尾部。
- 正常六面原 `FUN_0043c8b0` 震屏调用不属于本任务：已留 `Context.RequestShake`，原调用点 `0x416d85`，参数 5／120／0／0／0，调用前 EBX＝0x31。t7 接其构造／生命周期／采样；本树未主动接入额外主 RNG 消耗。ending／Staff 视觉未改。

## 7．建议的 `docs/fidelity.md` 改动

未直接修改主账。建议采用以下完整状态行（只调整本任务范围）：

```text
| ZUN RNG | 主 RNG 回放取种；备用 RNG 应用级种子策略已接线 | FUN_004201b0 0x4201d0…0x4201e1：启动毫秒低 16 位；Game 持有 DAT_004918a8，新局／跨关／回放不重置，sim 注入 0；原 ST 默认初始化另消耗 512 个主 RNG 单步 | 三份授权 RNG 测试，正常 ReplayConfig／RestoreReplayChapter 路径的六章种子隔离，完整逻辑／主 RNG 相同 |
| 动态 rank | 位精确（原模式入口已接线） | DAT_00474c98；FUN_00417870 只由 DAT_00474ca0 bit 8 决定新局 0／−512，普通关卡练习不误用；跨关与回放不重置，t4 原变化点保留 | TestNativePracticeAndRestartModes、原 rank 测试、六章 motion 无下降 |
| 存档/配置 | 原格式移植；真实存档字节已验证，双向运行时互通待用户 | FUN_0042b030/0042b1e0：TH10 v3、CR/ST unsigned checksum、0x35/0x10/0xac XOR、原树形 LZSS；24 B 排行／符卡 0x90 B 历史／练习八字节记录／原计数与标志；桌面 score.dat、WASM 会话同字节、旧 JSON 一次导入；cfg 与回放 t5 接口保留 | 原字节与类型视图往返、crypto/default/bounds/atomic/migration/共享历史测试、WASM build；真实 scoreth10.dat 17655 B／类型视图字节相同；需受控存档与原作读回验证 |
| 控制器/模式/结算 | 已移植本范围字段与计分／通关请求；完整监督者与 ending 视觉未做 | DAT_00474ca0、stage+0x5c、replay manager+0x10、fresh-run；FUN_00417040 按滚动分数更新高分／继续数字／自然 extend；MSG 0x14 原奖金，GUI+0x9eb4 bit 0x10/0x20、+0x9ecc 120 帧请求 14/2，练习/Extra 菜单与回放请求 4；仍走 Game.Scene、ending.After、OnContinue | 31/32帧加残与119/120帧场景测试，全仓/WASM/full smoke全绿；c0-c4 cmp相同，c5只在50288帧加7400000，motion覆盖15853/17733/29763/29469/39343/40099不降 |
```

按现有格式添加更新记录，日期为实际交付日期：

```text
- 2026-11-08 — t8 A–C：备用 RNG 改为 Game 应用生命周期墙钟低 16 位，sim 注入 0；原 RunFlags/stage mode/replay manager mode/fresh-run 接选关、练习、回放与续关，纠正 0x20 是 demo、0x10 才是练习。移植实际 FUN_0042b030/0042b1e0 的 scoreth10.dat 容器（任务后端名 score.dat），原 CR/ST 头／unsigned checksum／0x35/0x10/0xac XOR／树形 LZSS，共享符卡历史、原排行字节与练习／计数标志，旧 JSON 一次映射，WASM 同字节会话后端；真实原文件17655B／类型视图往返相同；更多受控存档与双向运行时互通待用户。A–C 六章 trace 与基线字节相同，原 cfg／回放接口保留。
- 2026-11-08 — t8 D–E：FUN_00417040 高分／继续数字／自然 extend 从 GUI 视图分离到原更新调用点，普通20/40/80/150百万显示分、Extra30/100百万，每帧一次，残机上限不通知但推进游标，不加 rank。MSG 0x14 按原 int32 次序加最终奖金，GUI 0x10/0x20 与 120 帧 scene request 替换旧 Context.cleared；练习/Extra直接菜单、回放最终返回，保留 ending.After/OnContinue。最终震屏留 t7 hook，不新增主 RNG 消耗。D六章cmp相同；E及最终仅六面50288最后一帧增加7400000，motion覆盖不降。七轮build/vet/test/WASM/full smoke全绿，gofmt空；额外lint默认33条／完整51条为与只读基线同类的遗留报告，无本任务报告，golden未改。
```
## 8．需要用户目视确认的项目

在合并树用 PowerShell 设置独立用户目录后运行，例如 `$env:TH10_DATA_DIR='D:\GoProjects\th10\.tmp\codex\t8\manual-data'; go run ./cmd`；不要写只读原资源目录。

1．第一次从旧 save.json 导入后出现 score.dat，之后只写原进度字节，重启后排行／名字／符卡尝试与收取保持；config.json／原 cfg／原回放接口不变。
2．练习后续关卡初始 9 残机、原 Power／rank；首关回收线提示读 mode／继续数，不因主 RNG 有种子隐藏。多次新局的备用演出不应每局重置。
3．得分跨普通 2000／4000／8000／15000 万、Extra 3000／10000 万门槛，检查滚动到门槛时的 SE／One Up／残机；上限 9 不重复通知，继续数字属于其高分。
4．普通六面 MSG 奖金入账、不因最终奖金再 extend；120 个后续 GUI 帧进入既有 ending，结束后同一个排行／回放菜单可用。练习／Extra 直接菜单，回放结束返回且存档不变。
5．WASM 启动墙钟／跨局共享与页面会话存档需要浏览器确认；刷新后持久化不是既有后端能力，本任务不声称实现 localStorage。
6．由用户操作原作验证原 scoreth10.dat 互通；字体／截图／Staff／最终震屏视觉仍分别属于既有 GUI 适配与 t7 接线范围。
## 9．与其他任务的接口

- `func (g *game.Game) AlternateRNG() *zun.RNG`：应用持有的 `DAT_004918a8`，所有备用 ANM 使用同一个引用，不能从主 RNG 抽数派生种子。
- `func (g *game.Game) BindState(*game.GameState)`：在关卡所有资源初始化之前绑定共享状态；交互 `stage.New` 自动调用。
- `func (g *game.Game) NewState(character, shotType, difficulty int) *game.GameState`：交互新局入口；独立 `game.NewState` 是确定性测试入口。
- `GameState.AlternateRNG()` 保留原签名，t6／t7 不需要修改现有 ANM 接线。
- `GameState.RunFlags uint32`、`StageMode int32`、`ReplayMode int32`、`FreshRun bool`、`PracticeLives int`：分别是原 `DAT_00474ca0`、stage＋0x5c、replay manager＋0x10、`DAT_00491fc4`、`DAT_00474cac`。普通／练习为 stage mode 0、manager mode 0，回放／sim 为 1／1；manager mode 2 只检查文件，不建活动世界。历史与进度必须保留 `!=1`，不能改成只接受 0。
- `GameState.Practice(stage int)` 设置原练习位与资源默认；`BeginStageTransition()` 消费原重用位并清 fresh-run；重开仍用 t5 `RestartStage`，按原 bit 8 调 `ResetRank`。
- `ScoreStore` 的 `Read() ([]byte,error)`／`Write([]byte) error`／`LegacyJSON() ([]byte,error)`；`Game.LoadProgress()` 用于注入后端，`Game.SaveProgress()` 只写原容器。
- `EncodeScoreFile(*ScoreFile) ([]byte,error)`／`DecodeScoreFile([]byte) (*ScoreFile,error)` 是纯 codec。`Game.SpellHistory` 为唯一跨局对象，`GameState.SpellHistory` 绑定它，t2 的三个历史 API 与 t7 调用签名不变。
- `GameState.RecordStageClear()`／`RecordFinalClear()` 的 MSG 调用点已接线；`RecordPlayFrame()` 在原 `FUN_00418190` 位置调用，`Game.RecordPracticeScore(*GameState)` 已接原菜单 case 7。其他任务不得重复计数；排行仍用 t5 `Game.InsertRanking`，新增 Raw 以保留其六字搬移。
- `GameState.UpdateScoreController(shown int32, noExtend bool) bool`：消费 GUI＋0x9e78 的滚动分数，写高分／继续次数字段与自然 extend，返回是否需要 SE／通知；stage 原调用点负责 `GUI.SetLives`／`ShowNotice(4,0)`。`GUIController.Flags` 的 0x20 已由 MSG 最终通关分支写入；不修改 t6 的自机和 t7 的符卡流程。
- `GameState.AwardClearBonus()`、`GUIController.TickClear(systemFlags uint32)`、`GUIController.SceneRequest int32`：MSG 与 GUI calc 已接线，Scene 层只递送请求。`Context.RequestShake func(kind,frames,x,y,z int32)` 留给 t7，调用点与隐藏 EBX 值见待决。`ending.Ending.After`／`Stage.OnContinue`／`Stage.OpenCompletion` 均保留原签名。
- 本阶段没有改符卡或自机逻辑。`ending.Ending.After`、`Stage.OnContinue` 与 t5 录制／播放接口保留。




